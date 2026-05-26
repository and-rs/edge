package signals

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/index/edge/backend/internal/config"
)

var (
	errAIConfig          = errors.New("ai config error")
	errAIInvalidResponse = errors.New("ai invalid response")
)

type SignalJudge interface {
	JudgeSignals(ctx context.Context, inputs []aiSignalInput) (map[int]aiSignalOutput, error)
}

type openAIJudge struct {
	client       *http.Client
	baseURL      string
	apiKey       string
	organization string
	project      string
	model        string
}

type aiSignalInput struct {
	Index           int      `json:"index" jsonschema_description:"candidate index"`
	Headline        string   `json:"headline" jsonschema_description:"news headline"`
	Source          string   `json:"source" jsonschema_description:"source publication"`
	PublishedAt     string   `json:"published_at" jsonschema_description:"publication time in RFC3339"`
	MarketQuestion  string   `json:"market_question" jsonschema_description:"best matched market question"`
	MarketStatus    string   `json:"market_status" jsonschema_description:"matched market status"`
	MarketVenue     string   `json:"market_venue" jsonschema_description:"matched market venue"`
	MarketVolume24h float64  `json:"market_volume_24h" jsonschema_description:"matched market 24h volume"`
	MatchedKeywords []string `json:"matched_keywords" jsonschema_description:"overlapping keywords between event and market"`
	BaseScore       float64  `json:"base_score" jsonschema_description:"rule based pre AI score"`
	BaseReason      string   `json:"base_reason" jsonschema_description:"rule based reason string"`
}

type aiSignalOutput struct {
	Index        int     `json:"index" jsonschema_description:"candidate index"`
	Thesis       string  `json:"thesis" jsonschema_description:"short actionable thesis"`
	WhyItMatters string  `json:"why_it_matters" jsonschema_description:"compact reason for operator attention"`
	MatchType    string  `json:"match_type" jsonschema:"enum=market-linked,enum=watchlist,enum=no-match" jsonschema_description:"quality of market linkage"`
	ScoreBoost   float64 `json:"score_boost" jsonschema_description:"score adjustment between -5 and 12"`
}

type aiSignalResponse struct {
	Signals []aiSignalOutput `json:"signals" jsonschema_description:"judged candidate signals"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func NewSignalJudge(cfg config.AIConfig) (SignalJudge, error) {
	if cfg.AuthMode == "disabled" {
		return nil, nil
	}
	if cfg.AuthMode != "api-key" {
		return nil, aiConfigErrorf("unsupported EDGE_AI_AUTH_MODE: %s", cfg.AuthMode)
	}
	if cfg.Model == "" {
		return nil, aiConfigErrorf("EDGE_AI_MODEL missing for api-key mode")
	}
	if cfg.APIKey == "" {
		return nil, aiConfigErrorf("EDGE_AI_API_KEY missing for api-key mode")
	}

	transport := http.DefaultTransport
	if cfg.DebugLogging {
		transport = newAILoggingTransport(transport)
	}
	httpClient := &http.Client{Timeout: cfg.Timeout, Transport: transport}
	return &openAIJudge{
		client:       httpClient,
		baseURL:      strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:       cfg.APIKey,
		organization: cfg.Organization,
		project:      cfg.Project,
		model:        cfg.Model,
	}, nil
}

func (j *openAIJudge) JudgeSignals(ctx context.Context, inputs []aiSignalInput) (map[int]aiSignalOutput, error) {
	if len(inputs) == 0 {
		return map[int]aiSignalOutput{}, nil
	}

	payload, err := json.Marshal(inputs)
	if err != nil {
		return nil, err
	}

	instructions := strings.Join([]string{
		"You rank event-to-market signals for crypto and prediction-market operators.",
		"Return JSON only.",
		"Return one top-level object with key signals.",
		"Return exactly one signals item per candidate index. Do not omit any candidate index.",
		"Each signals item must include index, thesis, why_it_matters, match_type, and score_boost.",
		"Keep thesis and why_it_matters compact and concrete.",
		"score_boost must stay between -5 and 12.",
		"Use match_type market-linked when event clearly maps to listed market.",
		"Use match_type watchlist when event matters but link is weak.",
		"Use match_type no-match when no usable market exists.",
	}, " ")

	requestBody, err := json.Marshal(map[string]any{
		"model": j.model,
		"messages": []map[string]string{
			{"role": "system", "content": instructions},
			{"role": "user", "content": "Candidates:\n" + string(payload)},
		},
		"response_format": map[string]string{"type": "json_object"},
		"temperature":     0.1,
	})
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, j.baseURL+"/chat/completions", bytes.NewReader(requestBody))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+j.apiKey)
	request.Header.Set("Content-Type", "application/json")
	if j.organization != "" {
		request.Header.Set("OpenAI-Organization", j.organization)
	}
	if j.project != "" {
		request.Header.Set("OpenAI-Project", j.project)
	}

	response, err := j.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, aiInvalidResponseErrorf("chat completion status %d: %s", response.StatusCode, string(responseBody))
	}

	var completion chatCompletionResponse
	if err := json.Unmarshal(responseBody, &completion); err != nil {
		return nil, aiInvalidResponseErrorf("decode chat completion response: %v", err)
	}
	if len(completion.Choices) == 0 {
		return nil, aiInvalidResponseErrorf("empty chat completion choices")
	}
	var parsed aiSignalResponse
	if err := json.Unmarshal([]byte(completion.Choices[0].Message.Content), &parsed); err != nil {
		return nil, aiInvalidResponseErrorf("decode chat completion body: %v", err)
	}
	return buildJudgments(parsed, len(inputs))
}

func clampFloat(value float64, minValue float64, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func buildJudgments(parsed aiSignalResponse, expectedCount int) (map[int]aiSignalOutput, error) {
	if len(parsed.Signals) == 0 {
		return nil, aiInvalidResponseErrorf("missing signals")
	}

	judgments := make(map[int]aiSignalOutput, len(parsed.Signals))
	for _, signal := range parsed.Signals {
		if signal.Index < 0 || signal.Index >= expectedCount {
			return nil, aiInvalidResponseErrorf("index %d out of range", signal.Index)
		}
		if signal.Thesis == "" || signal.WhyItMatters == "" || signal.MatchType == "" {
			return nil, aiInvalidResponseErrorf("incomplete signal for index %d", signal.Index)
		}
		if !isValidAIMatchType(signal.MatchType) {
			return nil, aiInvalidResponseErrorf("invalid match_type %q for index %d", signal.MatchType, signal.Index)
		}
		if _, exists := judgments[signal.Index]; exists {
			return nil, aiInvalidResponseErrorf("duplicate signal for index %d", signal.Index)
		}
		signal.ScoreBoost = clampFloat(signal.ScoreBoost, -5, 12)
		judgments[signal.Index] = signal
	}
	return judgments, nil
}

func isValidAIMatchType(value string) bool {
	switch value {
	case matchTypeMarketLinked, matchTypeWatchlist, matchTypeNoMatch:
		return true
	default:
		return false
	}
}

func aiConfigErrorf(format string, args ...any) error {
	return fmt.Errorf("%w: %s", errAIConfig, fmt.Sprintf(format, args...))
}

func aiInvalidResponseErrorf(format string, args ...any) error {
	return fmt.Errorf("%w: %s", errAIInvalidResponse, fmt.Sprintf(format, args...))
}

type aiLoggingTransport struct {
	next http.RoundTripper
}

func newAILoggingTransport(next http.RoundTripper) http.RoundTripper {
	if next == nil {
		next = http.DefaultTransport
	}
	return &aiLoggingTransport{next: next}
}

func (t *aiLoggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	requestBody := readAndRestoreRequestBody(req)
	log.Printf("ai request %s %s body=%s", req.Method, req.URL.String(), truncateForLog(requestBody, 4000))

	resp, err := t.next.RoundTrip(req)
	if err != nil {
		log.Printf("ai request failed %s %s err=%v", req.Method, req.URL.String(), err)
		return nil, err
	}

	responseBody := readAndRestoreResponseBody(resp)
	log.Printf("ai response %s %s status=%d body=%s", req.Method, req.URL.String(), resp.StatusCode, truncateForLog(responseBody, 4000))
	return resp, nil
}

func readAndRestoreRequestBody(req *http.Request) string {
	if req == nil || req.Body == nil {
		return ""
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return fmt.Sprintf("<read error: %v>", err)
	}
	req.Body = io.NopCloser(bytes.NewReader(body))
	return string(body)
}

func readAndRestoreResponseBody(resp *http.Response) string {
	if resp == nil || resp.Body == nil {
		return ""
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("<read error: %v>", err)
	}
	resp.Body = io.NopCloser(bytes.NewReader(body))
	return string(body)
}

func truncateForLog(body string, limit int) string {
	if len(body) <= limit {
		return body
	}
	return body[:limit] + "...<truncated>"
}

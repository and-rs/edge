package app

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	signalsv1 "github.com/index/stint/backend/gen/api/signals/v1"
	"github.com/index/stint/backend/internal/config"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	coindeskRSSURL         = "https://www.coindesk.com/arc/outboundfeeds/rss/"
	kalshiMarketsURL       = "https://external-api.kalshi.com/trade-api/v2/markets?limit=200&status=open"
	defaultSignalsCacheTTL = 20 * time.Second
	externalFetchTimeout   = 10 * time.Second
	matchTypeNoMatch       = "no-match"
	matchTypeWatchlist     = "watchlist"
	matchTypeMarketLinked  = "market-linked"
)

var (
	stopWords = map[string]struct{}{
		"a": {}, "an": {}, "and": {}, "are": {}, "as": {}, "at": {}, "be": {}, "by": {},
		"for": {}, "from": {}, "has": {}, "in": {}, "into": {}, "is": {}, "it": {},
		"its": {}, "of": {}, "on": {}, "or": {}, "that": {}, "the": {}, "their": {},
		"to": {}, "up": {}, "was": {}, "will": {}, "with": {},
	}
	errInvalidProviderResponse = errors.New("invalid provider response")
)

type rssFeed struct {
	Channel struct {
		Items []rssItem `xml:"item"`
	} `xml:"channel"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	PubDate     string `xml:"pubDate"`
	Description string `xml:"description"`
}

type kalshiMarketsResponse struct {
	Markets []kalshiMarket `json:"markets"`
}

type kalshiMarket struct {
	Ticker      string `json:"ticker"`
	Title       string `json:"title"`
	Status      string `json:"status"`
	Volume24hFP string `json:"volume_24h_fp"`
}

type newsSignal struct {
	Headline    string
	Source      string
	SourceURL   string
	PublishedAt time.Time
	Keywords    []string
}

type marketMatch struct {
	Question       string
	URL            string
	Venue          string
	Status         string
	Volume24h      float64
	SharedKeywords []string
	Score          float64
	Reason         string
	MatchType      string
}

type signalCandidate struct {
	Signal newsSignal
	Match  marketMatch
}

type SignalService struct {
	aiConfig       config.AIConfig
	judge          SignalJudge
	judgeInitErr   error
	newsClient     *http.Client
	marketClient   *http.Client
	cacheTTL       time.Duration
	mu             sync.RWMutex
	cachedResponse *signalsv1.ListSignalsResponse
	cacheExpiresAt time.Time
}

func NewSignalService(aiConfig config.AIConfig, judge SignalJudge, judgeInitErr error, cacheTTL time.Duration) *SignalService {
	if cacheTTL <= 0 {
		cacheTTL = defaultSignalsCacheTTL
	}

	return &SignalService{
		aiConfig:     aiConfig,
		judge:        judge,
		judgeInitErr: judgeInitErr,
		newsClient: &http.Client{
			Timeout: externalFetchTimeout,
		},
		marketClient: &http.Client{
			Timeout: externalFetchTimeout,
		},
		cacheTTL: cacheTTL,
	}
}

func (s *SignalService) ListSignals(
	ctx context.Context,
	_ *connect.Request[signalsv1.ListSignalsRequest],
) (*connect.Response[signalsv1.ListSignalsResponse], error) {
	if cached := s.cached(); cached != nil {
		return connect.NewResponse(cached), nil
	}

	response := s.buildResponse(ctx)
	s.store(response)
	return connect.NewResponse(cloneSignalsResponse(response)), nil
}

func (s *SignalService) cached() *signalsv1.ListSignalsResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.cachedResponse == nil || time.Now().After(s.cacheExpiresAt) {
		return nil
	}
	return cloneSignalsResponse(s.cachedResponse)
}

func (s *SignalService) store(response *signalsv1.ListSignalsResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cachedResponse = cloneSignalsResponse(response)
	s.cacheExpiresAt = time.Now().Add(s.cacheTTL)
}

func cloneSignalsResponse(response *signalsv1.ListSignalsResponse) *signalsv1.ListSignalsResponse {
	if response == nil {
		return nil
	}
	return proto.Clone(response).(*signalsv1.ListSignalsResponse)
}

func (s *SignalService) buildResponse(ctx context.Context) *signalsv1.ListSignalsResponse {
	var (
		newsSignals []newsSignal
		markets     []kalshiMarket
		newsErr     error
		marketsErr  error
	)

	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		newsSignals, newsErr = s.fetchCoinDeskSignals(groupCtx)
		return nil
	})
	group.Go(func() error {
		markets, marketsErr = s.fetchKalshiMarkets(groupCtx)
		return nil
	})
	_ = group.Wait()

	eventStage := buildIngestStage("CoinDesk ingest", newsErr, len(newsSignals), "events")
	marketStage := buildIngestStage("Kalshi ingest", marketsErr, len(markets), "markets")
	aiStage := s.aiSetupStage()

	candidates := make([]signalCandidate, 0, len(newsSignals))
	for _, signal := range newsSignals {
		candidates = append(candidates, signalCandidate{
			Signal: signal,
			Match:  matchSignalToMarket(signal, markets),
		})
	}

	sort.Slice(candidates, func(i int, j int) bool {
		return candidates[i].Match.Score > candidates[j].Match.Score
	})
	if len(candidates) > 10 {
		candidates = candidates[:10]
	}

	judgments := map[int]aiSignalOutput{}
	if len(candidates) == 0 {
		aiStage = newStage(
			signalsv1.StageStatus_STAGE_STATUS_SKIPPED,
			"AI judgment skipped",
			"No candidate events available for scoring.",
		)
	} else if s.judge != nil {
		var err error
		judgments, err = s.judge.JudgeSignals(ctx, buildAIInputs(candidates))
		if err != nil {
			aiStage = classifyStageError("AI judgment", err)
			judgments = map[int]aiSignalOutput{}
		} else {
			aiStage = newStage(
				signalsv1.StageStatus_STAGE_STATUS_READY,
				"AI judgment ready",
				fmt.Sprintf("Scored %d candidates.", len(judgments)),
			)
		}
	}

	responseSignals := buildResponseSignals(candidates, judgments, aiStage)

	sort.Slice(responseSignals, func(i int, j int) bool {
		return responseSignals[i].Score > responseSignals[j].Score
	})

	return &signalsv1.ListSignalsResponse{
		Signals:       responseSignals,
		EventIngest:   eventStage,
		MarketIngest:  marketStage,
		AiJudgment:    aiStage,
		Summary:       buildPipelineSummary(len(newsSignals), responseSignals, eventStage, marketStage, aiStage),
		FailureReason: strings.Join(collectFailureDetails(eventStage, marketStage, aiStage), " | "),
	}
}

type newsFetchResult struct {
	signals []newsSignal
	err     error
}

type marketsFetchResult struct {
	markets []kalshiMarket
	err     error
}

func (s *SignalService) StreamSignals(
	ctx context.Context,
	_ *connect.Request[signalsv1.ListSignalsRequest],
	stream *connect.ServerStream[signalsv1.SignalUpdate],
) error {
	if cached := s.cached(); cached != nil {
		return streamCachedSignals(cached, stream)
	}

	newsCh := make(chan newsFetchResult, 1)
	marketsCh := make(chan marketsFetchResult, 1)
	go func() {
		signals, err := s.fetchCoinDeskSignals(ctx)
		newsCh <- newsFetchResult{signals: signals, err: err}
	}()
	go func() {
		markets, err := s.fetchKalshiMarkets(ctx)
		marketsCh <- marketsFetchResult{markets: markets, err: err}
	}()

	var (
		newsSignals []newsSignal
		markets     []kalshiMarket
		eventStage  *signalsv1.Stage
		marketStage *signalsv1.Stage
	)

	for pending := 2; pending > 0; pending-- {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case result := <-newsCh:
			newsSignals = result.signals
			eventStage = buildIngestStage("CoinDesk ingest", result.err, len(newsSignals), "events")
			if err := stream.Send(&signalsv1.SignalUpdate{
				Type:        signalsv1.SignalUpdateType_SIGNAL_UPDATE_TYPE_STAGE,
				EventIngest: eventStage,
			}); err != nil {
				return err
			}
		case result := <-marketsCh:
			markets = result.markets
			marketStage = buildIngestStage("Kalshi ingest", result.err, len(markets), "markets")
			if err := stream.Send(&signalsv1.SignalUpdate{
				Type:         signalsv1.SignalUpdateType_SIGNAL_UPDATE_TYPE_STAGE,
				MarketIngest: marketStage,
			}); err != nil {
				return err
			}
		}
	}

	if eventStage == nil {
		eventStage = buildIngestStage("CoinDesk ingest", nil, len(newsSignals), "events")
	}
	if marketStage == nil {
		marketStage = buildIngestStage("Kalshi ingest", nil, len(markets), "markets")
	}

	candidates := make([]signalCandidate, 0, len(newsSignals))
	for _, signal := range newsSignals {
		candidates = append(candidates, signalCandidate{
			Signal: signal,
			Match:  matchSignalToMarket(signal, markets),
		})
	}
	sort.Slice(candidates, func(i int, j int) bool {
		return candidates[i].Match.Score > candidates[j].Match.Score
	})
	if len(candidates) > 10 {
		candidates = candidates[:10]
	}

	aiStage := s.aiSetupStage()
	if len(candidates) == 0 {
		aiStage = newStage(
			signalsv1.StageStatus_STAGE_STATUS_SKIPPED,
			"AI judgment skipped",
			"No candidate events available for scoring.",
		)
	} else if s.judge != nil {
		aiStage = newStage(
			signalsv1.StageStatus_STAGE_STATUS_RUNNING,
			"AI judgment running",
			fmt.Sprintf("Scoring %d candidates.", len(candidates)),
		)
	}
	if err := stream.Send(&signalsv1.SignalUpdate{
		Type:       signalsv1.SignalUpdateType_SIGNAL_UPDATE_TYPE_STAGE,
		AiJudgment: aiStage,
	}); err != nil {
		return err
	}

	ruleSignals := buildResponseSignals(candidates, map[int]aiSignalOutput{}, aiStage)
	for _, signal := range ruleSignals {
		if err := stream.Send(&signalsv1.SignalUpdate{
			Type:   signalsv1.SignalUpdateType_SIGNAL_UPDATE_TYPE_SIGNAL,
			Signal: signal,
		}); err != nil {
			return err
		}
	}

	judgments := map[int]aiSignalOutput{}
	if len(candidates) > 0 && s.judge != nil {
		var err error
		judgments, err = s.judge.JudgeSignals(ctx, buildAIInputs(candidates))
		if err != nil {
			aiStage = classifyStageError("AI judgment", err)
			judgments = map[int]aiSignalOutput{}
		} else {
			aiStage = newStage(
				signalsv1.StageStatus_STAGE_STATUS_READY,
				"AI judgment ready",
				fmt.Sprintf("Scored %d candidates.", len(judgments)),
			)
		}
		if err := stream.Send(&signalsv1.SignalUpdate{
			Type:       signalsv1.SignalUpdateType_SIGNAL_UPDATE_TYPE_STAGE,
			AiJudgment: aiStage,
		}); err != nil {
			return err
		}
	}

	responseSignals := buildResponseSignals(candidates, judgments, aiStage)
	for _, signal := range responseSignals {
		if err := stream.Send(&signalsv1.SignalUpdate{
			Type:   signalsv1.SignalUpdateType_SIGNAL_UPDATE_TYPE_SIGNAL,
			Signal: signal,
		}); err != nil {
			return err
		}
	}

	response := &signalsv1.ListSignalsResponse{
		Signals:       responseSignals,
		EventIngest:   eventStage,
		MarketIngest:  marketStage,
		AiJudgment:    aiStage,
		Summary:       buildPipelineSummary(len(newsSignals), responseSignals, eventStage, marketStage, aiStage),
		FailureReason: strings.Join(collectFailureDetails(eventStage, marketStage, aiStage), " | "),
	}
	s.store(response)

	if err := stream.Send(&signalsv1.SignalUpdate{
		Type:          signalsv1.SignalUpdateType_SIGNAL_UPDATE_TYPE_SUMMARY,
		Summary:       response.Summary,
		FailureReason: response.FailureReason,
	}); err != nil {
		return err
	}
	return stream.Send(&signalsv1.SignalUpdate{
		Type: signalsv1.SignalUpdateType_SIGNAL_UPDATE_TYPE_DONE,
		Done: true,
	})
}

func streamCachedSignals(response *signalsv1.ListSignalsResponse, stream *connect.ServerStream[signalsv1.SignalUpdate]) error {
	if err := stream.Send(&signalsv1.SignalUpdate{
		Type:         signalsv1.SignalUpdateType_SIGNAL_UPDATE_TYPE_STAGE,
		EventIngest:  response.EventIngest,
		MarketIngest: response.MarketIngest,
		AiJudgment:   response.AiJudgment,
	}); err != nil {
		return err
	}
	for _, signal := range response.Signals {
		if err := stream.Send(&signalsv1.SignalUpdate{
			Type:   signalsv1.SignalUpdateType_SIGNAL_UPDATE_TYPE_SIGNAL,
			Signal: signal,
		}); err != nil {
			return err
		}
	}
	if err := stream.Send(&signalsv1.SignalUpdate{
		Type:          signalsv1.SignalUpdateType_SIGNAL_UPDATE_TYPE_SUMMARY,
		Summary:       response.Summary,
		FailureReason: response.FailureReason,
	}); err != nil {
		return err
	}
	return stream.Send(&signalsv1.SignalUpdate{
		Type: signalsv1.SignalUpdateType_SIGNAL_UPDATE_TYPE_DONE,
		Done: true,
	})
}

func buildResponseSignals(candidates []signalCandidate, judgments map[int]aiSignalOutput, aiStage *signalsv1.Stage) []*signalsv1.Signal {
	responseSignals := make([]*signalsv1.Signal, 0, len(candidates))
	for index, candidate := range candidates {
		ruleMatchType := parseMatchType(candidate.Match.MatchType)
		finalMatchType := ruleMatchType
		whyItMatters := candidate.Match.Reason
		thesis := candidate.Match.Reason
		score := candidate.Match.Score
		signalAIStage := cloneStage(aiStage)
		signalFailureReason := ""

		if judgment, ok := judgments[index]; ok {
			if judgment.WhyItMatters != "" {
				whyItMatters = judgment.WhyItMatters
			}
			if judgment.Thesis != "" {
				thesis = judgment.Thesis
			} else {
				thesis = whyItMatters
			}
			finalMatchType = parseMatchType(judgment.MatchType)
			score = clampFloat(score+judgment.ScoreBoost, 0, 100)
			signalAIStage = newStage(
				signalsv1.StageStatus_STAGE_STATUS_READY,
				"AI judged",
				"Model returned thesis and score adjustment.",
			)
		} else if aiStage != nil && aiStage.Status != signalsv1.StageStatus_STAGE_STATUS_READY && aiStage.Status != signalsv1.StageStatus_STAGE_STATUS_RUNNING {
			signalFailureReason = aiStage.Detail
		}

		responseSignals = append(responseSignals, &signalsv1.Signal{
			Headline:         candidate.Signal.Headline,
			EventSource:      candidate.Signal.Source,
			SourceUrl:        candidate.Signal.SourceURL,
			PublishedAt:      timestamppb.New(candidate.Signal.PublishedAt),
			MarketQuestion:   candidate.Match.Question,
			MarketUrl:        candidate.Match.URL,
			WhyItMatters:     whyItMatters,
			Score:            score,
			MatchedKeywords:  candidate.Match.SharedKeywords,
			MarketVenue:      candidate.Match.Venue,
			MarketVolume_24H: candidate.Match.Volume24h,
			MarketStatus:     candidate.Match.Status,
			RuleMatchType:    ruleMatchType,
			FinalMatchType:   finalMatchType,
			Thesis:           thesis,
			AiJudgment:       signalAIStage,
			FailureReason:    signalFailureReason,
			LinkStateLabel:   buildLinkStateLabel(candidate.Signal.Source, finalMatchType),
		})
	}

	sort.Slice(responseSignals, func(i int, j int) bool {
		return responseSignals[i].Score > responseSignals[j].Score
	})
	return responseSignals
}

func (s *SignalService) fetchCoinDeskSignals(ctx context.Context) ([]newsSignal, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, coindeskRSSURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "IridiumEdge/0.1")

	response, err := s.newsClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status %d", response.StatusCode)
	}

	var feed rssFeed
	if err := xml.NewDecoder(response.Body).Decode(&feed); err != nil {
		return nil, fmt.Errorf("%w: decode Coindesk RSS: %v", errInvalidProviderResponse, err)
	}

	signals := make([]newsSignal, 0, min(10, len(feed.Channel.Items)))
	for _, item := range feed.Channel.Items {
		publishedAt, err := time.Parse(time.RFC1123Z, item.PubDate)
		if err != nil {
			publishedAt = time.Now()
		}
		signals = append(signals, newsSignal{
			Headline:    strings.TrimSpace(item.Title),
			Source:      "CoinDesk",
			SourceURL:   strings.TrimSpace(item.Link),
			PublishedAt: publishedAt,
			Keywords:    extractKeywords(item.Title + " " + item.Description),
		})
		if len(signals) == 10 {
			break
		}
	}

	return signals, nil
}

func (s *SignalService) fetchKalshiMarkets(ctx context.Context) ([]kalshiMarket, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, kalshiMarketsURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "IridiumEdge/0.1")

	response, err := s.marketClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status %d", response.StatusCode)
	}

	var payload kalshiMarketsResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("%w: decode Kalshi markets: %v", errInvalidProviderResponse, err)
	}

	markets := make([]kalshiMarket, 0, len(payload.Markets))
	for _, market := range payload.Markets {
		if market.Status != "active" {
			continue
		}
		markets = append(markets, market)
	}

	return markets, nil
}

func matchSignalToMarket(signal newsSignal, markets []kalshiMarket) marketMatch {
	best := marketMatch{
		Question:  "No obvious market match yet",
		Reason:    "Rule matcher found no strong live Kalshi market link.",
		Score:     8,
		MatchType: matchTypeNoMatch,
	}

	for _, market := range markets {
		marketKeywords := extractKeywords(market.Title)
		shared := intersectKeywords(signal.Keywords, marketKeywords)
		if len(shared) == 0 {
			continue
		}

		volume24h, err := strconv.ParseFloat(market.Volume24hFP, 64)
		if err != nil {
			volume24h = 0
		}
		score := float64(len(shared))*14 + minFloat(volume24h/1000, 8)
		if score <= best.Score {
			continue
		}

		best = marketMatch{
			Question:       market.Title,
			URL:            buildKalshiMarketURL(market.Ticker),
			Venue:          "Kalshi",
			Status:         market.Status,
			Volume24h:      volume24h,
			SharedKeywords: shared,
			Score:          score,
			Reason:         fmt.Sprintf("Matched to live Kalshi market on keywords: %s", strings.Join(shared, ", ")),
			MatchType:      matchTypeWatchlist,
		}
		if len(shared) >= 2 {
			best.MatchType = matchTypeMarketLinked
		}
	}

	return best
}

func buildAIInputs(candidates []signalCandidate) []aiSignalInput {
	inputs := make([]aiSignalInput, 0, len(candidates))
	for index, candidate := range candidates {
		inputs = append(inputs, aiSignalInput{
			Index:           index,
			Headline:        candidate.Signal.Headline,
			Source:          candidate.Signal.Source,
			PublishedAt:     candidate.Signal.PublishedAt.UTC().Format(time.RFC3339),
			MarketQuestion:  candidate.Match.Question,
			MarketStatus:    candidate.Match.Status,
			MarketVenue:     candidate.Match.Venue,
			MarketVolume24h: candidate.Match.Volume24h,
			MatchedKeywords: candidate.Match.SharedKeywords,
			BaseScore:       candidate.Match.Score,
			BaseReason:      candidate.Match.Reason,
		})
	}
	return inputs
}

func buildIngestStage(name string, err error, count int, noun string) *signalsv1.Stage {
	if err != nil {
		return classifyStageError(name, err)
	}
	if count == 0 {
		return newStage(signalsv1.StageStatus_STAGE_STATUS_READY, name+" ready", "No records returned.")
	}
	return newStage(signalsv1.StageStatus_STAGE_STATUS_READY, name+" ready", fmt.Sprintf("Loaded %d %s.", count, noun))
}

func (s *SignalService) aiSetupStage() *signalsv1.Stage {
	if s.aiConfig.AuthMode == "disabled" {
		return newStage(
			signalsv1.StageStatus_STAGE_STATUS_DISABLED,
			"AI judgment disabled",
			"Set STINT_AI_AUTH_MODE to api-key or openai-oauth to enable AI judging.",
		)
	}
	if s.judgeInitErr != nil {
		return classifyStageError("AI judgment", s.judgeInitErr)
	}
	if s.judge == nil {
		return newStage(
			signalsv1.StageStatus_STAGE_STATUS_MISCONFIGURED,
			"AI judgment misconfigured",
			"AI judge unavailable.",
		)
	}
	return newStage(signalsv1.StageStatus_STAGE_STATUS_READY, "AI judgment ready", "AI judge configured.")
}

func classifyStageError(name string, err error) *signalsv1.Stage {
	detail := err.Error()
	switch {
	case errors.Is(err, errAIConfig):
		return newStage(signalsv1.StageStatus_STAGE_STATUS_MISCONFIGURED, name+" misconfigured", detail)
	case errors.Is(err, errAIInvalidResponse), errors.Is(err, errInvalidProviderResponse):
		return newStage(signalsv1.StageStatus_STAGE_STATUS_INVALID, name+" invalid", detail)
	case errors.Is(err, context.DeadlineExceeded), isTimeoutError(err):
		return newStage(signalsv1.StageStatus_STAGE_STATUS_TIMEOUT, name+" timed out", detail)
	default:
		return newStage(signalsv1.StageStatus_STAGE_STATUS_UNAVAILABLE, name+" unavailable", detail)
	}
}

func isTimeoutError(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func newStage(status signalsv1.StageStatus, label string, detail string) *signalsv1.Stage {
	return &signalsv1.Stage{
		Status: status,
		Label:  label,
		Detail: detail,
	}
}

func cloneStage(stage *signalsv1.Stage) *signalsv1.Stage {
	if stage == nil {
		return nil
	}
	return proto.Clone(stage).(*signalsv1.Stage)
}

func parseMatchType(value string) signalsv1.MatchType {
	switch value {
	case matchTypeMarketLinked:
		return signalsv1.MatchType_MATCH_TYPE_MARKET_LINKED
	case matchTypeWatchlist:
		return signalsv1.MatchType_MATCH_TYPE_WATCHLIST
	default:
		return signalsv1.MatchType_MATCH_TYPE_NO_MATCH
	}
}

func buildLinkStateLabel(source string, matchType signalsv1.MatchType) string {
	switch matchType {
	case signalsv1.MatchType_MATCH_TYPE_MARKET_LINKED:
		return source + " → Kalshi linked"
	case signalsv1.MatchType_MATCH_TYPE_WATCHLIST:
		return source + " → possible Kalshi angle"
	default:
		return source + " → no live Kalshi market match"
	}
}

func collectFailureDetails(stages ...*signalsv1.Stage) []string {
	details := make([]string, 0, len(stages))
	for _, stage := range stages {
		if stage == nil || stage.Detail == "" || stage.Status == signalsv1.StageStatus_STAGE_STATUS_READY {
			continue
		}
		details = append(details, stage.Detail)
	}
	return details
}

func buildPipelineSummary(totalEvents int, signals []*signalsv1.Signal, eventStage *signalsv1.Stage, marketStage *signalsv1.Stage, aiStage *signalsv1.Stage) string {
	parts := make([]string, 0, 4)
	if eventStage != nil && eventStage.Status == signalsv1.StageStatus_STAGE_STATUS_READY {
		parts = append(parts, fmt.Sprintf("%d CoinDesk events scanned", totalEvents))
	} else if eventStage != nil && eventStage.Label != "" {
		parts = append(parts, eventStage.Label)
	}

	linked := 0
	watchlist := 0
	unmatched := 0
	for _, signal := range signals {
		switch signal.FinalMatchType {
		case signalsv1.MatchType_MATCH_TYPE_MARKET_LINKED:
			linked += 1
		case signalsv1.MatchType_MATCH_TYPE_WATCHLIST:
			watchlist += 1
		default:
			unmatched += 1
		}
	}
	parts = append(parts, fmt.Sprintf("%d live Kalshi links", linked))
	if watchlist > 0 {
		parts = append(parts, fmt.Sprintf("%d watchlist", watchlist))
	}
	if unmatched > 0 {
		parts = append(parts, fmt.Sprintf("%d unmatched", unmatched))
	}
	if marketStage != nil && marketStage.Status != signalsv1.StageStatus_STAGE_STATUS_READY {
		parts = append(parts, marketStage.Label)
	}
	if aiStage != nil && aiStage.Status != signalsv1.StageStatus_STAGE_STATUS_READY {
		parts = append(parts, aiStage.Label)
	}
	return strings.Join(parts, " · ")
}

func extractKeywords(input string) []string {
	replacer := strings.NewReplacer(
		",", " ", ".", " ", ":", " ", ";", " ", "?", " ", "!", " ",
		"(", " ", ")", " ", "[", " ", "]", " ", "{", " ", "}", " ",
		"/", " ", "\\", " ", "'", " ", "\"", " ", "-", " ", "_", " ",
	)
	parts := strings.Fields(strings.ToLower(replacer.Replace(input)))
	keywords := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		if len(part) < 4 {
			continue
		}
		if _, blocked := stopWords[part]; blocked {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		keywords = append(keywords, part)
	}
	return keywords
}

func intersectKeywords(left []string, right []string) []string {
	shared := make([]string, 0)
	for _, keyword := range left {
		if slices.Contains(right, keyword) {
			shared = append(shared, keyword)
		}
	}
	return shared
}

func buildKalshiMarketURL(ticker string) string {
	return "https://kalshi.com/markets/" + url.PathEscape(ticker)
}

func minFloat(left float64, right float64) float64 {
	if left < right {
		return left
	}
	return right
}

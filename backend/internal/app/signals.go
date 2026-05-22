package app

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	flightv1 "github.com/index/stint/backend/gen/api/flight/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	coindeskRSSURL   = "https://www.coindesk.com/arc/outboundfeeds/rss/"
	kalshiMarketsURL = "https://external-api.kalshi.com/trade-api/v2/markets?limit=200&status=open"
)

var stopWords = map[string]struct{}{
	"a": {}, "an": {}, "and": {}, "are": {}, "as": {}, "at": {}, "be": {}, "by": {},
	"for": {}, "from": {}, "has": {}, "in": {}, "into": {}, "is": {}, "it": {},
	"its": {}, "of": {}, "on": {}, "or": {}, "that": {}, "the": {}, "their": {},
	"to": {}, "up": {}, "was": {}, "will": {}, "with": {},
}

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
	Question string
	URL      string
	Score    float64
	Reason   string
}

func (s *FlightService) ListSignals(
	ctx context.Context,
	_ *connect.Request[flightv1.ListSignalsRequest],
) (*connect.Response[flightv1.ListSignalsResponse], error) {
	newsSignals, err := fetchCoinDeskSignals(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("fetch Coindesk RSS: %w", err))
	}

	markets, err := fetchKalshiMarkets(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("fetch Kalshi markets: %w", err))
	}

	responseSignals := make([]*flightv1.Signal, 0, len(newsSignals))
	for _, signal := range newsSignals {
		match := matchSignalToMarket(signal, markets)
		responseSignals = append(responseSignals, &flightv1.Signal{
			Headline:       signal.Headline,
			Source:         signal.Source,
			SourceUrl:      signal.SourceURL,
			PublishedAt:    timestamppb.New(signal.PublishedAt),
			MarketQuestion: match.Question,
			MarketUrl:      match.URL,
			WhyItMatters:   match.Reason,
			Score:          match.Score,
		})
	}

	sort.Slice(responseSignals, func(i int, j int) bool {
		return responseSignals[i].Score > responseSignals[j].Score
	})
	if len(responseSignals) > 10 {
		responseSignals = responseSignals[:10]
	}

	return connect.NewResponse(&flightv1.ListSignalsResponse{Signals: responseSignals}), nil
}

func fetchCoinDeskSignals(ctx context.Context) ([]newsSignal, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, coindeskRSSURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "IridiumEdge/0.1")

	client := &http.Client{Timeout: 10 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	var feed rssFeed
	if err := xml.NewDecoder(response.Body).Decode(&feed); err != nil {
		return nil, err
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

func fetchKalshiMarkets(ctx context.Context) ([]kalshiMarket, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, kalshiMarketsURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "IridiumEdge/0.1")

	client := &http.Client{Timeout: 10 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	var payload kalshiMarketsResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, err
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
		Question: "No obvious market match yet",
		Reason:   "Need stronger entity and market mapping than simple keyword overlap.",
		Score:    0,
	}

	for _, market := range markets {
		marketKeywords := extractKeywords(market.Title)
		shared := intersectKeywords(signal.Keywords, marketKeywords)
		if len(shared) < 2 {
			continue
		}

		volume24h, err := strconv.ParseFloat(market.Volume24hFP, 64)
		if err != nil {
			volume24h = 0
		}
		score := float64(len(shared))*10 + minFloat(volume24h/1000, 5)
		if score <= best.Score {
			continue
		}

		best = marketMatch{
			Question: market.Title,
			URL:      buildKalshiMarketURL(market.Ticker),
			Score:    score,
			Reason:   fmt.Sprintf("Matched on keywords: %s", strings.Join(shared, ", ")),
		}
	}

	return best
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

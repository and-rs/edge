package signals

import (
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

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

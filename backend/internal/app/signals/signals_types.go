package signals

import (
	"errors"
	"time"
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

type newsFetchResult struct {
	signals []newsSignal
	err     error
}

type marketsFetchResult struct {
	markets []kalshiMarket
	err     error
}

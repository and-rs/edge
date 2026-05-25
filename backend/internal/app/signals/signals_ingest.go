package signals

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
	"time"
)

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

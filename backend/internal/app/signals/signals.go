package signals

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	signalsv1 "github.com/index/edge/backend/gen/api/signals/v1"
	"github.com/index/edge/backend/internal/config"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
)

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

func NewService(aiConfig config.AIConfig, judge SignalJudge, judgeInitErr error, cacheTTL time.Duration) *SignalService {
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

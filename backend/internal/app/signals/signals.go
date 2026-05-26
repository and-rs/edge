package signals

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"connectrpc.com/connect"
	signalsv1 "github.com/index/edge/backend/gen/api/signals/v1"
	"github.com/index/edge/backend/internal/config"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

type SignalService struct {
	aiConfig       config.AIConfig
	judge          SignalJudge
	judgeInitErr   error
	newsClient     *http.Client
	marketClient   *http.Client
	cacheTTL       time.Duration
	mu             sync.RWMutex
	cachedState    *signalsv1.SignalHuntState
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

func (s *SignalService) cached() *signalsv1.SignalHuntState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.cachedState == nil || time.Now().After(s.cacheExpiresAt) {
		return nil
	}
	return cloneSignalHuntState(s.cachedState)
}

func (s *SignalService) store(state *signalsv1.SignalHuntState) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cachedState = cloneSignalHuntState(state)
	s.cacheExpiresAt = time.Now().Add(s.cacheTTL)
}

func cloneSignalHuntState(state *signalsv1.SignalHuntState) *signalsv1.SignalHuntState {
	if state == nil {
		return nil
	}
	return proto.Clone(state).(*signalsv1.SignalHuntState)
}

func (s *SignalService) StreamSignals(
	ctx context.Context,
	_ *connect.Request[emptypb.Empty],
	stream *connect.ServerStream[signalsv1.SignalHuntEvent],
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
			if err := sendStages(stream, buildPipelineStages(eventStage, marketStage, nil)); err != nil {
				return err
			}
		case result := <-marketsCh:
			markets = result.markets
			marketStage = buildIngestStage("Kalshi ingest", result.err, len(markets), "markets")
			if err := sendStages(stream, buildPipelineStages(eventStage, marketStage, nil)); err != nil {
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
	if len(candidates) > freeScanSignalLimit {
		candidates = candidates[:freeScanSignalLimit]
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
	if err := sendStages(stream, buildPipelineStages(eventStage, marketStage, aiStage)); err != nil {
		return err
	}

	ruleSignals := buildResponseSignals(candidates, map[int]aiSignalOutput{}, aiStage)
	for _, signal := range ruleSignals {
		if err := sendSignal(stream, signal); err != nil {
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
		if err := sendStages(stream, buildPipelineStages(eventStage, marketStage, aiStage)); err != nil {
			return err
		}
	}

	responseSignals := buildResponseSignals(candidates, judgments, aiStage)
	for _, signal := range responseSignals {
		if err := sendSignal(stream, signal); err != nil {
			return err
		}
	}

	state := &signalsv1.SignalHuntState{
		Signals: responseSignals,
		Stages:  buildPipelineStages(eventStage, marketStage, aiStage),
		Summary: buildSignalHuntSummary(len(newsSignals), responseSignals, eventStage, marketStage, aiStage),
	}
	s.store(state)

	if err := sendSummary(stream, state.Summary); err != nil {
		return err
	}
	return sendDone(stream)
}

func sendStages(stream *connect.ServerStream[signalsv1.SignalHuntEvent], stages *signalsv1.PipelineStages) error {
	return stream.Send(&signalsv1.SignalHuntEvent{
		Event: &signalsv1.SignalHuntEvent_Stages{Stages: stages},
	})
}

func sendSignal(stream *connect.ServerStream[signalsv1.SignalHuntEvent], signal *signalsv1.Signal) error {
	return stream.Send(&signalsv1.SignalHuntEvent{
		Event: &signalsv1.SignalHuntEvent_Signal{Signal: signal},
	})
}

func sendSummary(stream *connect.ServerStream[signalsv1.SignalHuntEvent], summary *signalsv1.SignalHuntSummary) error {
	return stream.Send(&signalsv1.SignalHuntEvent{
		Event: &signalsv1.SignalHuntEvent_Summary{Summary: summary},
	})
}

func sendDone(stream *connect.ServerStream[signalsv1.SignalHuntEvent]) error {
	return stream.Send(&signalsv1.SignalHuntEvent{
		Event: &signalsv1.SignalHuntEvent_Done{Done: &signalsv1.SignalHuntDone{}},
	})
}

func streamCachedSignals(state *signalsv1.SignalHuntState, stream *connect.ServerStream[signalsv1.SignalHuntEvent]) error {
	if err := sendStages(stream, state.Stages); err != nil {
		return err
	}
	for _, signal := range state.Signals {
		if err := sendSignal(stream, signal); err != nil {
			return err
		}
	}
	if err := sendSummary(stream, state.Summary); err != nil {
		return err
	}
	return sendDone(stream)
}

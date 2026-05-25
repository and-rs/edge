package signals

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sort"
	"strings"

	signalsv1 "github.com/index/edge/backend/gen/api/signals/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

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
			"Set EDGE_AI_AUTH_MODE=api-key to enable AI judging.",
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

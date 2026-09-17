package prediction

import (
	"testing"

	"TwitchChannelPointsMiner/internal/streamer"
)

func ptrFloat(val float64) *float64 { return &val }

func TestShouldSkipByFilterTotals(t *testing.T) {
	outcomes := []PredictionOutcome{
		{TotalUsers: 10, TotalPoints: 100},
		{TotalUsers: 15, TotalPoints: 200},
	}
	s := &streamer.Streamer{
		Settings: streamer.StreamerSettings{
			Bet: streamer.BetSettings{
				FilterCondition: &streamer.FilterCondition{
					By:    streamer.OutcomeTotalUsers,
					Where: streamer.ConditionGTE,
					Value: ptrFloat(20),
				},
			},
		},
	}
	event := &PredictionEvent{
		Streamer: s,
		Outcomes: outcomes,
		Decision: PredictionDecision{Choice: 1},
	}

	skip, compared, reason := event.ShouldSkipByFilter()
	if skip {
		t.Fatalf("expected bet allowed, got skip (compared %.0f, reason %s)", compared, reason)
	}
	if compared != 25 {
		t.Fatalf("expected compared total users 25, got %.0f", compared)
	}

	// ? force skip
	event.Streamer.Settings.Bet.FilterCondition.Value = ptrFloat(30)
	skip, compared, _ = event.ShouldSkipByFilter()
	if !skip {
		t.Fatalf("expected skip when total users below threshold")
	}
	if compared != 25 {
		t.Fatalf("expected compared total users 25, got %.0f", compared)
	}
}

func TestShouldSkipByFilterDecisionUsers(t *testing.T) {
	outcomes := []PredictionOutcome{
		{TotalUsers: 5, TotalPoints: 10},
		{TotalUsers: 50, TotalPoints: 20},
	}
	s := &streamer.Streamer{
		Settings: streamer.StreamerSettings{
			Bet: streamer.BetSettings{
				FilterCondition: &streamer.FilterCondition{
					By:    streamer.OutcomeDecisionUsers,
					Where: streamer.ConditionLT,
					Value: ptrFloat(40),
				},
			},
		},
	}
	event := &PredictionEvent{
		Streamer: s,
		Outcomes: outcomes,
		Decision: PredictionDecision{Choice: 1},
	}

	skip, compared, _ := event.ShouldSkipByFilter()
	if !skip {
		t.Fatalf("expected skip when decision users 50 !< 40")
	}
	if compared != 50 {
		t.Fatalf("expected decision users compared 50, got %.0f", compared)
	}

	// ? Loosen threshold to allow betting
	event.Streamer.Settings.Bet.FilterCondition.Where = streamer.ConditionGTE
	event.Streamer.Settings.Bet.FilterCondition.Value = ptrFloat(50)
	skip, compared, reason := event.ShouldSkipByFilter()
	if skip {
		t.Fatalf("expected bet allowed, got skip (compared %.0f, reason %s)", compared, reason)
	}
}

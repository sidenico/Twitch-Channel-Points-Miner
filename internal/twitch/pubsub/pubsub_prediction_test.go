package pubsub

import (
	"testing"

	"TwitchChannelPointsMiner/internal/constants"
	"TwitchChannelPointsMiner/internal/prediction"
	"TwitchChannelPointsMiner/internal/streamer"
)

type stubPubSubLogger struct {
	printfCalls int
	errorCalls  int
	emojiCalls  int
	debugCalls  int
}

func (s *stubPubSubLogger) Printf(string, ...interface{}) {
	s.printfCalls++
}

func (s *stubPubSubLogger) Errorf(string, ...interface{}) {
	s.errorCalls++
}

func (s *stubPubSubLogger) EmojiPrintf(string, string, ...interface{}) {
	s.emojiCalls++
}

func (s *stubPubSubLogger) Eventf(constants.Event, string, ...interface{}) {
	s.printfCalls++
}

func (s *stubPubSubLogger) EmojiEventf(string, constants.Event, string, ...interface{}) {
	s.emojiCalls++
}

func (s *stubPubSubLogger) ErrorEventf(constants.Event, string, ...interface{}) {
	s.errorCalls++
}

func (s *stubPubSubLogger) Debugf(string, ...interface{}) {
	s.debugCalls++
}

func (s *stubPubSubLogger) DebugEnabled() bool {
	return false
}

func TestPredictionEventDecideDoesNotMarkBetPlaced(t *testing.T) {
	s := &streamer.Streamer{
		Username:      "tester",
		ChannelPoints: 1_000,
		Settings: streamer.StreamerSettings{
			Bet: streamer.BetSettings{Strategy: streamer.StrategyMostVoted},
		},
	}
	event := prediction.NewPredictionEvent(s, map[string]interface{}{
		"id":     "ev1",
		"status": "ACTIVE",
		"title":  "test",
		"outcomes": []interface{}{
			map[string]interface{}{"id": "a", "title": "A", "color": "blue", "total_users": 10, "total_points": 100},
			map[string]interface{}{"id": "b", "title": "B", "color": "pink", "total_users": 5, "total_points": 50},
		},
	})
	if event == nil {
		t.Fatalf("expected event")
	}

	_ = event.Decide(s.ChannelPoints)
	if event.BetPlaced {
		t.Fatalf("Decide should not mark BetPlaced=true; BetPlaced is reserved for successful MakePrediction")
	}
}

func TestPlacePredictionStopsTrackingOnFilterSkip(t *testing.T) {
	logger := &stubPubSubLogger{}
	value := 1_000_000.0
	s := &streamer.Streamer{
		Username:      "tester",
		ChannelPoints: 1_000,
		Settings: streamer.StreamerSettings{
			Bet: streamer.BetSettings{
				Strategy: streamer.StrategyMostVoted,
				FilterCondition: &streamer.FilterCondition{
					By:    streamer.OutcomeTotalUsers,
					Where: streamer.ConditionGT,
					Value: &value,
				},
			},
		},
	}
	event := prediction.NewPredictionEvent(s, map[string]interface{}{
		"id":     "ev-skip",
		"status": "ACTIVE",
		"title":  "skip-me",
		"outcomes": []interface{}{
			map[string]interface{}{"id": "a", "title": "A", "color": "blue", "total_users": 10, "total_points": 100},
			map[string]interface{}{"id": "b", "title": "B", "color": "pink", "total_users": 5, "total_points": 50},
		},
	})
	if event == nil {
		t.Fatalf("expected event")
	}

	client := &PubSubClient{
		logger:      logger,
		predictions: map[string]*prediction.PredictionEvent{event.EventID: event},
	}

	client.placePrediction(event.EventID)
	if _, ok := client.predictions[event.EventID]; ok {
		t.Fatalf("expected prediction %q to be removed from tracking after filter skip", event.EventID)
	}
	if event.ResultType != "SKIPPED" {
		t.Fatalf("expected ResultType SKIPPED, got %q", event.ResultType)
	}
}

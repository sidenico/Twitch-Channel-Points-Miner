package twitchchannelpointsminer

import (
	"testing"
	"time"

	classpkg "TwitchChannelPointsMiner/TwitchChannelPointsMiner/classes"
	"TwitchChannelPointsMiner/TwitchChannelPointsMiner/classes/entities"
)

func onlineCandidate(username string, points int, game string, onlineAt time.Time) *entities.Streamer {
	return &entities.Streamer{
		Username:      username,
		ChannelID:     username + "-id",
		ChannelPoints: points,
		IsOnline:      true,
		PresenceKnown: true,
		OnlineAt:      onlineAt,
		Settings: entities.StreamerSettings{
			WatchStreak: false,
			ClaimDrops:  false,
			IRCMode:     entities.IRCModeNever,
		},
		Stream: &entities.Stream{
			BroadcastID: username + "-broadcast",
			Game:        map[string]interface{}{"displayName": game},
			CreatedAt:   onlineAt,
			StreamUpAt:  onlineAt,
		},
	}
}

func usernames(streamers []*entities.Streamer) []string {
	out := make([]string, 0, len(streamers))
	for _, s := range streamers {
		if s != nil {
			out = append(out, s.Username)
		}
	}
	return out
}

func TestPickStreamersToWatchOrderPriorityCapsAtTwo(t *testing.T) {
	now := time.Now()
	onlineAt := now.Add(-time.Minute)
	m := &Miner{
		logger:          discardLogger(),
		watchPriorities: []watchPriority{watchPriorityOrder},
	}
	streamers := []*entities.Streamer{
		onlineCandidate("a", 100, "GameA", onlineAt),
		onlineCandidate("b", 200, "GameB", onlineAt),
		onlineCandidate("c", 300, "GameC", onlineAt),
	}
	got := m.pickStreamersToWatch(streamers)
	if len(got) != maxConcurrentWatchers {
		t.Fatalf("expected %d watchers got %d", maxConcurrentWatchers, len(got))
	}
	if got[0].Username != "a" || got[1].Username != "b" {
		t.Fatalf("ORDER should keep list order for first two slots, got %s then %s", got[0].Username, got[1].Username)
	}
}

func TestPickStreamersToWatchPointsDescending(t *testing.T) {
	now := time.Now()
	onlineAt := now.Add(-time.Minute)
	m := &Miner{
		logger:          discardLogger(),
		watchPriorities: []watchPriority{watchPriorityPointsDescending},
	}
	streamers := []*entities.Streamer{
		onlineCandidate("low", 10, "GameA", onlineAt),
		onlineCandidate("mid", 100, "GameB", onlineAt),
		onlineCandidate("high", 1000, "GameC", onlineAt),
	}
	got := m.pickStreamersToWatch(streamers)
	if len(got) != 2 {
		t.Fatalf("expected 2 watchers got %d", len(got))
	}
	if got[0].Username != "high" || got[1].Username != "mid" {
		t.Fatalf("POINTS_DESC got %s then %s", got[0].Username, got[1].Username)
	}
}

func TestPickStreamersToWatchPointsAscending(t *testing.T) {
	now := time.Now()
	onlineAt := now.Add(-time.Minute)
	m := &Miner{
		logger:          discardLogger(),
		watchPriorities: []watchPriority{watchPriorityPointsAscending},
	}
	streamers := []*entities.Streamer{
		onlineCandidate("high", 1000, "GameA", onlineAt),
		onlineCandidate("low", 10, "GameB", onlineAt),
		onlineCandidate("mid", 100, "GameC", onlineAt),
	}
	got := m.pickStreamersToWatch(streamers)
	if len(got) != 2 {
		t.Fatalf("expected 2 watchers got %d", len(got))
	}
	if got[0].Username != "low" || got[1].Username != "mid" {
		t.Fatalf("POINTS_ASC got %s then %s", got[0].Username, got[1].Username)
	}
}

func TestPickStreamersToWatchDropsPriority(t *testing.T) {
	now := time.Now()
	onlineAt := now.Add(-time.Minute)
	m := &Miner{
		logger:          discardLogger(),
		watchPriorities: []watchPriority{watchPriorityDrops, watchPriorityOrder},
	}
	withDrops := onlineCandidate("drops", 50, "GameA", onlineAt)
	withDrops.Settings.ClaimDrops = true
	without := onlineCandidate("plain", 999, "GameB", onlineAt)
	third := onlineCandidate("also-plain", 1000, "GameC", onlineAt)
	got := m.pickStreamersToWatch([]*entities.Streamer{without, withDrops, third})
	if len(got) != 2 {
		t.Fatalf("expected 2 watchers got %d", len(got))
	}
	if got[0].Username != "drops" {
		t.Fatalf("DROPS priority should select claim_drops streamer first, got %s", got[0].Username)
	}
}

func TestPickStreamersToWatchSkipsOnlineGracePeriod(t *testing.T) {
	now := time.Now()
	m := &Miner{
		logger:          discardLogger(),
		watchPriorities: []watchPriority{watchPriorityOrder},
	}
	fresh := onlineCandidate("fresh", 100, "GameA", now)
	ready := onlineCandidate("ready", 100, "GameB", now.Add(-time.Minute))
	got := m.pickStreamersToWatch([]*entities.Streamer{fresh, ready})
	if len(got) != 1 || got[0].Username != "ready" {
		t.Fatalf("expected only ready streamer, got %#v", usernames(got))
	}
}

func TestPickStreamersToWatchSkipsOfflineAndExcludedGames(t *testing.T) {
	now := time.Now()
	onlineAt := now.Add(-time.Minute)
	m := &Miner{
		logger:          discardLogger(),
		watchPriorities: []watchPriority{watchPriorityOrder},
		gameExclusions:  map[string]struct{}{"banned": {}},
	}
	offline := onlineCandidate("offline", 100, "GameA", onlineAt)
	offline.IsOnline = false
	excluded := onlineCandidate("excluded", 100, "Banned", onlineAt)
	ok := onlineCandidate("ok", 100, "Allowed", onlineAt)
	got := m.pickStreamersToWatch([]*entities.Streamer{offline, excluded, ok})
	if len(got) != 1 || got[0].Username != "ok" {
		t.Fatalf("expected only ok streamer, got %#v", usernames(got))
	}
}

func TestPickStreamersToWatchGameDiversityPrefersDifferentGames(t *testing.T) {
	now := time.Now()
	onlineAt := now.Add(-time.Minute)
	m := &Miner{
		logger:          discardLogger(),
		watchPriorities: []watchPriority{watchPriorityOrder},
	}
	a1 := onlineCandidate("a1", 10, "SameGame", onlineAt)
	a2 := onlineCandidate("a2", 20, "SameGame", onlineAt)
	b1 := onlineCandidate("b1", 30, "OtherGame", onlineAt)
	got := m.pickStreamersToWatch([]*entities.Streamer{a1, a2, b1})
	if len(got) != 2 {
		t.Fatalf("expected 2 watchers got %d", len(got))
	}
	names := map[string]bool{got[0].Username: true, got[1].Username: true}
	if !names["a1"] || !names["b1"] {
		t.Fatalf("expected a1 and b1 for game diversity, got %#v", usernames(got))
	}
}

func TestPickStreamersToWatchDefersStreakWithoutPriorityGame(t *testing.T) {
	now := time.Now()
	onlineAt := now.Add(-time.Minute)
	m := &Miner{
		logger:          discardLogger(),
		watchPriorities: []watchPriority{watchPriorityStreak, watchPriorityOrder},
		gamePriority:    []string{"priority"},
		gamePriorityIndex: map[string]int{
			"priority": 0,
		},
	}
	streak := onlineCandidate("streak", 10, "other", onlineAt)
	streak.Settings.WatchStreak = true
	streak.Stream.WatchStreakMissing = true
	streak.Stream.MinuteWatched = 0
	orderFirst := onlineCandidate("order1", 100, "priority", onlineAt)
	orderSecond := onlineCandidate("order2", 200, "priority-b", onlineAt)
	got := m.pickStreamersToWatch([]*entities.Streamer{streak, orderFirst, orderSecond})
	if len(got) != 2 {
		t.Fatalf("expected 2 watchers got %d (%#v)", len(got), usernames(got))
	}
	if got[0].Username != "order1" {
		t.Fatalf("expected ORDER fill first when streak is not on priority game, got %s then %s", got[0].Username, got[1].Username)
	}
}

func TestNormalizeStreamerListDedupesAndLowercases(t *testing.T) {
	got := normalizeStreamerList([]string{" Alice ", "bob", "ALICE", "", "Bob"})
	want := []string{"alice", "bob"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("idx %d got %q want %q", i, got[i], want[i])
		}
	}
}

func TestSetPresenceFirstKnownOnlineSetsTimestampAndLogsOnce(t *testing.T) {
	m := &Miner{
		logger:       discardLogger(),
		chatWatchers: make(map[string]*classpkg.ChatClient),
	}
	s := &entities.Streamer{
		Username:  "newbie",
		ChannelID: "1",
		Settings:  entities.StreamerSettings{IRCMode: entities.IRCModeNever},
		Stream:    entities.NewStream(),
	}
	if s.PresenceKnown || s.IsOnline {
		t.Fatalf("precondition failed")
	}
	before := time.Now().Add(-time.Second)
	m.setPresence(s, true, "poll")
	after := time.Now().Add(time.Second)
	if !s.PresenceKnown || !s.IsOnline {
		t.Fatalf("expected known online state")
	}
	if s.OnlineAt.Before(before) || s.OnlineAt.After(after) {
		t.Fatalf("OnlineAt not set around now: %s", s.OnlineAt)
	}
}

func TestSetPresenceNoOpDoesNotResetTimestamps(t *testing.T) {
	m := &Miner{
		logger:       discardLogger(),
		chatWatchers: make(map[string]*classpkg.ChatClient),
	}
	onlineAt := time.Now().Add(-10 * time.Minute)
	s := &entities.Streamer{
		Username:      "stable",
		ChannelID:     "1",
		IsOnline:      true,
		PresenceKnown: true,
		OnlineAt:      onlineAt,
		Settings:      entities.StreamerSettings{IRCMode: entities.IRCModeNever},
		Stream:        entities.NewStream(),
	}
	m.setPresence(s, true, "poll")
	if !s.OnlineAt.Equal(onlineAt) {
		t.Fatalf("no-op online toggle should keep OnlineAt, got %s want %s", s.OnlineAt, onlineAt)
	}
}

func TestHandlePubSubPresencePrefixesReasonViaOfflineTransition(t *testing.T) {
	m := &Miner{
		logger:       discardLogger(),
		chatWatchers: make(map[string]*classpkg.ChatClient),
	}
	s := &entities.Streamer{
		Username:      "live",
		ChannelID:     "1",
		IsOnline:      true,
		PresenceKnown: true,
		OnlineAt:      time.Now().Add(-time.Hour),
		Settings:      entities.StreamerSettings{IRCMode: entities.IRCModeNever},
		Stream:        entities.NewStream(),
	}
	m.handlePubSubPresence(s, false, "stream-down")
	if s.IsOnline {
		t.Fatalf("expected offline after pubsub presence")
	}
	if s.OfflineAt.IsZero() {
		t.Fatalf("expected OfflineAt set")
	}
}

func TestSetPresenceOfflineTransitionArmsResolvedStreakCarryover(t *testing.T) {
	m := &Miner{
		logger:       discardLogger(),
		chatWatchers: make(map[string]*classpkg.ChatClient),
	}
	s := &entities.Streamer{
		Username:             "streaky",
		ChannelID:            "1",
		IsOnline:             true,
		PresenceKnown:        true,
		OnlineAt:             time.Now().Add(-time.Hour),
		CompletedWatchStreak: true,
		Settings:             entities.StreamerSettings{WatchStreak: true, IRCMode: entities.IRCModeNever},
		Stream: &entities.Stream{
			WatchStreakMissing: false,
			BroadcastID:        "b1",
			CreatedAt:          time.Now().Add(-time.Hour),
		},
	}
	m.setPresence(s, false, "poll")
	if !s.ResolvedStreakCarryover || s.ResolvedStreakCarryoverUntil.IsZero() {
		t.Fatalf("expected resolved streak carryover armed on offline transition")
	}
}

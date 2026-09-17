package miner

import (
	"testing"

	"TwitchChannelPointsMiner/internal/notify"
	st "TwitchChannelPointsMiner/internal/streamer"
)

func TestShouldJoinChat(t *testing.T) {
	tests := []struct {
		mode   st.IRCMode
		online bool
		want   bool
	}{
		{mode: st.IRCModeAlways, online: true, want: true},
		{mode: st.IRCModeAlways, online: false, want: true},
		{mode: st.IRCModeNever, online: true, want: false},
		{mode: st.IRCModeNever, online: false, want: false},
		{mode: st.IRCModeOnline, online: true, want: true},
		{mode: st.IRCModeOnline, online: false, want: false},
		{mode: st.IRCModeOffline, online: true, want: false},
		{mode: st.IRCModeOffline, online: false, want: true},
		{mode: st.IRCMode(""), online: true, want: true}, // ? default fallback
		{mode: st.IRCMode("unknown"), online: false, want: false},
	}
	for _, tt := range tests {
		if got := shouldJoinChat(tt.mode, tt.online); got != tt.want {
			t.Fatalf("mode=%s online=%t got %t want %t", tt.mode, tt.online, got, tt.want)
		}
	}
}

func TestNewMinerDisableAtInNicknameSetting(t *testing.T) {
	streamerSettings := st.StreamerSettings{}
	streamerSettings.Default()

	minr := NewMiner(
		"user",
		"",
		false,
		false,
		notify.LoggerSettings{},
		streamerSettings,
		nil,
		nil,
		nil,
		nil,
		nil,
		true,
		false,
		false,
		true,
		false,
	)

	if !minr.disableAtInNickname {
		t.Fatalf("expected disableAtInNickname to be enabled from constructor")
	}
}

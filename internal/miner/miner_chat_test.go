package miner

import (
	"testing"

	"TwitchChannelPointsMiner/internal/notify"
	"TwitchChannelPointsMiner/internal/streamer"
)

func TestShouldJoinChat(t *testing.T) {
	tests := []struct {
		mode   streamer.IRCMode
		online bool
		want   bool
	}{
		{mode: streamer.IRCModeAlways, online: true, want: true},
		{mode: streamer.IRCModeAlways, online: false, want: true},
		{mode: streamer.IRCModeNever, online: true, want: false},
		{mode: streamer.IRCModeNever, online: false, want: false},
		{mode: streamer.IRCModeOnline, online: true, want: true},
		{mode: streamer.IRCModeOnline, online: false, want: false},
		{mode: streamer.IRCModeOffline, online: true, want: false},
		{mode: streamer.IRCModeOffline, online: false, want: true},
		{mode: streamer.IRCMode(""), online: true, want: true}, // ? default fallback
		{mode: streamer.IRCMode("unknown"), online: false, want: false},
	}
	for _, tt := range tests {
		if got := shouldJoinChat(tt.mode, tt.online); got != tt.want {
			t.Fatalf("mode=%s online=%t got %t want %t", tt.mode, tt.online, got, tt.want)
		}
	}
}

func TestNewMinerDisableAtInNicknameSetting(t *testing.T) {
	streamerSettings := streamer.StreamerSettings{}
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

package config

import (
	"testing"

	"TwitchChannelPointsMiner/internal/streamer"
)

func TestParseIRCMode(t *testing.T) {
	fallback := streamer.IRCModeOnline
	tests := []struct {
		name string
		in   string
		want streamer.IRCMode
	}{
		{name: "always exact", in: "ALWAYS", want: streamer.IRCModeAlways},
		{name: "always lower", in: "always", want: streamer.IRCModeAlways},
		{name: "never padded", in: "  never ", want: streamer.IRCModeNever},
		{name: "offline", in: "offline", want: streamer.IRCModeOffline},
		{name: "online", in: "online", want: streamer.IRCModeOnline},
		{name: "empty", in: "", want: fallback},
		{name: "invalid", in: "nope", want: fallback},
	}
	for _, tt := range tests {
		if got := parseChatPresence(tt.in, fallback); got != tt.want {
			t.Fatalf("%s: parseChatPresence(%q)=%s want %s", tt.name, tt.in, got, tt.want)
		}
	}
}

func TestMergeStreamerSettingsIRCMode(t *testing.T) {
	base := streamer.StreamerSettings{IRCMode: streamer.IRCModeNever}

	// ? Valid override should replace base
	overrideVal := "online"
	out := mergeStreamerSettings(base, StreamerSettingsConfig{IRCMode: &overrideVal})
	if out.IRCMode != streamer.IRCModeOnline {
		t.Fatalf("expected override to set IRCMode to ONLINE, got %s", out.IRCMode)
	}

	// ? Invalid override should keep base
	invalid := "maybe"
	out = mergeStreamerSettings(base, StreamerSettingsConfig{IRCMode: &invalid})
	if out.IRCMode != base.IRCMode {
		t.Fatalf("invalid override should keep base IRCMode %s, got %s", base.IRCMode, out.IRCMode)
	}

	// ? Nil override should keep base
	out = mergeStreamerSettings(base, StreamerSettingsConfig{})
	if out.IRCMode != base.IRCMode {
		t.Fatalf("nil override should keep base IRCMode %s, got %s", base.IRCMode, out.IRCMode)
	}
}

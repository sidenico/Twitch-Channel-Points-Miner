package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"TwitchChannelPointsMiner/internal/streamer"
)

func TestResolveAppPathsPrefersDataDirFlag(t *testing.T) {
	dir := t.TempDir()
	paths, err := ResolvePaths("", dir)
	if err != nil {
		t.Fatalf("ResolvePaths: %v", err)
	}
	if paths.WorkDir != dir {
		t.Fatalf("WorkDir got %q want %q", paths.WorkDir, dir)
	}
	wantCfg := filepath.Join(dir, "config.json")
	if paths.ConfigPath != wantCfg {
		t.Fatalf("ConfigPath got %q want %q", paths.ConfigPath, wantCfg)
	}
}

func TestResolveAppPathsConfigFlagSetsWorkDirToParent(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "custom.json")
	paths, err := ResolvePaths(cfgPath, "")
	if err != nil {
		t.Fatalf("ResolvePaths: %v", err)
	}
	if paths.WorkDir != dir {
		t.Fatalf("WorkDir got %q want %q", paths.WorkDir, dir)
	}
	if paths.ConfigPath != cfgPath {
		t.Fatalf("ConfigPath got %q want %q", paths.ConfigPath, cfgPath)
	}
}

func TestResolveAppPathsUsesTCPMDataDirEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TCPM_DATA_DIR", dir)
	t.Setenv("TCPM_CONFIG", "")
	paths, err := ResolvePaths("", "")
	if err != nil {
		t.Fatalf("ResolvePaths: %v", err)
	}
	if paths.WorkDir != dir {
		t.Fatalf("WorkDir got %q want %q", paths.WorkDir, dir)
	}
	if paths.ConfigPath != filepath.Join(dir, "config.json") {
		t.Fatalf("ConfigPath got %q", paths.ConfigPath)
	}
}

func TestResolveAppPathsUsesTCPMConfigEnv(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "env-config.json")
	t.Setenv("TCPM_DATA_DIR", "")
	t.Setenv("TCPM_CONFIG", cfgPath)
	paths, err := ResolvePaths("", "")
	if err != nil {
		t.Fatalf("ResolvePaths: %v", err)
	}
	if paths.WorkDir != dir {
		t.Fatalf("WorkDir got %q want %q", paths.WorkDir, dir)
	}
	if paths.ConfigPath != cfgPath {
		t.Fatalf("ConfigPath got %q want %q", paths.ConfigPath, cfgPath)
	}
}

func TestResolveAppPathsPrefersCwdWhenConfigJSONExists(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"username":"x"}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Setenv("TCPM_DATA_DIR", "")
	t.Setenv("TCPM_CONFIG", "")

	paths, err := ResolvePaths("", "")
	if err != nil {
		t.Fatalf("ResolvePaths: %v", err)
	}
	if paths.WorkDir != dir {
		t.Fatalf("WorkDir got %q want %q", paths.WorkDir, dir)
	}
	if paths.ConfigPath != filepath.Join(dir, "config.json") {
		t.Fatalf("ConfigPath got %q", paths.ConfigPath)
	}
}

func TestShouldFallbackToUserConfig(t *testing.T) {
	if ShouldFallbackToUserConfig(nil) {
		t.Fatalf("nil error should not fallback")
	}
	if !ShouldFallbackToUserConfig(os.ErrPermission) {
		t.Fatalf("permission error should fallback")
	}
	if !ShouldFallbackToUserConfig(syscall.EROFS) {
		t.Fatalf("EROFS should fallback")
	}
	if ShouldFallbackToUserConfig(errors.New("other")) {
		t.Fatalf("unrelated error should not fallback")
	}
}

func TestLoadOrCreateConfigRejectsInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte(`{not-json`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := LoadOrCreate(path); err == nil {
		t.Fatalf("expected invalid JSON error")
	}
}

func TestLoadOrCreateConfigMergesMissingNestedKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "partial.json")
	partial := map[string]interface{}{
		"username": "tester",
		"bet": map[string]interface{}{
			"percentage": 9,
			// filter_condition intentionally missing
		},
		"privacy": map[string]interface{}{
			// anonymize_logs intentionally missing
		},
	}
	raw, err := json.Marshal(partial)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	cfg, err := LoadOrCreate(path)
	if err != nil {
		t.Fatalf("LoadOrCreate: %v", err)
	}
	if cfg.Username != "tester" {
		t.Fatalf("username got %q", cfg.Username)
	}
	if cfg.Bet.Percentage == nil || *cfg.Bet.Percentage != 9 {
		t.Fatalf("percentage override lost: %#v", cfg.Bet.Percentage)
	}

	rewritten, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read rewritten: %v", err)
	}
	var rewrittenMap map[string]interface{}
	if err := json.Unmarshal(rewritten, &rewrittenMap); err != nil {
		t.Fatalf("unmarshal rewritten: %v", err)
	}
	bet, ok := rewrittenMap["bet"].(map[string]interface{})
	if !ok {
		t.Fatalf("bet missing in rewritten file")
	}
	fc, ok := bet["filter_condition"].(map[string]interface{})
	if !ok {
		t.Fatalf("filter_condition should be filled by defaults")
	}
	for _, key := range []string{"by", "where", "value"} {
		if _, ok := fc[key]; !ok {
			t.Fatalf("filter_condition missing key %q", key)
		}
	}
	privacy, ok := rewrittenMap["privacy"].(map[string]interface{})
	if !ok {
		t.Fatalf("privacy missing")
	}
	if _, ok := privacy["anonymize_logs"]; !ok {
		t.Fatalf("privacy.anonymize_logs should be filled")
	}
	discord, ok := rewrittenMap["discord"].(map[string]interface{})
	if !ok {
		t.Fatalf("discord missing")
	}
	if _, ok := discord["webhook_api"]; !ok {
		t.Fatalf("discord.webhook_api should be filled")
	}
}

func TestMergeBetSettingsOverridesAndDefaults(t *testing.T) {
	pct := 12
	gap := 7
	maxPts := 1000
	minPts := 50
	stealth := true
	deduct := false
	delay := 3.5
	value := 25.0
	base := streamer.BetSettings{}
	base.Default()

	out := mergeBetSettings(base, BetConfig{
		Strategy:           "PERCENTAGE",
		Percentage:         &pct,
		PercentageGap:      &gap,
		MaxPoints:          &maxPts,
		MinimumPoints:      &minPts,
		StealthMode:        &stealth,
		DeductStakeOnPlace: &deduct,
		DelayMode:          "FROM_START",
		Delay:              &delay,
		FilterCondition: &FilterConditionConfig{
			By:    "TOTAL_USERS",
			Where: "GTE",
			Value: &value,
		},
	})

	if out.Strategy != streamer.StrategyPercentage {
		t.Fatalf("strategy got %s", out.Strategy)
	}
	if out.Percentage == nil || *out.Percentage != 12 {
		t.Fatalf("percentage got %#v", out.Percentage)
	}
	if out.PercentageGap == nil || *out.PercentageGap != 7 {
		t.Fatalf("percentage_gap got %#v", out.PercentageGap)
	}
	if out.MaxPoints == nil || *out.MaxPoints != 1000 {
		t.Fatalf("max_points got %#v", out.MaxPoints)
	}
	if out.MinimumPoints == nil || *out.MinimumPoints != 50 {
		t.Fatalf("minimum_points got %#v", out.MinimumPoints)
	}
	if out.StealthMode == nil || !*out.StealthMode {
		t.Fatalf("stealth_mode got %#v", out.StealthMode)
	}
	if out.DeductStakeOnPlace == nil || *out.DeductStakeOnPlace {
		t.Fatalf("deduct_stake_on_place got %#v", out.DeductStakeOnPlace)
	}
	if out.DelayMode != streamer.DelayModeFromStart {
		t.Fatalf("delay_mode got %s", out.DelayMode)
	}
	if out.Delay == nil || *out.Delay != 3.5 {
		t.Fatalf("delay got %#v", out.Delay)
	}
	if out.FilterCondition == nil || out.FilterCondition.By != streamer.OutcomeTotalUsers {
		t.Fatalf("filter by got %#v", out.FilterCondition)
	}
	if out.FilterCondition.Where != streamer.ConditionGTE {
		t.Fatalf("filter where got %s", out.FilterCondition.Where)
	}
	if out.FilterCondition.Value == nil || *out.FilterCondition.Value != 25 {
		t.Fatalf("filter value got %#v", out.FilterCondition.Value)
	}
}

func TestMergeStreamerSettingsAppliesFeatureFlags(t *testing.T) {
	base := streamer.StreamerSettings{
		MakePredictions: true,
		FollowRaid:      true,
		ClaimDrops:      true,
		ClaimMoments:    true,
		WatchStreak:     true,
		CommunityGoals:  false,
		IRCMode:         streamer.IRCModeOnline,
	}
	base.Default()

	makePred := false
	follow := false
	drops := false
	moments := false
	streak := false
	goals := true
	out := mergeStreamerSettings(base, StreamerSettingsConfig{
		MakePredictions: &makePred,
		FollowRaid:      &follow,
		ClaimDrops:      &drops,
		ClaimMoments:    &moments,
		WatchStreak:     &streak,
		CommunityGoals:  &goals,
	})
	if out.MakePredictions || out.FollowRaid || out.ClaimDrops || out.ClaimMoments || out.WatchStreak {
		t.Fatalf("expected feature flags overridden off, got %#v", out)
	}
	if !out.CommunityGoals {
		t.Fatalf("community goals should be enabled")
	}
}

func TestDefaultConfigWarmStartAndClaimDefaults(t *testing.T) {
	cfg := DefaultMap()
	if got, ok := cfg["watch_streak_warm_start_cache"].(bool); !ok || !got {
		t.Fatalf("watch_streak_warm_start_cache default got %#v want true", cfg["watch_streak_warm_start_cache"])
	}
	if got, ok := cfg["claim_drops_startup"].(bool); !ok || !got {
		t.Fatalf("claim_drops_startup default got %#v want true", cfg["claim_drops_startup"])
	}
	if got, ok := cfg["betting(make_predictions)"].(bool); !ok || !got {
		t.Fatalf("betting default got %#v want true", cfg["betting(make_predictions)"])
	}
}

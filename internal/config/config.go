package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"TwitchChannelPointsMiner/internal/constants"
	"TwitchChannelPointsMiner/internal/notify"
	"TwitchChannelPointsMiner/internal/persistence"
	"TwitchChannelPointsMiner/internal/streamer"
)

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info != nil && !info.IsDir()
}

func isGoRunExecutable(path string) bool {
	lower := strings.ToLower(path)
	if strings.Contains(lower, "go-build") {
		return true
	}
	temp := strings.ToLower(os.TempDir())
	return strings.HasPrefix(lower, temp)
}

type Paths struct {
	WorkDir    string
	ConfigPath string
}

func ResolvePaths(configFlag, dataDirFlag string) (Paths, error) {
	if dataDirFlag != "" {
		abs, err := filepath.Abs(dataDirFlag)
		if err != nil {
			return Paths{}, err
		}
		return Paths{
			WorkDir:    abs,
			ConfigPath: filepath.Join(abs, "config.json"),
		}, nil
	}

	if configFlag != "" {
		abs, err := filepath.Abs(configFlag)
		if err != nil {
			return Paths{}, err
		}
		return Paths{
			WorkDir:    filepath.Dir(abs),
			ConfigPath: abs,
		}, nil
	}

	// ? Environment overrides (useful for non-interactive launches).
	if raw := strings.TrimSpace(os.Getenv("TCPM_DATA_DIR")); raw != "" {
		abs, err := filepath.Abs(raw)
		if err != nil {
			return Paths{}, err
		}
		return Paths{
			WorkDir:    abs,
			ConfigPath: filepath.Join(abs, "config.json"),
		}, nil
	}

	if raw := strings.TrimSpace(os.Getenv("TCPM_CONFIG")); raw != "" {
		abs, err := filepath.Abs(raw)
		if err != nil {
			return Paths{}, err
		}
		return Paths{
			WorkDir:    filepath.Dir(abs),
			ConfigPath: abs,
		}, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		cwd = ""
	}

	// ? Preserve the historical behavior when the user runs from a folder that already has config.json.
	if cwd != "" && fileExists(filepath.Join(cwd, "config.json")) {
		return Paths{
			WorkDir:    cwd,
			ConfigPath: filepath.Join(cwd, "config.json"),
		}, nil
	}

	exePath, err := os.Executable()
	if err == nil && exePath != "" && !isGoRunExecutable(exePath) {
		exeDir := filepath.Dir(exePath)
		return Paths{
			WorkDir:    exeDir,
			ConfigPath: filepath.Join(exeDir, "config.json"),
		}, nil
	}

	// ? Fallback (dev runs, restricted environments).
	if cwd == "" {
		cwd = "."
	}
	return Paths{
		WorkDir:    cwd,
		ConfigPath: filepath.Join(cwd, "config.json"),
	}, nil
}

func ShouldFallbackToUserConfig(err error) bool {
	if err == nil {
		return false
	}
	if os.IsPermission(err) {
		return true
	}
	return errors.Is(err, syscall.EROFS)
}

type FilterConditionConfig struct {
	By    string   `json:"by"`
	Where string   `json:"where"`
	Value *float64 `json:"value"`
}

type BetConfig struct {
	Strategy           string                 `json:"strategy"`
	Percentage         *int                   `json:"percentage"`
	PercentageGap      *int                   `json:"percentage_gap"`
	MaxPoints          *int                   `json:"max_points"`
	StealthMode        *bool                  `json:"stealth_mode"`
	DeductStakeOnPlace *bool                  `json:"deduct_stake_on_place"`
	DelayMode          string                 `json:"delay_mode"`
	Delay              *float64               `json:"delay"`
	MinimumPoints      *int                   `json:"minimum_points"`
	FilterCondition    *FilterConditionConfig `json:"filter_condition"`
}

type StreamerSettingsConfig struct {
	MakePredictions *bool     `json:"make_predictions"`
	FollowRaid      *bool     `json:"follow_raid"`
	ClaimDrops      *bool     `json:"claim_drops"`
	ClaimMoments    *bool     `json:"claim_moments"`
	WatchStreak     *bool     `json:"watch_streak"`
	CommunityGoals  *bool     `json:"community_goals"`
	Bet             BetConfig `json:"bet"`
	IRCMode         *string   `json:"chat_presence"`
}

type PrivacyConfig struct {
	AnonymizeLogs bool `json:"anonymize_logs"`
}

type DiscordConfig struct {
	WebhookAPI string   `json:"webhook_api"`
	Events     []string `json:"events"`
}

type Config struct {
	Username                   string        `json:"username"`
	Password                   string        `json:"password"`
	AutoUpdate                 bool          `json:"auto_update"`
	Debug                      bool          `json:"debug"`
	DebugDeep                  bool          `json:"debug_deep"`
	WatchQueueLogging          bool          `json:"watch_queue_logging"`
	SmartLogging               bool          `json:"smart_logging"`
	DisableSSLCertVerification bool          `json:"disable_ssl_cert_verification"`
	ShowSeconds                bool          `json:"show_seconds"`
	ClaimDropsStartup          bool          `json:"claim_drops_startup"`
	ClaimDrops                 bool          `json:"claim_drops"`
	ClaimMoments               bool          `json:"claim_moments"`
	BettingMakePredictions     bool          `json:"betting(make_predictions)"`
	FollowRaid                 bool          `json:"follow_raid"`
	CommunityGoals             bool          `json:"community_goals"`
	Emojis                     bool          `json:"emojis"`
	SaveLogs                   bool          `json:"save_logs"`
	ShowUsernameInConsole      bool          `json:"show_username_in_console"`
	ShowClaimedBonusMsg        bool          `json:"show_claimed_bonus_msg"`
	ShowGame                   bool          `json:"show_game"`
	WatchStreakWarmStartCache  bool          `json:"watch_streak_warm_start_cache"`
	IRCMode                    string        `json:"chat_presence"`
	DisableAtInNickname        bool          `json:"disable_at_in_nickname"`
	ShowDropsProgress          bool          `json:"show_drops_progress"`
	Streamers                  []string      `json:"streamers"`
	StreamersExclude           []string      `json:"streamers_exclude"`
	GamePriority               []string      `json:"game_priority"`
	GameExclude                []string      `json:"game_exclude"`
	WatchPriority              []string      `json:"watch_priority"`
	Bet                        BetConfig     `json:"bet"`
	Timezone                   *string       `json:"timezone"`
	Privacy                    PrivacyConfig `json:"privacy"`
	Discord                    DiscordConfig `json:"discord"`

	StreamerOverrides map[string]StreamerSettingsConfig `json:"streamer_overrides"`
}

func mergeBetSettings(base streamer.BetSettings, override BetConfig) streamer.BetSettings {
	out := base
	if override.Strategy != "" {
		out.Strategy = streamer.Strategy(override.Strategy)
	}
	if override.Percentage != nil {
		out.Percentage = override.Percentage
	}
	if override.PercentageGap != nil {
		out.PercentageGap = override.PercentageGap
	}
	if override.MaxPoints != nil {
		out.MaxPoints = override.MaxPoints
	}
	if override.MinimumPoints != nil {
		out.MinimumPoints = override.MinimumPoints
	}
	if override.StealthMode != nil {
		out.StealthMode = override.StealthMode
	}
	if override.DeductStakeOnPlace != nil {
		out.DeductStakeOnPlace = override.DeductStakeOnPlace
	}
	if override.FilterCondition != nil {
		out.FilterCondition = mergeFilterCondition(out.FilterCondition, override.FilterCondition)
	}
	if override.DelayMode != "" {
		out.DelayMode = streamer.DelayMode(override.DelayMode)
	}
	if override.Delay != nil {
		out.Delay = override.Delay
	}
	out.Default()
	return out
}

func mergeStreamerSettings(base streamer.StreamerSettings, override StreamerSettingsConfig) streamer.StreamerSettings {
	out := base
	if override.MakePredictions != nil {
		out.MakePredictions = *override.MakePredictions
	}
	if override.FollowRaid != nil {
		out.FollowRaid = *override.FollowRaid
	}
	if override.ClaimDrops != nil {
		out.ClaimDrops = *override.ClaimDrops
	}
	if override.ClaimMoments != nil {
		out.ClaimMoments = *override.ClaimMoments
	}
	if override.WatchStreak != nil {
		out.WatchStreak = *override.WatchStreak
	}
	if override.CommunityGoals != nil {
		out.CommunityGoals = *override.CommunityGoals
	}
	out.Bet = mergeBetSettings(out.Bet, override.Bet)
	if override.IRCMode != nil {
		out.IRCMode = parseChatPresence(*override.IRCMode, out.IRCMode)
	}
	out.Default()
	return out
}

func parseChatPresence(mode string, fallback streamer.IRCMode) streamer.IRCMode {
	switch strings.ToUpper(strings.TrimSpace(mode)) {
	case string(streamer.IRCModeAlways):
		return streamer.IRCModeAlways
	case string(streamer.IRCModeNever):
		return streamer.IRCModeNever
	case string(streamer.IRCModeOffline):
		return streamer.IRCModeOffline
	case string(streamer.IRCModeOnline):
		return streamer.IRCModeOnline
	default:
		return fallback
	}
}

func mergeFilterCondition(base *streamer.FilterCondition, override *FilterConditionConfig) *streamer.FilterCondition {
	if override == nil {
		return base
	}
	var out streamer.FilterCondition
	if base != nil {
		out = *base
	}
	if override.By != "" {
		out.By = streamer.OutcomeKey(strings.ToUpper(strings.TrimSpace(override.By)))
	}
	if override.Where != "" {
		out.Where = streamer.Condition(strings.ToUpper(strings.TrimSpace(override.Where)))
	}
	if override.Value != nil {
		out.Value = override.Value
	}
	// ? If nothing was set, keep nil to avoid activating an empty filter
	if out.By == "" && out.Where == "" && out.Value == nil {
		return base
	}
	return &out
}

func DefaultMap() map[string]interface{} {
	return map[string]interface{}{
		"username":                      "your-twitch-username",
		"password":                      "your-twitch-password (Optional)",
		"auto_update":                   true,
		"debug":                         false,
		"debug_deep":                    false,
		"watch_queue_logging":           false,
		"smart_logging":                 true,
		"disable_ssl_cert_verification": false,
		"show_seconds":                  false,
		"claim_drops_startup":           true,
		"claim_drops":                   true,
		"claim_moments":                 true,
		"betting(make_predictions)":     true,
		"follow_raid":                   true,
		"community_goals":               false,
		"emojis":                        true,
		"save_logs":                     false,
		"show_username_in_console":      false,
		"show_claimed_bonus_msg":        true,
		"show_game":                     true,
		"watch_streak_warm_start_cache": true,
		"chat_presence":                 "ONLINE",
		"disable_at_in_nickname":        false,
		"show_drops_progress":           false,
		"timezone":                      nil,
		"privacy": map[string]interface{}{
			"anonymize_logs": false,
		},
		"discord": map[string]interface{}{
			"webhook_api": "",
			"events":      []interface{}{},
		},
		"streamers":         []interface{}{},
		"streamers_exclude": []interface{}{},
		"game_priority":     []interface{}{},
		"game_exclude":      []interface{}{},
		"watch_priority": []interface{}{
			"STREAK",
			"DROPS",
			"ORDER",
		},
		"streamer_overrides": map[string]interface{}{},
		"bet": map[string]interface{}{
			"strategy":              nil,
			"percentage":            nil,
			"percentage_gap":        nil,
			"max_points":            nil,
			"stealth_mode":          nil,
			"deduct_stake_on_place": true,
			"delay_mode":            nil,
			"delay":                 nil,
			"minimum_points":        nil,
			"filter_condition": map[string]interface{}{
				"by":    nil,
				"where": nil,
				"value": nil,
			},
		},
	}
}

func LoadOrCreate(path string) (Config, error) {
	cfgMap := map[string]interface{}{}
	fileData, err := os.ReadFile(path)
	if err == nil {
		if err := json.Unmarshal(fileData, &cfgMap); err != nil {
			return Config{}, fmt.Errorf("invalid config: %w", err)
		}
	}

	changed := false
	for key, value := range DefaultMap() {
		if _, ok := cfgMap[key]; !ok {
			cfgMap[key] = value
			changed = true
		}
	}

	privacyRaw, ok := cfgMap["privacy"].(map[string]interface{})
	if !ok {
		privacyRaw = DefaultMap()["privacy"].(map[string]interface{})
		cfgMap["privacy"] = privacyRaw
		changed = true
	} else {
		defaultPrivacy := DefaultMap()["privacy"].(map[string]interface{})
		for k, v := range defaultPrivacy {
			if _, ok := privacyRaw[k]; !ok {
				privacyRaw[k] = v
				changed = true
			}
		}
	}

	discordRaw, ok := cfgMap["discord"].(map[string]interface{})
	if !ok {
		discordRaw = DefaultMap()["discord"].(map[string]interface{})
		cfgMap["discord"] = discordRaw
		changed = true
	} else {
		defaultDiscord := DefaultMap()["discord"].(map[string]interface{})
		for k, v := range defaultDiscord {
			if _, ok := discordRaw[k]; !ok {
				discordRaw[k] = v
				changed = true
			}
		}
	}

	betRaw, ok := cfgMap["bet"].(map[string]interface{})
	if !ok {
		betRaw = DefaultMap()["bet"].(map[string]interface{})
		cfgMap["bet"] = betRaw
		changed = true
	} else {
		defaultBet := DefaultMap()["bet"].(map[string]interface{})
		for k, v := range defaultBet {
			if _, ok := betRaw[k]; !ok {
				betRaw[k] = v
				changed = true
			}
		}
		// ? Ensure nested filter_condition keys are present.
		if fcRaw, ok := betRaw["filter_condition"].(map[string]interface{}); ok {
			for k, v := range defaultBet["filter_condition"].(map[string]interface{}) {
				if _, ok := fcRaw[k]; !ok {
					fcRaw[k] = v
					changed = true
				}
			}
		} else {
			betRaw["filter_condition"] = defaultBet["filter_condition"]
			changed = true
		}
	}

	if err != nil || changed {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return Config{}, err
		}
		if err := persistence.SaveJSON(path, cfgMap); err != nil {
			return Config{}, err
		}
	}

	normalized, err := json.Marshal(cfgMap)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(normalized, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func ApplyTimezoneOverride(raw *string, logger *notify.Logger) {
	if raw == nil {
		return
	}
	zone := strings.TrimSpace(*raw)
	if zone == "" || strings.EqualFold(zone, "auto") {
		return
	}
	loc, err := time.LoadLocation(zone)
	if err != nil {
		logger.Errorf("%sTimezone override ignored; falling back to system time: %v%s", constants.ColorRed, err, constants.ColorReset)
		return
	}
	time.Local = loc
}

func BuildBaseStreamerSettings(cfg Config) streamer.StreamerSettings {
	betSettings := streamer.BetSettings{
		Strategy:           streamer.Strategy(cfg.Bet.Strategy),
		Percentage:         cfg.Bet.Percentage,
		PercentageGap:      cfg.Bet.PercentageGap,
		MaxPoints:          cfg.Bet.MaxPoints,
		StealthMode:        cfg.Bet.StealthMode,
		DeductStakeOnPlace: cfg.Bet.DeductStakeOnPlace,
		DelayMode:          streamer.DelayMode(cfg.Bet.DelayMode),
		Delay:              cfg.Bet.Delay,
		MinimumPoints:      cfg.Bet.MinimumPoints,
		FilterCondition:    mergeFilterCondition(nil, cfg.Bet.FilterCondition),
	}
	betSettings.Default()

	streamerSettings := streamer.StreamerSettings{
		MakePredictions: cfg.BettingMakePredictions,
		FollowRaid:      cfg.FollowRaid,
		ClaimDrops:      cfg.ClaimDrops,
		ClaimMoments:    cfg.ClaimMoments,
		WatchStreak:     true,
		CommunityGoals:  cfg.CommunityGoals,
		Bet:             betSettings,
		IRCMode:         parseChatPresence(cfg.IRCMode, streamer.IRCModeOnline),
	}
	streamerSettings.Default()
	return streamerSettings
}

func BuildOverrideSettings(base streamer.StreamerSettings, overrides map[string]StreamerSettingsConfig) map[string]streamer.StreamerSettings {
	overrideSettings := make(map[string]streamer.StreamerSettings, len(overrides))
	for name, override := range overrides {
		key := strings.ToLower(strings.TrimSpace(name))
		if key == "" {
			continue
		}
		overrideSettings[key] = mergeStreamerSettings(base, override)
	}
	return overrideSettings
}

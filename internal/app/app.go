package app

import (
	"context"
	"log"

	"TwitchChannelPointsMiner/internal/config"
	"TwitchChannelPointsMiner/internal/miner"
	"TwitchChannelPointsMiner/internal/notify"
	"TwitchChannelPointsMiner/internal/streamer"
	"TwitchChannelPointsMiner/internal/update"
)

// Run starts the miner lifecycle for the given config. Parent owns signal handling via ctx.
func Run(ctx context.Context, cfg config.Config) error {
	if cfg.AutoUpdate {
		updated, err := update.RunAutoUpdate()
		if err != nil {
			log.Printf("auto-update failed: %v", err)
		}
		if updated {
			log.Printf("auto-update installed a newer version; restarting...")
			return nil
		}
	}

	baseStreamerSettings := config.BuildBaseStreamerSettings(cfg)
	overrideSettings := config.BuildOverrideSettings(baseStreamerSettings, cfg.StreamerOverrides)

	loggerSettings := notify.LoggerSettings{
		Save:             cfg.SaveLogs,
		ConsoleLevel:     0,
		FileLevel:        0,
		Emoji:            cfg.Emojis,
		Smart:            cfg.SmartLogging,
		ShowSeconds:      cfg.ShowSeconds,
		ConsoleUsername:  cfg.ShowUsernameInConsole,
		ShowClaimedBonus: cfg.ShowClaimedBonusMsg,
		Less:             false,
		Debug:            cfg.Debug,
		DebugDeep:        cfg.DebugDeep,
		AnonymizeLogs:    cfg.Privacy.AnonymizeLogs,
		Discord: notify.DiscordSettings{
			WebhookAPI: cfg.Discord.WebhookAPI,
			Events:     cfg.Discord.Events,
		},
	}

	logger := notify.NewLogger(loggerSettings, cfg.Username)
	config.ApplyTimezoneOverride(cfg.Timezone, logger)

	minr := miner.NewMiner(
		cfg.Username,
		cfg.Password,
		cfg.ClaimDropsStartup,
		cfg.DisableSSLCertVerification,
		loggerSettings,
		baseStreamerSettings,
		overrideSettings,
		cfg.WatchPriority,
		cfg.StreamersExclude,
		cfg.GamePriority,
		cfg.GameExclude,
		cfg.DisableAtInNickname,
		cfg.ShowGame,
		cfg.WatchQueueLogging,
		cfg.WatchStreakWarmStartCache,
		cfg.ShowDropsProgress,
	)

	if len(cfg.Streamers) > 0 {
		return minr.Mine(ctx, cfg.Streamers)
	}
	return minr.MineFollowers(ctx, streamer.FollowersOrderDESC)
}

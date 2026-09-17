package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"

	"TwitchChannelPointsMiner/internal/config"
	"TwitchChannelPointsMiner/internal/miner"
	"TwitchChannelPointsMiner/internal/notify"
	"TwitchChannelPointsMiner/internal/streamer"
	"TwitchChannelPointsMiner/internal/update"
)

func clearConsole() {
	var c *exec.Cmd
	if runtime.GOOS == "windows" {
		c = exec.Command("cmd", "/c", "cls")
	} else {
		c = exec.Command("clear")
	}
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	_ = c.Run()
}

func setConsoleTitle(title string) {
	if runtime.GOOS != "windows" {
		return
	}
	cmd := exec.Command("cmd", "/c", fmt.Sprintf("title %s", title))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
}

// Run starts the miner lifecycle for the given config. Parent owns signal handling via ctx.
func Run(ctx context.Context, cfg config.Config) error {
	setConsoleTitle("Klaro's Twitch Miner")
	clearConsole()

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

	// Stage B: miner still owns signals; ctx reserved for Stage C wiring.
	_ = ctx
	if len(cfg.Streamers) > 0 {
		minr.Mine(cfg.Streamers)
	} else {
		minr.MineFollowers(streamer.FollowersOrderDESC)
	}
	return nil
}

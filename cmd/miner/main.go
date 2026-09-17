package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"TwitchChannelPointsMiner/internal/app"
	"TwitchChannelPointsMiner/internal/config"
)

func main() {
	configFlag := flag.String("config", "", "Path to config.json (default: ./config.json or next to the executable)")
	dataDirFlag := flag.String("data-dir", "", "Directory for config/cookies/log (default: current directory if config.json exists; otherwise the executable directory)")
	flag.Parse()

	hasOverride := *configFlag != "" || *dataDirFlag != "" || strings.TrimSpace(os.Getenv("TCPM_CONFIG")) != "" || strings.TrimSpace(os.Getenv("TCPM_DATA_DIR")) != ""
	paths, err := config.ResolvePaths(*configFlag, *dataDirFlag)
	if err != nil {
		log.Fatalf("failed to resolve config paths: %v", err)
	}
	if paths.WorkDir != "" {
		_ = os.MkdirAll(paths.WorkDir, 0o755)
		if err := os.Chdir(paths.WorkDir); err != nil {
			log.Printf("warning: failed to change working directory to %q: %v", paths.WorkDir, err)
		}
	}

	cfg, err := config.LoadOrCreate(paths.ConfigPath)
	if err != nil && !hasOverride && config.ShouldFallbackToUserConfig(err) {
		if base, derr := os.UserConfigDir(); derr == nil && base != "" {
			fallbackDir := filepath.Join(base, "TwitchChannelPointsMiner")
			_ = os.MkdirAll(fallbackDir, 0o755)
			if chErr := os.Chdir(fallbackDir); chErr == nil {
				fallbackCfg := filepath.Join(fallbackDir, "config.json")
				if cfg2, err2 := config.LoadOrCreate(fallbackCfg); err2 == nil {
					cfg = cfg2
					err = nil
					log.Printf("using config directory %q", fallbackDir)
				}
			}
		}
	}
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := app.Run(ctx, cfg); err != nil {
		log.Fatal(err)
	}
}

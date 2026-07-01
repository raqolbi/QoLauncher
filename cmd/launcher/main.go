package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/raqolbi/qolauncher/internal/config"
	"github.com/raqolbi/qolauncher/internal/logger"
	"github.com/raqolbi/qolauncher/internal/logwriter"
	"github.com/raqolbi/qolauncher/internal/rotator"
	"github.com/raqolbi/qolauncher/internal/supervisor"
	"github.com/raqolbi/qolauncher/internal/viewer"
)

var (
	version   = "0.1.0-dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	cfg := config.Default()
	if err := config.ApplyEnvFromOS(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	opts, err := config.ParseFlags(args, cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	if opts.Help() {
		config.Usage(os.Stdout)
		return 0
	}

	if opts.Version() {
		fmt.Printf("QoLauncher v%s (commit %s, built %s)\n", version, commit, buildDate)
		return 0
	}

	if opts.Config() {
		if err := cfg.Validate(); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return 1
		}
		cfg.Print(os.Stdout)
		return 0
	}

	if err := cfg.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	return runLauncher(cfg)
}

func runLauncher(cfg *config.Config) int {
	log := logger.New(cfg.LogLevel, os.Stderr)
	events := logger.NewSupervisor(log)
	events.LauncherStarted(version, map[string]any{"commit": commit})

	if err := rotator.Sweep(cfg.LogDir, cfg.LogRetentionDays, cfg.TZ); err != nil {
		log.Error("log retention sweep failed", map[string]any{"error": err.Error()})
		return 1
	}

	writer, err := logwriter.New(cfg.LogDir, cfg.TZ)
	if err != nil {
		log.Error("log writer init failed", map[string]any{"error": err.Error()})
		return 1
	}
	defer writer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rotator.StartPeriodic(ctx, 24*time.Hour, cfg.LogDir, cfg.LogRetentionDays, cfg.TZ)

	log.Info("logging initialized", map[string]any{"log_dir": cfg.LogDir})

	var viewServer *viewer.Server
	if cfg.ViewerEnabled {
		viewServer = viewer.New(cfg)
		if err := viewServer.Start(); err != nil {
			log.Error("log viewer start failed", map[string]any{"error": err.Error(), "port": cfg.LogPort})
			return 1
		}
		defer func() {
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			if err := viewServer.Shutdown(shutdownCtx); err != nil {
				log.Warn("log viewer shutdown error", map[string]any{"error": err.Error()})
			}
		}()
		log.Info("log viewer started", map[string]any{"port": cfg.LogPort})
	}

	sup := supervisor.New(cfg, events, writer)
	return sup.Run(ctx)
}

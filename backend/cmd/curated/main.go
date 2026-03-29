package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"go.uber.org/zap"

	"curated-backend/internal/app"
	"curated-backend/internal/config"
	"curated-backend/internal/logging"
	"curated-backend/internal/server"
	"curated-backend/internal/storage"
	"curated-backend/internal/version"
)

func main() {
	configPath := flag.String("config", "", "Path to backend config file")
	mode := flag.String("mode", "http", "Run mode: http (default), stdio, or both")
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cfg, err := config.Load(*configPath)
	if err != nil {
		exitWithInitError("failed to load config", err)
	}

	librarySettingsPath := config.DefaultLibrarySettingsPath()
	if err := config.MergeLibrarySettingsFile(&cfg, librarySettingsPath); err != nil {
		exitWithInitError("failed to merge library settings file", err)
	}

	logger, err := logging.New(cfg.LogLevel, logging.FileSink{
		LogDir:     cfg.LogDir,
		FilePrefix: cfg.LogFilePrefix,
		MaxAgeDays: cfg.LogMaxAgeDays,
	})
	if err != nil {
		exitWithInitError("failed to initialize logger", err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	startupFields := []zap.Field{
		zap.String("buildStamp", version.Stamp()),
		zap.String("channel", version.Channel),
		zap.String("httpAddr", cfg.HttpAddr),
		zap.String("databasePath", cfg.DatabasePath),
		zap.Int("libraryPathsConfigured", len(cfg.LibraryPaths)),
		zap.Bool("libraryWatchEnabled", cfg.LibraryWatchOn()),
		zap.Int("autoScanIntervalSeconds", cfg.AutoScanIntervalSeconds),
	}
	if strings.TrimSpace(cfg.LogDir) != "" {
		startupFields = append(startupFields, zap.String("logDir", cfg.LogDir))
	}
	if cfg.Proxy.Enabled && strings.TrimSpace(cfg.Proxy.URL) != "" {
		if u, perr := url.Parse(strings.TrimSpace(cfg.Proxy.URL)); perr == nil {
			startupFields = append(startupFields, zap.String("proxyURL", u.Redacted()))
		} else {
			startupFields = append(startupFields, zap.Bool("proxyEnabled", true))
		}
	} else {
		startupFields = append(startupFields, zap.Bool("proxyEnabled", false))
	}
	logger.Info("Curated backend starting", startupFields...)

	store, err := storage.NewSQLiteStore(cfg.DatabasePath)
	if err != nil {
		logger.Fatal("failed to open sqlite database", logging.Error(err))
	}
	defer func() {
		_ = store.Close()
	}()

	if err := store.Migrate(ctx); err != nil {
		logger.Fatal("failed to run sqlite migrations", logging.Error(err))
	}

	if err := store.SeedLibraryPathsIfEmpty(ctx, cfg.LibraryPaths); err != nil {
		logger.Fatal("failed to seed library paths", logging.Error(err))
	}

	backendApp, err := app.New(ctx, cfg, logger, store, librarySettingsPath)
	if err != nil {
		logger.Fatal("failed to initialize backend app", logging.Error(err))
	}

	backendApp.StartAutoScanLoop(ctx)

	if cfg.LibraryWatchOn() {
		if werr := backendApp.EnsureLibraryWatchRunning(); werr != nil {
			logger.Fatal("failed to init library fsnotify watcher", logging.Error(werr))
		}
	}

	switch *mode {
	case "http":
		if err := server.ListenAndServe(ctx, cfg.HttpAddr, backendApp.HTTPHandler(), logger); err != nil {
			logger.Fatal("HTTP server error", logging.Error(err))
		}
	case "stdio":
		if err := backendApp.Run(ctx, os.Stdin, os.Stdout); err != nil && !errors.Is(err, context.Canceled) {
			logger.Fatal("backend exited with error", logging.Error(err))
		}
	case "both":
		go func() {
			if err := server.ListenAndServe(ctx, cfg.HttpAddr, backendApp.HTTPHandler(), logger); err != nil {
				logger.Fatal("HTTP server error", logging.Error(err))
			}
		}()
		if err := backendApp.Run(ctx, os.Stdin, os.Stdout); err != nil && !errors.Is(err, context.Canceled) {
			logger.Fatal("backend exited with error", logging.Error(err))
		}
	default:
		exitWithInitError("unknown mode", fmt.Errorf("%q (expected http, stdio, or both)", *mode))
	}
}

func exitWithInitError(message string, err error) {
	_, _ = fmt.Fprintf(os.Stderr, "%s: %v\n", message, err)
	os.Exit(1)
}

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"jav-shadcn/backend/internal/app"
	"jav-shadcn/backend/internal/config"
	"jav-shadcn/backend/internal/logging"
	"jav-shadcn/backend/internal/server"
	"jav-shadcn/backend/internal/storage"
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

	logger, err := logging.New(cfg.LogLevel)
	if err != nil {
		exitWithInitError("failed to initialize logger", err)
	}
	defer func() {
		_ = logger.Sync()
	}()

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

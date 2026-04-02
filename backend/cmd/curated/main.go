package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"go.uber.org/zap"

	"curated-backend/internal/app"
	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/desktop"
	"curated-backend/internal/logging"
	"curated-backend/internal/server"
	"curated-backend/internal/shellopen"
	"curated-backend/internal/storage"
	"curated-backend/internal/version"
)

type bootstrap struct {
	cfg        config.Config
	logger     *zap.Logger
	store      *storage.SQLiteStore
	backendApp *app.App
	cleanup    func()
}

func main() {
	defaultMode := "http"
	if runtime.GOOS == "windows" && version.Channel == "release" {
		defaultMode = "tray"
	}

	configPath := flag.String("config", "", "Path to backend config file")
	mode := flag.String("mode", defaultMode, "Run mode: http, stdio, both, or tray")
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	boot, err := initialize(ctx, *configPath)
	if err != nil {
		exitWithInitError("failed to initialize backend", err)
	}
	defer boot.cleanup()

	switch *mode {
	case "http":
		if err := runHTTP(ctx, boot); err != nil {
			boot.logger.Fatal("HTTP server error", logging.Error(err))
		}
	case "stdio":
		if err := boot.backendApp.Run(ctx, os.Stdin, os.Stdout); err != nil && !errors.Is(err, context.Canceled) {
			boot.logger.Fatal("backend exited with error", logging.Error(err))
		}
	case "both":
		go func() {
			if err := runHTTP(ctx, boot); err != nil && !errors.Is(err, context.Canceled) {
				boot.logger.Fatal("HTTP server error", logging.Error(err))
			}
		}()
		if err := boot.backendApp.Run(ctx, os.Stdin, os.Stdout); err != nil && !errors.Is(err, context.Canceled) {
			boot.logger.Fatal("backend exited with error", logging.Error(err))
		}
	case "tray":
		if err := runTrayMode(ctx, cancel, boot); err != nil && !errors.Is(err, context.Canceled) {
			desktop.ShowErrorDialog("Curated startup failed", err.Error())
			boot.logger.Fatal("tray mode error", logging.Error(err))
		}
	default:
		exitWithInitError("unknown mode", fmt.Errorf("%q (expected http, stdio, both, or tray)", *mode))
	}
}

func initialize(ctx context.Context, configPath string) (*bootstrap, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	librarySettingsPath := config.DefaultLibrarySettingsPath()
	if err := config.MergeLibrarySettingsFile(&cfg, librarySettingsPath); err != nil {
		return nil, fmt.Errorf("merge library settings file: %w", err)
	}

	logger, err := logging.New(cfg.LogLevel, logging.FileSink{
		LogDir:     cfg.LogDir,
		FilePrefix: cfg.LogFilePrefix,
		MaxAgeDays: cfg.LogMaxAgeDays,
	})
	if err != nil {
		return nil, fmt.Errorf("initialize logger: %w", err)
	}

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
		_ = logger.Sync()
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}
	if err := store.Migrate(ctx); err != nil {
		_ = store.Close()
		_ = logger.Sync()
		return nil, fmt.Errorf("run sqlite migrations: %w", err)
	}
	if err := store.SeedLibraryPathsIfEmpty(ctx, cfg.LibraryPaths); err != nil {
		_ = store.Close()
		_ = logger.Sync()
		return nil, fmt.Errorf("seed library paths: %w", err)
	}

	backendApp, err := app.New(ctx, cfg, logger, store, librarySettingsPath)
	if err != nil {
		_ = store.Close()
		_ = logger.Sync()
		return nil, fmt.Errorf("initialize backend app: %w", err)
	}

	backendApp.StartAutoScanLoop(ctx)
	if cfg.LibraryWatchOn() {
		if err := backendApp.EnsureLibraryWatchRunning(); err != nil {
			_ = store.Close()
			_ = logger.Sync()
			return nil, fmt.Errorf("init library fsnotify watcher: %w", err)
		}
	}

	return &bootstrap{
		cfg:        cfg,
		logger:     logger,
		store:      store,
		backendApp: backendApp,
		cleanup: func() {
			if backendApp != nil {
				backendApp.Close()
			}
			_ = store.Close()
			_ = logger.Sync()
		},
	}, nil
}

func runHTTP(ctx context.Context, boot *bootstrap) error {
	return server.ListenAndServe(ctx, boot.cfg.HttpAddr, boot.backendApp.HTTPHandler(), boot.logger)
}

func runTrayMode(parentCtx context.Context, cancel context.CancelFunc, boot *bootstrap) error {
	lock, primary, err := desktop.AcquireSingleInstance(version.TrayMutexName())
	if err != nil {
		return fmt.Errorf("single instance: %w", err)
	}
	defer func() {
		_ = lock.Release()
	}()

	baseURL := desktop.ResolveBaseURL(boot.cfg.HttpAddr)
	if !primary {
		return shellopen.OpenURL(context.Background(), baseURL)
	}

	serverCtx, serverCancel := context.WithCancel(parentCtx)
	defer serverCancel()

	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- runHTTP(serverCtx, boot)
	}()

	readyCtx, readyCancel := context.WithTimeout(parentCtx, 15*time.Second)
	defer readyCancel()
	if err := waitForServerOrExit(readyCtx, serverErrCh, baseURL); err != nil {
		serverCancel()
		return err
	}

	if err := desktop.RunTray(parentCtx, desktop.TrayOptions{
		Logger:      boot.logger,
		Config:      boot.cfg,
		OpenURL:     baseURL,
		SettingsURL: strings.TrimRight(baseURL, "/") + "/#/settings",
		LogDir:      desktop.ResolveDefaultLogDir(boot.cfg),
		Cancel: func() {
			serverCancel()
			cancel()
		},
	}); err != nil {
		serverCancel()
		return err
	}

	if err := shellopen.OpenURL(context.Background(), baseURL); err != nil && boot.logger != nil {
		boot.logger.Warn("tray: initial browser launch failed", zap.Error(err))
	}

	select {
	case err := <-serverErrCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			return err
		}
		return nil
	case <-parentCtx.Done():
		serverCancel()
		err := <-serverErrCh
		if err != nil && !errors.Is(err, context.Canceled) {
			return err
		}
		return parentCtx.Err()
	}
}

func waitForServerOrExit(ctx context.Context, serverErrCh <-chan error, baseURL string) error {
	readyCh := make(chan error, 1)
	go func() {
		readyCh <- desktop.WaitForServerReady(ctx, baseURL)
	}()

	select {
	case err := <-serverErrCh:
		if err == nil {
			return nil
		}
		if isAddrInUseError(err) {
			if ok := isExistingCuratedServer(ctx, baseURL); ok {
				if openErr := shellopen.OpenURL(context.Background(), baseURL); openErr != nil {
					return openErr
				}
				return nil
			}
			return fmt.Errorf("Curated could not start because %s is already in use by another application. Close the conflicting process or change Curated's httpAddr and try again", baseURL)
		}
		return err
	case err := <-readyCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func isAddrInUseError(err error) bool {
	if err == nil {
		return false
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return strings.Contains(strings.ToLower(opErr.Err.Error()), "only one usage of each socket address") ||
			strings.Contains(strings.ToLower(opErr.Err.Error()), "address already in use")
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "only one usage of each socket address") ||
		strings.Contains(text, "address already in use")
}

func isExistingCuratedServer(ctx context.Context, baseURL string) bool {
	reqCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/api/health", nil)
	if err != nil {
		return false
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false
	}

	var health contracts.HealthDTO
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(health.Name), version.BackendName())
}

func exitWithInitError(message string, err error) {
	_, _ = fmt.Fprintf(os.Stderr, "%s: %v\n", message, err)
	os.Exit(1)
}

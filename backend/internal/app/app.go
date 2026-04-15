package app

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"net/http"
	"net/url"

	"curated-backend/internal/assets"
	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/desktop"
	"curated-backend/internal/devmetrics"
	"curated-backend/internal/library"
	"curated-backend/internal/library/moviecode"
	"curated-backend/internal/library/movieroot"
	"curated-backend/internal/librarywatch"
	"curated-backend/internal/nativeplayer"
	"curated-backend/internal/playback"
	"curated-backend/internal/proxyenv"
	"curated-backend/internal/scanner"
	"curated-backend/internal/scraper"
	"curated-backend/internal/scraper/metatube"
	"curated-backend/internal/server"
	"curated-backend/internal/storage"
	"curated-backend/internal/tasks"
	"curated-backend/internal/version"
	"curated-backend/internal/webui"
)

var (
	launchAtLoginSupportedFn = desktop.LaunchAtLoginSupported
	syncLaunchAtLoginFn      = desktop.SyncLaunchAtLogin
)

type App struct {
	cfg           config.Config
	logger        *zap.Logger
	store         *storage.SQLiteStore
	library       *library.Service
	scanner       *scanner.Service
	scraper       scraper.Service
	assets        *assets.Service
	tasks         *tasks.Manager
	player        *nativeplayer.Launcher
	streams       *playback.Manager
	devCPUSampler devmetrics.CPUSampler

	// organizeLibrary is toggled via Settings UI / PATCH and persisted to library-config.cfg.
	organizeLibrary bool
	organizeMu      sync.RWMutex
	// extendedLibraryImport: first-scan layout detection on newly added library roots (library-config.cfg).
	extendedLibraryImport bool
	extendedImportMu      sync.RWMutex
	// autoLibraryWatch gates fsnotify-driven scan enqueue; persisted to library-config.cfg.
	autoLibraryWatch   bool
	autoLibraryWatchMu sync.RWMutex
	// autoActorProfileScrape gates scan/import-time actor profile scrape enqueue; persisted to library-config.cfg.
	autoActorProfileScrape   bool
	autoActorProfileScrapeMu sync.RWMutex
	// launchAtLogin persists whether Curated should register Windows login autostart via the current-user Run key.
	launchAtLogin   bool
	launchAtLoginMu sync.RWMutex
	// autoActorProfileScrapePending dedupes auto-enqueued actor scrapes while they are in flight.
	autoActorProfileScrapePending   map[string]struct{}
	autoActorProfileScrapePendingMu sync.Mutex
	// playerSettingsMu protects cfg.Player and live playback-runtime updates.
	playerSettingsMu sync.RWMutex
	// metadataMovieMu protects cfg.MetadataMovieProvider/ProviderChain (library-config.cfg) during concurrent scrapes.
	metadataMovieMu            sync.RWMutex
	metadataMovieProviderChain []string // ordered list of providers to try in sequence
	librarySettingsPath        string   // JSON file under config/ (organizeLibrary, future keys)

	appCtx   context.Context
	writeMu  sync.Mutex
	scanning atomic.Bool

	// scrapeSem limits concurrent scrape.movie pipelines (network + DB).
	scrapeSem chan struct{}

	watchScanMu      sync.Mutex
	watchScanPending map[string]struct{}
	watchScanDrainMu sync.Mutex
	libWatchMu       sync.Mutex
	libWatch         *librarywatch.Watcher
	watchLoopCancel  context.CancelFunc
	watchLoopSession uint64 // bumped on each Start/Stop; goroutine clears cancel only if still current
}

// New 构造并返回可运行的后端 App（依赖注入入口），由 cmd/curated 在加载配置、合并 library-config.cfg、
// 打开数据库并完成迁移后调用。
//
// 参数说明：
//   - ctx：应用生命周期 context，用于取消与超时，存入 App.appCtx。
//   - cfg：已合并主配置与库设置文件的 config.Config（含 OrganizeLibrary、CacheDir、各类超时等）。
//   - logger / store：日志与 SQLite 存储。
//   - librarySettingsPath：持久化库行为开关的 JSON 文件路径（如 config/library-config.cfg），
//     供 SetOrganizeLibrary 原子写回；可为空字符串但此时 PATCH 整理库开关会失败。
//
// 若 Metatube 刮削服务初始化失败，返回 (nil, err)。
func New(ctx context.Context, cfg config.Config, logger *zap.Logger, store *storage.SQLiteStore, librarySettingsPath string) (*App, error) {
	proxyenv.Sync(cfg.Proxy, logger)

	scraperService, err := metatube.NewService(logger, time.Duration(cfg.Scraper.RequestTimeoutSeconds)*time.Second)
	if err != nil {
		return nil, err
	}

	scrapeConc := cfg.Scraper.MaxConcurrent
	if scrapeConc <= 0 {
		scrapeConc = 4
	}

	app := &App{
		cfg:     cfg,
		logger:  logger,
		store:   store,
		library: library.NewService(),
		scanner: scanner.NewService(logger),
		scraper: scraperService,
		assets: assets.NewService(
			logger,
			cfg.CacheDir,
			time.Duration(cfg.Assets.RequestTimeoutSeconds)*time.Second,
			cfg.Assets.MaxConcurrentDownloads,
			cfg.Assets.MaxResponseBodyMB,
		),
		tasks: tasks.NewManager(),
		player: nativeplayer.New(nativeplayer.Config{
			Enabled: cfg.Player.NativePlayerEnabled,
			Preset:  nativeplayer.NormalizePreset(cfg.Player.NativePlayerPreset, cfg.Player.NativePlayerCommand),
			Command: cfg.Player.NativePlayerCommand,
			Args:    cfg.Player.NativePlayerArgs,
		}),
		streams: playback.New(playback.Config{
			Enabled:         cfg.Player.StreamPushEnabled,
			HardwareDecode:  cfg.Player.HardwareDecode,
			HardwareEncoder: cfg.Player.HardwareEncoder,
			FFmpegCommand:   cfg.Player.FFmpegCommand,
			SessionRoot:     cfg.Player.StreamSessionRoot,
		}),
		devCPUSampler:                 devmetrics.NewCPUSampler(),
		organizeLibrary:               cfg.OrganizeLibrary,
		extendedLibraryImport:         cfg.ExtendedLibraryImport,
		autoLibraryWatch:              cfg.AutoLibraryWatch,
		autoActorProfileScrape:        cfg.AutoActorProfileScrape,
		launchAtLogin:                 cfg.LaunchAtLogin,
		autoActorProfileScrapePending: make(map[string]struct{}),
		metadataMovieProviderChain:    cfg.MetadataMovieProviderChain,
		librarySettingsPath:           strings.TrimSpace(librarySettingsPath),
		appCtx:                        ctx,
		scrapeSem:                     make(chan struct{}, scrapeConc),
		watchScanPending:              make(map[string]struct{}),
	}

	// 与设置页「保存代理设置」相同：持久化写回 + proxyenv.Sync。进程重启后仅依赖启动时第一次 Sync
	// 时，部分出站路径可能仍不按 HTTP_PROXY 生效；此处再应用一次，避免必须手动再点保存。
	if app.librarySettingsPath != "" && cfg.Proxy.Enabled && strings.TrimSpace(cfg.Proxy.URL) != "" {
		if err := app.SetProxy(cfg.Proxy); err != nil {
			logger.Warn("startup: reapply proxy settings failed", zap.Error(err))
		}
	}

	return app, nil
}

func (a *App) Close() {
	if a == nil {
		return
	}
	a.StopLibraryWatchLoop()
	if a.streams != nil {
		a.streams.Close()
	}
}

// ReloadLibraryWatches re-reads library roots from the database and rebuilds directory watches.
func (a *App) ReloadLibraryWatches(ctx context.Context) error {
	a.libWatchMu.Lock()
	w := a.libWatch
	a.libWatchMu.Unlock()
	if w == nil {
		return nil
	}
	return w.Reload(ctx)
}

// EnqueueLibraryWatchScanRoots queues library roots for a debounced fsnotify-driven scan.
func (a *App) EnqueueLibraryWatchScanRoots(roots []string) {
	if !a.AutoLibraryWatch() {
		return
	}
	if len(roots) == 0 {
		return
	}
	a.watchScanMu.Lock()
	for _, r := range roots {
		r = filepath.Clean(strings.TrimSpace(r))
		if r != "" {
			a.watchScanPending[r] = struct{}{}
		}
	}
	a.watchScanMu.Unlock()
	go a.tryDrainWatchScanQueue()
}

func (a *App) tryDrainWatchScanQueue() {
	a.watchScanDrainMu.Lock()
	defer a.watchScanDrainMu.Unlock()

	a.watchScanMu.Lock()
	if len(a.watchScanPending) == 0 {
		a.watchScanMu.Unlock()
		return
	}
	roots := make([]string, 0, len(a.watchScanPending))
	for r := range a.watchScanPending {
		roots = append(roots, r)
	}
	slices.Sort(roots)
	a.watchScanPending = make(map[string]struct{})
	a.watchScanMu.Unlock()

	_, err := a.startLibraryScan(a.appCtx, io.Discard, roots, map[string]any{"trigger": "fsnotify"})
	if err != nil {
		if errors.Is(err, contracts.ErrScanAlreadyRunning) {
			a.watchScanMu.Lock()
			for _, r := range roots {
				a.watchScanPending[r] = struct{}{}
			}
			a.watchScanMu.Unlock()
			return
		}
		a.logger.Warn("library watch: failed to start scan", zap.Error(err))
		a.watchScanMu.Lock()
		for _, r := range roots {
			a.watchScanPending[r] = struct{}{}
		}
		a.watchScanMu.Unlock()
	}
}

// OrganizeLibrary returns whether scan/scrape should move files and write NFO/assets into 番号 folders.
func (a *App) OrganizeLibrary() bool {
	a.organizeMu.RLock()
	defer a.organizeMu.RUnlock()
	return a.organizeLibrary
}

// SetOrganizeLibrary persists organizeLibrary to library-config.cfg, then updates in-memory state.
// Fails without mutating memory if the file cannot be written.
func (a *App) SetOrganizeLibrary(v bool) error {
	path := a.librarySettingsPath
	if path == "" {
		return fmt.Errorf("library settings path not configured")
	}
	if err := config.WriteLibrarySettingsMerge(path, func(m map[string]any) error {
		m["organizeLibrary"] = v
		return nil
	}); err != nil {
		return err
	}

	a.organizeMu.Lock()
	a.organizeLibrary = v
	a.cfg.OrganizeLibrary = v
	a.organizeMu.Unlock()
	return nil
}

// ExtendedLibraryImport reports whether first-scan import layout detection is enabled (library-config.cfg).
func (a *App) ExtendedLibraryImport() bool {
	a.extendedImportMu.RLock()
	defer a.extendedImportMu.RUnlock()
	return a.extendedLibraryImport
}

// SetExtendedLibraryImport persists extendedLibraryImport to library-config.cfg and updates in-memory state.
func (a *App) SetExtendedLibraryImport(v bool) error {
	path := a.librarySettingsPath
	if path == "" {
		return fmt.Errorf("library settings path not configured")
	}
	if err := config.WriteLibrarySettingsMerge(path, func(m map[string]any) error {
		m["extendedLibraryImport"] = v
		return nil
	}); err != nil {
		return err
	}
	a.extendedImportMu.Lock()
	a.extendedLibraryImport = v
	a.cfg.ExtendedLibraryImport = v
	a.extendedImportMu.Unlock()
	return nil
}

// AutoLibraryWatch reports whether directory watching may queue scans for new files under library roots.
func (a *App) AutoLibraryWatch() bool {
	a.autoLibraryWatchMu.RLock()
	defer a.autoLibraryWatchMu.RUnlock()
	return a.autoLibraryWatch
}

// AutoActorProfileScrape reports whether movie metadata scrapes may auto-enqueue missing actor profiles.
func (a *App) AutoActorProfileScrape() bool {
	a.autoActorProfileScrapeMu.RLock()
	defer a.autoActorProfileScrapeMu.RUnlock()
	return a.autoActorProfileScrape
}

// LaunchAtLogin reports whether the persisted login autostart preference is enabled.
func (a *App) LaunchAtLogin() bool {
	a.launchAtLoginMu.RLock()
	defer a.launchAtLoginMu.RUnlock()
	return a.launchAtLogin
}

// LaunchAtLoginSupported reports whether the current runtime can safely manage OS login autostart.
func (a *App) LaunchAtLoginSupported() bool {
	return launchAtLoginSupportedFn()
}

// SetAutoLibraryWatch persists autoLibraryWatch to library-config.cfg, updates in-memory state, and starts/stops the watcher loop when yaml allows watching.
func (a *App) SetAutoLibraryWatch(v bool) error {
	path := a.librarySettingsPath
	if path == "" {
		return fmt.Errorf("library settings path not configured")
	}
	if err := config.WriteLibrarySettingsMerge(path, func(m map[string]any) error {
		m["autoLibraryWatch"] = v
		return nil
	}); err != nil {
		return err
	}
	a.autoLibraryWatchMu.Lock()
	a.autoLibraryWatch = v
	a.cfg.AutoLibraryWatch = v
	a.autoLibraryWatchMu.Unlock()
	if v && a.cfg.LibraryWatchOn() {
		return a.EnsureLibraryWatchRunning()
	}
	a.StopLibraryWatchLoop()
	return nil
}

// SetAutoActorProfileScrape persists autoActorProfileScrape to library-config.cfg and updates in-memory state.
func (a *App) SetAutoActorProfileScrape(v bool) error {
	path := a.librarySettingsPath
	if path == "" {
		return fmt.Errorf("library settings path not configured")
	}
	if err := config.WriteLibrarySettingsMerge(path, func(m map[string]any) error {
		m["autoActorProfileScrape"] = v
		return nil
	}); err != nil {
		return err
	}
	a.autoActorProfileScrapeMu.Lock()
	a.autoActorProfileScrape = v
	a.cfg.AutoActorProfileScrape = v
	a.autoActorProfileScrapeMu.Unlock()
	return nil
}

// SetLaunchAtLogin persists launchAtLogin to library-config.cfg, synchronizes the OS autostart entry,
// and only updates in-memory state after both steps succeed.
func (a *App) SetLaunchAtLogin(v bool) error {
	path := a.librarySettingsPath
	if path == "" {
		return fmt.Errorf("library settings path not configured")
	}
	if v && !a.LaunchAtLoginSupported() {
		return fmt.Errorf("launch at login is not supported in this runtime")
	}

	prev := a.LaunchAtLogin()
	if err := config.WriteLibrarySettingsMerge(path, func(m map[string]any) error {
		m["launchAtLogin"] = v
		return nil
	}); err != nil {
		return err
	}
	if err := syncLaunchAtLoginFn(v); err != nil {
		revertErr := config.WriteLibrarySettingsMerge(path, func(m map[string]any) error {
			m["launchAtLogin"] = prev
			return nil
		})
		if revertErr != nil {
			return fmt.Errorf("sync launch at login: %w (revert settings: %v)", err, revertErr)
		}
		return fmt.Errorf("sync launch at login: %w", err)
	}

	a.launchAtLoginMu.Lock()
	a.launchAtLogin = v
	a.cfg.LaunchAtLogin = v
	a.launchAtLoginMu.Unlock()
	return nil
}

// EnsureLibraryWatchRunning starts the fsnotify Run loop when yaml LibraryWatchOn and AutoLibraryWatch are true.
func (a *App) EnsureLibraryWatchRunning() error {
	if !a.cfg.LibraryWatchOn() {
		return nil
	}
	if !a.AutoLibraryWatch() {
		return nil
	}
	a.libWatchMu.Lock()
	if a.watchLoopCancel != nil {
		a.libWatchMu.Unlock()
		return nil
	}
	if a.libWatch == nil {
		lw, err := librarywatch.New(librarywatch.Options{
			Enabled:  true,
			Debounce: a.cfg.LibraryWatchDebounce(),
			Logger:   a.logger,
			Lister:   a.store,
			Queue:    a,
		})
		if err != nil {
			a.libWatchMu.Unlock()
			return err
		}
		a.libWatch = lw
	}
	w := a.libWatch
	a.watchLoopSession++
	sess := a.watchLoopSession
	watchCtx, cancel := context.WithCancel(a.appCtx)
	a.watchLoopCancel = cancel
	a.libWatchMu.Unlock()

	go func(sess uint64) {
		err := w.Run(watchCtx)
		a.libWatchMu.Lock()
		if a.watchLoopSession == sess {
			a.watchLoopCancel = nil
		}
		a.libWatchMu.Unlock()
		if err != nil && !errors.Is(err, context.Canceled) {
			a.logger.Warn("library fsnotify watcher exited", zap.Error(err))
		}
	}(sess)

	return nil
}

// StopLibraryWatchLoop cancels the active fsnotify Run context (watcher instance is kept for Reload / restart).
func (a *App) StopLibraryWatchLoop() {
	a.libWatchMu.Lock()
	cancel := a.watchLoopCancel
	a.watchLoopCancel = nil
	a.watchLoopSession++
	a.libWatchMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// MetadataMovieProvider returns the persisted single-provider name for settings UI (not chain[0]).
// Effective scrape provider is chosen by movieScrapeOptionsForRun using MetadataMovieScrapeMode.
func (a *App) MetadataMovieProvider() string {
	a.metadataMovieMu.RLock()
	defer a.metadataMovieMu.RUnlock()
	return strings.TrimSpace(a.cfg.MetadataMovieProvider)
}

func (a *App) effectiveMetadataMovieScrapeModeLocked() string {
	m := strings.TrimSpace(strings.ToLower(a.cfg.MetadataMovieScrapeMode))
	if m == "auto" || m == "specified" || m == "chain" {
		return m
	}
	if len(a.metadataMovieProviderChain) > 0 {
		return "chain"
	}
	if strings.TrimSpace(a.cfg.MetadataMovieProvider) != "" {
		return "specified"
	}
	return "auto"
}

// MetadataMovieScrapeMode returns auto | specified | chain for API/settings UI.
func (a *App) MetadataMovieScrapeMode() string {
	a.metadataMovieMu.RLock()
	defer a.metadataMovieMu.RUnlock()
	return a.effectiveMetadataMovieScrapeModeLocked()
}

func (a *App) MetadataMovieStrategy() string {
	a.metadataMovieMu.RLock()
	defer a.metadataMovieMu.RUnlock()
	strategy := strings.TrimSpace(strings.ToLower(a.cfg.MetadataMovieStrategy))
	switch strategy {
	case "auto-global", "auto-cn-friendly", "custom-chain", "specified":
		return strategy
	default:
		mode := a.effectiveMetadataMovieScrapeModeLocked()
		switch mode {
		case "chain":
			return "custom-chain"
		case "specified":
			return "specified"
		default:
			return "auto-cn-friendly"
		}
	}
}

func (a *App) movieScrapeOptionsForRun() scraper.MovieScrapeOptions {
	a.metadataMovieMu.RLock()
	defer a.metadataMovieMu.RUnlock()
	if ms, ok := a.scraper.(*metatube.Service); ok {
		switch strings.TrimSpace(strings.ToLower(a.cfg.MetadataMovieStrategy)) {
		case "auto-cn-friendly":
			return scraper.MovieScrapeOptions{ProviderChain: ms.PreferredMovieProviderChain("auto-cn-friendly")}
		case "auto-global":
			return scraper.MovieScrapeOptions{}
		case "custom-chain":
			ch := make([]string, len(a.metadataMovieProviderChain))
			copy(ch, a.metadataMovieProviderChain)
			return scraper.MovieScrapeOptions{ProviderChain: ch}
		case "specified":
			return scraper.MovieScrapeOptions{Provider: strings.TrimSpace(a.cfg.MetadataMovieProvider)}
		}
	}
	mode := a.effectiveMetadataMovieScrapeModeLocked()
	switch mode {
	case "chain":
		ch := make([]string, len(a.metadataMovieProviderChain))
		copy(ch, a.metadataMovieProviderChain)
		return scraper.MovieScrapeOptions{Provider: "", ProviderChain: ch}
	case "specified":
		return scraper.MovieScrapeOptions{Provider: strings.TrimSpace(a.cfg.MetadataMovieProvider), ProviderChain: nil}
	default:
		return scraper.MovieScrapeOptions{Provider: "", ProviderChain: nil}
	}
}

// MetadataMovieProviderChain returns the ordered list of providers to try in sequence.
// Empty slice means "auto" (all sources).
func (a *App) MetadataMovieProviderChain() []string {
	a.metadataMovieMu.RLock()
	defer a.metadataMovieMu.RUnlock()
	out := make([]string, len(a.metadataMovieProviderChain))
	copy(out, a.metadataMovieProviderChain)
	return out
}

// SetMetadataMovieProvider persists the single provider and sets scrape mode to auto or specified.
// Does not remove metadataMovieProviderChain from disk so users can switch back to chain mode without re-entering the list.
func (a *App) SetMetadataMovieProvider(name string) error {
	path := a.librarySettingsPath
	if path == "" {
		return fmt.Errorf("library settings path not configured")
	}
	trimmed := strings.TrimSpace(name)
	mode := "auto"
	if trimmed != "" {
		mode = "specified"
	}
	if err := config.WriteLibrarySettingsMerge(path, func(m map[string]any) error {
		m["metadataMovieProvider"] = trimmed
		m["metadataMovieScrapeMode"] = mode
		return nil
	}); err != nil {
		return err
	}
	a.metadataMovieMu.Lock()
	a.cfg.MetadataMovieProvider = trimmed
	a.cfg.MetadataMovieScrapeMode = mode
	a.metadataMovieMu.Unlock()
	return nil
}

// SetMetadataMovieProviderChain persists the ordered provider list to library-config.cfg and updates memory.
// Empty slice means "auto". Also clears the single provider field.
func (a *App) SetMetadataMovieProviderChain(chain []string) error {
	path := a.librarySettingsPath
	if path == "" {
		return fmt.Errorf("library settings path not configured")
	}
	// Filter and trim
	filtered := make([]string, 0, len(chain))
	for _, p := range chain {
		if s := strings.TrimSpace(p); s != "" {
			filtered = append(filtered, s)
		}
	}
	if err := config.WriteLibrarySettingsMerge(path, func(m map[string]any) error {
		if len(filtered) == 0 {
			delete(m, "metadataMovieProviderChain")
			delete(m, "metadataMovieProvider")
			m["metadataMovieScrapeMode"] = "auto"
		} else {
			m["metadataMovieProviderChain"] = filtered
			delete(m, "metadataMovieProvider")
			m["metadataMovieScrapeMode"] = "chain"
		}
		return nil
	}); err != nil {
		return err
	}
	a.metadataMovieMu.Lock()
	a.metadataMovieProviderChain = filtered
	a.cfg.MetadataMovieProviderChain = filtered
	if len(filtered) == 0 {
		a.cfg.MetadataMovieProvider = ""
		a.cfg.MetadataMovieScrapeMode = "auto"
	} else {
		a.cfg.MetadataMovieProvider = ""
		a.cfg.MetadataMovieScrapeMode = "chain"
	}
	a.metadataMovieMu.Unlock()
	return nil
}

// SetMetadataMovieScrapeMode persists auto | specified | chain without changing saved provider or chain lists.
func (a *App) SetMetadataMovieScrapeMode(mode string) error {
	path := a.librarySettingsPath
	if path == "" {
		return fmt.Errorf("library settings path not configured")
	}
	normalized := strings.TrimSpace(strings.ToLower(mode))
	if normalized != "auto" && normalized != "specified" && normalized != "chain" {
		return fmt.Errorf("invalid metadataMovieScrapeMode %q", mode)
	}
	if err := config.WriteLibrarySettingsMerge(path, func(m map[string]any) error {
		m["metadataMovieScrapeMode"] = normalized
		return nil
	}); err != nil {
		return err
	}
	a.metadataMovieMu.Lock()
	a.cfg.MetadataMovieScrapeMode = normalized
	a.metadataMovieMu.Unlock()
	return nil
}

func (a *App) SetMetadataMovieStrategy(strategy string) error {
	path := a.librarySettingsPath
	if path == "" {
		return fmt.Errorf("library settings path not configured")
	}
	normalized := strings.TrimSpace(strings.ToLower(strategy))
	switch normalized {
	case "auto-global", "auto-cn-friendly", "custom-chain", "specified":
	default:
		return fmt.Errorf("invalid metadataMovieStrategy %q", strategy)
	}
	if err := config.WriteLibrarySettingsMerge(path, func(m map[string]any) error {
		m["metadataMovieStrategy"] = normalized
		return nil
	}); err != nil {
		return err
	}
	a.metadataMovieMu.Lock()
	a.cfg.MetadataMovieStrategy = normalized
	a.metadataMovieMu.Unlock()
	return nil
}

// ListMetadataMovieProviders returns registered Metatube movie provider names (sorted), or nil if scraper is not Metatube.
func (a *App) ListMetadataMovieProviders() []string {
	ms, ok := a.scraper.(*metatube.Service)
	if !ok {
		return nil
	}
	return ms.ListMovieProviderNames()
}

// Proxy returns the current HTTP proxy configuration.
func (a *App) Proxy() config.ProxyConfig {
	return a.cfg.Proxy
}

// SetProxy persists the proxy configuration to library-config.cfg and updates memory.
func (a *App) SetProxy(p config.ProxyConfig) error {
	path := a.librarySettingsPath
	if path == "" {
		return fmt.Errorf("library settings path not configured")
	}
	if err := config.WriteLibrarySettingsMerge(path, func(m map[string]any) error {
		if !p.Enabled {
			// When disabled, store enabled=false but keep other fields for UI convenience
			m["proxy"] = map[string]any{
				"enabled": false,
			}
		} else {
			proxyMap := map[string]any{
				"enabled": true,
				"url":     p.URL,
			}
			if p.Username != "" {
				proxyMap["username"] = p.Username
			}
			if p.Password != "" {
				proxyMap["password"] = p.Password
			}
			m["proxy"] = proxyMap
		}
		return nil
	}); err != nil {
		return err
	}
	a.cfg.Proxy = p
	proxyenv.Sync(p, a.logger)
	return nil
}

// BackendLogSettings returns current backend log fields (merged config + library-config.cfg).
func (a *App) BackendLogSettings() contracts.BackendLogSettingsDTO {
	return contracts.BackendLogSettingsDTO{
		LogDir:        config.ResolveLogDir(a.cfg.LogDir),
		LogFilePrefix: a.cfg.LogFilePrefix,
		LogMaxAgeDays: a.cfg.LogMaxAgeDays,
		LogLevel:      a.cfg.LogLevel,
	}
}

// PlayerSettings returns current player/playback preferences exposed to Settings UI.
func (a *App) PlayerSettings() contracts.PlayerSettingsDTO {
	a.playerSettingsMu.RLock()
	defer a.playerSettingsMu.RUnlock()
	nativePreset := nativeplayer.NormalizePreset(a.cfg.Player.NativePlayerPreset, a.cfg.Player.NativePlayerCommand)
	nativeCommand := strings.TrimSpace(a.cfg.Player.NativePlayerCommand)
	if nativeCommand == "" {
		nativeCommand = nativeplayer.DefaultCommandForPreset(nativePreset)
	}
	ffmpegCommand := strings.TrimSpace(a.cfg.Player.FFmpegCommand)
	if ffmpegCommand == "" {
		ffmpegCommand = "ffmpeg"
	}
	seekForward := a.cfg.Player.SeekForwardStepSec
	if seekForward <= 0 {
		seekForward = 10
	}
	seekBackward := a.cfg.Player.SeekBackwardStepSec
	if seekBackward <= 0 {
		seekBackward = 10
	}
	forceStreamPush := a.cfg.Player.ForceStreamPush
	if !a.cfg.Player.StreamPushEnabled {
		forceStreamPush = false
	}
	return contracts.PlayerSettingsDTO{
		HardwareDecode:      a.cfg.Player.HardwareDecode,
		HardwareEncoder:     config.NormalizeHardwareEncoderPreference(a.cfg.Player.HardwareEncoder),
		NativePlayerPreset:  nativePreset,
		NativePlayerEnabled: a.cfg.Player.NativePlayerEnabled,
		NativePlayerCommand: nativeCommand,
		StreamPushEnabled:   a.cfg.Player.StreamPushEnabled,
		ForceStreamPush:     forceStreamPush,
		FFmpegCommand:       ffmpegCommand,
		PreferNativePlayer:  a.cfg.Player.PreferNativePlayer,
		SeekForwardStepSec:  seekForward,
		SeekBackwardStepSec: seekBackward,
	}
}

// SetPlayerSettingsPatch merges playback settings into library-config.cfg, updates in-memory config,
// and hot-applies native-player / HLS session runtime behavior for subsequent requests.
func (a *App) SetPlayerSettingsPatch(p contracts.PatchPlayerSettingsDTO) error {
	path := a.librarySettingsPath
	if path == "" {
		return fmt.Errorf("library settings path not configured")
	}

	a.playerSettingsMu.RLock()
	next := a.cfg.Player
	a.playerSettingsMu.RUnlock()

	if p.HardwareDecode != nil {
		next.HardwareDecode = *p.HardwareDecode
	}
	if p.HardwareEncoder != nil {
		next.HardwareEncoder = config.NormalizeHardwareEncoderPreference(*p.HardwareEncoder)
	}
	if p.NativePlayerPreset != nil {
		next.NativePlayerPreset = nativeplayer.NormalizePreset(*p.NativePlayerPreset, next.NativePlayerCommand)
	}
	if p.NativePlayerEnabled != nil {
		next.NativePlayerEnabled = *p.NativePlayerEnabled
	}
	if p.NativePlayerCommand != nil {
		cmd := strings.TrimSpace(*p.NativePlayerCommand)
		if cmd == "" {
			cmd = nativeplayer.DefaultCommandForPreset(next.NativePlayerPreset)
		}
		next.NativePlayerCommand = cmd
	}
	if p.StreamPushEnabled != nil {
		next.StreamPushEnabled = *p.StreamPushEnabled
	}
	if p.ForceStreamPush != nil {
		next.ForceStreamPush = *p.ForceStreamPush
	}
	if p.FFmpegCommand != nil {
		cmd := strings.TrimSpace(*p.FFmpegCommand)
		if cmd == "" {
			cmd = "ffmpeg"
		}
		next.FFmpegCommand = cmd
	}
	if p.PreferNativePlayer != nil {
		next.PreferNativePlayer = *p.PreferNativePlayer
	}
	if p.SeekForwardStepSec != nil {
		if *p.SeekForwardStepSec <= 0 {
			return fmt.Errorf("seekForwardStepSec must be greater than 0")
		}
		next.SeekForwardStepSec = *p.SeekForwardStepSec
	}
	if p.SeekBackwardStepSec != nil {
		if *p.SeekBackwardStepSec <= 0 {
			return fmt.Errorf("seekBackwardStepSec must be greater than 0")
		}
		next.SeekBackwardStepSec = *p.SeekBackwardStepSec
	}

	next.NativePlayerPreset = nativeplayer.NormalizePreset(next.NativePlayerPreset, next.NativePlayerCommand)
	if strings.TrimSpace(next.NativePlayerCommand) == "" {
		next.NativePlayerCommand = nativeplayer.DefaultCommandForPreset(next.NativePlayerPreset)
	}
	if strings.TrimSpace(next.FFmpegCommand) == "" {
		next.FFmpegCommand = "ffmpeg"
	}
	next.HardwareEncoder = config.NormalizeHardwareEncoderPreference(next.HardwareEncoder)
	if next.SeekForwardStepSec <= 0 {
		next.SeekForwardStepSec = 10
	}
	if next.SeekBackwardStepSec <= 0 {
		next.SeekBackwardStepSec = 10
	}
	// Invariant: enabling StreamPushEnabled must never imply enabling ForceStreamPush (no auto-couple).
	// Only the converse is applied: when push is off, force-HLS is cleared.
	if !next.StreamPushEnabled {
		next.ForceStreamPush = false
	}

	if err := config.WriteLibrarySettingsMerge(path, func(m map[string]any) error {
		playerMap, _ := m["player"].(map[string]any)
		if playerMap == nil {
			playerMap = map[string]any{}
		}
		playerMap["hardwareDecode"] = next.HardwareDecode
		playerMap["hardwareEncoder"] = next.HardwareEncoder
		playerMap["nativePlayerPreset"] = next.NativePlayerPreset
		playerMap["nativePlayerEnabled"] = next.NativePlayerEnabled
		playerMap["nativePlayerCommand"] = next.NativePlayerCommand
		playerMap["streamPushEnabled"] = next.StreamPushEnabled
		playerMap["forceStreamPush"] = next.ForceStreamPush
		playerMap["ffmpegCommand"] = next.FFmpegCommand
		playerMap["preferNativePlayer"] = next.PreferNativePlayer
		playerMap["seekForwardStepSec"] = next.SeekForwardStepSec
		playerMap["seekBackwardStepSec"] = next.SeekBackwardStepSec
		m["player"] = playerMap
		return nil
	}); err != nil {
		return err
	}

	a.playerSettingsMu.Lock()
	a.cfg.Player = next
	a.playerSettingsMu.Unlock()

	if a.player != nil {
		a.player.SetConfig(nativeplayer.Config{
			Enabled: next.NativePlayerEnabled,
			Preset:  next.NativePlayerPreset,
			Command: next.NativePlayerCommand,
			Args:    next.NativePlayerArgs,
		})
	}
	if a.streams != nil {
		a.streams.SetConfig(playback.Config{
			Enabled:         next.StreamPushEnabled,
			HardwareDecode:  next.HardwareDecode,
			HardwareEncoder: next.HardwareEncoder,
			FFmpegCommand:   next.FFmpegCommand,
			SessionRoot:     next.StreamSessionRoot,
		})
	}
	return nil
}

// SetBackendLogPatch merges patch into current values, persists to library-config.cfg, and updates memory.
// Changing logDir/logLevel takes effect after backend restart.
func (a *App) SetBackendLogPatch(p contracts.PatchBackendLogSettings) error {
	path := a.librarySettingsPath
	if path == "" {
		return fmt.Errorf("library settings path not configured")
	}
	nextDir := a.cfg.LogDir
	nextPrefix := a.cfg.LogFilePrefix
	nextMaxAge := a.cfg.LogMaxAgeDays
	nextLevel := a.cfg.LogLevel
	if p.LogDir != nil {
		nextDir = strings.TrimSpace(*p.LogDir)
	}
	if p.LogFilePrefix != nil {
		nextPrefix = strings.TrimSpace(*p.LogFilePrefix)
	}
	if p.LogMaxAgeDays != nil {
		nextMaxAge = *p.LogMaxAgeDays
		if nextMaxAge < 0 {
			return fmt.Errorf("logMaxAgeDays must be non-negative")
		}
	}
	if p.LogLevel != nil {
		lvl := strings.TrimSpace(*p.LogLevel)
		if lvl == "" {
			lvl = "info"
		}
		var zl zapcore.Level
		if err := zl.UnmarshalText([]byte(lvl)); err != nil {
			return fmt.Errorf("invalid logLevel: %w", err)
		}
		nextLevel = lvl
	}
	if err := config.WriteLibrarySettingsMerge(path, func(m map[string]any) error {
		if strings.TrimSpace(nextDir) == "" {
			delete(m, "logDir")
		} else {
			m["logDir"] = nextDir
		}
		if strings.TrimSpace(nextPrefix) == "" {
			delete(m, "logFilePrefix")
		} else {
			m["logFilePrefix"] = nextPrefix
		}
		if nextMaxAge <= 0 {
			delete(m, "logMaxAgeDays")
		} else {
			m["logMaxAgeDays"] = nextMaxAge
		}
		m["logLevel"] = nextLevel
		return nil
	}); err != nil {
		return err
	}
	a.cfg.LogDir = config.ResolveLogDir(nextDir)
	a.cfg.LogFilePrefix = nextPrefix
	a.cfg.LogMaxAgeDays = nextMaxAge
	a.cfg.LogLevel = nextLevel
	return nil
}

func (a *App) Run(ctx context.Context, input io.Reader, output io.Writer) error {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return err
			}
			return nil
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var command contracts.Command
		if err := json.Unmarshal([]byte(line), &command); err != nil {
			if writeErr := a.writeResponse(output, contracts.Response{
				Kind:      "response",
				ID:        "",
				OK:        false,
				Error:     badRequest("invalid command json", map[string]any{"cause": err.Error()}),
				Timestamp: nowUTC(),
			}); writeErr != nil {
				return writeErr
			}
			continue
		}

		if err := a.handleCommand(ctx, output, command); err != nil {
			return err
		}
	}
}

func (a *App) handleCommand(ctx context.Context, output io.Writer, command contracts.Command) error {
	a.logger.Info("handling command", zap.String("command", command.Type), zap.String("commandId", command.ID))

	switch command.Type {
	case contracts.CommandSystemHealth:
		return a.respondOK(output, command.ID, contracts.HealthDTO{
			Name:             version.BackendName(),
			Version:          version.Stamp(),
			Channel:          version.Channel,
			InstallerVersion: version.PackageVersion(),
			Transport:        "stdio-jsonl",
			DatabasePath:     a.cfg.DatabasePath,
		})

	case contracts.CommandLibraryList:
		var request contracts.ListMoviesRequest
		if err := decodePayload(command.Payload, &request); err != nil {
			return a.respondError(output, command.ID, badRequest("invalid library.list payload", map[string]any{"cause": err.Error()}))
		}

		result, err := a.store.ListMovies(ctx, request)
		if err != nil {
			return a.respondError(output, command.ID, internalError("failed to list movies", map[string]any{"cause": err.Error()}))
		}
		return a.respondOK(output, command.ID, result)

	case contracts.CommandLibraryDetail:
		var request contracts.GetMovieDetailRequest
		if err := decodePayload(command.Payload, &request); err != nil {
			return a.respondError(output, command.ID, badRequest("invalid library.detail payload", map[string]any{"cause": err.Error()}))
		}
		if request.MovieID == "" {
			return a.respondError(output, command.ID, badRequest("movieId is required", nil))
		}

		movie, err := a.store.GetMovieDetail(ctx, request.MovieID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return a.respondError(output, command.ID, notFound("movie was not found", map[string]any{"movieId": request.MovieID}))
			}
			return a.respondError(output, command.ID, internalError("failed to load movie", map[string]any{"movieId": request.MovieID}))
		}

		return a.respondOK(output, command.ID, movie)

	case contracts.CommandSettingsGet:
		libraryPaths, err := a.store.ListLibraryPaths(ctx)
		if err != nil {
			return a.respondError(output, command.ID, internalError("failed to list library paths", map[string]any{"cause": err.Error()}))
		}

		p := a.Proxy()
		settings := contracts.SettingsDTO{
			LibraryPaths:               libraryPaths,
			Player:                     a.PlayerSettings(),
			OrganizeLibrary:            a.OrganizeLibrary(),
			ExtendedLibraryImport:      a.ExtendedLibraryImport(),
			AutoLibraryWatch:           a.AutoLibraryWatch(),
			AutoActorProfileScrape:     a.AutoActorProfileScrape(),
			MetadataMovieProvider:      a.MetadataMovieProvider(),
			MetadataMovieProviders:     a.ListMetadataMovieProviders(),
			MetadataMovieProviderChain: a.MetadataMovieProviderChain(),
			MetadataMovieScrapeMode:    a.MetadataMovieScrapeMode(),
			Proxy: contracts.ProxySettingsDTO{
				Enabled:  p.Enabled,
				URL:      p.URL,
				Username: p.Username,
				Password: p.Password,
			},
			BackendLog: a.BackendLogSettings(),
		}
		return a.respondOK(output, command.ID, settings)

	case contracts.CommandScanStart:
		var request contracts.StartScanRequest
		if err := decodePayload(command.Payload, &request); err != nil {
			return a.respondError(output, command.ID, badRequest("invalid scan.start payload", map[string]any{"cause": err.Error()}))
		}

		task, err := a.startLibraryScan(ctx, output, request.Paths, nil)
		if err != nil {
			if errors.Is(err, contracts.ErrScanAlreadyRunning) {
				return a.respondError(output, command.ID, &contracts.AppError{
					Code:      contracts.ErrorCodeConflict,
					Message:   "scan already in progress",
					Retryable: false,
				})
			}
			return a.respondError(output, command.ID, internalError("failed to start scan", map[string]any{"cause": err.Error()}))
		}

		if err := a.emitEvent(output, contracts.EventTaskStarted, contracts.TaskEventDTO{Task: task}); err != nil {
			return err
		}
		if err := a.emitEvent(output, contracts.EventScanStarted, contracts.TaskEventDTO{Task: task}); err != nil {
			return err
		}

		return a.respondOK(output, command.ID, task)

	case contracts.CommandScanStatus:
		var request contracts.GetTaskStatusRequest
		if err := decodePayload(command.Payload, &request); err != nil {
			return a.respondError(output, command.ID, badRequest("invalid scan.status payload", map[string]any{"cause": err.Error()}))
		}
		if request.TaskID == "" {
			return a.respondError(output, command.ID, badRequest("taskId is required", nil))
		}

		task, ok := a.tasks.Get(request.TaskID)
		if !ok {
			return a.respondError(output, command.ID, notFound("scan task was not found", map[string]any{"taskId": request.TaskID}))
		}

		return a.respondOK(output, command.ID, task)

	default:
		return a.respondError(output, command.ID, &contracts.AppError{
			Code:      contracts.ErrorCodeUnsupported,
			Message:   "unsupported command",
			Retryable: false,
			Details: map[string]any{
				"command": command.Type,
			},
		})
	}
}

func (a *App) runScan(parentCtx context.Context, output io.Writer, taskID string, paths []string) {
	ctx, cancel := context.WithTimeout(parentCtx, time.Duration(a.cfg.Tasks.ScanTimeoutSeconds)*time.Second)
	defer cancel()

	scanStarted := time.Now()
	importedCount := 0
	updatedCount := 0
	skippedCount := 0
	persistedResults := make([]contracts.ScanFileResultDTO, 0)

	const (
		minScanProgressInterval = 250 * time.Millisecond
		scanProgressFileStep    = 50
	)
	var lastProgressEmit time.Time

	libraryPathRows, listErr := a.store.ListLibraryPaths(ctx)
	if listErr != nil {
		task := a.tasks.Fail(taskID, contracts.ErrorCodeInternal, listErr.Error())
		_ = a.store.SaveTask(ctx, task)
		_ = a.emitEvent(output, contracts.EventTaskFailed, contracts.TaskEventDTO{Task: task})
		return
	}
	extendedSnapshot := a.ExtendedLibraryImport()
	seenMovieRoots := make(map[string]struct{})

	summary, err := a.scanner.Scan(ctx, taskID, paths, scanner.Hooks{
		OnProgress: func(processed, total int, message string) {
			now := time.Now()
			shouldEmit := processed == 0 || processed == total ||
				now.Sub(lastProgressEmit) >= minScanProgressInterval ||
				(processed > 0 && processed%scanProgressFileStep == 0)
			if !shouldEmit {
				return
			}
			lastProgressEmit = now

			progress := 100
			if total > 0 {
				progress = int(float64(processed) / float64(total) * 100)
			}

			patch := map[string]any{
				"scanTotal":     total,
				"scanProcessed": processed,
				"scanImported":  importedCount,
				"scanUpdated":   updatedCount,
				"scanSkipped":   skippedCount,
			}
			task := a.tasks.ProgressWithMetadata(taskID, progress, message, patch)
			if saveErr := a.store.SaveTask(ctx, task); saveErr != nil {
				a.logger.Error("failed to persist task progress", zap.Error(saveErr), zap.String("taskId", taskID))
			}
			if emitErr := a.emitEvent(output, contracts.EventTaskProgress, contracts.TaskEventDTO{Task: task}); emitErr != nil {
				a.logger.Error("failed to emit task progress", zap.Error(emitErr), zap.String("taskId", taskID))
				return
			}
			if emitErr := a.emitEvent(output, contracts.EventScanProgress, contracts.TaskEventDTO{Task: task}); emitErr != nil {
				a.logger.Error("failed to emit scan progress", zap.Error(emitErr), zap.String("taskId", taskID))
			}
		},
		OnFileDetected: func(result contracts.ScanFileResultDTO) error {
			if result.Number == "" {
				skippedCount++
				if err := a.store.SaveScanItem(ctx, result); err != nil {
					return err
				}
				persistedResults = append(persistedResults, result)

				if emitErr := a.emitEvent(output, contracts.EventScanFileSkipped, result); emitErr != nil {
					a.logger.Error("failed to emit scan skip event", zap.Error(emitErr), zap.String("taskId", taskID))
				}
				return nil
			}

			if a.OrganizeLibrary() {
				newPath, orgErr := library.OrganizeVideoFile(result.Path, result.Number)
				if orgErr != nil {
					skippedCount++
					result.Status = "skipped"
					if errors.Is(orgErr, library.ErrOrganizeConflict) {
						result.Reason = "organize_conflict: " + orgErr.Error()
					} else {
						result.Reason = "organize_failed: " + orgErr.Error()
					}
					if err := a.store.SaveScanItem(ctx, result); err != nil {
						return err
					}
					persistedResults = append(persistedResults, result)
					if emitErr := a.emitEvent(output, contracts.EventScanFileSkipped, result); emitErr != nil {
						a.logger.Error("failed to emit scan skip event", zap.Error(emitErr), zap.String("taskId", taskID))
					}
					return nil
				}
				result.Path = newPath
				result.FileName = filepath.Base(newPath)
			}

			lp := movieroot.ResolveConfiguredLibraryPath(result.Path, libraryPathRows)
			pending := lp != nil && lp.FirstLibraryScanPending
			kind, _ := movieroot.ClassifyVideoRoot(result.Path, result.Number, extendedSnapshot, pending)
			if extendedSnapshot && pending {
				result.ImportLayout = string(kind)
			}

			rootKey := strings.ToLower(filepath.Clean(filepath.Dir(result.Path))) + "\x00" + moviecode.NormalizeForStorageID(result.Number)
			if _, dup := seenMovieRoots[rootKey]; dup {
				skippedCount++
				result.Status = "skipped"
				result.Reason = "duplicate_movie_root"
				if err := a.store.SaveScanItem(ctx, result); err != nil {
					return err
				}
				persistedResults = append(persistedResults, result)
				if emitErr := a.emitEvent(output, contracts.EventScanFileSkipped, result); emitErr != nil {
					a.logger.Error("failed to emit scan skip event", zap.Error(emitErr), zap.String("taskId", taskID))
				}
				return nil
			}
			seenMovieRoots[rootKey] = struct{}{}

			outcome, err := a.store.PersistScanMovie(ctx, result)
			if err != nil {
				return err
			}

			result.MovieID = outcome.MovieID
			result.Status = outcome.Status
			result.Reason = outcome.Reason

			if err := a.store.SaveScanItem(ctx, result); err != nil {
				return err
			}

			if result.Status == "imported" || result.Status == "updated" {
				a.library.UpsertScannedMovie(result)
			}

			switch result.Status {
			case "imported":
				importedCount++
				if emitErr := a.emitEvent(output, contracts.EventScanFileImported, result); emitErr != nil {
					a.logger.Error("failed to emit scan import event", zap.Error(emitErr), zap.String("taskId", taskID))
				}
				a.enqueueScrape(parentCtx, output, result, taskID)
			case "updated":
				updatedCount++
				if emitErr := a.emitEvent(output, contracts.EventScanFileUpdated, result); emitErr != nil {
					a.logger.Error("failed to emit scan update event", zap.Error(emitErr), zap.String("taskId", taskID))
				}
				a.enqueueScrape(parentCtx, output, result, taskID)
			default:
				skippedCount++
				if emitErr := a.emitEvent(output, contracts.EventScanFileSkipped, result); emitErr != nil {
					a.logger.Error("failed to emit scan skip event", zap.Error(emitErr), zap.String("taskId", taskID))
				}
			}

			persistedResults = append(persistedResults, result)
			return nil
		},
	})
	if err != nil {
		code := contracts.ErrorCodeScanWalk
		if errors.Is(err, context.Canceled) {
			code = contracts.ErrorCodeScanCancelled
		}
		task := a.tasks.Fail(taskID, code, err.Error())
		if saveErr := a.store.SaveTask(ctx, task); saveErr != nil {
			a.logger.Error("failed to persist failed task", zap.Error(saveErr), zap.String("taskId", taskID))
		}
		_ = a.emitEvent(output, contracts.EventTaskFailed, contracts.TaskEventDTO{Task: task})
		return
	}

	summary.Results = persistedResults
	summary.FilesImported = importedCount
	summary.FilesUpdated = updatedCount
	summary.FilesSkipped = skippedCount

	taskMessage := fmt.Sprintf(
		"Scan finished: %d discovered, %d imported, %d updated, %d skipped",
		summary.FilesDiscovered,
		summary.FilesImported,
		summary.FilesUpdated,
		summary.FilesSkipped,
	)
	task := a.tasks.Complete(taskID, taskMessage)
	if saveErr := a.store.SaveTask(ctx, task); saveErr != nil {
		a.logger.Error("failed to persist completed task", zap.Error(saveErr), zap.String("taskId", taskID))
	}
	if err := a.emitEvent(output, contracts.EventTaskCompleted, contracts.TaskEventDTO{Task: task}); err != nil {
		a.logger.Error("failed to emit task completion", zap.Error(err), zap.String("taskId", taskID))
		return
	}

	if err := a.emitEvent(output, contracts.EventScanCompleted, struct {
		Task    contracts.TaskDTO        `json:"task"`
		Summary contracts.ScanSummaryDTO `json:"summary"`
	}{
		Task:    task,
		Summary: summary,
	}); err != nil {
		a.logger.Error("failed to emit scan completion", zap.Error(err), zap.String("taskId", taskID))
	}

	a.logger.Info("scan.library completed",
		zap.String("taskId", taskID),
		zap.Int64("duration_ms", time.Since(scanStarted).Milliseconds()),
		zap.Int("filesDiscovered", summary.FilesDiscovered),
		zap.Int("imported", importedCount),
		zap.Int("updated", updatedCount),
		zap.Int("skipped", skippedCount),
	)

	if clearErr := a.store.ClearFirstLibraryScanPendingAfterScan(ctx, paths); clearErr != nil {
		a.logger.Warn("failed to clear first_library_scan_pending", zap.Error(clearErr), zap.String("taskId", taskID))
	}
}

// enqueueScrape runs runScrape in a goroutine bounded by scrapeSem (config scraper.maxConcurrent).
func (a *App) enqueueScrape(parentCtx context.Context, output io.Writer, result contracts.ScanFileResultDTO, parentScanTaskID string) {
	go func(r contracts.ScanFileResultDTO, parent string) {
		a.scrapeSem <- struct{}{}
		defer func() { <-a.scrapeSem }()
		a.runScrape(parentCtx, output, r, parent)
	}(result, parentScanTaskID)
}

// beginMovieScrapeTask creates and persists the scrape.movie task and emits task started.
func (a *App) beginMovieScrapeTask(ctx context.Context, output io.Writer, result contracts.ScanFileResultDTO, parentScanTaskID string) contracts.TaskDTO {
	md := map[string]any{
		"movieId": result.MovieID,
		"number":  result.Number,
		"path":    result.Path,
	}
	if parentScanTaskID != "" {
		md["parentScanTaskId"] = parentScanTaskID
	}
	task := a.tasks.Create("scrape.movie", md)
	task = a.tasks.Start(task.TaskID, fmt.Sprintf("Scraping metadata for %s", result.Number))
	if err := a.store.SaveTask(ctx, task); err != nil {
		a.logger.Error("failed to persist scraper task", zap.Error(err), zap.String("taskId", task.TaskID))
	}
	if err := a.emitEvent(output, contracts.EventTaskStarted, contracts.TaskEventDTO{Task: task}); err != nil {
		a.logger.Error("failed to emit scraper task start", zap.Error(err), zap.String("taskId", task.TaskID))
	}
	return task
}

// runMovieScrapeBody runs scraper, persists metadata, NFO/assets hooks; ctx must be the scrape timeout context.
func (a *App) runMovieScrapeBody(ctx context.Context, parentCtx context.Context, output io.Writer, task contracts.TaskDTO, result contracts.ScanFileResultDTO) {
	scrapeStarted := time.Now()
	scrapeOpts := a.movieScrapeOptionsForRun()
	metadata, err := a.scraper.Scrape(ctx, result.MovieID, result.Number, scrapeOpts)
	if err != nil {
		code := contracts.ErrorCodeScraperRun
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			code = contracts.ErrorCodeScraperRun
		}
		task = a.tasks.Fail(task.TaskID, code, err.Error())
		task.ErrorCategory = classifyNetworkError(err)
		if saveErr := a.store.SaveTask(ctx, task); saveErr != nil {
			a.logger.Error("failed to persist failed scraper task", zap.Error(saveErr), zap.String("taskId", task.TaskID))
		}
		_ = a.emitEvent(output, contracts.EventTaskFailed, contracts.TaskEventDTO{Task: task})
		_ = a.emitEvent(output, contracts.EventScraperFailed, struct {
			Task    contracts.TaskDTO `json:"task"`
			MovieID string            `json:"movieId"`
			Number  string            `json:"number"`
			Error   string            `json:"error"`
		}{
			Task:    task,
			MovieID: result.MovieID,
			Number:  result.Number,
			Error:   err.Error(),
		})
		return
	}

	task = a.tasks.Progress(task.TaskID, 80, "Persisting scraped metadata")
	if err := a.store.SaveTask(ctx, task); err != nil {
		a.logger.Error("failed to persist scraper progress", zap.Error(err), zap.String("taskId", task.TaskID))
	}
	if err := a.emitEvent(output, contracts.EventTaskProgress, contracts.TaskEventDTO{Task: task}); err != nil {
		a.logger.Error("failed to emit scraper progress", zap.Error(err), zap.String("taskId", task.TaskID))
	}

	if err := a.store.SaveMovieMetadata(ctx, metadata); err != nil {
		task = a.tasks.Fail(task.TaskID, contracts.ErrorCodeScraperRun, err.Error())
		task.ErrorCategory = classifyNetworkError(err)
		task.Provider = strings.TrimSpace(metadata.Provider)
		if saveErr := a.store.SaveTask(ctx, task); saveErr != nil {
			a.logger.Error("failed to persist scraper failure", zap.Error(saveErr), zap.String("taskId", task.TaskID))
		}
		_ = a.emitEvent(output, contracts.EventTaskFailed, contracts.TaskEventDTO{Task: task})
		_ = a.emitEvent(output, contracts.EventScraperFailed, struct {
			Task    contracts.TaskDTO `json:"task"`
			MovieID string            `json:"movieId"`
			Number  string            `json:"number"`
			Error   string            `json:"error"`
		}{
			Task:    task,
			MovieID: result.MovieID,
			Number:  result.Number,
			Error:   err.Error(),
		})
		return
	}

	a.library.ApplyScrapedMetadata(metadata)
	a.enqueueAutoActorProfileScrapes(ctx, metadata.Actors)

	task = a.tasks.Complete(task.TaskID, fmt.Sprintf("Metadata saved for %s", result.Number))
	task.Provider = strings.TrimSpace(metadata.Provider)
	if err := a.store.SaveTask(ctx, task); err != nil {
		a.logger.Error("failed to persist completed scraper task", zap.Error(err), zap.String("taskId", task.TaskID))
	}
	a.logger.Info("scrape.movie completed",
		zap.String("taskId", task.TaskID),
		zap.String("movieId", result.MovieID),
		zap.String("number", result.Number),
		zap.Int64("duration_ms", time.Since(scrapeStarted).Milliseconds()),
	)
	if err := a.emitEvent(output, contracts.EventTaskCompleted, contracts.TaskEventDTO{Task: task}); err != nil {
		a.logger.Error("failed to emit scraper completion", zap.Error(err), zap.String("taskId", task.TaskID))
	}
	if err := a.emitEvent(output, contracts.EventScraperMetadataSaved, struct {
		Task     contracts.TaskDTO `json:"task"`
		MovieID  string            `json:"movieId"`
		Number   string            `json:"number"`
		Title    string            `json:"title"`
		Provider string            `json:"provider"`
	}{
		Task:     task,
		MovieID:  metadata.MovieID,
		Number:   metadata.Number,
		Title:    metadata.Title,
		Provider: metadata.Provider,
	}); err != nil {
		a.logger.Error("failed to emit metadata saved event", zap.Error(err), zap.String("taskId", task.TaskID))
	}

	if a.OrganizeLibrary() {
		nfoDir := filepath.Dir(result.Path)
		if err := library.WriteMovieNFO(nfoDir, metadata); err != nil {
			a.logger.Warn("failed to write movie.nfo", zap.Error(err), zap.String("taskId", task.TaskID), zap.String("dir", nfoDir))
		}
	}

	assetDest := filepath.Join(a.cfg.CacheDir, metadata.MovieID)
	if a.OrganizeLibrary() {
		assetDest = filepath.Dir(result.Path)
	}
	go a.runAssetDownload(parentCtx, output, metadata, assetDest)
}

func (a *App) runScrape(parentCtx context.Context, output io.Writer, result contracts.ScanFileResultDTO, parentScanTaskID string) {
	ctx, cancel := context.WithTimeout(parentCtx, time.Duration(a.cfg.Scraper.TaskTimeoutSeconds)*time.Second)
	defer cancel()
	task := a.beginMovieScrapeTask(ctx, output, result, parentScanTaskID)
	a.runMovieScrapeBody(ctx, parentCtx, output, task, result)
}

func scanFileResultFromMovieDetail(detail contracts.MovieDetailDTO) (contracts.ScanFileResultDTO, error) {
	if strings.TrimSpace(detail.Code) == "" {
		return contracts.ScanFileResultDTO{}, contracts.ErrScrapeMovieNoCode
	}
	if strings.TrimSpace(detail.Location) == "" {
		return contracts.ScanFileResultDTO{}, contracts.ErrScrapeMovieNoLocation
	}
	return contracts.ScanFileResultDTO{
		Path:     detail.Location,
		FileName: filepath.Base(detail.Location),
		Number:   detail.Code,
		MovieID:  detail.ID,
		Status:   "updated",
	}, nil
}

// startAsyncMovieMetadataScrape enqueues runMovieScrapeBody in a goroutine (same pipeline as scan-triggered scrape).
func (a *App) startAsyncMovieMetadataScrape(ctx context.Context, detail contracts.MovieDetailDTO) (contracts.TaskDTO, error) {
	result, err := scanFileResultFromMovieDetail(detail)
	if err != nil {
		return contracts.TaskDTO{}, err
	}
	task := a.beginMovieScrapeTask(ctx, io.Discard, result, "")
	go func() {
		scrapeCtx, cancel := context.WithTimeout(a.appCtx, time.Duration(a.cfg.Scraper.TaskTimeoutSeconds)*time.Second)
		defer cancel()
		a.runMovieScrapeBody(scrapeCtx, a.appCtx, io.Discard, task, result)
	}()
	return task, nil
}

// StartMovieMetadataRefresh enqueues a single-movie rescrape (same pipeline as scan-triggered scrape).
// It returns the scrape task immediately; work continues in a background goroutine.
func (a *App) StartMovieMetadataRefresh(ctx context.Context, movieID string) (contracts.TaskDTO, error) {
	detail, err := a.store.GetMovieDetail(ctx, movieID)
	if errors.Is(err, sql.ErrNoRows) {
		return contracts.TaskDTO{}, contracts.ErrScrapeMovieNotFound
	}
	if err != nil {
		return contracts.TaskDTO{}, err
	}
	return a.startAsyncMovieMetadataScrape(ctx, detail)
}

func (a *App) runActorScrapeBody(ctx context.Context, task contracts.TaskDTO, actorName string) {
	actorScrapeStarted := time.Now()
	profile, err := a.scraper.ScrapeActor(ctx, actorName)
	if err != nil {
		task = a.tasks.Fail(task.TaskID, contracts.ErrorCodeScraperRun, err.Error())
		task.ErrorCategory = classifyNetworkError(err)
		if saveErr := a.store.SaveTask(ctx, task); saveErr != nil {
			a.logger.Error("failed to persist failed actor scrape task", zap.Error(saveErr), zap.String("taskId", task.TaskID))
		}
		return
	}

	task = a.tasks.Progress(task.TaskID, 80, "Saving actor profile")
	if err := a.store.SaveTask(ctx, task); err != nil {
		a.logger.Error("failed to persist actor scrape progress", zap.Error(err), zap.String("taskId", task.TaskID))
	}

	if err := a.store.UpdateActorProfile(ctx, profile); err != nil {
		task = a.tasks.Fail(task.TaskID, contracts.ErrorCodeScraperRun, err.Error())
		task.ErrorCategory = classifyNetworkError(err)
		if saveErr := a.store.SaveTask(ctx, task); saveErr != nil {
			a.logger.Error("failed to persist failed actor scrape task", zap.Error(saveErr), zap.String("taskId", task.TaskID))
		}
		return
	}
	if avatarURL := strings.TrimSpace(profile.AvatarURL); avatarURL != "" {
		go a.downloadActorAvatar(actorName, avatarURL, profile.Homepage)
	}

	task = a.tasks.Complete(task.TaskID, fmt.Sprintf("Actor profile saved: %s", actorName))
	task.Provider = strings.TrimSpace(profile.Provider)
	if err := a.store.SaveTask(ctx, task); err != nil {
		a.logger.Error("failed to persist completed actor scrape task", zap.Error(err), zap.String("taskId", task.TaskID))
	}
	a.logger.Info("scrape.actor completed",
		zap.String("taskId", task.TaskID),
		zap.String("actorName", actorName),
		zap.Int64("duration_ms", time.Since(actorScrapeStarted).Milliseconds()),
	)
}

func normalizeActorScrapePendingKey(actorName string) string {
	return strings.ToLower(strings.TrimSpace(actorName))
}

func (a *App) claimAutoActorProfileScrape(actorName string) bool {
	key := normalizeActorScrapePendingKey(actorName)
	if key == "" {
		return false
	}
	a.autoActorProfileScrapePendingMu.Lock()
	defer a.autoActorProfileScrapePendingMu.Unlock()
	if a.autoActorProfileScrapePending == nil {
		a.autoActorProfileScrapePending = make(map[string]struct{})
	}
	if _, exists := a.autoActorProfileScrapePending[key]; exists {
		return false
	}
	a.autoActorProfileScrapePending[key] = struct{}{}
	return true
}

func (a *App) releaseAutoActorProfileScrape(actorName string) {
	key := normalizeActorScrapePendingKey(actorName)
	if key == "" {
		return
	}
	a.autoActorProfileScrapePendingMu.Lock()
	delete(a.autoActorProfileScrapePending, key)
	a.autoActorProfileScrapePendingMu.Unlock()
}

func (a *App) createActorProfileScrapeTask(ctx context.Context, actorName string, metadata map[string]any) (contracts.TaskDTO, error) {
	task := a.tasks.Create("scrape.actor", metadata)
	task = a.tasks.Start(task.TaskID, fmt.Sprintf("Scraping actor profile: %s", actorName))
	if err := a.store.SaveTask(ctx, task); err != nil {
		return contracts.TaskDTO{}, err
	}
	return task, nil
}

func (a *App) enqueueActorProfileScrapeTask(task contracts.TaskDTO, actorName string, onDone func()) {
	go func(t contracts.TaskDTO, name string) {
		if onDone != nil {
			defer onDone()
		}
		a.scrapeSem <- struct{}{}
		defer func() { <-a.scrapeSem }()
		scrapeCtx, cancel := context.WithTimeout(a.appCtx, time.Duration(a.cfg.Scraper.TaskTimeoutSeconds)*time.Second)
		defer cancel()
		a.runActorScrapeBody(scrapeCtx, t, name)
	}(task, actorName)
}

func (a *App) enqueueAutoActorProfileScrapes(ctx context.Context, actorNames []string) {
	if !a.AutoActorProfileScrape() {
		return
	}
	seen := make(map[string]struct{}, len(actorNames))
	for _, rawName := range actorNames {
		name := strings.TrimSpace(rawName)
		key := normalizeActorScrapePendingKey(name)
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}

		needsScrape, err := a.store.ActorProfileNeedsScrape(ctx, name)
		if err != nil {
			a.logger.Warn("failed to check actor scrape eligibility", zap.Error(err), zap.String("actorName", name))
			continue
		}
		if !needsScrape {
			continue
		}
		if !a.claimAutoActorProfileScrape(name) {
			continue
		}

		task, err := a.createActorProfileScrapeTask(ctx, name, map[string]any{
			"actorName": name,
			"trigger":   "auto.movie-metadata",
		})
		if err != nil {
			a.releaseAutoActorProfileScrape(name)
			a.logger.Warn("failed to enqueue auto actor profile scrape", zap.Error(err), zap.String("actorName", name))
			continue
		}
		a.enqueueActorProfileScrapeTask(task, name, func() {
			a.releaseAutoActorProfileScrape(name)
		})
	}
}

// StartActorProfileScrape enqueues Metatube actor lookup for an existing library actor row (exact name).
func (a *App) StartActorProfileScrape(ctx context.Context, actorName string) (contracts.TaskDTO, error) {
	actorName = strings.TrimSpace(actorName)
	if actorName == "" {
		return contracts.TaskDTO{}, fmt.Errorf("actor name is required")
	}
	exists, err := a.store.ActorNameExists(ctx, actorName)
	if err != nil {
		return contracts.TaskDTO{}, err
	}
	if !exists {
		return contracts.TaskDTO{}, contracts.ErrActorNotFound
	}

	task, err := a.createActorProfileScrapeTask(ctx, actorName, map[string]any{"actorName": actorName})
	if err != nil {
		return contracts.TaskDTO{}, err
	}
	a.enqueueActorProfileScrapeTask(task, actorName, nil)
	return task, nil
}

// matchConfiguredLibraryPath returns the canonical configured path if req matches a library path (case-fold on Windows).
func matchConfiguredLibraryPath(req string, configured []contracts.LibraryPathDTO) (canonical string, ok bool) {
	reqClean := filepath.Clean(strings.TrimSpace(req))
	if reqClean == "." || reqClean == "" {
		return "", false
	}
	for _, c := range configured {
		cp := filepath.Clean(strings.TrimSpace(c.Path))
		if cp == "" || cp == "." {
			continue
		}
		if strings.EqualFold(reqClean, cp) {
			return cp, true
		}
	}
	return "", false
}

// StartMetadataRefreshForLibraryPaths queues metadata rescrape for all indexed movies under the given paths.
// Each requested path must match a configured library root; non-matching paths are listed in invalidPaths.
func (a *App) StartMetadataRefreshForLibraryPaths(ctx context.Context, requestedPaths []string) (contracts.MetadataRefreshQueuedDTO, error) {
	out := contracts.MetadataRefreshQueuedDTO{}
	if len(requestedPaths) == 0 {
		return out, nil
	}

	configured, err := a.store.ListLibraryPaths(ctx)
	if err != nil {
		return out, err
	}

	resolvedRoots := make([]string, 0, len(requestedPaths))
	seenRoot := make(map[string]struct{})
	for _, raw := range requestedPaths {
		p := strings.TrimSpace(raw)
		if p == "" {
			continue
		}
		canon, ok := matchConfiguredLibraryPath(p, configured)
		if !ok {
			out.InvalidPaths = append(out.InvalidPaths, raw)
			continue
		}
		key := strings.ToLower(canon)
		if _, dup := seenRoot[key]; dup {
			continue
		}
		seenRoot[key] = struct{}{}
		resolvedRoots = append(resolvedRoots, canon)
	}

	if len(resolvedRoots) == 0 {
		return out, nil
	}

	ids, err := a.store.ListMovieIDsUnderLibraryRoots(ctx, resolvedRoots)
	if err != nil {
		return out, err
	}

	queued, skipped := 0, 0
	for _, id := range ids {
		detail, derr := a.store.GetMovieDetail(ctx, id)
		if derr != nil {
			skipped++
			continue
		}
		_, serr := a.startAsyncMovieMetadataScrape(ctx, detail)
		if serr != nil {
			skipped++
			continue
		}
		queued++
	}
	out.Queued = queued
	out.Skipped = skipped
	return out, nil
}

func (a *App) runAssetDownload(parentCtx context.Context, output io.Writer, metadata scraper.Metadata, destDir string) {
	ctx, cancel := context.WithTimeout(parentCtx, time.Duration(a.cfg.Assets.TaskTimeoutSeconds)*time.Second)
	defer cancel()

	task := a.tasks.Create("asset.download", map[string]any{
		"movieId": metadata.MovieID,
		"number":  metadata.Number,
	})
	task = a.tasks.Start(task.TaskID, fmt.Sprintf("Downloading assets for %s", metadata.Number))
	if err := a.store.SaveTask(ctx, task); err != nil {
		a.logger.Error("failed to persist asset task", zap.Error(err), zap.String("taskId", task.TaskID))
	}
	_ = a.emitEvent(output, contracts.EventTaskStarted, contracts.TaskEventDTO{Task: task})

	downloaded, err := a.assets.DownloadAllTo(ctx, metadata, destDir)
	if err != nil {
		task = a.tasks.Fail(task.TaskID, contracts.ErrorCodeAssetDownload, err.Error())
		task.ErrorCategory = classifyNetworkError(err)
		task.Provider = strings.TrimSpace(metadata.Provider)
		if saveErr := a.store.SaveTask(ctx, task); saveErr != nil {
			a.logger.Error("failed to persist failed asset task", zap.Error(saveErr), zap.String("taskId", task.TaskID))
		}
		_ = a.emitEvent(output, contracts.EventTaskFailed, contracts.TaskEventDTO{Task: task})
		_ = a.emitEvent(output, contracts.EventAssetDownloadFailed, struct {
			Task    contracts.TaskDTO `json:"task"`
			MovieID string            `json:"movieId"`
			Number  string            `json:"number"`
			Error   string            `json:"error"`
		}{
			Task:    task,
			MovieID: metadata.MovieID,
			Number:  metadata.Number,
			Error:   err.Error(),
		})
		return
	}

	for index, asset := range downloaded {
		progress := int(float64(index+1) / float64(len(downloaded)) * 100)
		task = a.tasks.Progress(task.TaskID, progress, fmt.Sprintf("Downloaded %s asset", asset.Type))
		if err := a.store.SaveTask(ctx, task); err != nil {
			a.logger.Error("failed to persist asset progress", zap.Error(err), zap.String("taskId", task.TaskID))
		}
		if err := a.store.UpdateMediaAssetLocalPath(ctx, metadata.MovieID, asset.Type, asset.SourceURL, asset.LocalPath); err != nil {
			a.logger.Error("failed to persist asset local path", zap.Error(err), zap.String("taskId", task.TaskID))
		}
		_ = a.emitEvent(output, contracts.EventTaskProgress, contracts.TaskEventDTO{Task: task})
		_ = a.emitEvent(output, contracts.EventAssetDownloaded, struct {
			Task      contracts.TaskDTO `json:"task"`
			MovieID   string            `json:"movieId"`
			Number    string            `json:"number"`
			Type      string            `json:"type"`
			SourceURL string            `json:"sourceUrl"`
			LocalPath string            `json:"localPath"`
		}{
			Task:      task,
			MovieID:   metadata.MovieID,
			Number:    metadata.Number,
			Type:      asset.Type,
			SourceURL: asset.SourceURL,
			LocalPath: asset.LocalPath,
		})
	}

	task = a.tasks.Complete(task.TaskID, fmt.Sprintf("Downloaded %d assets for %s", len(downloaded), metadata.Number))
	task.Provider = strings.TrimSpace(metadata.Provider)
	if err := a.store.SaveTask(ctx, task); err != nil {
		a.logger.Error("failed to persist completed asset task", zap.Error(err), zap.String("taskId", task.TaskID))
	}
	_ = a.emitEvent(output, contracts.EventTaskCompleted, contracts.TaskEventDTO{Task: task})
}

func (a *App) downloadActorAvatar(actorName, sourceURL, referer string) {
	ctx, cancel := context.WithTimeout(a.appCtx, 45*time.Second)
	defer cancel()
	localPath, httpStatus, err := a.assets.DownloadActorAvatar(ctx, actorName, sourceURL, assets.ImageFetchOptions{Referer: referer})
	if err != nil {
		if saveErr := a.store.UpdateActorAvatarCache(ctx, actorName, "", httpStatus, err.Error()); saveErr != nil && a.logger != nil {
			a.logger.Warn("failed to persist actor avatar failure", zap.Error(saveErr), zap.String("actorName", actorName))
		}
		return
	}
	if err := a.store.UpdateActorAvatarCache(ctx, actorName, localPath, httpStatus, ""); err != nil && a.logger != nil {
		a.logger.Warn("failed to persist actor avatar cache", zap.Error(err), zap.String("actorName", actorName))
	}
}

func classifyNetworkError(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "no such host"), strings.Contains(msg, "lookup "):
		return "dns_failure"
	case strings.Contains(msg, "timeout"):
		return "connect_timeout"
	case strings.Contains(msg, "tls"):
		return "tls_failure"
	case strings.Contains(msg, "restricted"), strings.Contains(msg, "geo"), strings.Contains(msg, "region"):
		return "region_restricted"
	case strings.Contains(msg, "hotlink"), strings.Contains(msg, "referer"), strings.Contains(msg, "forbidden"):
		return "hotlink_denied"
	case strings.Contains(msg, "no results"):
		return "provider_empty_result"
	case strings.Contains(msg, "parse"):
		return "parser_failed"
	default:
		return "provider_invalid_content"
	}
}

func (a *App) respondOK(output io.Writer, id string, data any) error {
	return a.writeResponse(output, contracts.Response{
		Kind:      "response",
		ID:        id,
		OK:        true,
		Data:      data,
		Timestamp: nowUTC(),
	})
}

func (a *App) respondError(output io.Writer, id string, appErr *contracts.AppError) error {
	return a.writeResponse(output, contracts.Response{
		Kind:      "response",
		ID:        id,
		OK:        false,
		Error:     appErr,
		Timestamp: nowUTC(),
	})
}

func (a *App) emitEvent(output io.Writer, eventType string, payload any) error {
	return a.writeEvent(output, contracts.Event{
		Kind:      "event",
		Type:      eventType,
		Payload:   payload,
		Timestamp: nowUTC(),
	})
}

func (a *App) writeResponse(output io.Writer, response contracts.Response) error {
	a.writeMu.Lock()
	defer a.writeMu.Unlock()

	return json.NewEncoder(output).Encode(response)
}

func (a *App) writeEvent(output io.Writer, event contracts.Event) error {
	a.writeMu.Lock()
	defer a.writeMu.Unlock()

	return json.NewEncoder(output).Encode(event)
}

func decodePayload(payload json.RawMessage, target any) error {
	if len(payload) == 0 {
		return nil
	}
	return json.Unmarshal(payload, target)
}

func badRequest(message string, details map[string]any) *contracts.AppError {
	return &contracts.AppError{
		Code:      contracts.ErrorCodeBadRequest,
		Message:   message,
		Retryable: false,
		Details:   details,
	}
}

func notFound(message string, details map[string]any) *contracts.AppError {
	return &contracts.AppError{
		Code:      contracts.ErrorCodeNotFound,
		Message:   message,
		Retryable: false,
		Details:   details,
	}
}

func internalError(message string, details map[string]any) *contracts.AppError {
	return &contracts.AppError{
		Code:      contracts.ErrorCodeInternal,
		Message:   message,
		Retryable: true,
		Details:   details,
	}
}

func (a *App) resolveScanPaths(ctx context.Context, paths []string) ([]string, error) {
	if len(paths) > 0 {
		return paths, nil
	}
	dbPaths, err := a.store.ListLibraryPathStrings(ctx)
	if err != nil {
		return nil, err
	}
	if len(dbPaths) > 0 {
		return dbPaths, nil
	}
	return append([]string(nil), a.cfg.LibraryPaths...), nil
}

// StartAutoScanLoop runs periodic library scans until ctx is cancelled. No-op if AutoScanIntervalSeconds <= 0.
func (a *App) StartAutoScanLoop(ctx context.Context) {
	secs := a.cfg.AutoScanIntervalSeconds
	if secs <= 0 {
		return
	}
	interval := time.Duration(secs) * time.Second
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				task, err := a.StartScan(context.Background(), nil)
				if err != nil {
					if errors.Is(err, contracts.ErrScanAlreadyRunning) {
						a.logger.Debug("auto scan skipped: scan already in progress")
						continue
					}
					a.logger.Warn("auto scan failed", zap.Error(err))
					continue
				}
				a.logger.Info("auto scan started",
					zap.String("taskId", task.TaskID),
					zap.Int("intervalSeconds", secs),
				)
			}
		}
	}()
}

func (a *App) StartScan(ctx context.Context, paths []string) (contracts.TaskDTO, error) {
	return a.startLibraryScan(ctx, io.Discard, paths, nil)
}

func (a *App) ResolvePlayback(ctx context.Context, movieID string) (contracts.PlaybackDescriptorDTO, error) {
	detail, err := a.store.GetMovieDetail(ctx, movieID)
	if err != nil {
		return contracts.PlaybackDescriptorDTO{}, err
	}
	progress, err := a.store.GetPlaybackProgress(ctx, movieID)
	if err != nil && a.logger != nil {
		a.logger.Warn("get playback progress failed", zap.Error(err), zap.String("movieId", movieID))
	}

	location := strings.TrimSpace(detail.Location)
	mediaInfo := a.resolvePlaybackMediaInfo(ctx, location)
	durationSec := a.resolvePlaybackDurationSec(ctx, location, detail, progress, mediaInfo)
	decision := buildPlaybackDecision(playbackDecisionInput{
		Location:          location,
		MediaInfo:         mediaInfo,
		StreamPushEnabled: a.streams != nil && a.streams.Enabled(),
		ForceStreamPush:   a.cfg.Player.ForceStreamPush,
	})
	descriptor := buildDirectPlaybackDescriptor(movieID, detail, progress, durationSec, decision)
	if decision.Mode == contracts.PlaybackModeHLS {
		startPositionSec := 0.0
		if progress != nil && progress.PositionSec > 0 {
			startPositionSec = progress.PositionSec
		}
		if durationSec > 0 && startPositionSec > durationSec {
			startPositionSec = durationSec
		}
		session, err := a.streams.StartHLSSession(ctx, movieID, location, playback.StartHLSSessionOptions{
			StartPositionSec: startPositionSec,
			PreferRemux:      decision.PreferRemux,
			SourceVideoCodec: decision.SourceVideoCodec,
			SourceAudioCodec: decision.SourceAudioCodec,
		})
		if err != nil {
			if a.logger != nil {
				a.logger.Warn("failed to start HLS playback session; falling back to direct", zap.Error(err), zap.String("movieId", movieID))
			}
			descriptor.ReasonCode = "hls_session_start_failed"
			descriptor.ReasonMessage = "HLS session startup failed; fell back to direct playback."
			descriptor.Reason = descriptor.ReasonMessage + " " + strings.TrimSpace(err.Error())
			return descriptor, nil
		}
		descriptor.Mode = contracts.PlaybackModeHLS
		descriptor.SessionID = session.ID
		descriptor.URL = "/api/playback/sessions/" + session.ID + "/hls/index.m3u8"
		descriptor.MimeType = "application/vnd.apple.mpegurl"
		descriptor.StartPositionSec = session.StartPositionSec
		descriptor.TranscodeProfile = session.ProfileName
		descriptor.SessionKind = session.Kind
		descriptor.CanDirectPlay = false
		descriptor.ReasonCode = decision.ReasonCode
		descriptor.ReasonMessage = decision.ReasonMessage
		descriptor.Reason = decision.ReasonMessage
	}
	return descriptor, nil
}

func (a *App) CreatePlaybackSession(ctx context.Context, movieID string, mode contracts.PlaybackMode, startPositionSec float64) (contracts.PlaybackDescriptorDTO, error) {
	if mode == "" || mode == contracts.PlaybackModeDirect {
		return a.ResolvePlayback(ctx, movieID)
	}
	detail, err := a.store.GetMovieDetail(ctx, movieID)
	if err != nil {
		return contracts.PlaybackDescriptorDTO{}, err
	}
	progress, err := a.store.GetPlaybackProgress(ctx, movieID)
	if err != nil && a.logger != nil {
		a.logger.Warn("get playback progress failed", zap.Error(err), zap.String("movieId", movieID))
	}
	location := strings.TrimSpace(detail.Location)
	mediaInfo := a.resolvePlaybackMediaInfo(ctx, location)
	durationSec := a.resolvePlaybackDurationSec(ctx, location, detail, progress, mediaInfo)
	decision := buildPlaybackDecision(playbackDecisionInput{
		Location:          location,
		MediaInfo:         mediaInfo,
		StreamPushEnabled: a.streams != nil && a.streams.Enabled(),
		ForceStreamPush:   a.cfg.Player.ForceStreamPush,
	})
	switch mode {
	case contracts.PlaybackModeHLS:
		if startPositionSec < 0 {
			startPositionSec = 0
		}
		if durationSec > 0 && startPositionSec > durationSec {
			startPositionSec = durationSec
		}
		session, err := a.streams.StartHLSSession(ctx, movieID, location, playback.StartHLSSessionOptions{
			StartPositionSec: startPositionSec,
			PreferRemux:      decision.PreferRemux,
			SourceVideoCodec: decision.SourceVideoCodec,
			SourceAudioCodec: decision.SourceAudioCodec,
		})
		if err != nil {
			return contracts.PlaybackDescriptorDTO{}, err
		}
		dto := buildDirectPlaybackDescriptor(movieID, detail, progress, durationSec, decision)
		dto.Mode = contracts.PlaybackModeHLS
		dto.SessionID = session.ID
		dto.URL = "/api/playback/sessions/" + session.ID + "/hls/index.m3u8"
		dto.MimeType = "application/vnd.apple.mpegurl"
		dto.StartPositionSec = session.StartPositionSec
		dto.ResumePositionSec = startPositionSec
		dto.TranscodeProfile = session.ProfileName
		dto.SessionKind = session.Kind
		dto.CanDirectPlay = false
		dto.ReasonCode = decision.ReasonCode
		dto.ReasonMessage = "Explicit HLS playback session created from the current playback plan."
		dto.Reason = dto.ReasonMessage
		return dto, nil
	default:
		return contracts.PlaybackDescriptorDTO{}, fmt.Errorf("unsupported playback mode %q", mode)
	}
}

func (a *App) LaunchNativePlayback(ctx context.Context, movieID string, startPositionSec float64) (contracts.NativePlaybackLaunchDTO, error) {
	targetPath, err := a.store.ResolvePrimaryVideoPath(ctx, movieID)
	if err != nil {
		return contracts.NativePlaybackLaunchDTO{}, err
	}
	detail, err := a.store.GetMovieDetail(ctx, movieID)
	if err != nil {
		return contracts.NativePlaybackLaunchDTO{}, err
	}
	title := strings.TrimSpace(detail.Code)
	if title == "" {
		title = strings.TrimSpace(detail.Title)
	}
	if err := a.player.Launch(context.Background(), targetPath, startPositionSec, title); err != nil {
		return contracts.NativePlaybackLaunchDTO{}, err
	}
	return contracts.NativePlaybackLaunchDTO{
		OK:        true,
		Command:   a.player.Command(),
		Target:    targetPath,
		Mode:      string(contracts.PlaybackModeNative),
		Message:   "native player launched",
		MovieID:   movieID,
		StartedAt: nowUTC(),
	}, nil
}

func (a *App) ResolvePlaybackSessionFile(sessionID string, name string) (string, error) {
	return a.streams.ResolveFile(sessionID, name)
}

func (a *App) DeletePlaybackSession(sessionID string) error {
	return a.streams.DeleteSession(sessionID)
}

func (a *App) GetPlaybackSession(ctx context.Context, sessionID string) (contracts.PlaybackSessionStatusDTO, error) {
	_ = ctx
	snapshot, err := a.streams.GetSessionSnapshot(sessionID)
	if err != nil {
		return contracts.PlaybackSessionStatusDTO{}, err
	}
	return playbackSessionStatusDTO(snapshot), nil
}

func (a *App) ListRecentPlaybackSessions(ctx context.Context, limit int) (contracts.PlaybackSessionListDTO, error) {
	_ = ctx
	snapshots := a.streams.ListSessionSnapshots(limit)
	items := make([]contracts.PlaybackSessionStatusDTO, 0, len(snapshots))
	for _, snapshot := range snapshots {
		items = append(items, playbackSessionStatusDTO(snapshot))
	}
	return contracts.PlaybackSessionListDTO{Items: items}, nil
}

func (a *App) shouldPreferHLS(location string) bool {
	decision := buildPlaybackDecision(playbackDecisionInput{
		Location:          location,
		StreamPushEnabled: a.streams != nil && a.streams.Enabled(),
		ForceStreamPush:   a.cfg.Player.ForceStreamPush,
	})
	return decision.Mode == contracts.PlaybackModeHLS
}

func (a *App) resolvePlaybackDurationSec(
	ctx context.Context,
	location string,
	detail contracts.MovieDetailDTO,
	progress *storage.PlaybackProgressRow,
	mediaInfo playback.MediaInfo,
) float64 {
	runtimeDurationSec := 0.0
	if detail.RuntimeMinutes > 0 {
		runtimeDurationSec = float64(detail.RuntimeMinutes) * 60
	}
	progressDurationSec := 0.0
	if progress != nil && progress.DurationSec > 0 {
		progressDurationSec = progress.DurationSec
	}
	probedDurationSec := mediaInfo.DurationSec
	if probedDurationSec <= 0 && strings.TrimSpace(location) != "" {
		duration, err := playback.ProbeMediaDuration(ctx, location, a.cfg.Player.FFmpegCommand)
		if err == nil {
			probedDurationSec = duration
		} else if a.logger != nil {
			a.logger.Debug("playback duration probe failed", zap.Error(err), zap.String("location", location))
		}
	}
	return choosePlaybackDurationSec(probedDurationSec, runtimeDurationSec, progressDurationSec)
}

func choosePlaybackDurationSec(probedDurationSec float64, runtimeDurationSec float64, progressDurationSec float64) float64 {
	if probedDurationSec > 0 {
		return probedDurationSec
	}
	if runtimeDurationSec > 0 && progressDurationSec > 0 {
		if runtimeDurationSec >= progressDurationSec {
			return runtimeDurationSec
		}
		return progressDurationSec
	}
	if runtimeDurationSec > 0 {
		return runtimeDurationSec
	}
	if progressDurationSec > 0 {
		return progressDurationSec
	}
	return 0
}

func buildDirectPlaybackDescriptor(
	movieID string,
	detail contracts.MovieDetailDTO,
	progress *storage.PlaybackProgressRow,
	durationSec float64,
	decision playbackDecision,
) contracts.PlaybackDescriptorDTO {
	fileName := filepath.Base(strings.TrimSpace(detail.Location))
	mimeType, canDirectPlay := resolveDirectPlaybackMimeType(fileName)
	if decision.CanDirectPlay {
		canDirectPlay = true
	}

	dto := contracts.PlaybackDescriptorDTO{
		MovieID:          movieID,
		Mode:             contracts.PlaybackModeDirect,
		URL:              "/api/library/movies/" + url.PathEscape(movieID) + "/stream",
		MimeType:         mimeType,
		FileName:         fileName,
		DurationSec:      durationSec,
		CanDirectPlay:    canDirectPlay,
		SessionKind:      decision.SessionKind,
		ReasonCode:       decision.ReasonCode,
		ReasonMessage:    decision.ReasonMessage,
		Reason:           decision.ReasonMessage,
		SourceContainer:  decision.SourceContainer,
		SourceVideoCodec: decision.SourceVideoCodec,
		SourceAudioCodec: decision.SourceAudioCodec,
		AudioTracks:      []contracts.PlaybackAudioTrackDTO{},
		SubtitleTracks:   []contracts.PlaybackSubtitleTrackDTO{},
	}
	if progress != nil && progress.PositionSec > 0 {
		dto.ResumePositionSec = progress.PositionSec
		if durationSec > 0 && dto.ResumePositionSec > durationSec {
			dto.ResumePositionSec = durationSec
		}
	}
	return dto
}

func playbackSessionStatusDTO(snapshot playback.SessionSnapshot) contracts.PlaybackSessionStatusDTO {
	return contracts.PlaybackSessionStatusDTO{
		SessionID:        snapshot.Session.ID,
		MovieID:          snapshot.Session.MovieID,
		SessionKind:      snapshot.Session.Kind,
		TranscodeProfile: snapshot.Session.ProfileName,
		StartPositionSec: snapshot.Session.StartPositionSec,
		StartedAt:        formatOptionalPlaybackTime(snapshot.Session.StartedAt),
		LastAccessedAt:   formatOptionalPlaybackTime(snapshot.LastAccessedAt),
		ExpiresAt:        formatOptionalPlaybackTime(snapshot.ExpiresAt),
		FinishedAt:       formatOptionalPlaybackTime(snapshot.FinishedAt),
		State:            snapshot.State,
		LastError:        strings.TrimSpace(snapshot.LastError),
	}
}

func formatOptionalPlaybackTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func (a *App) resolvePlaybackMediaInfo(ctx context.Context, location string) playback.MediaInfo {
	location = strings.TrimSpace(location)
	if location == "" {
		return playback.MediaInfo{}
	}

	mediaInfo, err := playback.ProbeMediaInfo(ctx, location, a.cfg.Player.FFmpegCommand)
	if err != nil {
		if a.logger != nil {
			a.logger.Debug("playback media probe failed", zap.Error(err), zap.String("location", location))
		}
		return playback.MediaInfo{}
	}
	return mediaInfo
}

func resolveDirectPlaybackMimeType(fileName string) (mimeType string, canDirectPlay bool) {
	switch strings.ToLower(filepath.Ext(strings.TrimSpace(fileName))) {
	case ".mp4", ".m4v":
		return "video/mp4", true
	case ".webm":
		return "video/webm", true
	case ".ogv":
		return "video/ogg", true
	case ".m3u8":
		return "application/vnd.apple.mpegurl", true
	default:
		return "application/octet-stream", false
	}
}

// startLibraryScan starts a library scan task. extraMeta is merged into task metadata (e.g. trigger: fsnotify).
func (a *App) startLibraryScan(ctx context.Context, output io.Writer, paths []string, extraMeta map[string]any) (contracts.TaskDTO, error) {
	if !a.scanning.CompareAndSwap(false, true) {
		return contracts.TaskDTO{}, contracts.ErrScanAlreadyRunning
	}

	resolved, err := a.resolveScanPaths(ctx, paths)
	if err != nil {
		a.scanning.Store(false)
		return contracts.TaskDTO{}, err
	}
	paths = resolved

	meta := map[string]any{"paths": paths}
	for k, v := range extraMeta {
		meta[k] = v
	}
	task := a.tasks.Create("scan.library", meta)
	task = a.tasks.Start(task.TaskID, fmt.Sprintf("Scanning %d library path(s)", len(paths)))
	if err := a.store.SaveTask(a.appCtx, task); err != nil {
		a.scanning.Store(false)
		return contracts.TaskDTO{}, err
	}

	trigger := "manual"
	if extraMeta != nil {
		if v, ok := extraMeta["trigger"]; ok {
			if s, ok := v.(string); ok && s != "" {
				trigger = s
			}
		}
	}
	pathsPreview := strings.Join(paths, ", ")
	if len(pathsPreview) > 200 {
		pathsPreview = pathsPreview[:200] + "..."
	}
	a.logger.Info("scan.library started",
		zap.String("taskId", task.TaskID),
		zap.String("trigger", trigger),
		zap.Int("pathCount", len(paths)),
		zap.String("pathsPreview", pathsPreview),
	)

	go func() {
		defer func() {
			a.scanning.Store(false)
			a.tryDrainWatchScanQueue()
		}()
		a.runScan(a.appCtx, output, task.TaskID, paths)
	}()

	return task, nil
}

func (a *App) HTTPHandler() http.Handler {
	apiHandler := server.NewHandler(server.Deps{
		Cfg:                       a.cfg,
		Logger:                    a.logger,
		Store:                     a.store,
		Tasks:                     a.tasks,
		ScanStarter:               a,
		OrganizeLibraryCtl:        a,
		ExtendedLibraryImportCtl:  a,
		AutoLibraryWatchCtl:       a,
		AutoActorProfileScrapeCtl: a,
		LaunchAtLoginCtl:          a,
		MetadataScrapeCtl:         a,
		ProviderHealthChecker:     a.scraper,
		ProxyCtl:                  a,
		BackendLogCtl:             a,
		PlayerSettingsCtl:         a,
		MovieMetadataRefresher:    a,
		ActorProfileRefresher:     a,
		LibraryWatchReloader:      a,
		DevPerformanceProvider:    a,
		PlaybackResolver:          a,
		NativePlaybackLauncher:    a,
		HomepageRecommendations:   a,
	}).Routes()
	return webui.WrapHandler(apiHandler)
}

func (a *App) GetDevPerformanceSummary(ctx context.Context) contracts.DevPerformanceSummaryDTO {
	if a == nil || a.devCPUSampler == nil {
		return contracts.DevPerformanceSummaryDTO{Supported: false}
	}
	snapshot := a.devCPUSampler.Snapshot(ctx)
	return contracts.DevPerformanceSummaryDTO{
		Supported:         snapshot.Supported,
		SampledAt:         snapshot.SampledAt,
		SystemCPUPercent:  snapshot.SystemCPUPercent,
		BackendCPUPercent: snapshot.BackendCPUPercent,
	}
}

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}

package config

import (
	"curated-backend/internal/version"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	LogLevel string `json:"logLevel"`
	// LogDir stores the effective backend log directory. Empty config values are normalized
	// to the build-specific default (dev: project runtime/logs; release: app-data logs).
	LogDir string `json:"logDir,omitempty"`
	// LogFilePrefix is the base name for files like {prefix}-20060102.log; default is channel-specific when LogDir is set.
	LogFilePrefix string `json:"logFilePrefix,omitempty"`
	// LogMaxAgeDays removes rotated files older than this many days; 0 means default 7 when file logging is enabled.
	LogMaxAgeDays int      `json:"logMaxAgeDays,omitempty"`
	HttpAddr      string   `json:"httpAddr"`
	DatabasePath  string   `json:"databasePath"`
	CacheDir      string   `json:"cacheDir"`
	LibraryPaths  []string `json:"libraryPaths"`
	// ScanIntervalSeconds is deprecated: kept in JSON for backward compatibility with older config files; ignored (no scheduled scan).
	ScanIntervalSeconds int `json:"scanIntervalSeconds,omitempty"`
	// OrganizeLibrary moves/renames video files into {parent}/{番号}/{番号}.ext and stores NFO/assets beside the video when enabled.
	OrganizeLibrary bool `json:"organizeLibrary"`
	// AutoLibraryWatch: when true (default), fsnotify on library roots may queue debounced scans (and follow-on scrape). Persisted in library-config.cfg.
	AutoLibraryWatch bool `json:"autoLibraryWatch"`
	// ExtendedLibraryImport: first scan on a newly added library root may run curated/external layout detection (library-config.cfg). Default false for zero impact on existing libraries.
	ExtendedLibraryImport bool `json:"extendedLibraryImport,omitempty"`
	// MetadataMovieProvider is the Metatube movie provider name for scrapes; empty = auto (SearchMovieAll). Usually set via library-config.cfg merge, not main config.yaml.
	MetadataMovieProvider string `json:"metadataMovieProvider,omitempty"`
	// MetadataMovieProviderChain is an ordered list of providers to try in sequence; empty = auto. Takes precedence over MetadataMovieProvider when non-empty.
	MetadataMovieProviderChain []string `json:"metadataMovieProviderChain,omitempty"`
	// MetadataMovieScrapeMode is auto | specified | chain; empty means infer from legacy chain/provider (pre-migration configs).
	MetadataMovieScrapeMode string `json:"metadataMovieScrapeMode,omitempty"`
	// MetadataMovieStrategy refines provider scheduling while keeping legacy mode compatibility.
	MetadataMovieStrategy string `json:"metadataMovieStrategy,omitempty"`
	// AutoScanIntervalSeconds runs a full library scan on this interval; 0 disables (manual POST /api/scans only).
	AutoScanIntervalSeconds int `json:"autoScanIntervalSeconds"`
	// LibraryWatchEnabled: nil = default on (fsnotify on library roots); explicit false disables.
	LibraryWatchEnabled *bool `json:"libraryWatchEnabled,omitempty"`
	// LibraryWatchDebounceMs merges fsnotify events before starting a scan; 0 = default 1500ms.
	LibraryWatchDebounceMs int           `json:"libraryWatchDebounceMs,omitempty"`
	Tasks                  TaskConfig    `json:"tasks"`
	Scraper                ScraperConfig `json:"scraper"`
	Assets                 AssetConfig   `json:"assets"`
	Player                 PlayerConfig  `json:"player"`
	// Proxy configures HTTP/SOCKS5 proxy for outbound metadata scraping requests. Persisted in library-config.cfg.
	Proxy ProxyConfig `json:"proxy,omitempty"`
}

type TaskConfig struct {
	ScanTimeoutSeconds int `json:"scanTimeoutSeconds"`
}

type ScraperConfig struct {
	RequestTimeoutSeconds int `json:"requestTimeoutSeconds"`
	TaskTimeoutSeconds    int `json:"taskTimeoutSeconds"`
	// MaxConcurrent limits parallel scrape.movie goroutines after scan (0 = default 4).
	MaxConcurrent int `json:"maxConcurrent,omitempty"`
}

type AssetConfig struct {
	RequestTimeoutSeconds int `json:"requestTimeoutSeconds"`
	TaskTimeoutSeconds    int `json:"taskTimeoutSeconds"`
	// MaxConcurrentDownloads limits parallel HTTP fetches per asset.download task (0 = default 3).
	MaxConcurrentDownloads int `json:"maxConcurrentDownloads,omitempty"`
	// MaxResponseBodyMB caps a single HTTP response body when saving an asset (0 = default 50).
	MaxResponseBodyMB int `json:"maxResponseBodyMB,omitempty"`
}

type PlayerConfig struct {
	HardwareDecode bool `json:"hardwareDecode"`
	// HardwareEncoder sets the preferred hardware encoder order for HLS stream push:
	// auto | amf | qsv | nvenc | videotoolbox | software.
	HardwareEncoder string `json:"hardwareEncoder,omitempty"`
	// NativePlayerPreset controls which argument style Curated uses when launching
	// an external player: mpv | potplayer | custom.
	NativePlayerPreset string `json:"nativePlayerPreset,omitempty"`
	// NativePlayerEnabled allows the backend to launch an external native player process (for example mpv).
	NativePlayerEnabled bool `json:"nativePlayerEnabled,omitempty"`
	// NativePlayerCommand is the executable used for native playback; default "mpv".
	NativePlayerCommand string `json:"nativePlayerCommand,omitempty"`
	// NativePlayerArgs are prepended before Curated appends the media path/URL.
	NativePlayerArgs []string `json:"nativePlayerArgs,omitempty"`
	// StreamPushEnabled allows the backend to create HLS playback sessions with ffmpeg and is disabled by default.
	StreamPushEnabled bool `json:"streamPushEnabled,omitempty"`
	// ForceStreamPush forces browser playback to prefer HLS stream push even for formats that are usually direct-play friendly.
	ForceStreamPush bool `json:"forceStreamPush,omitempty"`
	// FFmpegCommand is the executable used for HLS transcoding; default "ffmpeg".
	// When left at the default command, runtime prefers a bundled third_party/ffmpeg binary if present.
	FFmpegCommand string `json:"ffmpegCommand,omitempty"`
	// StreamSessionRoot stores generated HLS playlists and segments; default under cacheDir/playback-sessions.
	StreamSessionRoot string `json:"streamSessionRoot,omitempty"`
	// PreferNativePlayer can be used by future UI flows to default into native playback when available.
	PreferNativePlayer bool `json:"preferNativePlayer,omitempty"`
	// SeekForwardStepSec is the default skip-forward duration for keyboard/UI seeks.
	SeekForwardStepSec int `json:"seekForwardStepSec,omitempty"`
	// SeekBackwardStepSec is the default skip-back duration for keyboard/UI seeks.
	SeekBackwardStepSec int `json:"seekBackwardStepSec,omitempty"`
}

type ProxyConfig struct {
	// Enabled toggles HTTP proxy for outbound scraper and metadata requests.
	Enabled bool `json:"enabled"`
	// URL is the proxy URL (e.g., http://proxy.example.com:8080 or socks5://proxy:1080).
	URL string `json:"url,omitempty"`
	// Username for proxy authentication (optional).
	Username string `json:"username,omitempty"`
	// Password for proxy authentication (optional).
	Password string `json:"password,omitempty"`
}

// DefaultHTTPAddr returns the compiled default HTTP listen address: :8080 for normal builds,
// :8081 for release builds (go build -tags release).
func DefaultHTTPAddr() string {
	return defaultHTTPAddr()
}

func Default() Config {
	cacheDir := defaultCacheDir()
	return Config{
		LogLevel:     "info",
		LogDir:       defaultLogDir(),
		HttpAddr:     defaultHTTPAddr(),
		DatabasePath: defaultDatabasePath(),
		CacheDir:     cacheDir,
		LibraryPaths: defaultLibraryPaths(),
		Tasks: TaskConfig{
			ScanTimeoutSeconds: 600,
		},
		Scraper: ScraperConfig{
			RequestTimeoutSeconds: 45,
			TaskTimeoutSeconds:    120,
			MaxConcurrent:         4,
		},
		Assets: AssetConfig{
			RequestTimeoutSeconds:  30,
			TaskTimeoutSeconds:     180,
			MaxConcurrentDownloads: 3,
			MaxResponseBodyMB:      50,
		},
		Player: PlayerConfig{
			HardwareDecode:      false,
			HardwareEncoder:     "auto",
			NativePlayerPreset:  "mpv",
			NativePlayerEnabled: true,
			NativePlayerCommand: "mpv",
			StreamPushEnabled:   false,
			ForceStreamPush:     false,
			FFmpegCommand:       "ffmpeg",
			StreamSessionRoot:   filepath.Join(cacheDir, "playback-sessions"),
			SeekForwardStepSec:  10,
			SeekBackwardStepSec: 10,
		},
		OrganizeLibrary:  true,
		AutoLibraryWatch: true,
	}
}

func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		return cfg, nil
	}

	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return Config{}, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, err
	}

	if cfg.DatabasePath == "" {
		cfg.DatabasePath = defaultDatabasePath()
	}
	if cfg.CacheDir == "" {
		cfg.CacheDir = defaultCacheDir()
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}
	cfg.LogDir = ResolveLogDir(cfg.LogDir)
	if strings.TrimSpace(cfg.LogDir) != "" {
		if cfg.LogFilePrefix == "" {
			cfg.LogFilePrefix = version.DefaultLogFilePrefix()
		}
		if cfg.LogMaxAgeDays <= 0 {
			cfg.LogMaxAgeDays = 7
		}
	}
	if cfg.HttpAddr == "" {
		cfg.HttpAddr = defaultHTTPAddr()
	}
	if cfg.Tasks.ScanTimeoutSeconds <= 0 {
		cfg.Tasks.ScanTimeoutSeconds = 600
	}
	if cfg.Scraper.RequestTimeoutSeconds <= 0 {
		cfg.Scraper.RequestTimeoutSeconds = 45
	}
	if cfg.Scraper.TaskTimeoutSeconds <= 0 {
		cfg.Scraper.TaskTimeoutSeconds = 120
	}
	if cfg.Scraper.MaxConcurrent <= 0 {
		cfg.Scraper.MaxConcurrent = 4
	}
	if cfg.Assets.RequestTimeoutSeconds <= 0 {
		cfg.Assets.RequestTimeoutSeconds = 30
	}
	if cfg.Assets.TaskTimeoutSeconds <= 0 {
		cfg.Assets.TaskTimeoutSeconds = 180
	}
	if cfg.Assets.MaxConcurrentDownloads <= 0 {
		cfg.Assets.MaxConcurrentDownloads = 3
	}
	if cfg.Assets.MaxResponseBodyMB <= 0 {
		cfg.Assets.MaxResponseBodyMB = 50
	}
	if strings.TrimSpace(cfg.Player.NativePlayerCommand) == "" {
		cfg.Player.NativePlayerCommand = "mpv"
	}
	cfg.Player.NativePlayerPreset = NormalizeNativePlayerPreset(cfg.Player.NativePlayerPreset)
	if strings.TrimSpace(cfg.Player.FFmpegCommand) == "" {
		cfg.Player.FFmpegCommand = "ffmpeg"
	}
	cfg.Player.HardwareEncoder = NormalizeHardwareEncoderPreference(cfg.Player.HardwareEncoder)
	if strings.TrimSpace(cfg.Player.StreamSessionRoot) == "" {
		cfg.Player.StreamSessionRoot = filepath.Join(cfg.CacheDir, "playback-sessions")
	}
	if cfg.Player.SeekForwardStepSec <= 0 {
		cfg.Player.SeekForwardStepSec = 10
	}
	if cfg.Player.SeekBackwardStepSec <= 0 {
		cfg.Player.SeekBackwardStepSec = 10
	}

	return cfg, nil
}

func defaultLibraryPaths() []string {
	if root := curatedDataRoot(); root != "" {
		return nil
	}
	cwd, err := os.Getwd()
	if err == nil && filepath.Base(cwd) == "backend" {
		return []string{
			filepath.FromSlash("../videos_test"),
			filepath.FromSlash("../docs/film-scanner/videos_test"),
		}
	}

	return []string{
		filepath.FromSlash("videos_test"),
		filepath.FromSlash("docs/film-scanner/videos_test"),
	}
}

func defaultCacheDir() string {
	if root := curatedDataRoot(); root != "" {
		return filepath.Join(root, "cache")
	}
	cwd, err := os.Getwd()
	if err == nil && filepath.Base(cwd) == "backend" {
		return filepath.FromSlash("runtime/cache")
	}

	return filepath.FromSlash("backend/runtime/cache")
}

func defaultDatabasePath() string {
	if root := curatedDataRoot(); root != "" {
		return filepath.Join(root, "data", "curated.db")
	}
	cwd, err := os.Getwd()
	if err == nil && filepath.Base(cwd) == "backend" {
		return filepath.FromSlash("runtime/curated.db")
	}

	return filepath.FromSlash("backend/runtime/curated.db")
}

func NormalizeHardwareEncoderPreference(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "auto":
		return "auto"
	case "amf", "qsv", "nvenc", "software", "videotoolbox":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "auto"
	}
}

func NormalizeNativePlayerPreset(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "mpv":
		return "mpv"
	case "potplayer", "custom":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "mpv"
	}
}

// LibraryWatchOn reports whether fsnotify-based library watching should run.
func (c Config) LibraryWatchOn() bool {
	if c.LibraryWatchEnabled == nil {
		return true
	}
	return *c.LibraryWatchEnabled
}

// LibraryWatchDebounce returns debounce duration for coalescing watch events before scan.
func (c Config) LibraryWatchDebounce() time.Duration {
	ms := c.LibraryWatchDebounceMs
	if ms <= 0 {
		ms = 1500
	}
	return time.Duration(ms) * time.Millisecond
}

package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	LogLevel     string   `json:"logLevel"`
	// LogDir, if non-empty, enables daily rotated log files under this directory (e.g. "logs" or "runtime/logs").
	LogDir string `json:"logDir,omitempty"`
	// LogFilePrefix is the base name for files like {prefix}-20060102.log; default "javd" when LogDir is set.
	LogFilePrefix string `json:"logFilePrefix,omitempty"`
	// LogMaxAgeDays removes rotated files older than this many days; 0 means default 7 when file logging is enabled.
	LogMaxAgeDays int `json:"logMaxAgeDays,omitempty"`
	HttpAddr     string   `json:"httpAddr"`
	DatabasePath string   `json:"databasePath"`
	CacheDir     string   `json:"cacheDir"`
	LibraryPaths []string `json:"libraryPaths"`
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

func Default() Config {
	return Config{
		LogLevel:     "info",
		HttpAddr:     ":8080",
		DatabasePath: defaultDatabasePath(),
		CacheDir:     defaultCacheDir(),
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
			HardwareDecode: true,
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
	if strings.TrimSpace(cfg.LogDir) != "" {
		if cfg.LogFilePrefix == "" {
			cfg.LogFilePrefix = "javd"
		}
		if cfg.LogMaxAgeDays <= 0 {
			cfg.LogMaxAgeDays = 7
		}
	}
	if cfg.HttpAddr == "" {
		cfg.HttpAddr = ":8080"
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

	return cfg, nil
}

func defaultLibraryPaths() []string {
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
	cwd, err := os.Getwd()
	if err == nil && filepath.Base(cwd) == "backend" {
		return filepath.FromSlash("runtime/cache")
	}

	return filepath.FromSlash("backend/runtime/cache")
}

func defaultDatabasePath() string {
	cwd, err := os.Getwd()
	if err == nil && filepath.Base(cwd) == "backend" {
		return filepath.FromSlash("runtime/jav-library.db")
	}

	return filepath.FromSlash("backend/runtime/jav-library.db")
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

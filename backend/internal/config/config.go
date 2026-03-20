package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type Config struct {
	LogLevel            string        `json:"logLevel"`
	HttpAddr            string        `json:"httpAddr"`
	DatabasePath        string        `json:"databasePath"`
	CacheDir            string        `json:"cacheDir"`
	LibraryPaths []string `json:"libraryPaths"`
	// ScanIntervalSeconds is deprecated: kept in JSON for backward compatibility with older config files; ignored (no scheduled scan).
	ScanIntervalSeconds int `json:"scanIntervalSeconds,omitempty"`
	// OrganizeLibrary moves/renames video files into {parent}/{番号}/{番号}.ext and stores NFO/assets beside the video when enabled.
	OrganizeLibrary bool `json:"organizeLibrary"`
	// AutoScanIntervalSeconds runs a full library scan on this interval; 0 disables (manual POST /api/scans only).
	AutoScanIntervalSeconds int `json:"autoScanIntervalSeconds"`
	Tasks               TaskConfig `json:"tasks"`
	Scraper             ScraperConfig `json:"scraper"`
	Assets              AssetConfig   `json:"assets"`
	Player              PlayerConfig  `json:"player"`
}

type TaskConfig struct {
	ScanTimeoutSeconds int `json:"scanTimeoutSeconds"`
}

type ScraperConfig struct {
	RequestTimeoutSeconds int `json:"requestTimeoutSeconds"`
	TaskTimeoutSeconds    int `json:"taskTimeoutSeconds"`
}

type AssetConfig struct {
	RequestTimeoutSeconds int `json:"requestTimeoutSeconds"`
	TaskTimeoutSeconds    int `json:"taskTimeoutSeconds"`
}

type PlayerConfig struct {
	HardwareDecode bool `json:"hardwareDecode"`
}

func Default() Config {
	return Config{
		LogLevel:            "info",
		HttpAddr:            ":8080",
		DatabasePath:        defaultDatabasePath(),
		CacheDir:            defaultCacheDir(),
		LibraryPaths: defaultLibraryPaths(),
		Tasks: TaskConfig{
			ScanTimeoutSeconds: 600,
		},
		Scraper: ScraperConfig{
			RequestTimeoutSeconds: 45,
			TaskTimeoutSeconds:    120,
		},
		Assets: AssetConfig{
			RequestTimeoutSeconds: 30,
			TaskTimeoutSeconds:    180,
		},
		Player: PlayerConfig{
			HardwareDecode: true,
		},
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
	if cfg.Assets.RequestTimeoutSeconds <= 0 {
		cfg.Assets.RequestTimeoutSeconds = 30
	}
	if cfg.Assets.TaskTimeoutSeconds <= 0 {
		cfg.Assets.TaskTimeoutSeconds = 180
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

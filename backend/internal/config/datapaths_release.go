//go:build release

package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// curatedDataRoot is the Jellyfin-style per-user data directory for release builds.
// Override with env CURATED_DATA_DIR (absolute or home-relative path).
//
// Default layout under this root:
//
//	config/library-config.cfg
//	data/curated.db
//	cache/
func curatedDataRoot() string {
	if custom := strings.TrimSpace(os.Getenv("CURATED_DATA_DIR")); custom != "" {
		// Expect an absolute path; relative values resolve against process cwd.
		return filepath.Clean(custom)
	}
	switch runtime.GOOS {
	case "windows":
		if d := strings.TrimSpace(os.Getenv("LOCALAPPDATA")); d != "" {
			return filepath.Join(d, "Curated")
		}
		dir, err := os.UserCacheDir()
		if err == nil {
			return filepath.Join(dir, "Curated")
		}
	case "darwin":
		dir, err := os.UserConfigDir()
		if err == nil {
			return filepath.Join(dir, "Curated")
		}
	default:
		if xdg := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); xdg != "" {
			return filepath.Join(xdg, "curated")
		}
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, ".local", "share", "curated")
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "Curated"
	}
	return filepath.Join(home, ".curated")
}

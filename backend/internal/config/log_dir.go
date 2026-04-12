package config

import "strings"

// DefaultLogDir returns the platform/build-specific default backend log directory.
func DefaultLogDir() string {
	return defaultLogDir()
}

// ResolveLogDir maps an optional config value to the effective backend log directory.
// Empty means "use the default directory", not "disable file logging".
func ResolveLogDir(dir string) string {
	if trimmed := strings.TrimSpace(dir); trimmed != "" {
		return trimmed
	}
	return defaultLogDir()
}

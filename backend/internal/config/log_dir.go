package config

// DefaultLogDir returns the platform/build-specific default backend log directory.
func DefaultLogDir() string {
	return defaultLogDir()
}

// ResolveLogDir keeps legacy configuration readable while always using the
// application-managed directory. Existing custom log files are left untouched.
func ResolveLogDir(_ string) string {
	return defaultLogDir()
}

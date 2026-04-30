//go:build !windows

package desktop

// LaunchAtLoginSupported always returns false on non-Windows platforms.
func LaunchAtLoginSupported() bool {
	return false
}

// SyncLaunchAtLogin is a no-op on non-Windows platforms unless enabling is requested.
func SyncLaunchAtLogin(enabled bool) error {
	if enabled {
		return ErrLaunchAtLoginUnsupported
	}
	return nil
}

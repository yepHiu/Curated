//go:build !windows

package desktop

func LaunchAtLoginSupported() bool {
	return false
}

func SyncLaunchAtLogin(enabled bool) error {
	if enabled {
		return ErrLaunchAtLoginUnsupported
	}
	return nil
}

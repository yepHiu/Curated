package version

import "runtime"

// BackendName is the backend/runtime identity exposed by health checks and logs.
// Dev builds use a distinct name so they can coexist with release installs.
func BackendName() string {
	if Channel == "release" {
		return "curated"
	}
	return "curated-dev"
}

// DefaultLogFilePrefix keeps dev/release log files easy to distinguish on the same machine.
func DefaultLogFilePrefix() string {
	return BackendName()
}

// TrayMutexName isolates Windows single-instance tray mode across dev/release channels.
func TrayMutexName() string {
	if Channel == "release" {
		return `Local\Curated.Tray.Singleton`
	}
	return `Local\Curated.Dev.Tray.Singleton`
}

// SuggestedBinaryName is the recommended local build artifact name for the current channel.
func SuggestedBinaryName() string {
	if runtime.GOOS == "windows" {
		return BackendName() + ".exe"
	}
	return BackendName()
}

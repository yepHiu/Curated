// Package desktop provides desktop integration features including tray icon,
// single-instance locking, launch-at-login, and error dialogs.
package desktop

import (
	"errors"

	"curated-backend/internal/version"
)

// ErrLaunchAtLoginUnsupported is returned when the runtime cannot manage launch-at-login registration.
var ErrLaunchAtLoginUnsupported = errors.New("launch at login is not supported in this runtime")

// LaunchAtLoginValueName returns the registry value name for launch-at-login, varying by release channel.
func LaunchAtLoginValueName() string {
	if version.Channel == "release" {
		return "Curated"
	}
	return "Curated Dev"
}

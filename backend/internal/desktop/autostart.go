package desktop

import (
	"errors"

	"curated-backend/internal/version"
)

var ErrLaunchAtLoginUnsupported = errors.New("launch at login is not supported in this runtime")

func LaunchAtLoginValueName() string {
	if version.Channel == "release" {
		return "Curated"
	}
	return "Curated Dev"
}

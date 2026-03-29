//go:build !release

package config

// curatedDataRoot is empty for non-release builds: defaults stay cwd/repo-relative (development).
func curatedDataRoot() string {
	return ""
}

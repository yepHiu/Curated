//go:build !release

package config

import (
	"os"
	"path/filepath"
)

func defaultLogDir() string {
	cwd, err := os.Getwd()
	if err == nil && filepath.Base(cwd) == "backend" {
		return filepath.FromSlash("runtime/logs")
	}
	return filepath.FromSlash("backend/runtime/logs")
}

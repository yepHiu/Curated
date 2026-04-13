//go:build windows

package desktop

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"

	"curated-backend/internal/version"
)

const runRegistryPath = `Software\Microsoft\Windows\CurrentVersion\Run`

type runKeyStore interface {
	SetStringValue(name string, value string) error
	DeleteValue(name string) error
}

var osExecutable = os.Executable

func LaunchAtLoginSupported() bool {
	exe, err := osExecutable()
	if err != nil {
		return false
	}
	return isLaunchAtLoginExecutableSupported(exe)
}

func SyncLaunchAtLogin(enabled bool) error {
	command := ""
	if enabled {
		exe, err := osExecutable()
		if err != nil {
			return fmt.Errorf("resolve executable: %w", err)
		}
		if !isLaunchAtLoginExecutableSupported(exe) {
			return ErrLaunchAtLoginUnsupported
		}
		command, err = buildLaunchAtLoginCommand(exe)
		if err != nil {
			return err
		}
	}

	key, _, err := registry.CreateKey(registry.CURRENT_USER, runRegistryPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open launch-at-login registry key: %w", err)
	}
	defer key.Close()

	return syncLaunchAtLoginStore(key, enabled, LaunchAtLoginValueName(), command)
}

func buildLaunchAtLoginCommand(executable string) (string, error) {
	executable = strings.TrimSpace(executable)
	if executable == "" {
		return "", fmt.Errorf("empty executable path")
	}
	return `"` + executable + `" -mode tray -autostart`, nil
}

func syncLaunchAtLoginStore(store runKeyStore, enabled bool, valueName string, command string) error {
	if enabled {
		return store.SetStringValue(valueName, command)
	}
	if err := store.DeleteValue(valueName); err != nil && !errors.Is(err, registry.ErrNotExist) {
		return err
	}
	return nil
}

func isLaunchAtLoginExecutableSupported(executable string) bool {
	executable = strings.TrimSpace(executable)
	if executable == "" {
		return false
	}
	base := strings.ToLower(filepath.Base(executable))
	return base == strings.ToLower(version.SuggestedBinaryName())
}

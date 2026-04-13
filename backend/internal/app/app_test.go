package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"curated-backend/internal/config"
)

func TestSetLaunchAtLogin_PersistsAndSyncs(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "library-config.cfg")
	var synced []bool

	prevSupported := launchAtLoginSupportedFn
	prevSync := syncLaunchAtLoginFn
	t.Cleanup(func() {
		launchAtLoginSupportedFn = prevSupported
		syncLaunchAtLoginFn = prevSync
	})
	launchAtLoginSupportedFn = func() bool { return true }
	syncLaunchAtLoginFn = func(v bool) error {
		synced = append(synced, v)
		return nil
	}

	a := &App{
		cfg:                 config.Default(),
		librarySettingsPath: path,
	}

	if err := a.SetLaunchAtLogin(true); err != nil {
		t.Fatal(err)
	}

	if !a.LaunchAtLogin() {
		t.Fatal("expected LaunchAtLogin true in memory")
	}
	if len(synced) != 1 || !synced[0] {
		t.Fatalf("sync calls = %#v, want [true]", synced)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) == "" || !containsLaunchAtLoginValue(string(raw), true) {
		t.Fatalf("expected launchAtLogin true in file, got %s", string(raw))
	}
}

func TestSetLaunchAtLogin_RevertsConfigWhenDesktopSyncFails(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "library-config.cfg")
	if err := os.WriteFile(path, []byte(`{"launchAtLogin": false}`), 0o644); err != nil {
		t.Fatal(err)
	}

	prevSupported := launchAtLoginSupportedFn
	prevSync := syncLaunchAtLoginFn
	t.Cleanup(func() {
		launchAtLoginSupportedFn = prevSupported
		syncLaunchAtLoginFn = prevSync
	})
	launchAtLoginSupportedFn = func() bool { return true }
	syncLaunchAtLoginFn = func(bool) error {
		return os.ErrPermission
	}

	a := &App{
		cfg:                 config.Default(),
		librarySettingsPath: path,
	}

	if err := a.SetLaunchAtLogin(true); err == nil {
		t.Fatal("expected sync failure")
	}
	if a.LaunchAtLogin() {
		t.Fatal("expected LaunchAtLogin to remain false after failed sync")
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !containsLaunchAtLoginValue(string(raw), false) {
		t.Fatalf("expected launchAtLogin reverted to false, got %s", string(raw))
	}
}

func containsLaunchAtLoginValue(raw string, want bool) bool {
	if want {
		return strings.Contains(raw, `"launchAtLogin": true`) || strings.Contains(raw, `"launchAtLogin":true`)
	}
	return strings.Contains(raw, `"launchAtLogin": false`) || strings.Contains(raw, `"launchAtLogin":false`)
}

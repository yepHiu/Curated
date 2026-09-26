package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"curated-backend/internal/config"
)

// TestSetLANEnabled_PersistsWithoutPIN 确认没有 PIN 时也能写入局域网访问偏好，且不改当前监听。
func TestSetLANEnabled_PersistsWithoutPIN(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "library-config.cfg")

	a := &App{
		cfg:                 config.Default(),
		librarySettingsPath: path,
	}
	a.cfg.HttpAddr = "127.0.0.1:8081"
	if err := a.SetLANEnabled(true); err != nil {
		t.Fatal(err)
	}
	if !a.LANEnabled() {
		t.Fatal("expected lanEnabled true in memory")
	}
	if a.LANListening() {
		t.Fatal("saving the preference must not rebind the current listener")
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"lanEnabled": true`) &&
		!strings.Contains(string(raw), `"lanEnabled":true`) {
		t.Fatalf("expected lanEnabled true in file, got %s", string(raw))
	}
}

func TestDiscoveryPreferencePersistsWithoutChangingLiveBind(t *testing.T) {
	file := filepath.Join(t.TempDir(), "library-config.cfg")
	a := &App{cfg: config.Default(), librarySettingsPath: file, discoveryEnabled: true}
	if err := a.SetDiscoveryEnabled(false); err != nil {
		t.Fatal(err)
	}
	if a.DiscoveryEnabled() {
		t.Fatal("preference not saved")
	}
	raw, err := os.ReadFile(file)
	if err != nil || !strings.Contains(string(raw), `"discoveryEnabled": false`) {
		t.Fatalf("%s %v", raw, err)
	}
	if !a.cfg.DiscoveryOn() {
		t.Fatal("live discovery configuration changed before restart")
	}
}

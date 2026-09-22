package app

import (
	"curated-backend/internal/config"
	"path/filepath"
	"testing"
)

// TestBrowserPluginPreference verifies persistence and failed-write behavior.
func TestBrowserPluginPreference(t *testing.T) {
	path := filepath.Join(t.TempDir(), "library-config.cfg")
	a := &App{librarySettingsPath: path}
	if a.BrowserPluginEnabled() {
		t.Fatal("must default off")
	}
	for _, enabled := range []bool{true, false} {
		if err := a.SetBrowserPluginEnabled(enabled); err != nil {
			t.Fatal(err)
		}
		var restored config.Config
		if err := config.MergeLibrarySettingsFile(&restored, path); err != nil {
			t.Fatal(err)
		}
		if restored.BrowserPluginEnabled != enabled || a.BrowserPluginEnabled() != enabled {
			t.Fatal("preference not persisted")
		}
	}
	a.librarySettingsPath = ""
	if err := a.SetBrowserPluginEnabled(true); err == nil {
		t.Fatal("expected failed write")
	}
	if a.BrowserPluginEnabled() {
		t.Fatal("failed write changed active preference")
	}
}

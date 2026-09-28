package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
)

func TestSetBackendLogPatch_EmptyLogDirFallsBackToDefault(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "library-config.cfg")
	if err := os.WriteFile(path, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := &App{
		cfg: config.Config{
			LogDir:        `C:\custom\logs`,
			LogMaxAgeDays: 7,
			LogLevel:      "info",
		},
		librarySettingsPath: path,
	}

	empty := ""
	if err := a.SetBackendLogPatch(contracts.PatchBackendLogSettings{
		LogDir: &empty,
	}); err != nil {
		t.Fatal(err)
	}

	if got, want := a.cfg.LogDir, config.DefaultLogDir(); got != want {
		t.Fatalf("cfg.LogDir = %q, want %q", got, want)
	}
}

func TestSetBackendLogPatchIgnoresLegacyDirectoryAndPreservesOtherSettings(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "library-config.cfg")
	if err := os.WriteFile(path, []byte(`{"logDir":"legacy","organizeLibrary":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	a := &App{cfg: config.Config{LogDir: "legacy", LogLevel: "info"}, librarySettingsPath: path}
	custom, level, days := "another-custom-dir", "debug", 10
	if err := a.SetBackendLogPatch(contracts.PatchBackendLogSettings{LogDir: &custom, LogLevel: &level, LogMaxAgeDays: &days}); err != nil {
		t.Fatal(err)
	}
	if a.cfg.LogDir != config.DefaultLogDir() || a.cfg.LogLevel != level || a.cfg.LogMaxAgeDays != days {
		t.Fatalf("unexpected logging configuration: %+v", a.BackendLogSettings())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var saved map[string]any
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	if _, ok := saved["logDir"]; ok || saved["organizeLibrary"] != true || saved["logMaxAgeDays"] != float64(days) || saved["logLevel"] != level {
		t.Fatalf("unexpected persisted settings: %s", data)
	}
}

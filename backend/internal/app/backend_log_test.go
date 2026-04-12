package app

import (
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

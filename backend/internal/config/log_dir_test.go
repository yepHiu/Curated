package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadIgnoresLegacyCustomLogDirectory(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"logDir":"old-custom-logs"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LogDir != DefaultLogDir() {
		t.Fatalf("got %q, want default %q", cfg.LogDir, DefaultLogDir())
	}
}

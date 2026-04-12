//go:build release

package config

import (
	"path/filepath"
	"testing"
)

func TestDefaultLogDir_ReleaseBuildUsesLocalAppDataLogs(t *testing.T) {
	root := t.TempDir()
	t.Setenv("LOCALAPPDATA", root)
	t.Setenv("CURATED_DATA_DIR", "")

	if got, want := Default().LogDir, filepath.Join(root, "Curated", "logs"); got != want {
		t.Fatalf("Default().LogDir = %q, want %q", got, want)
	}
}

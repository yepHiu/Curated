//go:build release

package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDefaultBackupDirectoryRelease(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CURATED_DATA_DIR", filepath.Join(root, "custom-data"))
	want := filepath.Join(root, "custom-data", "backups")
	if got := DefaultBackupDirectory(); got != want {
		t.Fatalf("custom data destination = %q, want %q", got, want)
	}
	if _, err := os.Stat(want); !os.IsNotExist(err) {
		t.Fatalf("resolving destination created a directory: %v", err)
	}
	if runtime.GOOS == "windows" {
		t.Setenv("CURATED_DATA_DIR", "")
		t.Setenv("LOCALAPPDATA", root)
		if got, want := DefaultBackupDirectory(), filepath.Join(root, "Curated", "backups"); got != want {
			t.Fatalf("user destination = %q, want %q", got, want)
		}
	}
}

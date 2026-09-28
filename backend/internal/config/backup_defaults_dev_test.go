//go:build !release

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultBackupDirectoryDevelopment(t *testing.T) {
	root := t.TempDir()
	backend := filepath.Join(root, "backend")
	if err := os.Mkdir(backend, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, cwd := range []string{root, backend} {
		t.Run(filepath.Base(cwd), func(t *testing.T) {
			t.Chdir(cwd)
			want := filepath.Join(backend, "runtime", "backups")
			if got := DefaultBackupDirectory(); got != want {
				t.Fatalf("destination = %q, want %q", got, want)
			}
			if _, err := os.Stat(want); !os.IsNotExist(err) {
				t.Fatalf("resolving destination created a directory: %v", err)
			}
		})
	}
}

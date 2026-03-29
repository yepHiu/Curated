package movieroot

import (
	"path/filepath"
	"testing"

	"curated-backend/internal/contracts"
)

func TestResolveConfiguredLibraryPath(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	libA := filepath.Join(root, "libA")
	libB := filepath.Join(root, "libB")
	paths := []contracts.LibraryPathDTO{
		{ID: "1", Path: libA},
		{ID: "2", Path: libB},
	}
	file := filepath.Join(libB, "sub", "x.mp4")
	got := ResolveConfiguredLibraryPath(file, paths)
	if got == nil || got.Path != libB {
		t.Fatalf("got %+v", got)
	}
}

package app

import (
	"context"
	"path/filepath"
	"testing"

	"curated-backend/internal/config"
	"curated-backend/internal/storage"
)

func TestResolveScanPathsRespectsDeletedLibraryPaths(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	store, err := storage.NewSQLiteStore(filepath.Join(root, "library.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	legacyPath := filepath.Join(root, "legacy")
	a := &App{store: store, cfg: config.Config{LibraryPaths: []string{legacyPath}}}
	path, err := store.AddLibraryPath(ctx, filepath.Join(root, "media"), "Media")
	if err != nil {
		t.Fatal(err)
	}
	paths, err := a.resolveScanPaths(ctx, nil)
	if err != nil || len(paths) != 1 || paths[0] != path.Path {
		t.Fatalf("configured paths: %#v, err=%v", paths, err)
	}
	if _, err := store.DeleteLibraryPathAndPruneOrphanMovies(ctx, path.ID); err != nil {
		t.Fatal(err)
	}
	paths, err = a.resolveScanPaths(ctx, nil)
	if err != nil || len(paths) != 0 {
		t.Fatalf("empty library must not scan legacy config: %#v, err=%v", paths, err)
	}
	paths, err = a.resolveScanPaths(ctx, []string{legacyPath})
	if err != nil || len(paths) != 1 || paths[0] != legacyPath {
		t.Fatalf("explicit scan paths: %#v, err=%v", paths, err)
	}
}

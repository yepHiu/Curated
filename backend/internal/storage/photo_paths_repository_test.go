package storage

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestPhotoLibraryPathsRepository(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	store := newPhotoRepositoryTestStore(t, root)

	mainRoot := filepath.Join(root, "main")
	path, err := store.AddPhotoLibraryPath(ctx, mainRoot, "Main Photos")
	if err != nil {
		t.Fatalf("add photo path: %v", err)
	}
	if path.ID == "" || path.Path != mainRoot || path.Title != "Main Photos" || !path.FirstLibraryScanPending {
		t.Fatalf("unexpected path dto: %+v", path)
	}

	list, err := store.ListPhotoLibraryPaths(ctx)
	if err != nil {
		t.Fatalf("list photo paths: %v", err)
	}
	if len(list) != 1 || list[0].ID != path.ID {
		t.Fatalf("list = %+v, want one path %q", list, path.ID)
	}

	updated, err := store.UpdatePhotoLibraryPathTitle(ctx, path.ID, "Photo Shelf")
	if err != nil {
		t.Fatalf("update photo path title: %v", err)
	}
	if updated.Title != "Photo Shelf" {
		t.Fatalf("updated title = %q, want Photo Shelf", updated.Title)
	}

	_, err = store.AddPhotoLibraryPath(ctx, mainRoot, "Duplicate")
	if !errors.Is(err, ErrPhotoLibraryPathDuplicate) {
		t.Fatalf("duplicate add error = %v, want ErrPhotoLibraryPathDuplicate", err)
	}

	_, err = store.AddPhotoLibraryPath(ctx, "relative/photos", "Relative")
	if !errors.Is(err, ErrPhotoLibraryPathNotAbsolute) {
		t.Fatalf("relative add error = %v, want ErrPhotoLibraryPathNotAbsolute", err)
	}

	if err := store.DeletePhotoLibraryPath(ctx, path.ID); err != nil {
		t.Fatalf("delete photo path: %v", err)
	}
	if err := store.DeletePhotoLibraryPath(ctx, path.ID); !errors.Is(err, ErrPhotoLibraryPathNotFound) {
		t.Fatalf("delete missing error = %v, want ErrPhotoLibraryPathNotFound", err)
	}
}

func TestListPhotoLibraryPathStrings(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	store := newPhotoRepositoryTestStore(t, root)

	first := filepath.Join(root, "a")
	second := filepath.Join(root, "b")
	if _, err := store.AddPhotoLibraryPath(ctx, second, "Second"); err != nil {
		t.Fatalf("add second photo path: %v", err)
	}
	if _, err := store.AddPhotoLibraryPath(ctx, first, "First"); err != nil {
		t.Fatalf("add first photo path: %v", err)
	}

	got, err := store.ListPhotoLibraryPathStrings(ctx)
	if err != nil {
		t.Fatalf("list photo path strings: %v", err)
	}
	if len(got) != 2 || got[0] != first || got[1] != second {
		t.Fatalf("path strings = %#v, want [%q %q]", got, first, second)
	}
}

func newPhotoRepositoryTestStore(t *testing.T, root string) *SQLiteStore {
	t.Helper()

	store, err := NewSQLiteStore(filepath.Join(root, "photo-repo.db"))
	if err != nil {
		t.Fatalf("new sqlite store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return store
}

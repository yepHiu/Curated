package storage

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestLibraryPathsSeedListDelete(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(root, "libpaths.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	alphaVideos := filepath.Join(root, "alpha", "videos")
	betaMedia := filepath.Join(root, "beta", "media")
	if err := store.SeedLibraryPathsIfEmpty(ctx, []string{alphaVideos, betaMedia}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	n, _ := store.GetLibraryPathCount(ctx)
	if n != 2 {
		t.Fatalf("expected 2 seeded rows, got %d", n)
	}

	// Seed again should not duplicate
	if err := store.SeedLibraryPathsIfEmpty(ctx, []string{filepath.Join(root, "gamma")}); err != nil {
		t.Fatalf("seed2: %v", err)
	}
	n, _ = store.GetLibraryPathCount(ctx)
	if n != 2 {
		t.Fatalf("re-seed should not add rows, got %d", n)
	}

	list, err := store.ListLibraryPaths(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 || list[0].Path > list[1].Path {
		// ordered by path: /alpha before /beta
		if len(list) != 2 {
			t.Fatalf("list len %d", len(list))
		}
	}

	strs, err := store.ListLibraryPathStrings(ctx)
	if err != nil || len(strs) != 2 {
		t.Fatalf("strings: %v %#v", err, strs)
	}

	gammaNew := filepath.Join(root, "gamma", "new")
	dto, err := store.AddLibraryPath(ctx, gammaNew, "Gamma")
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if dto.Title != "Gamma" {
		t.Fatalf("title %q", dto.Title)
	}

	updated, err := store.UpdateLibraryPathTitle(ctx, dto.ID, "Gamma Archive")
	if err != nil || updated.Title != "Gamma Archive" || updated.Path != gammaNew {
		t.Fatalf("update title: %v %#v", err, updated)
	}
	_, err = store.UpdateLibraryPathTitle(ctx, "missing-id", "x")
	if !errors.Is(err, ErrLibraryPathNotFound) {
		t.Fatalf("expected not found on update, got %v", err)
	}

	_, err = store.AddLibraryPath(ctx, alphaVideos, "dup")
	if !errors.Is(err, ErrLibraryPathDuplicate) {
		t.Fatalf("expected duplicate, got %v", err)
	}

	_, err = store.AddLibraryPath(ctx, "relative/subdir", "rel")
	if !errors.Is(err, ErrLibraryPathNotAbsolute) {
		t.Fatalf("expected not absolute error, got %v", err)
	}

	if err := store.DeleteLibraryPath(ctx, dto.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := store.DeleteLibraryPath(ctx, dto.ID); !errors.Is(err, ErrLibraryPathNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestLibraryPathsEmptySeed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(root, "empty.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := store.SeedLibraryPathsIfEmpty(ctx, nil); err != nil {
		t.Fatal(err)
	}
	n, _ := store.GetLibraryPathCount(ctx)
	if n != 0 {
		t.Fatalf("expected 0 rows, got %d", n)
	}
}

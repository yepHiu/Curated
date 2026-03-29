package storage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"curated-backend/internal/contracts"
)

func TestResolvePrimaryVideoPath_OK(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(root, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	libDir := filepath.Join(root, "lib")
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		t.Fatal(err)
	}
	videoPath := filepath.Join(libDir, "ABC-123.mp4")
	if err := os.WriteFile(videoPath, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddLibraryPath(ctx, libDir, "Lib"); err != nil {
		t.Fatal(err)
	}

	outcome, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
		TaskID:   "task-1",
		Path:     videoPath,
		FileName: "ABC-123.mp4",
		Number:   "ABC-123",
	})
	if err != nil {
		t.Fatal(err)
	}

	absWant, err := filepath.Abs(videoPath)
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.ResolvePrimaryVideoPath(ctx, outcome.MovieID)
	if err != nil {
		t.Fatalf("ResolvePrimaryVideoPath: %v", err)
	}
	if got != absWant {
		t.Fatalf("got %q want %q", got, absWant)
	}
}

func TestResolvePrimaryVideoPath_OutsideLibraryRoot(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(root, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	libDir := filepath.Join(root, "lib")
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		t.Fatal(err)
	}
	otherDir := filepath.Join(root, "other")
	if err := os.MkdirAll(otherDir, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(otherDir, "X-999.mp4")
	if err := os.WriteFile(outside, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddLibraryPath(ctx, libDir, "Lib"); err != nil {
		t.Fatal(err)
	}

	outcome, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
		TaskID:   "task-1",
		Path:     outside,
		FileName: "X-999.mp4",
		Number:   "X-999",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.ResolvePrimaryVideoPath(ctx, outcome.MovieID)
	if !errors.Is(err, ErrMovieVideoForbidden) {
		t.Fatalf("want ErrMovieVideoForbidden, got %v", err)
	}
}

func TestResolvePrimaryVideoPath_UnknownMovie(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(root, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	_, err = store.ResolvePrimaryVideoPath(ctx, "missing-id")
	if !errors.Is(err, ErrMovieVideoNotFound) {
		t.Fatalf("want ErrMovieVideoNotFound, got %v", err)
	}
}

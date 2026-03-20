package storage

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"jav-shadcn/backend/internal/contracts"
	"jav-shadcn/backend/internal/scraper"
)

func TestDeleteMovie_NotFound(t *testing.T) {
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
	err = store.DeleteMovie(ctx, "missing-id")
	if !errors.Is(err, ErrMovieNotFound) {
		t.Fatalf("expected ErrMovieNotFound, got %v", err)
	}
}

func TestDeleteMovie_RemovesRowsAndFiles(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	videoPath := filepath.Join(root, "DEL-001.mp4")
	if err := os.WriteFile(videoPath, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	nfoPath := filepath.Join(root, "movie.nfo")
	if err := os.WriteFile(nfoPath, []byte("<movie/>"), 0o644); err != nil {
		t.Fatal(err)
	}
	coverPath := filepath.Join(root, "cover.jpg")
	if err := os.WriteFile(coverPath, []byte("jpg"), 0o644); err != nil {
		t.Fatal(err)
	}

	store, err := NewSQLiteStore(filepath.Join(root, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	outcome, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
		TaskID:   "task-1",
		Path:     videoPath,
		FileName: "DEL-001.mp4",
		Number:   "DEL-001",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveMovieMetadata(ctx, scraper.Metadata{
		MovieID:  outcome.MovieID,
		Number:   "DEL-001",
		Title:    "T",
		Summary:  "S",
		Studio:   "St",
		CoverURL: "http://x",
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateMediaAssetLocalPath(ctx, outcome.MovieID, "cover", "http://x", coverPath); err != nil {
		t.Fatal(err)
	}

	if err := store.DeleteMovie(ctx, outcome.MovieID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(videoPath); !os.IsNotExist(err) {
		t.Fatalf("video should be gone: %v", err)
	}
	if _, err := os.Stat(nfoPath); !os.IsNotExist(err) {
		t.Fatalf("nfo should be gone: %v", err)
	}
	if _, err := os.Stat(coverPath); !os.IsNotExist(err) {
		t.Fatalf("cover should be gone: %v", err)
	}

	_, err = store.GetMovieDetail(ctx, outcome.MovieID)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows after delete, got %v", err)
	}
}

package storage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"jav-shadcn/backend/internal/contracts"
	"jav-shadcn/backend/internal/scraper"
)

func TestRecordPlayedMovieIfMovieExists_NotFound(t *testing.T) {
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
	err = store.RecordPlayedMovieIfMovieExists(ctx, "no-such-movie")
	if !errors.Is(err, ErrPlayedMovieMovieNotFound) {
		t.Fatalf("expected ErrPlayedMovieMovieNotFound, got %v", err)
	}
}

func TestRecordPlayedMovieIfMovieExists_ListIdempotent(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	videoPath := filepath.Join(root, "PLAY-001.mp4")
	if err := os.WriteFile(videoPath, []byte("x"), 0o644); err != nil {
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
		FileName: "PLAY-001.mp4",
		Number:   "PLAY-001",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveMovieMetadata(ctx, scraper.Metadata{
		MovieID:  outcome.MovieID,
		Number:   "PLAY-001",
		Title:    "T",
		Summary:  "S",
		Studio:   "St",
		CoverURL: "http://x",
	}); err != nil {
		t.Fatal(err)
	}
	mid := outcome.MovieID
	if err := store.RecordPlayedMovieIfMovieExists(ctx, mid); err != nil {
		t.Fatal(err)
	}
	if err := store.RecordPlayedMovieIfMovieExists(ctx, mid); err != nil {
		t.Fatal(err)
	}
	ids, err := store.ListPlayedMovieIDs(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != mid {
		t.Fatalf("expected one id %q, got %v", mid, ids)
	}
	n, err := store.CountPlayedMovies(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("count want 1 got %d", n)
	}
}

package storage

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"jav-shadcn/backend/internal/contracts"
	"jav-shadcn/backend/internal/scraper"
)

func TestSaveMovieMetadata(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(root, "test.db"))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() {
		_ = store.Close()
	}()

	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("failed to migrate store: %v", err)
	}

	outcome, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
		TaskID:   "task-1",
		Path:     "D:/Media/JAV/Main/ABC-123.mp4",
		FileName: "ABC-123.mp4",
		Number:   "ABC-123",
	})
	if err != nil {
		t.Fatalf("failed to persist scan movie: %v", err)
	}

	err = store.SaveMovieMetadata(ctx, scraper.Metadata{
		MovieID:         outcome.MovieID,
		Number:          "ABC-123",
		Title:           "Example Title",
		Summary:         "Example Summary",
		Provider:        "javbus",
		Homepage:        "https://example.com/movie",
		Director:        "Jane Doe",
		Studio:          "Sample Studio",
		Actors:          []string{"Actor A", "Actor B"},
		Tags:            []string{"Drama", "Sample"},
		RuntimeMinutes:  120,
		Rating:          4.6,
		ReleaseDate:     "2025-03-01",
		CoverURL:        "https://example.com/poster.jpg",
		ThumbURL:        "https://example.com/thumb.jpg",
		PreviewVideoURL: "https://example.com/preview.mp4",
		PreviewImages:   []string{"https://example.com/1.jpg", "https://example.com/2.jpg"},
	})
	if err != nil {
		t.Fatalf("failed to save movie metadata: %v", err)
	}

	var (
		title, studio, provider, coverURL string
		actorCount, tagCount, assetCount  int
	)
	if err := store.db.QueryRowContext(ctx, `SELECT title, studio, provider, cover_url FROM movies WHERE id = ?`, outcome.MovieID).
		Scan(&title, &studio, &provider, &coverURL); err != nil {
		t.Fatalf("failed to query saved movie: %v", err)
	}
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM movie_actors WHERE movie_id = ?`, outcome.MovieID).Scan(&actorCount); err != nil {
		t.Fatalf("failed to count movie actors: %v", err)
	}
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM movie_tags WHERE movie_id = ?`, outcome.MovieID).Scan(&tagCount); err != nil {
		t.Fatalf("failed to count movie tags: %v", err)
	}
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM media_assets WHERE movie_id = ?`, outcome.MovieID).Scan(&assetCount); err != nil {
		t.Fatalf("failed to count media assets: %v", err)
	}

	if title != "Example Title" || studio != "Sample Studio" || provider != "javbus" || coverURL == "" {
		t.Fatalf("unexpected movie metadata values: title=%q studio=%q provider=%q cover=%q", title, studio, provider, coverURL)
	}
	if actorCount != 2 || tagCount != 2 {
		t.Fatalf("unexpected relation counts: actors=%d tags=%d", actorCount, tagCount)
	}
	if assetCount != 4 {
		t.Fatalf("expected 4 media assets, got %d", assetCount)
	}
}

func TestSaveMovieMetadata_UnknownMovieID(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(root, "test.db"))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("failed to migrate store: %v", err)
	}

	err = store.SaveMovieMetadata(ctx, scraper.Metadata{
		MovieID: "no-such-movie",
		Number:  "X-1",
		Title:   "T",
	})
	if err == nil {
		t.Fatal("expected error for unknown movie id")
	}
	if !errors.Is(err, ErrMovieNotFoundForMetadata) {
		t.Fatalf("expected ErrMovieNotFoundForMetadata, got %v", err)
	}
}

package storage

import (
	"context"
	"path/filepath"
	"testing"

	"jav-shadcn/backend/internal/contracts"
	"jav-shadcn/backend/internal/scraper"
)

func TestListMoviesAndGetMovieDetail(t *testing.T) {
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

	if err := store.SaveMovieMetadata(ctx, scraper.Metadata{
		MovieID:        outcome.MovieID,
		Number:         "ABC-123",
		Title:          "Example Title",
		Summary:        "Example Summary",
		Studio:         "Sample Studio",
		Actors:         []string{"Actor B", "Actor A"},
		Tags:           []string{"Tag B", "Tag A"},
		RuntimeMinutes: 120,
		Rating:         4.5,
	}); err != nil {
		t.Fatalf("failed to save movie metadata: %v", err)
	}

	page, err := store.ListMovies(ctx, contracts.ListMoviesRequest{Limit: 10})
	if err != nil {
		t.Fatalf("failed to list movies: %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 {
		t.Fatalf("unexpected page result: total=%d items=%d", page.Total, len(page.Items))
	}
	if page.Items[0].Title != "Example Title" || len(page.Items[0].Actors) != 2 || len(page.Items[0].Tags) != 2 {
		t.Fatalf("unexpected list item contents: %+v", page.Items[0])
	}

	movie, err := store.GetMovieDetail(ctx, outcome.MovieID)
	if err != nil {
		t.Fatalf("failed to get movie detail: %v", err)
	}
	if movie.Summary != "Example Summary" || movie.Code != "ABC-123" {
		t.Fatalf("unexpected movie detail: %+v", movie)
	}
	if movie.MetadataRating != 4.5 || movie.Rating != 4.5 || movie.UserRating != nil {
		t.Fatalf("expected metadata rating 4.5 and no user override, got %+v", movie)
	}

	if err := store.PatchMovieUserPrefs(ctx, outcome.MovieID, contracts.PatchMovieInput{
		UserRatingSet: true,
		UserRating:    2.0,
	}); err != nil {
		t.Fatalf("patch user rating: %v", err)
	}
	movie, err = store.GetMovieDetail(ctx, outcome.MovieID)
	if err != nil {
		t.Fatalf("get after patch: %v", err)
	}
	if movie.Rating != 2.0 || movie.UserRating == nil || *movie.UserRating != 2.0 || movie.MetadataRating != 4.5 {
		t.Fatalf("expected user override 2.0 and metadata 4.5, got %+v", movie)
	}
	page, err = store.ListMovies(ctx, contracts.ListMoviesRequest{Limit: 10})
	if err != nil {
		t.Fatalf("list after patch: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].Rating != 2.0 {
		t.Fatalf("list effective rating want 2.0, got %+v", page.Items[0])
	}
}

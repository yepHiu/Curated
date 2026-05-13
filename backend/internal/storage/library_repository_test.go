package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"curated-backend/internal/contracts"
	"curated-backend/internal/scraper"
)

func TestListMovies_EmptyRelationsEncodeAsArrays(t *testing.T) {
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

	if _, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
		TaskID:   "task-empty-relations",
		Path:     "D:/Media/JAV/Main/NOACT-001.mp4",
		FileName: "NOACT-001.mp4",
		Number:   "NOACT-001",
	}); err != nil {
		t.Fatalf("failed to persist scan movie: %v", err)
	}

	page, err := store.ListMovies(ctx, contracts.ListMoviesRequest{Limit: 10})
	if err != nil {
		t.Fatalf("failed to list movies: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(page.Items))
	}
	if page.Items[0].Actors == nil {
		t.Fatal("Actors is nil, want empty slice")
	}
	if page.Items[0].Tags == nil {
		t.Fatal("Tags is nil, want empty slice")
	}

	b, err := json.Marshal(page)
	if err != nil {
		t.Fatalf("marshal page: %v", err)
	}
	body := string(b)
	if strings.Contains(body, `"actors":null`) || strings.Contains(body, `"tags":null`) {
		t.Fatalf("empty relation arrays encoded as null: %s", body)
	}
	if !strings.Contains(body, `"actors":[]`) || !strings.Contains(body, `"tags":[]`) {
		t.Fatalf("empty relation arrays not encoded as arrays: %s", body)
	}
}

func TestListMovies_LargePageDoesNotHitSQLiteVariableLimit(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(root, "large-page.db"))
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

	const movieCount = 33_000
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin seed tx: %v", err)
	}
	for i := 0; i < movieCount; i++ {
		code := fmt.Sprintf("BULK-%05d", i)
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO movies (
				id, title, code, studio, summary, runtime_minutes, rating, is_favorite,
				added_at, location, resolution, year, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			strings.ToLower(code),
			code,
			code,
			"Studio",
			"Summary",
			0,
			0,
			0,
			"2026-01-01",
			filepath.Join(root, code+".mp4"),
			"mp4",
			0,
			"2026-01-01T00:00:00Z",
			"2026-01-01T00:00:00Z",
		); err != nil {
			_ = tx.Rollback()
			t.Fatalf("seed movie %d: %v", i, err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit seed tx: %v", err)
	}

	page, err := store.ListMovies(ctx, contracts.ListMoviesRequest{Limit: movieCount})
	if err != nil {
		t.Fatalf("large list movies: %v", err)
	}
	if page.Total != movieCount || len(page.Items) != movieCount {
		t.Fatalf("page total=%d items=%d, want %d", page.Total, len(page.Items), movieCount)
	}
}

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

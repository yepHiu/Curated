package storage

import (
	"context"
	"path/filepath"
	"testing"

	"curated-backend/internal/contracts"
)

func TestPersistScanMovie(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(root, "test.db"))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() {
		_ = store.Close()
	}()

	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("failed to migrate store: %v", err)
	}

	first, err := store.PersistScanMovie(context.Background(), contracts.ScanFileResultDTO{
		TaskID:   "task-1",
		Path:     "D:/Media/JAV/Main/ABC-123.mp4",
		FileName: "ABC-123.mp4",
		Number:   "ABC-123",
	})
	if err != nil {
		t.Fatalf("failed to persist first movie: %v", err)
	}
	if first.Status != "imported" {
		t.Fatalf("expected imported status, got %q", first.Status)
	}

	second, err := store.PersistScanMovie(context.Background(), contracts.ScanFileResultDTO{
		TaskID:   "task-2",
		Path:     "E:/Vault/JAV/New/ABC-123.mkv",
		FileName: "ABC-123.mkv",
		Number:   "ABC-123",
	})
	if err != nil {
		t.Fatalf("failed to persist duplicate movie: %v", err)
	}
	if second.Status != "updated" {
		t.Fatalf("expected updated status, got %q", second.Status)
	}
	if second.MovieID != first.MovieID {
		t.Fatalf("expected same movie id, got %q and %q", first.MovieID, second.MovieID)
	}
}

func TestPersistScanMovieSkipsWhenTargetPathBelongsToAnotherMovie(t *testing.T) {
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

	original, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
		TaskID:   "task-1",
		Path:     "D:/Media/ABC-123/ABC-123.mp4",
		FileName: "ABC-123.mp4",
		Number:   "ABC-123",
	})
	if err != nil {
		t.Fatalf("failed to persist original movie: %v", err)
	}

	occupiedPath := "D:/Media/DEF-999/DEF-999.mp4"
	occupant, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
		TaskID:   "task-2",
		Path:     occupiedPath,
		FileName: "DEF-999.mp4",
		Number:   "DEF-999",
	})
	if err != nil {
		t.Fatalf("failed to persist path occupant: %v", err)
	}

	outcome, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
		TaskID:   "task-3",
		Path:     occupiedPath,
		FileName: "ABC-123.mp4",
		Number:   "ABC-123",
	})
	if err != nil {
		t.Fatalf("persisting colliding scan path should not fail: %v", err)
	}
	if outcome.Status != "skipped" || outcome.Reason != "path_already_indexed" {
		t.Fatalf("outcome = %+v, want skipped path_already_indexed", outcome)
	}
	if outcome.MovieID != occupant.MovieID {
		t.Fatalf("expected colliding path owner movie id %q, got %q", occupant.MovieID, outcome.MovieID)
	}

	var originalLocation string
	if err := store.db.QueryRowContext(ctx, `SELECT location FROM movies WHERE id = ?`, original.MovieID).Scan(&originalLocation); err != nil {
		t.Fatalf("query original location: %v", err)
	}
	if originalLocation != "D:/Media/ABC-123/ABC-123.mp4" {
		t.Fatalf("original movie location changed to %q", originalLocation)
	}
}

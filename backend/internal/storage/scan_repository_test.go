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

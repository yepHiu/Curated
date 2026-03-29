package storage

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"curated-backend/internal/contracts"
)

func TestMovieComment_UpsertAndGet(t *testing.T) {
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

	outcome, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
		TaskID:   "task-1",
		Path:     "D:/Media/JAV/COM-001.mp4",
		FileName: "COM-001.mp4",
		Number:   "COM-001",
	})
	if err != nil {
		t.Fatal(err)
	}
	mid := outcome.MovieID

	dto, err := store.GetMovieComment(ctx, mid)
	if err != nil {
		t.Fatal(err)
	}
	if dto.Body != "" || dto.UpdatedAt != "" {
		t.Fatalf("expected empty, got %+v", dto)
	}

	saved, err := store.UpsertMovieComment(ctx, mid, "  first note  ")
	if err != nil {
		t.Fatal(err)
	}
	if saved.Body != "first note" || saved.UpdatedAt == "" {
		t.Fatalf("unexpected saved: %+v", saved)
	}

	dto, err = store.GetMovieComment(ctx, mid)
	if err != nil {
		t.Fatal(err)
	}
	if dto.Body != "first note" {
		t.Fatalf("get body = %q", dto.Body)
	}

	saved2, err := store.UpsertMovieComment(ctx, mid, "edited")
	if err != nil {
		t.Fatal(err)
	}
	if saved2.Body != "edited" {
		t.Fatalf("second save = %q", saved2.Body)
	}
	dto, err = store.GetMovieComment(ctx, mid)
	if err != nil || dto.Body != "edited" {
		t.Fatalf("after edit: %+v err=%v", dto, err)
	}
}

func TestMovieComment_UpsertMovieNotFound(t *testing.T) {
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
	_, err = store.UpsertMovieComment(ctx, "no-such-id", "x")
	if err != ErrMovieNotFound {
		t.Fatalf("want ErrMovieNotFound, got %v", err)
	}
}

func TestMovieComment_TooLong(t *testing.T) {
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
	outcome, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
		TaskID: "t", Path: "D:/x/LONG-1.mp4", FileName: "LONG-1.mp4", Number: "LONG-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	long := strings.Repeat("あ", contracts.MaxMovieCommentRunes+1)
	_, err = store.UpsertMovieComment(ctx, outcome.MovieID, long)
	if err != ErrMovieCommentTooLong {
		t.Fatalf("want ErrMovieCommentTooLong, got %v", err)
	}
}

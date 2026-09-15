package storage

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"curated-backend/internal/contracts"
)

// TestPhotoCommentUpsertAndGet 覆盖写真备注的读写、过长拒绝和缺书。
func TestPhotoCommentUpsertAndGet(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "photo-comment.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	photoRoot := filepath.Join(t.TempDir(), "photos")
	path, err := store.AddPhotoLibraryPath(ctx, photoRoot, "Photos")
	if err != nil {
		t.Fatal(err)
	}
	book, err := store.UpsertPhotoBook(ctx, PhotoBookUpsert{
		LibraryPathID:  path.ID,
		Location:       filepath.Join(photoRoot, "Note.cbz"),
		SourceFileName: "Note.cbz",
		Title:          "Note",
		FileSize:       1,
		FileModifiedAt: "2026-09-12T00:00:00Z",
		PageCount:      1,
	})
	if err != nil {
		t.Fatal(err)
	}

	empty, err := store.GetPhotoComment(ctx, book.ID)
	if err != nil {
		t.Fatal(err)
	}
	if empty.Body != "" || empty.UpdatedAt != "" {
		t.Fatalf("expected empty comment, got %+v", empty)
	}

	saved, err := store.UpsertPhotoComment(ctx, book.ID, "  photo note  ")
	if err != nil {
		t.Fatal(err)
	}
	if saved.Body != "photo note" || saved.UpdatedAt == "" {
		t.Fatalf("unexpected saved comment: %+v", saved)
	}
	got, err := store.GetPhotoComment(ctx, book.ID)
	if err != nil || got.Body != "photo note" {
		t.Fatalf("get after upsert = %+v err=%v", got, err)
	}

	if _, err := store.UpsertPhotoComment(ctx, "missing", "x"); err != ErrPhotoBookNotFound {
		t.Fatalf("missing book err = %v", err)
	}
	long := strings.Repeat("あ", contracts.MaxBookCommentRunes+1)
	if _, err := store.UpsertPhotoComment(ctx, book.ID, long); err != ErrBookCommentTooLong {
		t.Fatalf("too long err = %v", err)
	}
}

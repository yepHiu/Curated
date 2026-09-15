package storage

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"curated-backend/internal/contracts"
)

// TestComicCommentUpsertGetAndDelete 覆盖漫画备注的读写、过长拒绝、缺书以及删除后级联清理。
func TestComicCommentUpsertGetAndDelete(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	store := newComicRepositoryTestStore(t, root)
	path, err := store.AddComicLibraryPath(ctx, filepath.Join(root, "comics"), "Comics")
	if err != nil {
		t.Fatal(err)
	}
	book, err := store.UpsertComicBook(ctx, ComicBookUpsert{
		LibraryPathID:  path.ID,
		Location:       filepath.Join(root, "comics", "Note.cbz"),
		SourceFileName: "Note.cbz",
		Title:          "Note",
		FileSize:       1,
		FileModifiedAt: "2026-09-12T00:00:00Z",
		PageCount:      1,
	})
	if err != nil {
		t.Fatal(err)
	}

	empty, err := store.GetComicComment(ctx, book.ID)
	if err != nil {
		t.Fatal(err)
	}
	if empty.Body != "" || empty.UpdatedAt != "" {
		t.Fatalf("expected empty comment, got %+v", empty)
	}

	saved, err := store.UpsertComicComment(ctx, book.ID, "  first note  ")
	if err != nil {
		t.Fatal(err)
	}
	if saved.Body != "first note" || saved.UpdatedAt == "" {
		t.Fatalf("unexpected saved comment: %+v", saved)
	}
	got, err := store.GetComicComment(ctx, book.ID)
	if err != nil || got.Body != "first note" {
		t.Fatalf("get after upsert = %+v err=%v", got, err)
	}

	if _, err := store.UpsertComicComment(ctx, "missing", "x"); err != ErrComicBookNotFound {
		t.Fatalf("missing book err = %v", err)
	}
	long := strings.Repeat("あ", contracts.MaxBookCommentRunes+1)
	if _, err := store.UpsertComicComment(ctx, book.ID, long); err != ErrBookCommentTooLong {
		t.Fatalf("too long err = %v", err)
	}

	if err := store.DeleteComicBookIndex(ctx, book.ID); err != nil {
		t.Fatal(err)
	}
	afterDelete, err := store.GetComicComment(ctx, book.ID)
	if err != nil {
		t.Fatal(err)
	}
	if afterDelete.Body != "" || afterDelete.UpdatedAt != "" {
		t.Fatalf("comment should cascade away, got %+v", afterDelete)
	}
}

package storage

import (
	"context"
	"path/filepath"
	"testing"

	"curated-backend/internal/contracts"
)

func TestPhotoBookRepositories(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "photo-books.db"))
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
		Location:       filepath.Join(photoRoot, "Portrait Set.cbz"),
		SourceFileName: "Portrait Set.cbz",
		Title:          "Portrait Set",
		FileSize:       123,
		FileModifiedAt: "2026-07-05T00:00:00Z",
		PageCount:      2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if book.Title != "Portrait Set" || book.PageCount != 2 {
		t.Fatalf("book = %#v", book)
	}
	if err := store.ReplacePhotoPages(ctx, book.ID, []PhotoPageInput{
		{Index: 0, EntryPath: "001.jpg", FileName: "001.jpg", ImageExt: ".jpg", SizeBytes: 10},
		{Index: 1, EntryPath: "002.png", FileName: "002.png", ImageExt: ".png", SizeBytes: 20},
	}); err != nil {
		t.Fatal(err)
	}
	detail, err := store.GetPhotoBookDetail(ctx, book.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.PageCount != 2 || len(detail.Pages) != 2 {
		t.Fatalf("detail = %#v, want 2 pages", detail)
	}
	page, err := store.ListPhotoBooks(ctx, contracts.ListPhotoBooksRequest{Query: "portrait", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || page.Items[0].ID != book.ID {
		t.Fatalf("list = %#v, want created photo book", page)
	}
}

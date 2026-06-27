package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"curated-backend/internal/contracts"
)

func TestComicBookRepositories(t *testing.T) {
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
		Location:       filepath.Join(root, "comics", "Sample 01.cbz"),
		SourceFileName: "Sample 01.cbz",
		Title:          "Sample 01",
		FileSize:       1234,
		FileModifiedAt: "2026-06-28T01:00:00Z",
		PageCount:      3,
	})
	if err != nil {
		t.Fatalf("upsert comic book: %v", err)
	}
	if book.ID == "" || book.Title != "Sample 01" || book.SourceFileName != "Sample 01.cbz" {
		t.Fatalf("unexpected book: %+v", book)
	}

	if err := store.ReplaceComicPages(ctx, book.ID, []ComicPageInput{
		{Index: 0, EntryPath: "chapter/001.jpg", FileName: "001.jpg", ImageExt: ".jpg", SizeBytes: 101},
		{Index: 1, EntryPath: "chapter/002.jpg", FileName: "002.jpg", ImageExt: ".jpg", SizeBytes: 102},
		{Index: 2, EntryPath: "chapter/010.png", FileName: "010.png", ImageExt: ".png", SizeBytes: 103},
	}); err != nil {
		t.Fatalf("replace pages: %v", err)
	}

	rating := 4.5
	tags := []string{"作者:alice", "系列:sample", "color"}
	detail, err := store.PatchComicBook(ctx, book.ID, contracts.PatchComicBookRequest{
		Title:     stringPtr("Sample Patched"),
		Tags:      &tags,
		Favorite:  boolPtr(true),
		RatingSet: true,
		Rating:    &rating,
	})
	if err != nil {
		t.Fatalf("patch comic book: %v", err)
	}
	if detail.Title != "Sample Patched" || !detail.IsFavorite || detail.Rating == nil || *detail.Rating != rating {
		t.Fatalf("patched detail = %+v", detail)
	}
	if len(detail.Tags) != 3 || len(detail.Pages) != 3 {
		t.Fatalf("patched detail tags/pages = %+v / %+v", detail.Tags, detail.Pages)
	}

	page, err := store.ListComicBooks(ctx, contracts.ListComicBooksRequest{
		Query:      "sample",
		Tag:        "作者:alice",
		Favorite:   boolPtr(true),
		ReadStatus: "unread",
		Limit:      20,
	})
	if err != nil {
		t.Fatalf("list comic books: %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].ID != book.ID {
		t.Fatalf("list page = %+v, want one patched book", page)
	}

	if err := store.SaveComicProgress(ctx, book.ID, 2, false); err != nil {
		t.Fatalf("save progress: %v", err)
	}
	progress, err := store.GetComicProgress(ctx, book.ID)
	if err != nil {
		t.Fatalf("get progress: %v", err)
	}
	if progress.PageIndex != 2 || progress.Completed {
		t.Fatalf("progress = %+v, want page 2 incomplete", progress)
	}
	bookAfterProgress, err := store.GetComicBookDetail(ctx, book.ID)
	if err != nil {
		t.Fatalf("get book after progress: %v", err)
	}
	if bookAfterProgress.ReadStatus != "reading" || bookAfterProgress.CurrentPageIndex != 2 {
		t.Fatalf("book after progress = %+v, want reading at page 2", bookAfterProgress)
	}

	if err := store.SaveComicReadingPreferences(ctx, book.ID, contracts.ComicReadingPreferencesDTO{
		Mode:      "scroll",
		Fit:       "width",
		Direction: "rtl",
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("save preferences: %v", err)
	}
	prefs, err := store.GetComicReadingPreferences(ctx, book.ID)
	if err != nil {
		t.Fatalf("get preferences: %v", err)
	}
	if prefs.Mode != "scroll" || prefs.Fit != "width" || prefs.Direction != "rtl" {
		t.Fatalf("preferences = %+v", prefs)
	}

	if err := store.SaveComicProgress(ctx, book.ID, 2, true); err != nil {
		t.Fatalf("save completed progress: %v", err)
	}
	readPage, err := store.ListComicBooks(ctx, contracts.ListComicBooksRequest{ReadStatus: "read", Limit: 20})
	if err != nil {
		t.Fatalf("list read comic books: %v", err)
	}
	if readPage.Total != 1 {
		t.Fatalf("read page total = %d, want 1", readPage.Total)
	}
}

func newComicRepositoryTestStore(t *testing.T, root string) *SQLiteStore {
	t.Helper()

	store, err := NewSQLiteStore(filepath.Join(root, "comic-repo.db"))
	if err != nil {
		t.Fatalf("new sqlite store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return store
}

func stringPtr(v string) *string {
	return &v
}

func boolPtr(v bool) *bool {
	return &v
}

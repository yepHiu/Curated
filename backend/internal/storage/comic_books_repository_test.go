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

// TestListComicBooksFillsTagsInBatch 确认列表一次带回多本标签。
func TestListComicBooksFillsTagsInBatch(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	store := newComicRepositoryTestStore(t, root)
	path, err := store.AddComicLibraryPath(ctx, filepath.Join(root, "comics"), "Comics")
	if err != nil {
		t.Fatal(err)
	}

	first, err := store.UpsertComicBook(ctx, ComicBookUpsert{
		LibraryPathID:  path.ID,
		Location:       filepath.Join(root, "comics", "A.cbz"),
		SourceFileName: "A.cbz",
		Title:          "Alpha",
		FileSize:       1,
		FileModifiedAt: "2026-09-11T00:00:00Z",
		PageCount:      1,
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.UpsertComicBook(ctx, ComicBookUpsert{
		LibraryPathID:  path.ID,
		Location:       filepath.Join(root, "comics", "B.cbz"),
		SourceFileName: "B.cbz",
		Title:          "Beta",
		FileSize:       1,
		FileModifiedAt: "2026-09-11T00:00:00Z",
		PageCount:      1,
	})
	if err != nil {
		t.Fatal(err)
	}
	firstTags := []string{"color", "作者:alice"}
	if _, err := store.PatchComicBook(ctx, first.ID, contracts.PatchComicBookRequest{Tags: &firstTags}); err != nil {
		t.Fatal(err)
	}
	secondTags := []string{"mono"}
	if _, err := store.PatchComicBook(ctx, second.ID, contracts.PatchComicBookRequest{Tags: &secondTags}); err != nil {
		t.Fatal(err)
	}

	page, err := store.ListComicBooks(ctx, contracts.ListComicBooksRequest{Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 || len(page.Items) != 2 {
		t.Fatalf("list = %+v, want two books", page)
	}
	byID := map[string][]string{}
	for _, item := range page.Items {
		byID[item.ID] = item.Tags
	}
	if len(byID[first.ID]) != 2 || byID[first.ID][0] != "color" || byID[first.ID][1] != "作者:alice" {
		t.Fatalf("first tags = %#v", byID[first.ID])
	}
	if len(byID[second.ID]) != 1 || byID[second.ID][0] != "mono" {
		t.Fatalf("second tags = %#v", byID[second.ID])
	}
}

// TestComicDisplayTitleSurvivesRescan 确认 user_title 覆盖在重扫更新来源标题后仍保留。
func TestComicDisplayTitleSurvivesRescan(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	store := newComicRepositoryTestStore(t, root)
	location := filepath.Join(root, "comics", "Sample 01.cbz")
	book, err := store.UpsertComicBook(ctx, ComicBookUpsert{
		Location:       location,
		SourceFileName: "Sample 01.cbz",
		Title:          "Sample 01",
		PageCount:      1,
	})
	if err != nil {
		t.Fatal(err)
	}
	patched, err := store.PatchComicBook(ctx, book.ID, contracts.PatchComicBookRequest{Title: stringPtr("展示标题")})
	if err != nil {
		t.Fatal(err)
	}
	if patched.Title != "展示标题" {
		t.Fatalf("patched title = %q", patched.Title)
	}
	rescanned, err := store.UpsertComicBook(ctx, ComicBookUpsert{
		Location:       location,
		SourceFileName: "Sample 01.cbz",
		Title:          "Renamed From File",
		PageCount:      1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rescanned.ID != book.ID || rescanned.Title != "展示标题" {
		t.Fatalf("rescanned = %+v, want overlay kept", rescanned)
	}
	page, err := store.ListComicBooks(ctx, contracts.ListComicBooksRequest{Query: "展示", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || page.Items[0].Title != "展示标题" {
		t.Fatalf("search overlay = %+v", page)
	}
	if _, err := store.PatchComicBook(ctx, book.ID, contracts.PatchComicBookRequest{Title: stringPtr("")}); err == nil {
		t.Fatal("expected empty title to fail")
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

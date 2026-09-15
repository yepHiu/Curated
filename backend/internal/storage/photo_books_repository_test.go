package storage

import (
	"context"
	"path/filepath"
	"testing"

	"curated-backend/internal/contracts"
)

// TestPhotoBookRepositories 覆盖写真入库、列表、评分写入与清除。
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

	rating := 4.5
	rated, err := store.PatchPhotoBook(ctx, book.ID, contracts.PatchPhotoBookRequest{
		RatingSet: true,
		Rating:    &rating,
	})
	if err != nil {
		t.Fatalf("patch photo rating: %v", err)
	}
	if rated.Rating == nil || *rated.Rating != rating {
		t.Fatalf("rated = %#v", rated)
	}
	cleared, err := store.PatchPhotoBook(ctx, book.ID, contracts.PatchPhotoBookRequest{
		RatingSet:   true,
		RatingClear: true,
	})
	if err != nil {
		t.Fatalf("clear photo rating: %v", err)
	}
	if cleared.Rating != nil {
		t.Fatalf("cleared = %#v", cleared)
	}
	tooHigh := 5.1
	if _, err := store.PatchPhotoBook(ctx, book.ID, contracts.PatchPhotoBookRequest{
		RatingSet: true,
		Rating:    &tooHigh,
	}); err == nil {
		t.Fatal("expected over-range rating to fail")
	}
	if _, err := store.PatchPhotoBook(ctx, "missing", contracts.PatchPhotoBookRequest{
		RatingSet: true,
		Rating:    &rating,
	}); err == nil {
		t.Fatal("expected missing photo patch to fail")
	}
}

// TestListPhotoBooksFillsTagsInBatch 确认列表一次带回多本标签。
func TestListPhotoBooksFillsTagsInBatch(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "photo-tags.db"))
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
	first, err := store.UpsertPhotoBook(ctx, PhotoBookUpsert{
		LibraryPathID:  path.ID,
		Location:       filepath.Join(photoRoot, "A.cbz"),
		SourceFileName: "A.cbz",
		Title:          "Alpha",
		FileSize:       1,
		FileModifiedAt: "2026-09-11T00:00:00Z",
		PageCount:      1,
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.UpsertPhotoBook(ctx, PhotoBookUpsert{
		LibraryPathID:  path.ID,
		Location:       filepath.Join(photoRoot, "B.cbz"),
		SourceFileName: "B.cbz",
		Title:          "Beta",
		FileSize:       1,
		FileModifiedAt: "2026-09-11T00:00:00Z",
		PageCount:      1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ReplacePhotoTags(ctx, first.ID, []string{"portrait", "studio"}); err != nil {
		t.Fatal(err)
	}
	if err := store.ReplacePhotoTags(ctx, second.ID, []string{"night"}); err != nil {
		t.Fatal(err)
	}

	page, err := store.ListPhotoBooks(ctx, contracts.ListPhotoBooksRequest{Limit: 20})
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
	if len(byID[first.ID]) != 2 || byID[first.ID][0] != "portrait" || byID[first.ID][1] != "studio" {
		t.Fatalf("first tags = %#v", byID[first.ID])
	}
	if len(byID[second.ID]) != 1 || byID[second.ID][0] != "night" {
		t.Fatalf("second tags = %#v", byID[second.ID])
	}
}

// TestPhotoDisplayTitleSurvivesRescan 确认写真 user_title 覆盖在重扫后仍保留。
func TestPhotoDisplayTitleSurvivesRescan(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "photo-title.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	location := filepath.Join(t.TempDir(), "photos", "Portrait Set.cbz")
	book, err := store.UpsertPhotoBook(ctx, PhotoBookUpsert{
		Location:       location,
		SourceFileName: "Portrait Set.cbz",
		Title:          "Portrait Set",
		PageCount:      1,
	})
	if err != nil {
		t.Fatal(err)
	}
	title := "展示写真"
	patched, err := store.PatchPhotoBook(ctx, book.ID, contracts.PatchPhotoBookRequest{Title: &title})
	if err != nil {
		t.Fatal(err)
	}
	if patched.Title != title {
		t.Fatalf("patched = %#v", patched)
	}
	rescanned, err := store.UpsertPhotoBook(ctx, PhotoBookUpsert{
		Location:       location,
		SourceFileName: "Portrait Set.cbz",
		Title:          "Renamed From File",
		PageCount:      1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rescanned.ID != book.ID || rescanned.Title != title {
		t.Fatalf("rescanned = %#v", rescanned)
	}
	page, err := store.ListPhotoBooks(ctx, contracts.ListPhotoBooksRequest{Query: "展示", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || page.Items[0].Title != title {
		t.Fatalf("search overlay = %+v", page)
	}
	empty := "  "
	if _, err := store.PatchPhotoBook(ctx, book.ID, contracts.PatchPhotoBookRequest{Title: &empty}); err == nil {
		t.Fatal("expected empty title to fail")
	}
}


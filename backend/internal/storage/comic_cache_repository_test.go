package storage

import (
	"context"
	"path/filepath"
	"testing"
)

func TestComicCacheRepository(t *testing.T) {
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
		Location:       filepath.Join(root, "comics", "Cache.cbz"),
		SourceFileName: "Cache.cbz",
		Title:          "Cache",
		PageCount:      1,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := store.SaveComicCacheEntry(ctx, ComicCacheEntryInput{
		CacheKey:  "comic:" + book.ID + ":thumb:0",
		ComicID:   book.ID,
		Kind:      "thumbnail",
		PageIndex: 0,
		Path:      filepath.Join(root, "cache", "thumb.webp"),
		SizeBytes: 512,
	}); err != nil {
		t.Fatalf("save cache entry: %v", err)
	}

	entries, err := store.ListComicCacheEntries(ctx, 10)
	if err != nil {
		t.Fatalf("list cache entries: %v", err)
	}
	if len(entries) != 1 || entries[0].CacheKey == "" || entries[0].SizeBytes != 512 {
		t.Fatalf("entries = %+v", entries)
	}

	if err := store.TouchComicCacheEntry(ctx, entries[0].CacheKey); err != nil {
		t.Fatalf("touch cache entry: %v", err)
	}
	if err := store.DeleteComicCacheEntry(ctx, entries[0].CacheKey); err != nil {
		t.Fatalf("delete cache entry: %v", err)
	}
	entries, err = store.ListComicCacheEntries(ctx, 10)
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("entries after delete = %+v, want empty", entries)
	}
}

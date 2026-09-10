package comicscanner

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"testing"

	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

func TestScanComicRootsDiscoversSupportedArchives(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	store := newScannerTestStore(t, root)
	comicRoot := filepath.Join(root, "comics")
	if err := os.MkdirAll(comicRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	path, err := store.AddComicLibraryPath(ctx, comicRoot, "Comics")
	if err != nil {
		t.Fatal(err)
	}

	makeScannerZip(t, filepath.Join(comicRoot, "Book One.cbz"), map[string]string{
		"chapter/010.jpg": "ten",
		"chapter/001.jpg": "one",
	})
	makeScannerZip(t, filepath.Join(comicRoot, "Book Two.zip"), map[string]string{
		"001.png": "one",
	})
	makeScannerZip(t, filepath.Join(comicRoot, "Empty.cbz"), map[string]string{
		"notes.txt": "ignored",
	})
	if err := os.WriteFile(filepath.Join(comicRoot, "Ignored.rar"), []byte("rar"), 0o644); err != nil {
		t.Fatal(err)
	}

	svc := NewService(store)
	summary, err := svc.Scan(ctx, []contracts.ComicLibraryPathDTO{path})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if summary.FilesDiscovered != 3 || summary.Imported != 2 || summary.Skipped != 1 || len(summary.Errors) != 1 {
		t.Fatalf("summary = %+v, want 3 discovered, 2 imported, 1 skipped, 1 error", summary)
	}

	page, err := store.ListComicBooks(ctx, contracts.ListComicBooksRequest{Query: "Book", Limit: 10})
	if err != nil {
		t.Fatalf("list books: %v", err)
	}
	if page.Total != 2 {
		t.Fatalf("indexed books = %d, want 2", page.Total)
	}
	first, err := store.GetComicBookByLocation(ctx, filepath.Join(comicRoot, "Book One.cbz"))
	if err != nil {
		t.Fatalf("get book one: %v", err)
	}
	if first.Title != "Book One" || first.PageCount != 2 || len(first.Pages) != 2 {
		t.Fatalf("book one = %+v", first)
	}
	if first.Pages[0].EntryPath != "chapter/001.jpg" || first.Pages[1].EntryPath != "chapter/010.jpg" {
		t.Fatalf("pages not naturally sorted: %+v", first.Pages)
	}
}

func TestScanComicRootsUpdatesExistingBookPages(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	store := newScannerTestStore(t, root)
	comicRoot := filepath.Join(root, "comics")
	if err := os.MkdirAll(comicRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	path, err := store.AddComicLibraryPath(ctx, comicRoot, "Comics")
	if err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(comicRoot, "Mutable.cbz")
	makeScannerZip(t, archivePath, map[string]string{"001.jpg": "one"})

	svc := NewService(store)
	if _, err := svc.Scan(ctx, []contracts.ComicLibraryPathDTO{path}); err != nil {
		t.Fatalf("first scan: %v", err)
	}
	makeScannerZip(t, archivePath, map[string]string{
		"001.jpg": "one",
		"002.jpg": "two",
		"003.jpg": "three",
	})
	summary, err := svc.Scan(ctx, []contracts.ComicLibraryPathDTO{path})
	if err != nil {
		t.Fatalf("second scan: %v", err)
	}
	if summary.Updated != 1 {
		t.Fatalf("second scan updated = %d, want 1", summary.Updated)
	}
	detail, err := store.GetComicBookByLocation(ctx, archivePath)
	if err != nil {
		t.Fatal(err)
	}
	if detail.PageCount != 3 || len(detail.Pages) != 3 {
		t.Fatalf("detail after update = %+v, want 3 pages", detail)
	}
}

func newScannerTestStore(t *testing.T, root string) *storage.SQLiteStore {
	t.Helper()
	store, err := storage.NewSQLiteStore(filepath.Join(root, "scanner.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	return store
}

func makeScannerZip(t *testing.T, path string, entries map[string]string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(file)
	for entry, body := range entries {
		w, err := zw.Create(entry)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

package photoscanner

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"testing"

	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

func TestScanPhotoRootsDiscoversSupportedArchives(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	store := newPhotoScannerTestStore(t, root)
	photoRoot := filepath.Join(root, "photos")
	if err := os.MkdirAll(photoRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	path, err := store.AddPhotoLibraryPath(ctx, photoRoot, "Photos")
	if err != nil {
		t.Fatal(err)
	}

	makePhotoScannerZip(t, filepath.Join(photoRoot, "Portrait One.cbz"), map[string]string{
		"set/010.jpg": "ten",
		"set/001.jpg": "one",
	})
	makePhotoScannerZip(t, filepath.Join(photoRoot, "Portrait Two.zip"), map[string]string{
		"001.png": "one",
	})
	makePhotoScannerZip(t, filepath.Join(photoRoot, "Empty.cbz"), map[string]string{
		"notes.txt": "ignored",
	})

	svc := NewService(store)
	summary, err := svc.Scan(ctx, []contracts.PhotoLibraryPathDTO{path})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if summary.FilesDiscovered != 3 || summary.Imported != 2 || summary.Skipped != 1 || len(summary.Errors) != 1 {
		t.Fatalf("summary = %+v, want 3 discovered, 2 imported, 1 skipped, 1 error", summary)
	}

	page, err := store.ListPhotoBooks(ctx, contracts.ListPhotoBooksRequest{Query: "Portrait", Limit: 10})
	if err != nil {
		t.Fatalf("list photo books: %v", err)
	}
	if page.Total != 2 {
		t.Fatalf("indexed photo books = %d, want 2", page.Total)
	}
	first, err := store.GetPhotoBookByLocation(ctx, filepath.Join(photoRoot, "Portrait One.cbz"))
	if err != nil {
		t.Fatalf("get portrait one: %v", err)
	}
	if first.Title != "Portrait One" || first.PageCount != 2 || len(first.Pages) != 2 {
		t.Fatalf("portrait one = %+v", first)
	}
	if first.Pages[0].EntryPath != "set/001.jpg" || first.Pages[1].EntryPath != "set/010.jpg" {
		t.Fatalf("pages not naturally sorted: %+v", first.Pages)
	}
}

func newPhotoScannerTestStore(t *testing.T, root string) *storage.SQLiteStore {
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

func makePhotoScannerZip(t *testing.T, path string, entries map[string]string) {
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

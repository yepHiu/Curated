package comiccache

import (
	"archive/zip"
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"curated-backend/internal/comicscanner"
	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

func TestComicCacheServiceCreatesStatusAndCleansOnlyCacheFiles(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	store := newComicCacheServiceTestStore(t, root)
	comicRoot := filepath.Join(root, "comics")
	if err := os.MkdirAll(comicRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(comicRoot, "Book One.cbz")
	if err := writeComicZipBytes(archivePath, map[string][]byte{
		"001.png": tinyPNG(t),
	}); err != nil {
		t.Fatal(err)
	}
	path, err := store.AddComicLibraryPath(ctx, comicRoot, "Comics")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := comicscanner.NewService(store).Scan(ctx, []contracts.ComicLibraryPathDTO{path}); err != nil {
		t.Fatal(err)
	}
	page, err := store.ListComicBooks(ctx, contracts.ListComicBooksRequest{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	detail, err := store.GetComicBookDetail(ctx, page.Items[0].ID)
	if err != nil {
		t.Fatal(err)
	}

	cacheRoot := filepath.Join(root, "cache")
	svc := NewService(cacheRoot, 2*1024*1024*1024, store)
	file, err := svc.GetOrCreateThumbnail(ctx, detail, detail.Pages[0])
	if err != nil {
		t.Fatal(err)
	}
	if file.Path == "" || file.ContentType != "image/jpeg" || file.SizeBytes <= 0 {
		t.Fatalf("thumbnail file = %#v", file)
	}
	if _, err := os.Stat(file.Path); err != nil {
		t.Fatal(err)
	}

	status, err := svc.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if status.EntryCount != 1 || status.UsedBytes != file.SizeBytes {
		t.Fatalf("status = %#v, want one thumbnail entry", status)
	}

	status, err = svc.Cleanup(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if status.EntryCount != 0 || status.UsedBytes != 0 {
		t.Fatalf("status after cleanup = %#v, want empty", status)
	}
	if _, err := os.Stat(file.Path); !os.IsNotExist(err) {
		t.Fatalf("thumbnail should be removed, stat err=%v", err)
	}
	if _, err := os.Stat(archivePath); err != nil {
		t.Fatalf("source archive must not be deleted: %v", err)
	}
}

func newComicCacheServiceTestStore(t *testing.T, root string) *storage.SQLiteStore {
	t.Helper()
	store, err := storage.NewSQLiteStore(filepath.Join(root, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	return store
}

func tinyPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 8), G: uint8(y * 8), B: 180, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func writeComicZipBytes(path string, entries map[string][]byte) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := zip.NewWriter(file)
	for name, content := range entries {
		w, err := writer.Create(name)
		if err != nil {
			_ = writer.Close()
			return err
		}
		if _, err := w.Write(content); err != nil {
			_ = writer.Close()
			return err
		}
	}
	return writer.Close()
}

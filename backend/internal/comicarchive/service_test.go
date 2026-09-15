package comicarchive

import (
	"archive/zip"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSupportedArchiveAndImageEntries(t *testing.T) {
	t.Parallel()

	for _, path := range []string{"book.zip", "book.cbz", "C:/Comics/Book.CBZ"} {
		if !IsSupportedArchivePath(path) {
			t.Fatalf("IsSupportedArchivePath(%q) = false, want true", path)
		}
	}
	for _, path := range []string{"book.rar", "book.cbr", "book.7z", "book.pdf"} {
		if IsSupportedArchivePath(path) {
			t.Fatalf("IsSupportedArchivePath(%q) = true, want false", path)
		}
	}

	for _, entry := range []string{"001.jpg", "002.jpeg", "003.png", "004.webp", "005.gif"} {
		if !IsSupportedImageEntry(entry) {
			t.Fatalf("IsSupportedImageEntry(%q) = false, want true", entry)
		}
	}
	for _, entry := range []string{"nested.zip", "book.pdf", "video.mp4", "folder/"} {
		if IsSupportedImageEntry(entry) {
			t.Fatalf("IsSupportedImageEntry(%q) = true, want false", entry)
		}
	}
}

func TestListPagesUsesDirectoryHierarchyAndNaturalSort(t *testing.T) {
	t.Parallel()

	archivePath := makeComicZip(t, "sorted.cbz", map[string]string{
		"chapter2/001.jpg":  "c2-1",
		"chapter1/010.jpg":  "c1-10",
		"chapter1/002.jpg":  "c1-2",
		"chapter1/001.jpg":  "c1-1",
		"chapter1/info.txt": "ignore",
		"nested.zip":        "ignore",
	})

	pages, err := ListPages(context.Background(), archivePath)
	if err != nil {
		t.Fatalf("ListPages: %v", err)
	}
	got := make([]string, 0, len(pages))
	for _, page := range pages {
		got = append(got, page.EntryPath)
	}
	want := []string{
		"chapter1/001.jpg",
		"chapter1/002.jpg",
		"chapter1/010.jpg",
		"chapter2/001.jpg",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("page order = %#v, want %#v", got, want)
	}
}

func TestListPagesEmptyArchive(t *testing.T) {
	t.Parallel()

	archivePath := makeComicZip(t, "empty.zip", map[string]string{"notes.txt": "ignored"})
	_, err := ListPages(context.Background(), archivePath)
	if !errors.Is(err, ErrArchiveEmpty) {
		t.Fatalf("ListPages empty error = %v, want ErrArchiveEmpty", err)
	}
}

func TestOpenPageReadsSelectedImage(t *testing.T) {
	t.Parallel()

	archivePath := makeComicZip(t, "read.zip", map[string]string{
		"001.jpg": "first",
		"002.png": "second",
	})

	body, entry, err := OpenPage(context.Background(), archivePath, "002.png")
	if err != nil {
		t.Fatalf("OpenPage: %v", err)
	}
	defer body.Close()

	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("read page body: %v", err)
	}
	if string(data) != "second" || entry.EntryPath != "002.png" || entry.FileName != "002.png" {
		t.Fatalf("OpenPage body=%q entry=%+v, want second page", string(data), entry)
	}
}

// TestOpenPageMissingEntry 确认按条目打开时缺失页仍返回 ErrPageNotFound。
func TestOpenPageMissingEntry(t *testing.T) {
	t.Parallel()

	archivePath := makeComicZip(t, "missing.zip", map[string]string{
		"001.jpg": "first",
	})
	_, _, err := OpenPage(context.Background(), archivePath, "002.png")
	if !errors.Is(err, ErrPageNotFound) {
		t.Fatalf("OpenPage missing error = %v, want ErrPageNotFound", err)
	}
}

func makeComicZip(t *testing.T, name string, entries map[string]string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
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
	return path
}

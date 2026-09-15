package photoarchive

import (
	"archive/zip"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// TestOpenPageReadsSelectedImage 确认按 ZIP 条目名直接打开目标页。
func TestOpenPageReadsSelectedImage(t *testing.T) {
	t.Parallel()

	archivePath := makePhotoZip(t, "read.zip", map[string]string{
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

// TestOpenPageMissingEntry 确认缺失条目返回 ErrPageNotFound。
func TestOpenPageMissingEntry(t *testing.T) {
	t.Parallel()

	archivePath := makePhotoZip(t, "missing.zip", map[string]string{
		"001.jpg": "first",
	})
	_, _, err := OpenPage(context.Background(), archivePath, "002.png")
	if !errors.Is(err, ErrPageNotFound) {
		t.Fatalf("OpenPage missing error = %v, want ErrPageNotFound", err)
	}
}

// makePhotoZip 写入临时 ZIP 供 OpenPage 测试使用。
func makePhotoZip(t *testing.T, name string, entries map[string]string) string {
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

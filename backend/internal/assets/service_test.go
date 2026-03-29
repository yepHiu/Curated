package assets

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap"

	"curated-backend/internal/scraper"
)

func TestDownloadAll(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("asset"))
	}))
	defer server.Close()

	cacheDir := filepath.Join(t.TempDir(), "cache")
	service := NewService(zap.NewNop(), cacheDir, 5*time.Second, 0, 0)

	results, err := service.DownloadAll(context.Background(), scraper.Metadata{
		MovieID:       "abc-123",
		Number:        "ABC-123",
		CoverURL:      server.URL + "/cover.jpg",
		ThumbURL:      server.URL + "/thumb.webp",
		PreviewImages: []string{server.URL + "/preview-1.png"},
	})
	if err != nil {
		t.Fatalf("download failed: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 downloaded assets, got %d", len(results))
	}

	for _, result := range results {
		info, err := os.Stat(result.LocalPath)
		if err != nil {
			t.Fatalf("expected downloaded file %s: %v", result.LocalPath, err)
		}
		if info.Size() == 0 {
			t.Fatalf("expected non-empty downloaded file: %s", result.LocalPath)
		}
	}
}

func TestDownloadOverwritesExistingStableFileName(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("new-bytes"))
	}))
	defer server.Close()

	cacheDir := filepath.Join(t.TempDir(), "cache")
	destDir := filepath.Join(cacheDir, "movie-1")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		t.Fatal(err)
	}
	coverPath := filepath.Join(destDir, "cover.jpg")
	if err := os.WriteFile(coverPath, []byte("stale-on-disk"), 0o644); err != nil {
		t.Fatal(err)
	}

	service := NewService(zap.NewNop(), cacheDir, 5*time.Second, 0, 0)
	results, err := service.DownloadAllTo(context.Background(), scraper.Metadata{
		MovieID:  "movie-1",
		Number:   "ABC-123",
		CoverURL: server.URL + "/a.jpg",
	}, destDir)
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 asset, got %d", len(results))
	}
	b, err := os.ReadFile(results[0].LocalPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "new-bytes" {
		t.Fatalf("cover should be re-fetched over stale file, got %q", string(b))
	}
}

func TestPreviewSkipsDownloadWhenFileNonempty(t *testing.T) {
	t.Parallel()

	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("from-network"))
	}))
	defer server.Close()

	cacheDir := filepath.Join(t.TempDir(), "cache")
	destDir := filepath.Join(cacheDir, "movie-p")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		t.Fatal(err)
	}
	previewPath := filepath.Join(destDir, "preview-01.jpg")
	if err := os.WriteFile(previewPath, []byte("cached-preview"), 0o644); err != nil {
		t.Fatal(err)
	}

	service := NewService(zap.NewNop(), cacheDir, 5*time.Second, 0, 0)
	results, err := service.DownloadAllTo(context.Background(), scraper.Metadata{
		MovieID:       "movie-p",
		Number:        "PRE-1",
		PreviewImages: []string{server.URL + "/p1.jpg"},
	}, destDir)
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	if calls != 0 {
		t.Fatalf("expected preview skip (no HTTP), got %d requests", calls)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 asset, got %d", len(results))
	}
	b, err := os.ReadFile(results[0].LocalPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "cached-preview" {
		t.Fatalf("preview file should be unchanged, got %q", string(b))
	}
}

func TestCoverUnchangedOnDownloadFailure(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cacheDir := filepath.Join(t.TempDir(), "cache")
	destDir := filepath.Join(cacheDir, "movie-fail")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		t.Fatal(err)
	}
	coverPath := filepath.Join(destDir, "cover.jpg")
	if err := os.WriteFile(coverPath, []byte("keep-me"), 0o644); err != nil {
		t.Fatal(err)
	}

	service := NewService(zap.NewNop(), cacheDir, 5*time.Second, 0, 0)
	_, err := service.DownloadAllTo(context.Background(), scraper.Metadata{
		MovieID:  "movie-fail",
		Number:   "X",
		CoverURL: server.URL + "/c.jpg",
	}, destDir)
	if err == nil {
		t.Fatal("expected error from failed cover download")
	}
	b, err := os.ReadFile(coverPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "keep-me" {
		t.Fatalf("cover should be preserved after failed download, got %q", string(b))
	}
}

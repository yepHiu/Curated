package assets

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"

	"curated-backend/internal/scraper"
)

func TestDownloadAll(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("asset-image-content"))
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
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("new-bytes-image-content"))
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
	if string(b) != "new-bytes-image-content" {
		t.Fatalf("cover should be re-fetched over stale file, got %q", string(b))
	}
}

func TestPreviewSkipsDownloadWhenFileNonempty(t *testing.T) {
	t.Parallel()

	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("from-network-image-content"))
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

func TestDownloadActorAvatarRejectsNonImage(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html>blocked</html>"))
	}))
	defer server.Close()

	cacheDir := filepath.Join(t.TempDir(), "cache")
	service := NewService(zap.NewNop(), cacheDir, 5*time.Second, 0, 0)
	_, status, err := service.DownloadActorAvatar(context.Background(), "Alice", server.URL+"/avatar.jpg", ImageFetchOptions{})
	if err == nil {
		t.Fatal("expected non-image avatar download to fail")
	}
	if status != http.StatusOK {
		t.Fatalf("status=%d want %d", status, http.StatusOK)
	}
}

// TestDownloadSlotsAreProcessWide 确认两部影片同时下载时共享一个名额。
func TestDownloadSlotsAreProcessWide(t *testing.T) {
	t.Parallel()
	var inflight atomic.Int32
	var maxSeen atomic.Int32
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := inflight.Add(1)
		for {
			seen := maxSeen.Load()
			if current <= seen || maxSeen.CompareAndSwap(seen, current) {
				break
			}
		}
		<-release
		inflight.Add(-1)
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte("shared-slot-image"))
	}))
	defer server.Close()
	service := NewService(zap.NewNop(), t.TempDir(), 5*time.Second, 1, 1)
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, movieID := range []string{"movie-a", "movie-b"} {
		wg.Add(1)
		go func(movieID string) {
			defer wg.Done()
			_, err := service.DownloadAll(context.Background(), scraper.Metadata{
				MovieID: movieID, Number: movieID, CoverURL: server.URL + "/" + movieID + ".jpg",
			})
			errs <- err
		}(movieID)
	}
	time.Sleep(150 * time.Millisecond)
	if maxSeen.Load() != 1 {
		t.Fatalf("in-flight downloads = %d, want 1", maxSeen.Load())
	}
	close(release)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
}

// TestDownloadPrefersCoverBeforePreviews 确认名额为 1 时封面请求先于预览图。
func TestDownloadPrefersCoverBeforePreviews(t *testing.T) {
	t.Parallel()
	var order []string
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		order = append(order, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte("ordered-image"))
	}))
	defer server.Close()
	service := NewService(zap.NewNop(), t.TempDir(), 5*time.Second, 1, 1)
	_, err := service.DownloadAll(context.Background(), scraper.Metadata{
		MovieID: "movie-order", Number: "ORDER-1",
		CoverURL: server.URL + "/cover.jpg", PreviewImages: []string{server.URL + "/preview.jpg"},
	})
	if err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(order) != 2 || order[0] != "/cover.jpg" || order[1] != "/preview.jpg" {
		t.Fatalf("request order = %v", order)
	}
}

// TestDownloadSlotWaitCanBeCancelled 确认等待名额时取消不会占住槽位。
func TestDownloadSlotWaitCanBeCancelled(t *testing.T) {
	t.Parallel()
	release := make(chan struct{})
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte("held-slot-image"))
	}))
	defer server.Close()
	service := NewService(zap.NewNop(), t.TempDir(), 5*time.Second, 1, 1)
	holderDone := make(chan error, 1)
	go func() {
		_, err := service.DownloadAll(context.Background(), scraper.Metadata{
			MovieID: "holder", Number: "HOLD-1", CoverURL: server.URL + "/hold.jpg",
		})
		holderDone <- err
	}()
	<-started
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := service.DownloadAll(ctx, scraper.Metadata{
		MovieID: "waiter", Number: "WAIT-1", CoverURL: server.URL + "/wait.jpg",
	})
	if err == nil {
		t.Fatal("expected cancelled waiter")
	}
	close(release)
	if err := <-holderDone; err != nil {
		t.Fatal(err)
	}
}

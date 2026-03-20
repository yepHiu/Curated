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

	"jav-shadcn/backend/internal/scraper"
)

func TestDownloadAll(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("asset"))
	}))
	defer server.Close()

	cacheDir := filepath.Join(t.TempDir(), "cache")
	service := NewService(zap.NewNop(), cacheDir, 5*time.Second)

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

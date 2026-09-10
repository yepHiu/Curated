package photothumb

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
	"time"
)

func archive(t *testing.T, path string, width, height int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 100, 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	z := zip.NewWriter(f)
	w, err := z.Create("page.png")
	if err != nil {
		t.Fatal(err)
	}
	if err = png.Encode(w, img); err != nil {
		t.Fatal(err)
	}
	if err = z.Close(); err != nil {
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
}
func TestThumbnailBoundsCachingAndRevision(t *testing.T) {
	path := filepath.Join(t.TempDir(), "photos.cbz")
	archive(t, path, 1600, 1000)
	service := New()
	body, tag, err := service.Get(context.Background(), path, "page.png")
	if err != nil {
		t.Fatal(err)
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if format != "jpeg" || cfg.Width != 420 || cfg.Height != 262 {
		t.Fatalf("%s %+v", format, cfg)
	}
	cached, tag2, err := service.Get(context.Background(), path, "page.png")
	if err != nil || tag != tag2 || !bytes.Equal(body, cached) {
		t.Fatal("cache miss")
	}
	archive(t, path, 300, 600)
	future := time.Now().Add(time.Second)
	_ = os.Chtimes(path, future, future)
	changed, tag3, err := service.Get(context.Background(), path, "page.png")
	if err != nil {
		t.Fatal(err)
	}
	if tag3 == tag || bytes.Equal(body, changed) {
		t.Fatal("stale cache")
	}
	if service.bytes > cacheBudget {
		t.Fatal("cache budget exceeded")
	}
}
func TestMissingPageAndCancelledGeneration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "photos.cbz")
	archive(t, path, 50, 50)
	s := New()
	if _, _, err := s.Get(context.Background(), path, "missing.png"); err == nil {
		t.Fatal("missing page accepted")
	}
	s.gate <- struct{}{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := s.Get(ctx, path, "page.png"); err != context.Canceled {
		t.Fatalf("%v", err)
	}
}
func TestRejectInvalidImage(t *testing.T) {
	if _, err := encode([]byte("invalid")); err == nil {
		t.Fatal("invalid accepted")
	}
}

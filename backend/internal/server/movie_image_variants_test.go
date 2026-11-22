package server

import (
	"context"
	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/scraper"
	"curated-backend/internal/storage"
	"encoding/json"
	"go.uber.org/zap"
	"image"
	"image/color"
	"image/jpeg"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMovieImageVariantsKeepSourceValidationAndOriginalRoute(t *testing.T) {
	root := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store, err := storage.NewSQLiteStore(filepath.Join(root, "db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	movie, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{Path: filepath.Join(root, "ABC-123.mp4"), FileName: "ABC-123.mp4", Number: "ABC-123"})
	if err != nil {
		t.Fatal(err)
	}
	const sourceURL = "https://example.com/cover.jpg"
	if err := store.SaveMovieMetadata(ctx, scraper.Metadata{MovieID: movie.MovieID, Number: "ABC-123", Title: "sample", CoverURL: sourceURL}); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "cover.jpg")
	f, _ := os.Create(source)
	img := image.NewRGBA(image.Rect(0, 0, 1200, 1800))
	for y := 0; y < 1800; y++ {
		for x := 0; x < 1200; x++ {
			img.SetRGBA(x, y, color.RGBA{R: uint8(x), G: uint8(y), A: 255})
		}
	}
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	f.Close()
	if err := store.UpdateMediaAssetLocalPath(ctx, movie.MovieID, "cover", sourceURL, source); err != nil {
		t.Fatal(err)
	}
	h := NewHandler(Deps{Store: store, Cfg: config.Config{CacheDir: root}, Logger: zap.NewNop(), RuntimeContext: ctx})
	routes := h.Routes()
	var dto contracts.MovieDetailDTO
	deadline := time.Now().Add(5 * time.Second)
	for {
		w := httptest.NewRecorder()
		routes.ServeHTTP(w, httptest.NewRequest("GET", "/api/library/movies/"+movie.MovieID, nil))
		if w.Code != 200 {
			t.Fatalf("detail: %d %s", w.Code, w.Body.String())
		}
		json.Unmarshal(w.Body.Bytes(), &dto)
		if len(dto.CoverVariants) == 3 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("variants not generated")
		}
		time.Sleep(10 * time.Millisecond)
	}
	small := dto.CoverVariants[0].URL
	w := httptest.NewRecorder()
	routes.ServeHTTP(w, httptest.NewRequest("GET", small, nil))
	if w.Code != 200 || w.Header().Get("X-Curated-Image-Cache") != "hit" {
		t.Fatalf("variant: %d %s", w.Code, w.Body.String())
	}
	cfg, _, err := image.DecodeConfig(w.Body)
	if err != nil || cfg.Width != 320 || cfg.Height != 480 {
		t.Fatalf("dimensions %+v %v", cfg, err)
	}
	w = httptest.NewRecorder()
	routes.ServeHTTP(w, httptest.NewRequest("GET", dto.CoverURL, nil))
	cfg, _, err = image.DecodeConfig(w.Body)
	if err != nil || cfg.Width != 1200 {
		t.Fatalf("original changed: %+v %v", cfg, err)
	}
	w = httptest.NewRecorder()
	routes.ServeHTTP(w, httptest.NewRequest("GET", dto.CoverURL+"&width=123", nil))
	if w.Code != 400 {
		t.Fatal("invalid width allowed")
	}
	if err := os.Remove(source); err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	routes.ServeHTTP(w, httptest.NewRequest("GET", small, nil))
	if w.Code != 404 {
		t.Fatalf("derived image bypassed missing source: %d", w.Code)
	}
}

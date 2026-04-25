package server

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

func makeCuratedExportTestPNG(t *testing.T) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 8, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 8; x++ {
			img.Set(x, y, color.RGBA{R: uint8(10 * x), G: uint8(20 * y), B: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func newCuratedExportHandlerTestServer(t *testing.T) (*storage.SQLiteStore, *httptest.Server) {
	t.Helper()

	root := t.TempDir()
	store, err := storage.NewSQLiteStore(filepath.Join(root, "curated-export.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(NewHandler(Deps{
		Cfg:    config.Default(),
		Logger: zap.NewNop(),
		Store:  store,
	}).Routes())
	t.Cleanup(srv.Close)
	return store, srv
}

func addMovieForCuratedExportHandlerTest(t *testing.T, store *storage.SQLiteStore, code string) string {
	t.Helper()

	root := t.TempDir()
	videoPath := filepath.Join(root, code+".mp4")
	if err := os.WriteFile(videoPath, []byte("video"), 0o644); err != nil {
		t.Fatal(err)
	}
	outcome, err := store.PersistScanMovie(context.Background(), contracts.ScanFileResultDTO{
		TaskID:   "task-" + code,
		Path:     videoPath,
		FileName: code + ".mp4",
		Number:   code,
	})
	if err != nil {
		t.Fatal(err)
	}
	return outcome.MovieID
}

func TestHandlePostCuratedFramesExport_DefaultsToJPG(t *testing.T) {
	t.Parallel()

	store, srv := newCuratedExportHandlerTestServer(t)
	movieID := addMovieForCuratedExportHandlerTest(t, store, "EXP-001")
	if err := store.InsertCuratedFrame(context.Background(), storage.CuratedFrameMeta{
		ID:          "frame-1",
		MovieID:     movieID,
		Title:       "Export Frame",
		Code:        "EXP-001",
		Actors:      []string{"Airi"},
		PositionSec: 15,
		CapturedAt:  "2026-04-26T09:00:00Z",
		Tags:        []string{"closeup"},
	}, makeCuratedExportTestPNG(t)); err != nil {
		t.Fatal(err)
	}

	body := `{"ids":["frame-1"]}`
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/curated-frames/export", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		payload, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d body=%s", resp.StatusCode, string(payload))
	}
	if got, want := resp.Header.Get("Content-Type"), "image/jpeg"; got != want {
		t.Fatalf("Content-Type = %q, want %q", got, want)
	}
	if got := resp.Header.Get("Content-Disposition"); !strings.Contains(strings.ToLower(got), ".jpg") {
		t.Fatalf("Content-Disposition = %q, want .jpg filename", got)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("expected exported image bytes")
	}
}

func TestHandlePostCuratedFramesExport_AcceptsJPGFormat(t *testing.T) {
	t.Parallel()

	store, srv := newCuratedExportHandlerTestServer(t)
	movieID := addMovieForCuratedExportHandlerTest(t, store, "EXP-002")
	if err := store.InsertCuratedFrame(context.Background(), storage.CuratedFrameMeta{
		ID:          "frame-2",
		MovieID:     movieID,
		Title:       "Export Frame 2",
		Code:        "EXP-002",
		Actors:      []string{"Airi"},
		PositionSec: 18,
		CapturedAt:  "2026-04-26T09:05:00Z",
		Tags:        []string{"favorite"},
	}, makeCuratedExportTestPNG(t)); err != nil {
		t.Fatal(err)
	}

	payload, err := json.Marshal(contracts.PostCuratedFramesExportBody{
		IDs:    []string{"frame-2"},
		Format: "jpg",
	})
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/curated-frames/export", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d body=%s", resp.StatusCode, string(body))
	}
	if got, want := resp.Header.Get("Content-Type"), "image/jpeg"; got != want {
		t.Fatalf("Content-Type = %q, want %q", got, want)
	}
}

func TestHandlePostCuratedFramesExport_RejectsUnsupportedFormat(t *testing.T) {
	t.Parallel()

	store, srv := newCuratedExportHandlerTestServer(t)
	movieID := addMovieForCuratedExportHandlerTest(t, store, "EXP-003")
	if err := store.InsertCuratedFrame(context.Background(), storage.CuratedFrameMeta{
		ID:          "frame-3",
		MovieID:     movieID,
		Title:       "Export Frame 3",
		Code:        "EXP-003",
		Actors:      []string{"Airi"},
		PositionSec: 20,
		CapturedAt:  "2026-04-26T09:10:00Z",
		Tags:        []string{"closeup"},
	}, makeCuratedExportTestPNG(t)); err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/curated-frames/export", strings.NewReader(`{"ids":["frame-3"],"format":"gif"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d body=%s", resp.StatusCode, string(body))
	}
}

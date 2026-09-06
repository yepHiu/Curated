package server

import (
	"bytes"
	"context"
	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/executil"
	"curated-backend/internal/storage"
	"image"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSourceFrameFromSyntheticLibrary(t *testing.T) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("FFmpeg not installed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	root := t.TempDir()
	path := filepath.Join(root, "TEST-001.mp4")
	if output, err := executil.CommandContext(ctx, ffmpeg, "-hide_banner", "-loglevel", "error", "-f", "lavfi", "-i", "testsrc2=size=320x180:rate=10", "-t", "1", "-c:v", "libx264", path).CombinedOutput(); err != nil {
		t.Fatalf("source: %v %s", err, output)
	}
	store, err := storage.NewSQLiteStore(filepath.Join(root, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddLibraryPath(ctx, root, "Test"); err != nil {
		t.Fatal(err)
	}
	movie, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{TaskID: "test", Path: path, FileName: "TEST-001.mp4", Number: "TEST-001"})
	if err != nil {
		t.Fatal(err)
	}
	h := &Handler{store: store, cfg: config.Config{}}
	req := httptest.NewRequest(http.MethodPost, "/frame", strings.NewReader(`{"positionSec":0.4}`))
	req.SetPathValue("movieId", movie.MovieID)
	response := httptest.NewRecorder()
	h.handleExtractMovieFrame(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("frame: %d %s", response.Code, response.Body.String())
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(response.Body.Bytes()))
	if err != nil || cfg.Width != 320 || cfg.Height != 180 {
		t.Fatalf("source dimensions: %+v %v", cfg, err)
	}
}

func TestSourceFrameRejectsInvalidPosition(t *testing.T) {
	h := &Handler{}
	for _, body := range []string{`{"positionSec":-1}`, `{"positionSec":1e999}`, `{"positionSec":999999999}`, `broken`} {
		r := httptest.NewRequest(http.MethodPost, "/api/library/movies/test/frame", strings.NewReader(body))
		w := httptest.NewRecorder()
		h.handleExtractMovieFrame(w, r)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("%s: %d", body, w.Code)
		}
	}
}

func TestFrameOutputMemoryBound(t *testing.T) {
	b := boundedFrameBuffer{limit: 4}
	if _, err := b.Write([]byte("1234")); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Write([]byte("5")); err == nil {
		t.Fatal("oversize output accepted")
	}
	if b.Len() != 4 {
		t.Fatalf("buffer grew to %d", b.Len())
	}
}

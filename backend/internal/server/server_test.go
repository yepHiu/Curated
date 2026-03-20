package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"

	"jav-shadcn/backend/internal/config"
	"jav-shadcn/backend/internal/contracts"
	"jav-shadcn/backend/internal/storage"
)

func TestHandleDeleteMovie_NotFound(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	store, err := storage.NewSQLiteStore(filepath.Join(root, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}

	h := NewHandler(Deps{
		Cfg:    config.Config{},
		Logger: zap.NewNop(),
		Store:  store,
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodDelete, srv.URL+"/api/library/movies/does-not-exist", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
	var appErr contracts.AppError
	if err := json.NewDecoder(resp.Body).Decode(&appErr); err != nil {
		t.Fatal(err)
	}
	if appErr.Code != contracts.ErrorCodeNotFound {
		t.Fatalf("error code = %q, want %q", appErr.Code, contracts.ErrorCodeNotFound)
	}
}

func TestHandleDeleteMovie_Success204(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	videoPath := filepath.Join(root, "SRV-DEL.mp4")
	if err := os.WriteFile(videoPath, []byte("v"), 0o644); err != nil {
		t.Fatal(err)
	}

	store, err := storage.NewSQLiteStore(filepath.Join(root, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	outcome, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
		TaskID:   "t1",
		Path:     videoPath,
		FileName: "SRV-DEL.mp4",
		Number:   "SRV-DEL",
	})
	if err != nil {
		t.Fatal(err)
	}

	h := NewHandler(Deps{
		Cfg:    config.Config{},
		Logger: zap.NewNop(),
		Store:  store,
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodDelete, srv.URL+"/api/library/movies/"+outcome.MovieID, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
	if cl := resp.Header.Get("Content-Length"); cl != "" && cl != "0" {
		// 204 typically has no body; some stacks still set Content-Length: 0
	}

	_, err = store.GetMovieDetail(ctx, outcome.MovieID)
	if err == nil {
		t.Fatal("expected movie gone from DB")
	}
}

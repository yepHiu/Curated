package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
	"curated-backend/internal/tasks"
	"go.uber.org/zap"
)

// TestSelectedMovieFileHTTP verifies descriptor, range reads, and file ownership through actual routes.
func TestSelectedMovieFileHTTP(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	store, err := storage.NewSQLiteStore(filepath.Join(root, "http.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddLibraryPath(ctx, root, "Fixture"); err != nil {
		t.Fatal(err)
	}
	var movieID string
	for index, name := range []string{"ABC-123-CD1.mp4", "ABC-123-CD2.mp4"} {
		path := filepath.Join(root, name)
		content := []byte("first-source")
		if index == 1 {
			content = []byte("second-source")
		}
		if err := os.WriteFile(path, content, 0644); err != nil {
			t.Fatal(err)
		}
		row, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{Number: "ABC-123", Path: path})
		if err != nil {
			t.Fatal(err)
		}
		movieID = row.MovieID
	}
	other, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{Number: "DEF-123", Path: filepath.Join(root, "other.mp4")})
	if err != nil {
		t.Fatal(err)
	}
	detail, err := store.GetMovieDetail(ctx, movieID)
	if err != nil {
		t.Fatal(err)
	}
	selected := detail.Files[1].ID
	if err := store.UpsertPlaybackProgress(contracts.WithMovieFileSelection(ctx, selected), movieID, 45, 100); err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(Deps{Cfg: config.Config{}, Store: store, Tasks: tasks.NewManager(), Logger: zap.NewNop()})
	server := httptest.NewServer(handler.Routes())
	defer server.Close()
	query := "?fileId=" + url.QueryEscape(selected)
	response, err := http.Get(server.URL + "/api/library/movies/" + movieID + "/playback" + query)
	if err != nil {
		t.Fatal(err)
	}
	var descriptor contracts.PlaybackDescriptorDTO
	if err := json.NewDecoder(response.Body).Decode(&descriptor); err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 200 || descriptor.FileID != selected || descriptor.ResumePositionSec != 45 {
		t.Fatalf("descriptor=%+v status=%d", descriptor, response.StatusCode)
	}
	request, _ := http.NewRequest(http.MethodGet, server.URL+descriptor.URL, http.NoBody)
	request.Header.Set("Range", "bytes=0-5")
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	bytes, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != 206 || string(bytes) != "second" {
		t.Fatalf("range status=%d body=%q", response.StatusCode, bytes)
	}
	response, err = http.Get(server.URL + "/api/library/movies/" + other.MovieID + "/stream" + query)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 404 {
		t.Fatalf("foreign file status=%d", response.StatusCode)
	}
	// 同一作品的另一分片不能替换静态萃取帧的动图片段。
	if err := store.InsertCuratedFrame(ctx, storage.CuratedFrameMeta{ID: "first-frame", MovieID: movieID, FileID: detail.Files[0].ID, Actors: []string{}, Tags: []string{}}, []byte("png")); err != nil {
		t.Fatal(err)
	}
	response, err = http.Post(server.URL+"/api/library/movies/"+movieID+"/clips"+query, "application/json", strings.NewReader(`{"startSec":1,"endSec":3,"curatedFrameId":"first-frame"}`))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("foreign frame status=%d", response.StatusCode)
	}

}

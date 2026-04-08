package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	"curated-backend/internal/scraper"
	"curated-backend/internal/storage"
	"curated-backend/internal/tasks"
)

// testMovieMetadataRefresher registers a scrape.movie task in tm without running a real scraper.
type testMovieMetadataRefresher struct {
	tm *tasks.Manager
}

func (f *testMovieMetadataRefresher) StartMovieMetadataRefresh(ctx context.Context, movieID string) (contracts.TaskDTO, error) {
	_ = ctx
	task := f.tm.Create("scrape.movie", map[string]any{"movieId": movieID})
	return f.tm.Start(task.TaskID, "Scraping metadata (test)"), nil
}

func (f *testMovieMetadataRefresher) StartMetadataRefreshForLibraryPaths(ctx context.Context, paths []string) (contracts.MetadataRefreshQueuedDTO, error) {
	_ = ctx
	_ = paths
	return contracts.MetadataRefreshQueuedDTO{Queued: 0, Skipped: 0, InvalidPaths: nil}, nil
}

func (f *testMovieMetadataRefresher) StartActorProfileScrape(ctx context.Context, actorName string) (contracts.TaskDTO, error) {
	_ = ctx
	task := f.tm.Create("scrape.actor", map[string]any{"actorName": actorName})
	return f.tm.Start(task.TaskID, "Scraping actor profile (test)"), nil
}

type errMovieMetadataRefresher struct {
	err error
}

func (e *errMovieMetadataRefresher) StartMovieMetadataRefresh(ctx context.Context, movieID string) (contracts.TaskDTO, error) {
	_ = ctx
	_ = movieID
	return contracts.TaskDTO{}, e.err
}

func (e *errMovieMetadataRefresher) StartMetadataRefreshForLibraryPaths(ctx context.Context, paths []string) (contracts.MetadataRefreshQueuedDTO, error) {
	_ = ctx
	_ = paths
	return contracts.MetadataRefreshQueuedDTO{}, e.err
}

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

func TestHandleRevealMovie_NotFound(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	store, err := storage.NewSQLiteStore(filepath.Join(root, "reveal.db"))
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

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/library/movies/no-such-movie/reveal", http.NoBody)
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

func TestHandleDeleteMovie_SoftTrashThenPermanent204(t *testing.T) {
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

	cacheRoot := filepath.Join(root, "api-cache")
	movieCacheDir := filepath.Join(cacheRoot, outcome.MovieID)
	if err := os.MkdirAll(movieCacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(movieCacheDir, "poster.jpg"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	h := NewHandler(Deps{
		Cfg: config.Config{
			CacheDir: cacheRoot,
		},
		Logger: zap.NewNop(),
		Store:  store,
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	reqSoft, err := http.NewRequest(http.MethodDelete, srv.URL+"/api/library/movies/"+outcome.MovieID, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	respSoft, err := http.DefaultClient.Do(reqSoft)
	if err != nil {
		t.Fatal(err)
	}
	defer respSoft.Body.Close()
	if respSoft.StatusCode != http.StatusNoContent {
		t.Fatalf("soft delete status = %d, want 204", respSoft.StatusCode)
	}

	trashed, terr := store.IsMovieTrashed(ctx, outcome.MovieID)
	if terr != nil {
		t.Fatal(terr)
	}
	if !trashed {
		t.Fatal("expected movie in trash after soft DELETE")
	}
	if _, err := os.Stat(movieCacheDir); err != nil {
		t.Fatalf("cache dir should still exist after soft delete: %v", err)
	}

	reqPerm, err := http.NewRequest(
		http.MethodDelete,
		srv.URL+"/api/library/movies/"+outcome.MovieID+"?permanent=true",
		http.NoBody,
	)
	if err != nil {
		t.Fatal(err)
	}
	respPerm, err := http.DefaultClient.Do(reqPerm)
	if err != nil {
		t.Fatal(err)
	}
	defer respPerm.Body.Close()
	if respPerm.StatusCode != http.StatusNoContent {
		t.Fatalf("permanent delete status = %d, want 204", respPerm.StatusCode)
	}

	_, err = store.GetMovieDetail(ctx, outcome.MovieID)
	if err == nil {
		t.Fatal("expected movie gone from DB after permanent delete")
	}
	if _, err := os.Stat(movieCacheDir); !os.IsNotExist(err) {
		t.Fatalf("asset cache dir should be removed after permanent DELETE API: %v", err)
	}
}

func TestHandleDeleteMovie_PermanentWithoutTrash400(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	videoPath := filepath.Join(root, "SRV-PERM.mp4")
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
		FileName: "SRV-PERM.mp4",
		Number:   "SRV-PERM",
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

	req, err := http.NewRequest(
		http.MethodDelete,
		srv.URL+"/api/library/movies/"+outcome.MovieID+"?permanent=true",
		http.NoBody,
	)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	_, err = store.GetMovieDetail(ctx, outcome.MovieID)
	if err != nil {
		t.Fatalf("movie should still exist: %v", err)
	}
}

func TestHandleRestoreMovie_204(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	videoPath := filepath.Join(root, "SRV-RS.mp4")
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
		FileName: "SRV-RS.mp4",
		Number:   "SRV-RS",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.TrashMovie(ctx, outcome.MovieID); err != nil {
		t.Fatal(err)
	}

	h := NewHandler(Deps{
		Cfg:    config.Config{},
		Logger: zap.NewNop(),
		Store:  store,
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/library/movies/"+outcome.MovieID+"/restore", http.NoBody)
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
	ok, err := store.IsMovieTrashed(ctx, outcome.MovieID)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected movie restored (not trashed)")
	}
}

func TestHandleRefreshMovieMetadata_Accepted202AndTaskPoll(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	videoPath := filepath.Join(root, "SRV-SCRAPE.mp4")
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
		TaskID:   "t-scrape",
		Path:     videoPath,
		FileName: "SRV-SCRAPE.mp4",
		Number:   "SRV-SCRAPE",
	})
	if err != nil {
		t.Fatal(err)
	}

	tm := tasks.NewManager()
	ref := &testMovieMetadataRefresher{tm: tm}
	h := NewHandler(Deps{
		Cfg:                    config.Config{},
		Logger:                 zap.NewNop(),
		Store:                  store,
		Tasks:                  tm,
		MovieMetadataRefresher: ref,
		ActorProfileRefresher:  ref,
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/library/movies/"+outcome.MovieID+"/scrape", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", resp.StatusCode)
	}
	var task contracts.TaskDTO
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		t.Fatal(err)
	}
	if task.TaskID == "" {
		t.Fatal("expected taskId in body")
	}
	if task.Type != "scrape.movie" {
		t.Fatalf("task type = %q, want scrape.movie", task.Type)
	}

	pollReq, err := http.NewRequest(http.MethodGet, srv.URL+"/api/tasks/"+task.TaskID, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	pollResp, err := http.DefaultClient.Do(pollReq)
	if err != nil {
		t.Fatal(err)
	}
	defer pollResp.Body.Close()
	if pollResp.StatusCode != http.StatusOK {
		t.Fatalf("poll status = %d, want 200", pollResp.StatusCode)
	}
	var polled contracts.TaskDTO
	if err := json.NewDecoder(pollResp.Body).Decode(&polled); err != nil {
		t.Fatal(err)
	}
	if polled.TaskID != task.TaskID {
		t.Fatalf("polled task id = %q, want %q", polled.TaskID, task.TaskID)
	}
}

func TestHandleRefreshMovieMetadata_NotFound404(t *testing.T) {
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

	tm := tasks.NewManager()
	h := NewHandler(Deps{
		Cfg:    config.Config{},
		Logger: zap.NewNop(),
		Store:  store,
		Tasks:  tm,
		MovieMetadataRefresher: &errMovieMetadataRefresher{
			err: contracts.ErrScrapeMovieNotFound,
		},
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/library/movies/missing-id/scrape", http.NoBody)
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
}

func TestHandleRefreshMovieMetadata_BadRequestWhenNotConfiguredUsesNilRefresher(t *testing.T) {
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
		Tasks:  tasks.NewManager(),
		// MovieMetadataRefresher nil
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/library/movies/any/scrape", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", resp.StatusCode)
	}
}

func TestHandleRefreshMovieMetadata_ErrorsIsWrappedNotFound(t *testing.T) {
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

	tm := tasks.NewManager()
	h := NewHandler(Deps{
		Cfg:    config.Config{},
		Logger: zap.NewNop(),
		Store:  store,
		Tasks:  tm,
		MovieMetadataRefresher: &errMovieMetadataRefresher{
			err: fmt.Errorf("wrapped: %w", contracts.ErrScrapeMovieNotFound),
		},
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/library/movies/x/scrape", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 for fmt.Errorf-wrapped sentinel", resp.StatusCode)
	}
}

func TestHandleStreamMovie_OKAndRange(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	libRoot := filepath.Join(root, "lib")
	if err := os.MkdirAll(libRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	videoPath := filepath.Join(libRoot, "PLAY-001.mp4")
	content := []byte("fake-mp4-bytes-for-range-test")
	if err := os.WriteFile(videoPath, content, 0o644); err != nil {
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
	if _, err := store.AddLibraryPath(ctx, libRoot, ""); err != nil {
		t.Fatal(err)
	}
	outcome, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
		TaskID:   "t1",
		Path:     videoPath,
		FileName: "PLAY-001.mp4",
		Number:   "PLAY-001",
	})
	if err != nil {
		t.Fatal(err)
	}

	h := NewHandler(Deps{
		Cfg:    config.Config{},
		Logger: zap.NewNop(),
		Store:  store,
		Tasks:  tasks.NewManager(),
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	fullURL := srv.URL + "/api/library/movies/" + outcome.MovieID + "/stream"
	req, err := http.NewRequest(http.MethodGet, fullURL, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET stream status = %d, want 200", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != string(content) {
		t.Fatalf("body mismatch")
	}
	if acceptRanges := resp.Header.Get("Accept-Ranges"); acceptRanges != "bytes" {
		t.Fatalf("Accept-Ranges = %q, want bytes", acceptRanges)
	}

	req2, err := http.NewRequest(http.MethodGet, fullURL, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	req2.Header.Set("Range", "bytes=4-7")
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusPartialContent {
		t.Fatalf("Range status = %d, want 206", resp2.StatusCode)
	}
	partial, err := io.ReadAll(resp2.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(partial) != "-mp4" {
		t.Fatalf("partial body = %q, want -mp4", string(partial))
	}
}

func TestHandleGetMoviePlaybackDescriptor_OK(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	libRoot := filepath.Join(root, "lib")
	if err := os.MkdirAll(libRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	videoPath := filepath.Join(libRoot, "PLAY-DESC-001.mp4")
	if err := os.WriteFile(videoPath, []byte("fake-mp4-bytes"), 0o644); err != nil {
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
	if _, err := store.AddLibraryPath(ctx, libRoot, ""); err != nil {
		t.Fatal(err)
	}
	outcome, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
		TaskID:   "t1",
		Path:     videoPath,
		FileName: "PLAY-DESC-001.mp4",
		Number:   "PLAY-DESC-001",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertPlaybackProgress(ctx, outcome.MovieID, 123, 456); err != nil {
		t.Fatal(err)
	}

	h := NewHandler(Deps{
		Cfg:    config.Config{},
		Logger: zap.NewNop(),
		Store:  store,
		Tasks:  tasks.NewManager(),
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/library/movies/"+outcome.MovieID+"/playback", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var dto contracts.PlaybackDescriptorDTO
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		t.Fatal(err)
	}
	if dto.MovieID != outcome.MovieID {
		t.Fatalf("movieId = %q, want %q", dto.MovieID, outcome.MovieID)
	}
	if dto.Mode != contracts.PlaybackModeDirect {
		t.Fatalf("mode = %q, want %q", dto.Mode, contracts.PlaybackModeDirect)
	}
	if dto.URL != "/api/library/movies/"+outcome.MovieID+"/stream" {
		t.Fatalf("url = %q", dto.URL)
	}
	if dto.FileName != "PLAY-DESC-001.mp4" {
		t.Fatalf("fileName = %q", dto.FileName)
	}
	if dto.MimeType != "video/mp4" {
		t.Fatalf("mimeType = %q, want video/mp4", dto.MimeType)
	}
	if dto.ResumePositionSec != 123 {
		t.Fatalf("resumePositionSec = %v, want 123", dto.ResumePositionSec)
	}
	if dto.DurationSec != 456 {
		t.Fatalf("durationSec = %v, want 456", dto.DurationSec)
	}
	if !dto.CanDirectPlay {
		t.Fatal("expected canDirectPlay true")
	}
}

func TestHandleCreatePlaybackSession_OK(t *testing.T) {
	t.Parallel()
	h := NewHandler(Deps{
		Cfg:    config.Config{},
		Logger: zap.NewNop(),
		PlaybackResolver: stubPlaybackResolver{
			createDTO: contracts.PlaybackDescriptorDTO{
				Mode:          contracts.PlaybackModeHLS,
				SessionID:     "sess-1",
				URL:           "/api/playback/sessions/sess-1/hls/index.m3u8",
				MimeType:      "application/vnd.apple.mpegurl",
				CanDirectPlay: false,
			},
		},
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/library/movies/demo-1/playback-session", strings.NewReader(`{"mode":"hls"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
	var dto contracts.PlaybackDescriptorDTO
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		t.Fatal(err)
	}
	if dto.Mode != contracts.PlaybackModeHLS {
		t.Fatalf("mode = %q, want %q", dto.Mode, contracts.PlaybackModeHLS)
	}
	if dto.SessionID != "sess-1" {
		t.Fatalf("sessionId = %q, want sess-1", dto.SessionID)
	}
}

func TestHandleGetPlaybackSessionFile_M3U8ServedWithoutCache(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	playlistPath := filepath.Join(root, "index.m3u8")
	if err := os.WriteFile(playlistPath, []byte("#EXTM3U\n#EXTINF:2.0,\nsegment-00000.ts\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	h := NewHandler(Deps{
		Cfg:    config.Config{},
		Logger: zap.NewNop(),
		PlaybackResolver: stubPlaybackResolver{
			filePath: playlistPath,
		},
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/playback/sessions/sess-1/hls/index.m3u8", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Range", "bytes=0-8")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Type"); !strings.Contains(got, "application/vnd.apple.mpegurl") {
		t.Fatalf("content-type = %q", got)
	}
	if got := resp.Header.Get("Cache-Control"); !strings.Contains(got, "no-store") {
		t.Fatalf("cache-control = %q", got)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "segment-00000.ts") {
		t.Fatalf("playlist body = %q", string(body))
	}
}

func TestHandleLaunchNativePlayback_OK(t *testing.T) {
	t.Parallel()
	h := NewHandler(Deps{
		Cfg:                    config.Config{},
		Logger:                 zap.NewNop(),
		NativePlaybackLauncher: stubNativePlaybackLauncher{},
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/library/movies/demo-1/native-play", strings.NewReader(`{"startPositionSec":12}`))
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
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var dto contracts.NativePlaybackLaunchDTO
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		t.Fatal(err)
	}
	if !dto.OK {
		t.Fatal("expected ok=true")
	}
	if dto.Command != "mpv" {
		t.Fatalf("command = %q, want mpv", dto.Command)
	}
}

func TestHandleStreamMovie_NotFoundUnknownMovie(t *testing.T) {
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
		Tasks:  tasks.NewManager(),
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/library/movies/no-such-id/stream", http.NoBody)
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
}

func TestHandlePatchMovie_SQLiteFavoriteAndRating(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	videoPath := filepath.Join(root, "PATCH-RAT.mp4")
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
		TaskID:   "t-patch",
		Path:     videoPath,
		FileName: "PATCH-RAT.mp4",
		Number:   "PATCH-RAT",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveMovieMetadata(ctx, scraper.Metadata{
		MovieID: outcome.MovieID,
		Number:  "PATCH-RAT",
		Title:   "Patch Rating Title",
		Summary: "S",
		Studio:  "St",
		Rating:  4.5,
	}); err != nil {
		t.Fatal(err)
	}

	h := NewHandler(Deps{
		Cfg:    config.Config{},
		Logger: zap.NewNop(),
		Store:  store,
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	body := `{"isFavorite":true,"rating":4.25}`
	req, err := http.NewRequest(http.MethodPatch, srv.URL+"/api/library/movies/"+outcome.MovieID, strings.NewReader(body))
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
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var detail contracts.MovieDetailDTO
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		t.Fatal(err)
	}
	if !detail.IsFavorite || detail.Rating != 4.25 {
		t.Fatalf("detail = %+v, want favorite=true rating=4.25", detail)
	}
	if detail.UserRating == nil || *detail.UserRating != 4.25 {
		t.Fatalf("userRating = %v, want 4.25", detail.UserRating)
	}
	if detail.MetadataRating != 4.5 {
		t.Fatalf("metadataRating = %v, want 4.5", detail.MetadataRating)
	}
}

func TestHandlePatchMovie_ClearUserRating(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	videoPath := filepath.Join(root, "PATCH-CLR.mp4")
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
		TaskID: "t-clr", Path: videoPath, FileName: "PATCH-CLR.mp4", Number: "PATCH-CLR",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveMovieMetadata(ctx, scraper.Metadata{
		MovieID: outcome.MovieID, Number: "PATCH-CLR", Title: "T", Summary: "S", Studio: "St", Rating: 3.0,
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.PatchMovieUserPrefs(ctx, outcome.MovieID, contracts.PatchMovieInput{
		UserRatingSet: true, UserRating: 1.0,
	}); err != nil {
		t.Fatal(err)
	}

	h := NewHandler(Deps{Cfg: config.Config{}, Logger: zap.NewNop(), Store: store})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodPatch, srv.URL+"/api/library/movies/"+outcome.MovieID, strings.NewReader(`{"rating":null}`))
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
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var detail contracts.MovieDetailDTO
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		t.Fatal(err)
	}
	if detail.Rating != 3.0 || detail.UserRating != nil {
		t.Fatalf("after clear: rating=%v userRating=%v, want rating=3 userRating=nil", detail.Rating, detail.UserRating)
	}
}

func TestHandlePatchMovie_UserTags(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	videoPath := filepath.Join(root, "PATCH-UT.mp4")
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
		TaskID: "t-ut", Path: videoPath, FileName: "PATCH-UT.mp4", Number: "PATCH-UT",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveMovieMetadata(ctx, scraper.Metadata{
		MovieID: outcome.MovieID, Number: "PATCH-UT", Title: "T", Summary: "S", Studio: "St", Tags: []string{"NfoTag"},
	}); err != nil {
		t.Fatal(err)
	}

	h := NewHandler(Deps{Cfg: config.Config{}, Logger: zap.NewNop(), Store: store})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	body := `{"userTags":["mine","local"]}`
	req, err := http.NewRequest(http.MethodPatch, srv.URL+"/api/library/movies/"+outcome.MovieID, strings.NewReader(body))
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
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var detail contracts.MovieDetailDTO
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		t.Fatal(err)
	}
	if len(detail.UserTags) != 2 || detail.UserTags[0] != "local" || detail.UserTags[1] != "mine" {
		t.Fatalf("userTags sorted = %#v", detail.UserTags)
	}
	if len(detail.Tags) != 1 || detail.Tags[0] != "NfoTag" {
		t.Fatalf("nfo tags = %#v", detail.Tags)
	}
}

func TestHandlePatchMovie_MetadataTags(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	videoPath := filepath.Join(root, "PATCH-MT.mp4")
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
		TaskID: "t-mt", Path: videoPath, FileName: "PATCH-MT.mp4", Number: "PATCH-MT",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveMovieMetadata(ctx, scraper.Metadata{
		MovieID: outcome.MovieID, Number: "PATCH-MT", Title: "T", Summary: "S", Studio: "St",
		Tags: []string{"KeepA", "DropB"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.PatchMovieUserPrefs(ctx, outcome.MovieID, contracts.PatchMovieInput{
		UserTagsSet: true,
		UserTags:    []string{"user-only"},
	}); err != nil {
		t.Fatal(err)
	}

	h := NewHandler(Deps{Cfg: config.Config{}, Logger: zap.NewNop(), Store: store})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	body := `{"metadataTags":["KeepA"]}`
	req, err := http.NewRequest(http.MethodPatch, srv.URL+"/api/library/movies/"+outcome.MovieID, strings.NewReader(body))
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
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var detail contracts.MovieDetailDTO
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		t.Fatal(err)
	}
	if len(detail.Tags) != 1 || detail.Tags[0] != "KeepA" {
		t.Fatalf("metadata tags = %#v, want [KeepA]", detail.Tags)
	}
	if len(detail.UserTags) != 1 || detail.UserTags[0] != "user-only" {
		t.Fatalf("user tags should be untouched, got %#v", detail.UserTags)
	}
}

func TestHandlePatchMovie_DisplayOverrides(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	videoPath := filepath.Join(root, "PATCH-DO.mp4")
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
		TaskID: "t-do", Path: videoPath, FileName: "PATCH-DO.mp4", Number: "PATCH-DO",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveMovieMetadata(ctx, scraper.Metadata{
		MovieID: outcome.MovieID, Number: "PATCH-DO", Title: "BaseTitle", Summary: "BaseSum", Studio: "BaseStudio",
		RuntimeMinutes: 60, Rating: 3, ReleaseDate: "2019-06-15",
	}); err != nil {
		t.Fatal(err)
	}

	h := NewHandler(Deps{Cfg: config.Config{}, Logger: zap.NewNop(), Store: store})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	patchBody := `{"userTitle":"User T","userStudio":"User St","userSummary":"User Sum","userReleaseDate":"2022-03-04","userRuntimeMinutes":99}`
	req, err := http.NewRequest(http.MethodPatch, srv.URL+"/api/library/movies/"+outcome.MovieID, strings.NewReader(patchBody))
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
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var detail contracts.MovieDetailDTO
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		t.Fatal(err)
	}
	if detail.Title != "User T" || detail.Studio != "User St" || detail.Summary != "User Sum" {
		t.Fatalf("display fields = %#v %#v %#v", detail.Title, detail.Studio, detail.Summary)
	}
	if detail.ReleaseDate != "2022-03-04" || detail.Year != 2022 || detail.RuntimeMinutes != 99 {
		t.Fatalf("release/year/runtime = %s %d %d", detail.ReleaseDate, detail.Year, detail.RuntimeMinutes)
	}

	clearBody := `{"userTitle":null,"userStudio":null,"userSummary":null,"userReleaseDate":null,"userRuntimeMinutes":null}`
	req2, err := http.NewRequest(http.MethodPatch, srv.URL+"/api/library/movies/"+outcome.MovieID, strings.NewReader(clearBody))
	if err != nil {
		t.Fatal(err)
	}
	req2.Header.Set("Content-Type", "application/json")
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("clear status = %d, want 200", resp2.StatusCode)
	}
	if err := json.NewDecoder(resp2.Body).Decode(&detail); err != nil {
		t.Fatal(err)
	}
	if detail.Title != "BaseTitle" || detail.Studio != "BaseStudio" || detail.Summary != "BaseSum" {
		t.Fatalf("after clear = %#v %#v %#v", detail.Title, detail.Studio, detail.Summary)
	}
	if detail.RuntimeMinutes != 60 {
		t.Fatalf("runtime = %d, want 60", detail.RuntimeMinutes)
	}
}

func TestHandlePatchMovie_InvalidRating(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	videoPath := filepath.Join(root, "PATCH-BAD.mp4")
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
		TaskID: "t-bad", Path: videoPath, FileName: "PATCH-BAD.mp4", Number: "PATCH-BAD",
	})
	if err != nil {
		t.Fatal(err)
	}

	h := NewHandler(Deps{Cfg: config.Config{}, Logger: zap.NewNop(), Store: store})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodPatch, srv.URL+"/api/library/movies/"+outcome.MovieID, strings.NewReader(`{"rating":5.01}`))
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
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestHandlePatchMovie_NotFoundWhenMovieMissing(t *testing.T) {
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

	h := NewHandler(Deps{Cfg: config.Config{}, Logger: zap.NewNop(), Store: store})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodPatch, srv.URL+"/api/library/movies/mkb-100", strings.NewReader(`{"isFavorite":false,"rating":2.5}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

type errActorProfileRefresher struct {
	err error
}

func (e *errActorProfileRefresher) StartActorProfileScrape(ctx context.Context, actorName string) (contracts.TaskDTO, error) {
	_ = ctx
	_ = actorName
	return contracts.TaskDTO{}, e.err
}

func TestHandleGetActorProfile_MissingName(t *testing.T) {
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

	h := NewHandler(Deps{Cfg: config.Config{}, Logger: zap.NewNop(), Store: store})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/library/actors/profile", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestHandleGetActorProfile_NotFound(t *testing.T) {
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

	h := NewHandler(Deps{Cfg: config.Config{}, Logger: zap.NewNop(), Store: store})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/library/actors/profile?name=NoSuchActor", http.NoBody)
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
}

func TestHandleScrapeActorProfile_NotConfigured(t *testing.T) {
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

	h := NewHandler(Deps{Cfg: config.Config{}, Logger: zap.NewNop(), Store: store})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/library/actors/scrape?name=Anyone", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", resp.StatusCode)
	}
}

func TestHandleScrapeActorProfile_NotFound(t *testing.T) {
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
		Cfg:                   config.Config{},
		Logger:                zap.NewNop(),
		Store:                 store,
		ActorProfileRefresher: &errActorProfileRefresher{err: contracts.ErrActorNotFound},
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/library/actors/scrape?name=Ghost", http.NoBody)
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
}

func TestHandleListActors_Empty(t *testing.T) {
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

	h := NewHandler(Deps{Cfg: config.Config{}, Logger: zap.NewNop(), Store: store})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/library/actors", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body contracts.ListActorsResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Total != 0 || len(body.Actors) != 0 {
		t.Fatalf("body = %+v", body)
	}
}

func TestHandlePatchActorUserTags_NotFound(t *testing.T) {
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

	h := NewHandler(Deps{Cfg: config.Config{}, Logger: zap.NewNop(), Store: store})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodPatch, srv.URL+"/api/library/actors/tags?name=Nobody", strings.NewReader(`{"userTags":["x"]}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestHandleGetRecentTasks(t *testing.T) {
	t.Parallel()
	tm := tasks.NewManager()
	x := tm.Create("scan.library", map[string]any{"trigger": "fsnotify"})
	tm.Start(x.TaskID, "scan")
	tm.Complete(x.TaskID, "ok")

	h := NewHandler(Deps{
		Cfg:    config.Config{},
		Logger: zap.NewNop(),
		Store:  nil,
		Tasks:  tm,
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/tasks/recent?limit=5", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body contracts.RecentTasksDTO
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Tasks) != 1 || body.Tasks[0].TaskID != x.TaskID {
		t.Fatalf("unexpected body: %+v", body.Tasks)
	}
}

func TestHandleMovieComment_GetNotFoundMovie(t *testing.T) {
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
	h := NewHandler(Deps{Cfg: config.Config{}, Logger: zap.NewNop(), Store: store})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/library/movies/missing-id/comment", http.NoBody)
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
}

func TestHandleMovieComment_PutGet(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
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
		TaskID: "t", Path: "D:/Media/JAV/CMT-1.mp4", FileName: "CMT-1.mp4", Number: "CMT-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	mid := outcome.MovieID

	h := NewHandler(Deps{Cfg: config.Config{}, Logger: zap.NewNop(), Store: store})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	getURL := srv.URL + "/api/library/movies/" + mid + "/comment"
	req, err := http.NewRequest(http.MethodGet, getURL, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get status = %d body=%s", resp.StatusCode, string(b))
	}
	var empty contracts.MovieCommentDTO
	if err := json.Unmarshal(b, &empty); err != nil {
		t.Fatal(err)
	}
	if empty.Body != "" {
		t.Fatalf("want empty body, got %q", empty.Body)
	}

	putBody := []byte(`{"body":"hello comment"}`)
	req2, err := http.NewRequest(http.MethodPut, getURL, bytes.NewReader(putBody))
	if err != nil {
		t.Fatal(err)
	}
	req2.Header.Set("Content-Type", "application/json")
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	b2, _ := io.ReadAll(resp2.Body)
	_ = resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("put status = %d body=%s", resp2.StatusCode, string(b2))
	}
	var saved contracts.MovieCommentDTO
	if err := json.Unmarshal(b2, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Body != "hello comment" {
		t.Fatalf("saved body = %q", saved.Body)
	}

	resp3, err := http.DefaultClient.Get(getURL)
	if err != nil {
		t.Fatal(err)
	}
	b3, _ := io.ReadAll(resp3.Body)
	_ = resp3.Body.Close()
	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("get2 status = %d", resp3.StatusCode)
	}
	var got contracts.MovieCommentDTO
	if err := json.Unmarshal(b3, &got); err != nil {
		t.Fatal(err)
	}
	if got.Body != "hello comment" {
		t.Fatalf("get2 body = %q", got.Body)
	}
}

type stubOrganizeCtl struct{}

func (stubOrganizeCtl) OrganizeLibrary() bool         { return true }
func (stubOrganizeCtl) SetOrganizeLibrary(bool) error { return nil }

type stubExtendedImportCtl struct{ v bool }

func (s stubExtendedImportCtl) ExtendedLibraryImport() bool { return s.v }
func (stubExtendedImportCtl) SetExtendedLibraryImport(bool) error {
	return nil
}

type stubAutoWatchCtl struct{}

func (stubAutoWatchCtl) AutoLibraryWatch() bool         { return true }
func (stubAutoWatchCtl) SetAutoLibraryWatch(bool) error { return nil }

type stubMetadataCtl struct{}

func (stubMetadataCtl) MetadataMovieProvider() string                { return "" }
func (stubMetadataCtl) SetMetadataMovieProvider(string) error        { return nil }
func (stubMetadataCtl) MetadataMovieProviderChain() []string         { return nil }
func (stubMetadataCtl) SetMetadataMovieProviderChain([]string) error { return nil }
func (stubMetadataCtl) MetadataMovieScrapeMode() string              { return "auto" }
func (stubMetadataCtl) SetMetadataMovieScrapeMode(string) error      { return nil }
func (stubMetadataCtl) MetadataMovieStrategy() string                { return "auto-cn-friendly" }
func (stubMetadataCtl) SetMetadataMovieStrategy(string) error        { return nil }
func (stubMetadataCtl) ListMetadataMovieProviders() []string         { return nil }

type stubBackendLogCtl struct {
	v contracts.BackendLogSettingsDTO
}

func (s *stubBackendLogCtl) BackendLogSettings() contracts.BackendLogSettingsDTO {
	return s.v
}

func (s *stubBackendLogCtl) SetBackendLogPatch(p contracts.PatchBackendLogSettings) error {
	if p.LogDir != nil {
		s.v.LogDir = strings.TrimSpace(*p.LogDir)
	}
	if p.LogFilePrefix != nil {
		s.v.LogFilePrefix = strings.TrimSpace(*p.LogFilePrefix)
	}
	if p.LogMaxAgeDays != nil {
		s.v.LogMaxAgeDays = *p.LogMaxAgeDays
	}
	if p.LogLevel != nil {
		l := strings.TrimSpace(*p.LogLevel)
		if l == "" {
			l = "info"
		}
		s.v.LogLevel = l
	}
	return nil
}

type stubPlayerSettingsCtl struct {
	v contracts.PlayerSettingsDTO
}

func (s *stubPlayerSettingsCtl) PlayerSettings() contracts.PlayerSettingsDTO {
	return s.v
}

func (s *stubPlayerSettingsCtl) SetPlayerSettingsPatch(p contracts.PatchPlayerSettingsDTO) error {
	if p.HardwareDecode != nil {
		s.v.HardwareDecode = *p.HardwareDecode
	}
	if p.NativePlayerEnabled != nil {
		s.v.NativePlayerEnabled = *p.NativePlayerEnabled
	}
	if p.NativePlayerCommand != nil {
		cmd := strings.TrimSpace(*p.NativePlayerCommand)
		if cmd == "" {
			cmd = "mpv"
		}
		s.v.NativePlayerCommand = cmd
	}
	if p.StreamPushEnabled != nil {
		s.v.StreamPushEnabled = *p.StreamPushEnabled
	}
	if p.FFmpegCommand != nil {
		cmd := strings.TrimSpace(*p.FFmpegCommand)
		if cmd == "" {
			cmd = "ffmpeg"
		}
		s.v.FFmpegCommand = cmd
	}
	if p.PreferNativePlayer != nil {
		s.v.PreferNativePlayer = *p.PreferNativePlayer
	}
	if p.SeekForwardStepSec != nil {
		s.v.SeekForwardStepSec = *p.SeekForwardStepSec
	}
	if p.SeekBackwardStepSec != nil {
		s.v.SeekBackwardStepSec = *p.SeekBackwardStepSec
	}
	return nil
}

type stubPlaybackResolver struct {
	resolveDTO  contracts.PlaybackDescriptorDTO
	createDTO   contracts.PlaybackDescriptorDTO
	filePath    string
	sessionByID contracts.PlaybackSessionStatusDTO
	sessions    []contracts.PlaybackSessionStatusDTO
}

func (s stubPlaybackResolver) ResolvePlayback(ctx context.Context, movieID string) (contracts.PlaybackDescriptorDTO, error) {
	_ = ctx
	dto := s.resolveDTO
	dto.MovieID = movieID
	return dto, nil
}

func (s stubPlaybackResolver) CreatePlaybackSession(ctx context.Context, movieID string, mode contracts.PlaybackMode, startPositionSec float64) (contracts.PlaybackDescriptorDTO, error) {
	_ = ctx
	_ = startPositionSec
	dto := s.createDTO
	dto.MovieID = movieID
	dto.Mode = mode
	return dto, nil
}

func (s stubPlaybackResolver) ResolvePlaybackSessionFile(sessionID string, name string) (string, error) {
	_ = sessionID
	_ = name
	return s.filePath, nil
}

func (s stubPlaybackResolver) DeletePlaybackSession(sessionID string) error {
	_ = sessionID
	return nil
}

type stubNativePlaybackLauncher struct{}

func (stubNativePlaybackLauncher) LaunchNativePlayback(ctx context.Context, movieID string, startPositionSec float64) (contracts.NativePlaybackLaunchDTO, error) {
	_ = ctx
	_ = startPositionSec
	return contracts.NativePlaybackLaunchDTO{
		OK:      true,
		Command: "mpv",
		Target:  "C:/media/demo.mkv",
		Mode:    "native",
		Message: "native player launched",
		MovieID: movieID,
	}, nil
}

func TestHandleGetSettings_ExtendedLibraryImportFromController(t *testing.T) {
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
		Cfg:                      config.Config{Player: config.PlayerConfig{HardwareDecode: true}},
		Logger:                   zap.NewNop(),
		Store:                    store,
		OrganizeLibraryCtl:       stubOrganizeCtl{},
		ExtendedLibraryImportCtl: stubExtendedImportCtl{v: true},
		AutoLibraryWatchCtl:      stubAutoWatchCtl{},
		MetadataScrapeCtl:        stubMetadataCtl{},
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/api/settings")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var dto contracts.SettingsDTO
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		t.Fatal(err)
	}
	if !dto.ExtendedLibraryImport {
		t.Fatal("expected extendedLibraryImport true from controller")
	}
}

func TestHandlePatchSettings_BackendLog(t *testing.T) {
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

	logCtl := &stubBackendLogCtl{v: contracts.BackendLogSettingsDTO{LogLevel: "info"}}
	h := NewHandler(Deps{
		Cfg:                 config.Config{Player: config.PlayerConfig{HardwareDecode: true}},
		Logger:              zap.NewNop(),
		Store:               store,
		BackendLogCtl:       logCtl,
		OrganizeLibraryCtl:  stubOrganizeCtl{},
		AutoLibraryWatchCtl: stubAutoWatchCtl{},
		MetadataScrapeCtl:   stubMetadataCtl{},
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	dir := filepath.Join(root, "logs")
	body := fmt.Sprintf(`{"backendLog":{"logDir":%q,"logMaxAgeDays":3,"logLevel":"warn"}}`, dir)
	req, err := http.NewRequest(http.MethodPatch, srv.URL+"/api/settings", strings.NewReader(body))
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
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d body=%s", resp.StatusCode, string(b))
	}
	var dto contracts.SettingsDTO
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		t.Fatal(err)
	}
	if dto.BackendLog.LogDir != dir {
		t.Fatalf("BackendLog.LogDir = %q, want %q", dto.BackendLog.LogDir, dir)
	}
	if dto.BackendLog.LogMaxAgeDays != 3 {
		t.Fatalf("BackendLog.LogMaxAgeDays = %d, want 3", dto.BackendLog.LogMaxAgeDays)
	}
	if dto.BackendLog.LogLevel != "warn" {
		t.Fatalf("BackendLog.LogLevel = %q, want warn", dto.BackendLog.LogLevel)
	}
}

func TestHandlePatchSettings_Player(t *testing.T) {
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

	playerCtl := &stubPlayerSettingsCtl{v: contracts.PlayerSettingsDTO{
		HardwareDecode:      true,
		NativePlayerEnabled: true,
		NativePlayerCommand: "mpv",
		StreamPushEnabled:   true,
		FFmpegCommand:       "ffmpeg",
		SeekForwardStepSec:  10,
		SeekBackwardStepSec: 10,
	}}
	h := NewHandler(Deps{
		Cfg: config.Config{
			Player: config.PlayerConfig{
				HardwareDecode:      true,
				NativePlayerEnabled: true,
				NativePlayerCommand: "mpv",
				StreamPushEnabled:   true,
				FFmpegCommand:       "ffmpeg",
				SeekForwardStepSec:  10,
				SeekBackwardStepSec: 10,
			},
		},
		Logger:              zap.NewNop(),
		Store:               store,
		PlayerSettingsCtl:   playerCtl,
		OrganizeLibraryCtl:  stubOrganizeCtl{},
		AutoLibraryWatchCtl: stubAutoWatchCtl{},
		MetadataScrapeCtl:   stubMetadataCtl{},
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	body := `{"player":{"hardwareDecode":false,"nativePlayerEnabled":false,"nativePlayerCommand":"vlc","streamPushEnabled":true,"ffmpegCommand":"ffmpeg-custom","preferNativePlayer":true,"seekForwardStepSec":30,"seekBackwardStepSec":7}}`
	req, err := http.NewRequest(http.MethodPatch, srv.URL+"/api/settings", strings.NewReader(body))
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
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d body=%s", resp.StatusCode, string(b))
	}
	var dto contracts.SettingsDTO
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		t.Fatal(err)
	}
	if dto.Player.HardwareDecode {
		t.Fatal("expected hardwareDecode false")
	}
	if dto.Player.NativePlayerEnabled {
		t.Fatal("expected nativePlayerEnabled false")
	}
	if dto.Player.NativePlayerCommand != "vlc" {
		t.Fatalf("nativePlayerCommand = %q, want vlc", dto.Player.NativePlayerCommand)
	}
	if !dto.Player.StreamPushEnabled {
		t.Fatal("expected streamPushEnabled true")
	}
	if dto.Player.FFmpegCommand != "ffmpeg-custom" {
		t.Fatalf("ffmpegCommand = %q, want ffmpeg-custom", dto.Player.FFmpegCommand)
	}
	if !dto.Player.PreferNativePlayer {
		t.Fatal("expected preferNativePlayer true")
	}
	if dto.Player.SeekForwardStepSec != 30 {
		t.Fatalf("seekForwardStepSec = %d, want 30", dto.Player.SeekForwardStepSec)
	}
	if dto.Player.SeekBackwardStepSec != 7 {
		t.Fatalf("seekBackwardStepSec = %d, want 7", dto.Player.SeekBackwardStepSec)
	}
}

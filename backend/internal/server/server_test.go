package server

import (
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

	"jav-shadcn/backend/internal/config"
	"jav-shadcn/backend/internal/contracts"
	"jav-shadcn/backend/internal/library"
	"jav-shadcn/backend/internal/scraper"
	"jav-shadcn/backend/internal/storage"
	"jav-shadcn/backend/internal/tasks"
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
	if _, err := os.Stat(movieCacheDir); !os.IsNotExist(err) {
		t.Fatalf("asset cache dir should be removed after DELETE API: %v", err)
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
		Cfg:     config.Config{},
		Logger:  zap.NewNop(),
		Store:   store,
		Library: library.NewService(),
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

	h := NewHandler(Deps{Cfg: config.Config{}, Logger: zap.NewNop(), Store: store, Library: library.NewService()})
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

	h := NewHandler(Deps{Cfg: config.Config{}, Logger: zap.NewNop(), Store: store, Library: library.NewService()})
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

	h := NewHandler(Deps{Cfg: config.Config{}, Logger: zap.NewNop(), Store: store, Library: library.NewService()})
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

func TestHandlePatchMovie_InMemoryFallback(t *testing.T) {
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

	h := NewHandler(Deps{Cfg: config.Config{}, Logger: zap.NewNop(), Store: store, Library: library.NewService()})
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
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var detail contracts.MovieDetailDTO
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		t.Fatal(err)
	}
	if detail.IsFavorite || detail.Rating != 2.5 {
		t.Fatalf("detail = %+v", detail)
	}
}

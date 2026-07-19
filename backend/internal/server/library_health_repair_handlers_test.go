package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
	"curated-backend/internal/tasks"
)

type completingMetadataRefresher struct {
	tasks   *tasks.Manager
	mu      sync.Mutex
	started []string
	fail    map[string]bool
}

func (f *completingMetadataRefresher) StartMovieMetadataRefresh(_ context.Context, movieID string) (contracts.TaskDTO, error) {
	f.mu.Lock()
	f.started = append(f.started, movieID)
	f.mu.Unlock()
	task := f.tasks.Create("scrape.movie", map[string]any{"movieId": movieID})
	task = f.tasks.Start(task.TaskID, "scraping")
	if f.fail[movieID] {
		return f.tasks.Fail(task.TaskID, contracts.ErrorCodeScraperRun, "provider failed"), nil
	}
	return f.tasks.Complete(task.TaskID, "metadata saved"), nil
}

func (f *completingMetadataRefresher) StartMetadataRefreshForLibraryPaths(context.Context, []string) (contracts.MetadataRefreshQueuedDTO, error) {
	return contracts.MetadataRefreshQueuedDTO{}, nil
}

func (f *completingMetadataRefresher) startedMovieIDs() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.started...)
}

func TestLibraryHealthMetadataRepairRequiresConfirmationAndPersistsPerItemResults(t *testing.T) {
	root := t.TempDir()
	store, err := storage.NewSQLiteStore(filepath.Join(t.TempDir(), "health-repair-api.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	for _, code := range []string{"QUEUE-OK", "QUEUE-FAIL"} {
		path := filepath.Join(root, code+".mp4")
		if err := os.WriteFile(path, []byte("video"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
			TaskID: "scan-" + code, Path: path, FileName: filepath.Base(path), Number: code,
		}); err != nil {
			t.Fatal(err)
		}
	}
	taskManager := tasks.NewManager()
	refresher := &completingMetadataRefresher{tasks: taskManager, fail: map[string]bool{"queue-fail": true}}
	provider := &libraryHealthStorageProvider{items: []contracts.LibraryPathStorageStatusDTO{{
		LibraryPathID: "library-queue", Path: root, Title: "Queue Library",
		Status: contracts.LibraryPathStorageStatusOnline, Message: "online", CanRescan: true, CanImport: true,
	}}}
	h := NewHandler(Deps{
		RuntimeContext: context.Background(), Cfg: config.Default(), Logger: zap.NewNop(), Store: store,
		Tasks: taskManager, MovieMetadataRefresher: refresher, LibraryPathStorageStatusProvider: provider,
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	request := contracts.StartLibraryHealthRepairRequest{
		Action: "rescrape_metadata", Categories: []string{"metadata_missing"}, Limit: 2,
	}
	body, _ := json.Marshal(request)
	resp, err := http.Post(srv.URL+"/api/library/health/repairs", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	var appErr contracts.AppError
	if err := json.NewDecoder(resp.Body).Decode(&appErr); err != nil {
		_ = resp.Body.Close()
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusConflict || appErr.Code != contracts.ErrorCodeHealthRepairConfirmationRequired {
		t.Fatalf("unconfirmed response = %d %#v", resp.StatusCode, appErr)
	}
	if len(refresher.startedMovieIDs()) != 0 {
		t.Fatal("unconfirmed repair started child tasks")
	}

	request.Confirm = true
	body, _ = json.Marshal(request)
	resp, err = http.Post(srv.URL+"/api/library/health/repairs", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusAccepted {
		_ = resp.Body.Close()
		t.Fatalf("status = %d, want 202", resp.StatusCode)
	}
	var created contracts.LibraryHealthRepairDTO
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		_ = resp.Body.Close()
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if created.TotalItems != 2 || created.TaskID == "" || created.RepairID == "" {
		t.Fatalf("created = %#v", created)
	}

	finished := waitForLibraryHealthRepair(t, srv.URL, created.RepairID)
	if finished.Status != contracts.TaskPartialFailed || finished.CompletedItems != 2 || finished.SucceededItems != 1 || finished.FailedItems != 1 {
		t.Fatalf("finished = %#v", finished)
	}
	if len(finished.Items) != 2 || finished.Items[0].ChildTaskID == "" || finished.Items[1].ChildTaskID == "" {
		t.Fatalf("items = %#v", finished.Items)
	}
	statuses := map[string]string{}
	for _, item := range finished.Items {
		statuses[item.MovieID] = item.Status
	}
	if statuses["queue-ok"] != "succeeded" || statuses["queue-fail"] != "failed" {
		t.Fatalf("item statuses = %#v", statuses)
	}
	parent, ok := taskManager.Get(created.TaskID)
	if !ok || parent.Status != contracts.TaskPartialFailed {
		t.Fatalf("parent task = %#v, ok=%v", parent, ok)
	}
}

func TestLibraryHealthMetadataRepairFiltersFailedCategory(t *testing.T) {
	root := t.TempDir()
	store, err := storage.NewSQLiteStore(filepath.Join(t.TempDir(), "health-repair-filter.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	for _, code := range []string{"ONLY-MISSING", "FAILED-ONLY"} {
		path := filepath.Join(root, code+".mp4")
		if err := os.WriteFile(path, []byte("video"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
			TaskID: "scan-" + code, Path: path, FileName: filepath.Base(path), Number: code,
		}); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now().UTC()
	if err := store.StartMovieMetadataScrapeAttempt(ctx, "failed-only", "scrape-failed", now); err != nil {
		t.Fatal(err)
	}
	if err := store.FinishMovieMetadataScrapeAttempt(ctx, storage.MovieMetadataScrapeAttempt{
		MovieID: "failed-only", TaskID: "scrape-failed", Status: "failed", ErrorCode: contracts.ErrorCodeScraperRun,
		ErrorCategory: "connect_timeout", ErrorMessage: "timeout",
	}, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}

	taskManager := tasks.NewManager()
	refresher := &completingMetadataRefresher{tasks: taskManager, fail: map[string]bool{}}
	provider := &libraryHealthStorageProvider{items: []contracts.LibraryPathStorageStatusDTO{{
		LibraryPathID: "library-filter", Path: root, Title: "Filter Library",
		Status: contracts.LibraryPathStorageStatusOnline, Message: "online", CanRescan: true, CanImport: true,
	}}}
	h := NewHandler(Deps{
		RuntimeContext: context.Background(), Cfg: config.Default(), Logger: zap.NewNop(), Store: store,
		Tasks: taskManager, MovieMetadataRefresher: refresher, LibraryPathStorageStatusProvider: provider,
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	request := contracts.StartLibraryHealthRepairRequest{
		Action: "rescrape_metadata", Categories: []string{"metadata_failed"}, Limit: 10, Confirm: true,
	}
	body, _ := json.Marshal(request)
	resp, err := http.Post(srv.URL+"/api/library/health/repairs", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var created contracts.LibraryHealthRepairDTO
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	finished := waitForLibraryHealthRepair(t, srv.URL, created.RepairID)
	if finished.TotalItems != 1 || finished.Items[0].MovieID != "failed-only" {
		t.Fatalf("filtered repair = %#v", finished)
	}
	started := refresher.startedMovieIDs()
	if len(started) != 1 || started[0] != "failed-only" {
		t.Fatalf("started movie ids = %#v", started)
	}
}

func waitForLibraryHealthRepair(t *testing.T, baseURL, repairID string) contracts.LibraryHealthRepairDTO {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(baseURL + "/api/library/health/repairs/" + repairID)
		if err != nil {
			t.Fatal(err)
		}
		var dto contracts.LibraryHealthRepairDTO
		decodeErr := json.NewDecoder(resp.Body).Decode(&dto)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK || decodeErr != nil {
			t.Fatalf("get repair status=%d decode=%v", resp.StatusCode, decodeErr)
		}
		if isTerminalTaskStatus(dto.Status) {
			return dto
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("timed out waiting for library health repair")
	return contracts.LibraryHealthRepairDTO{}
}

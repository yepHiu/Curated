package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
	"curated-backend/internal/tasks"
)

func TestMovieImportUploadRecoversProgressAndCommitsAfterHandlerRestart(t *testing.T) {
	root := t.TempDir()
	store := newImportTestStore(t, root)
	libRoot := filepath.Join(root, "library")
	if err := os.MkdirAll(libRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	libraryPath, err := store.AddLibraryPath(context.Background(), libRoot, "library")
	if err != nil {
		t.Fatal(err)
	}

	firstTasks := tasks.NewManager()
	firstHandler := NewHandler(Deps{
		Cfg:                         config.Config{},
		Logger:                      zap.NewNop(),
		Store:                       store,
		Tasks:                       firstTasks,
		DefaultImportLibraryPathCtl: &stubDefaultImportLibraryPathCtl{id: libraryPath.ID},
	})
	firstServer := httptest.NewServer(firstHandler.Routes())
	created := createMovieImportUploadForTest(t, firstServer.URL, "folder/restart.mkv", 8)
	putMovieImportUploadChunkForTest(t, firstServer.URL, created.UploadID, created.Files[0].FileID, 0, 0, "half")
	firstServer.Close()

	persisted, err := store.GetMovieImportUploadSession(context.Background(), created.UploadID)
	if err != nil {
		t.Fatalf("GetMovieImportUploadSession: %v", err)
	}
	if persisted.BytesReceived != 4 || persisted.Files[0].BytesReceived != 4 || len(persisted.Files[0].Chunks) != 1 {
		t.Fatalf("persisted=%+v", persisted)
	}
	if info, err := os.Stat(persisted.Files[0].StagingPath); err != nil || info.Size() != 8 {
		t.Fatalf("preallocated staging size=%v err=%v", info, err)
	}

	restartedTasks := tasks.NewManager()
	restartedHandler := NewHandler(Deps{
		Cfg:                         config.Config{},
		Logger:                      zap.NewNop(),
		Store:                       store,
		Tasks:                       restartedTasks,
		DefaultImportLibraryPathCtl: &stubDefaultImportLibraryPathCtl{id: libraryPath.ID},
	})
	restartedServer := httptest.NewServer(restartedHandler.Routes())
	defer restartedServer.Close()

	status := getMovieImportUploadForTest(t, restartedServer.URL, created.UploadID)
	if status.BytesReceived != 4 || status.RecoveryStatus != movieImportUploadRecoveryReady || status.Task.TaskID != created.Task.TaskID {
		t.Fatalf("recovered status=%+v", status)
	}
	if status.Files[0].BytesReceived != 4 || status.Files[0].Complete {
		t.Fatalf("recovered file=%+v", status.Files[0])
	}
	if restoredTask, ok := restartedTasks.Get(created.Task.TaskID); !ok || restoredTask.Progress != 50 {
		t.Fatalf("restored task=%+v ok=%v", restoredTask, ok)
	}

	putMovieImportUploadChunkForTest(t, restartedServer.URL, created.UploadID, created.Files[0].FileID, 1, 4, "done")
	commitTask := commitMovieImportUploadForTest(t, restartedServer.URL, created.UploadID)
	if commitTask.Status != contracts.TaskCompleted {
		t.Fatalf("commit task=%+v", commitTask)
	}
	content, err := os.ReadFile(filepath.Join(libRoot, "folder", "restart.mkv"))
	if err != nil || string(content) != "halfdone" {
		t.Fatalf("final content=%q err=%v", content, err)
	}
	if _, err := store.GetMovieImportUploadSession(context.Background(), created.UploadID); !errors.Is(err, storage.ErrMovieImportUploadNotFound) {
		t.Fatalf("terminal upload persisted after cleanup: %v", err)
	}
	audits, err := store.ListMovieImportUploadCleanupAudits(context.Background(), 10)
	if err != nil || len(audits) != 1 || audits[0].Outcome != "removed" || audits[0].Reason != "session_committed" {
		t.Fatalf("cleanup audits=%+v err=%v", audits, err)
	}
}

func TestMovieImportUploadResumesCommitAfterFileMoveBeforeDatabaseMarker(t *testing.T) {
	root := t.TempDir()
	store := newImportTestStore(t, root)
	libRoot := filepath.Join(root, "library")
	if err := os.MkdirAll(libRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	libraryPath, err := store.AddLibraryPath(context.Background(), libRoot, "library")
	if err != nil {
		t.Fatal(err)
	}
	firstHandler := NewHandler(Deps{
		Cfg:                         config.Config{},
		Logger:                      zap.NewNop(),
		Store:                       store,
		Tasks:                       tasks.NewManager(),
		DefaultImportLibraryPathCtl: &stubDefaultImportLibraryPathCtl{id: libraryPath.ID},
	})
	firstServer := httptest.NewServer(firstHandler.Routes())
	created := createMovieImportUploadForTest(t, firstServer.URL, "commit-restart.mp4", 4)
	putMovieImportUploadChunkForTest(t, firstServer.URL, created.UploadID, created.Files[0].FileID, 0, 0, "data")
	firstServer.Close()

	now := time.Now().UTC()
	if err := store.TransitionMovieImportUploadState(context.Background(), created.UploadID,
		[]string{movieImportUploadStateUploading}, movieImportUploadStateCommitting,
		now, formatUploadRuntimeTime(now.Add(time.Hour)), "", "", ""); err != nil {
		t.Fatalf("TransitionMovieImportUploadState: %v", err)
	}
	record, err := store.GetMovieImportUploadSession(context.Background(), created.UploadID)
	if err != nil {
		t.Fatal(err)
	}
	if err := commitMovieImportStagingFile(record.Files[0].StagingPath, record.Files[0].FinalPath); err != nil {
		t.Fatalf("simulate file commit: %v", err)
	}

	restartedHandler := NewHandler(Deps{
		Cfg:                         config.Config{},
		Logger:                      zap.NewNop(),
		Store:                       store,
		Tasks:                       tasks.NewManager(),
		DefaultImportLibraryPathCtl: &stubDefaultImportLibraryPathCtl{id: libraryPath.ID},
	})
	restartedServer := httptest.NewServer(restartedHandler.Routes())
	defer restartedServer.Close()
	status := getMovieImportUploadForTest(t, restartedServer.URL, created.UploadID)
	if status.State != movieImportUploadStateCommitting || status.Files[0].State != storage.MovieImportUploadFileStateCommitted {
		t.Fatalf("reconciled status=%+v", status)
	}
	commitTask := commitMovieImportUploadForTest(t, restartedServer.URL, created.UploadID)
	if commitTask.Status != contracts.TaskCompleted {
		t.Fatalf("commit task=%+v", commitTask)
	}
	content, err := os.ReadFile(record.Files[0].FinalPath)
	if err != nil || string(content) != "data" {
		t.Fatalf("final content=%q err=%v", content, err)
	}
}

func TestMovieImportUploadRecoveryDiagnosesInvalidStagingBeforeJanitorCleanup(t *testing.T) {
	root := t.TempDir()
	store := newImportTestStore(t, root)
	libRoot := filepath.Join(root, "library")
	if err := os.MkdirAll(libRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	libraryPath, err := store.AddLibraryPath(context.Background(), libRoot, "library")
	if err != nil {
		t.Fatal(err)
	}
	firstHandler := NewHandler(Deps{
		Cfg:                         config.Config{},
		Logger:                      zap.NewNop(),
		Store:                       store,
		Tasks:                       tasks.NewManager(),
		DefaultImportLibraryPathCtl: &stubDefaultImportLibraryPathCtl{id: libraryPath.ID},
	})
	firstServer := httptest.NewServer(firstHandler.Routes())
	created := createMovieImportUploadForTest(t, firstServer.URL, "invalid.mkv", 8)
	firstServer.Close()
	record, err := store.GetMovieImportUploadSession(context.Background(), created.UploadID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(record.Files[0].StagingPath, 1); err != nil {
		t.Fatal(err)
	}

	restartedTasks := tasks.NewManager()
	restartedHandler := NewHandler(Deps{
		Cfg:                         config.Config{},
		Logger:                      zap.NewNop(),
		Store:                       store,
		Tasks:                       restartedTasks,
		DefaultImportLibraryPathCtl: &stubDefaultImportLibraryPathCtl{id: libraryPath.ID},
	})
	status := restartedHandler.importUploads.sessions[created.UploadID]
	if status == nil || status.state != movieImportUploadStateUnrecoverable || !strings.Contains(status.recoveryError, "declared size") {
		t.Fatalf("recovered invalid session=%+v", status)
	}
	persisted, err := store.GetMovieImportUploadSession(context.Background(), created.UploadID)
	if err != nil || persisted.State != movieImportUploadStateUnrecoverable || persisted.DiagnosticCode != "staging_validation_failed" {
		t.Fatalf("persisted invalid session=%+v err=%v", persisted, err)
	}
	cleanupAt, err := parseUploadRuntimeTime(persisted.CleanupAfter)
	if err != nil {
		t.Fatal(err)
	}
	restartedHandler.importUploads.now = func() time.Time { return cleanupAt.Add(time.Second) }

	if err := restartedHandler.importUploads.runJanitor(context.Background()); err != nil {
		t.Fatalf("runJanitor: %v", err)
	}
	if _, err := os.Stat(record.StagingDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("invalid staging still exists: %v", err)
	}
	if _, err := store.GetMovieImportUploadSession(context.Background(), created.UploadID); !errors.Is(err, storage.ErrMovieImportUploadNotFound) {
		t.Fatalf("invalid session still exists: %v", err)
	}
	audits, err := store.ListMovieImportUploadCleanupAudits(context.Background(), 10)
	if err != nil || len(audits) != 1 || audits[0].Outcome != "removed" || !strings.Contains(audits[0].Diagnostic, "declared size") {
		t.Fatalf("audits=%+v err=%v", audits, err)
	}
}

func TestMovieImportUploadRecoveryDefersExpiredCleanupWhileTargetStorageIsOffline(t *testing.T) {
	root := t.TempDir()
	store := newImportTestStore(t, root)
	libRoot := filepath.Join(root, "library")
	offlineRoot := filepath.Join(root, "library-offline")
	if err := os.MkdirAll(libRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	libraryPath, err := store.AddLibraryPath(context.Background(), libRoot, "library")
	if err != nil {
		t.Fatal(err)
	}
	firstHandler := NewHandler(Deps{
		Cfg:                         config.Config{},
		Logger:                      zap.NewNop(),
		Store:                       store,
		Tasks:                       tasks.NewManager(),
		DefaultImportLibraryPathCtl: &stubDefaultImportLibraryPathCtl{id: libraryPath.ID},
	})
	firstServer := httptest.NewServer(firstHandler.Routes())
	created := createMovieImportUploadForTest(t, firstServer.URL, "offline.mkv", 4)
	firstServer.Close()
	if err := os.Rename(libRoot, offlineRoot); err != nil {
		t.Fatal(err)
	}

	restartedHandler := NewHandler(Deps{
		Cfg:                         config.Config{},
		Logger:                      zap.NewNop(),
		Store:                       store,
		Tasks:                       tasks.NewManager(),
		DefaultImportLibraryPathCtl: &stubDefaultImportLibraryPathCtl{id: libraryPath.ID},
	})
	restartedServer := httptest.NewServer(restartedHandler.Routes())
	status := getMovieImportUploadForTest(t, restartedServer.URL, created.UploadID)
	if status.RecoveryStatus != movieImportUploadRecoveryUnavailable || status.RecoveryError == "" {
		t.Fatalf("offline status=%+v", status)
	}
	if err := os.Rename(offlineRoot, libRoot); err != nil {
		t.Fatal(err)
	}
	status = getMovieImportUploadForTest(t, restartedServer.URL, created.UploadID)
	if status.RecoveryStatus != movieImportUploadRecoveryReady {
		t.Fatalf("reattached storage did not resume session: %+v", status)
	}
	if err := os.Rename(libRoot, offlineRoot); err != nil {
		t.Fatal(err)
	}
	restartedServer.Close()
	record, err := store.GetMovieImportUploadSession(context.Background(), created.UploadID)
	if err != nil {
		t.Fatal(err)
	}
	future, err := parseUploadRuntimeTime(record.ExpiresAt)
	if err != nil {
		t.Fatal(err)
	}
	future = future.Add(time.Minute)
	restartedHandler.importUploads.now = func() time.Time { return future }
	if err := restartedHandler.importUploads.runJanitor(context.Background()); err != nil {
		t.Fatalf("runJanitor while offline: %v", err)
	}
	persisted, err := store.GetMovieImportUploadSession(context.Background(), created.UploadID)
	if err != nil || persisted.State != movieImportUploadStateExpired {
		t.Fatalf("offline expired session=%+v err=%v", persisted, err)
	}
	audits, err := store.ListMovieImportUploadCleanupAudits(context.Background(), 10)
	if err != nil || len(audits) != 0 {
		t.Fatalf("offline cleanup should not claim removal: audits=%+v err=%v", audits, err)
	}
	if err := os.Rename(offlineRoot, libRoot); err != nil {
		t.Fatal(err)
	}
	restartedHandler.importUploads.now = func() time.Time { return future.Add(2 * time.Minute) }
	if err := restartedHandler.importUploads.runJanitor(context.Background()); err != nil {
		t.Fatalf("runJanitor after storage returns: %v", err)
	}
	if _, err := store.GetMovieImportUploadSession(context.Background(), created.UploadID); !errors.Is(err, storage.ErrMovieImportUploadNotFound) {
		t.Fatalf("expired session not cleaned after storage returned: %v", err)
	}
	audits, err = store.ListMovieImportUploadCleanupAudits(context.Background(), 10)
	if err != nil || len(audits) != 1 || audits[0].Outcome != "removed" {
		t.Fatalf("cleanup audits=%+v err=%v", audits, err)
	}
}

func TestMovieImportUploadRejectsBodyBeyondManifestWithoutExtendingStagingFile(t *testing.T) {
	root := t.TempDir()
	store := newImportTestStore(t, root)
	libRoot := filepath.Join(root, "library")
	if err := os.MkdirAll(libRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	libraryPath, err := store.AddLibraryPath(context.Background(), libRoot, "library")
	if err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(Deps{
		Cfg:                         config.Config{},
		Logger:                      zap.NewNop(),
		Store:                       store,
		Tasks:                       tasks.NewManager(),
		DefaultImportLibraryPathCtl: &stubDefaultImportLibraryPathCtl{id: libraryPath.ID},
	})
	server := httptest.NewServer(handler.Routes())
	defer server.Close()
	created := createMovieImportUploadForTest(t, server.URL, "bounded.mp4", 4)
	url := server.URL + "/api/import/movies/uploads/" + created.UploadID + "/files/" + created.Files[0].FileID + "/chunks/0"
	req, err := http.NewRequest(http.MethodPut, url, strings.NewReader("12345"))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("X-Curated-Offset", "0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("oversized chunk status=%d body=%s", resp.StatusCode, data)
	}
	record, err := store.GetMovieImportUploadSession(context.Background(), created.UploadID)
	if err != nil {
		t.Fatal(err)
	}
	if record.BytesReceived != 0 || record.Files[0].BytesReceived != 0 || len(record.Files[0].Chunks) != 0 {
		t.Fatalf("oversized chunk was persisted: %+v", record)
	}
	info, err := os.Stat(record.Files[0].StagingPath)
	if err != nil || info.Size() != 4 {
		t.Fatalf("staging file size=%v err=%v", info, err)
	}
}

func TestMovieImportUploadAbortPersistsTerminalStateBeforeAuditedCleanup(t *testing.T) {
	root := t.TempDir()
	store := newImportTestStore(t, root)
	libRoot := filepath.Join(root, "library")
	if err := os.MkdirAll(libRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	libraryPath, err := store.AddLibraryPath(context.Background(), libRoot, "library")
	if err != nil {
		t.Fatal(err)
	}
	taskManager := tasks.NewManager()
	handler := NewHandler(Deps{
		Cfg:                         config.Config{},
		Logger:                      zap.NewNop(),
		Store:                       store,
		Tasks:                       taskManager,
		DefaultImportLibraryPathCtl: &stubDefaultImportLibraryPathCtl{id: libraryPath.ID},
	})
	server := httptest.NewServer(handler.Routes())
	defer server.Close()
	created := createMovieImportUploadForTest(t, server.URL, "abort.mp4", 4)
	record, err := store.GetMovieImportUploadSession(context.Background(), created.UploadID)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodDelete, server.URL+"/api/import/movies/uploads/"+created.UploadID, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("abort status=%d", resp.StatusCode)
	}
	if _, err := store.GetMovieImportUploadSession(context.Background(), created.UploadID); !errors.Is(err, storage.ErrMovieImportUploadNotFound) {
		t.Fatalf("aborted session still exists: %v", err)
	}
	if _, err := os.Stat(record.StagingDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("aborted staging still exists: %v", err)
	}
	task, ok := taskManager.Get(created.Task.TaskID)
	if !ok || task.Status != contracts.TaskFailed || task.ErrorCode != contracts.ErrorCodeImportCancelled {
		t.Fatalf("abort task=%+v ok=%v", task, ok)
	}
	audits, err := store.ListMovieImportUploadCleanupAudits(context.Background(), 10)
	if err != nil || len(audits) != 1 || audits[0].Reason != "session_aborted" || audits[0].PriorState != movieImportUploadStateAborted {
		t.Fatalf("abort audits=%+v err=%v", audits, err)
	}
}

func TestMovieImportUploadManifestIsStrictAndRejectsDuplicateDestinations(t *testing.T) {
	root := t.TempDir()
	store := newImportTestStore(t, root)
	libRoot := filepath.Join(root, "library")
	if err := os.MkdirAll(libRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	libraryPath, err := store.AddLibraryPath(context.Background(), libRoot, "library")
	if err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(Deps{
		Cfg:                         config.Config{},
		Logger:                      zap.NewNop(),
		Store:                       store,
		Tasks:                       tasks.NewManager(),
		DefaultImportLibraryPathCtl: &stubDefaultImportLibraryPathCtl{id: libraryPath.ID},
	})
	server := httptest.NewServer(handler.Routes())
	defer server.Close()
	for _, body := range []string{
		`{"files":[{"relativePath":"duplicate.mp4","size":4},{"relativePath":"duplicate.mp4","size":4}]}`,
		`{"files":[{"relativePath":"movie.mp4","size":4}],"unknown":true}`,
		`{"files":[{"relativePath":"movie.mp4","size":4}]} {}`,
	} {
		req, err := http.NewRequest(http.MethodPost, server.URL+"/api/import/movies/uploads", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		data, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("manifest %s status=%d body=%s", body, resp.StatusCode, data)
		}
	}
	ids, err := store.ListMovieImportUploadSessionIDs(context.Background())
	if err != nil || len(ids) != 0 {
		t.Fatalf("rejected manifests persisted sessions=%v err=%v", ids, err)
	}
}

func TestMovieImportUploadJanitorOnlyRemovesStrictOldOrphanDirectories(t *testing.T) {
	root := t.TempDir()
	store := newImportTestStore(t, root)
	libRoot := filepath.Join(root, "library")
	if err := os.MkdirAll(libRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddLibraryPath(context.Background(), libRoot, "library"); err != nil {
		t.Fatal(err)
	}
	stagingRoot := filepath.Join(libRoot, movieImportUploadStagingDirName)
	oldOrphan := filepath.Join(stagingRoot, "upload_0123456789abcdef")
	freshOrphan := filepath.Join(stagingRoot, "upload_fedcba9876543210")
	invalidName := filepath.Join(stagingRoot, "not-an-upload")
	unrelatedHidden := filepath.Join(libRoot, ".unrelated", "upload_1111111111111111")
	for _, path := range []string{oldOrphan, freshOrphan, invalidName, unrelatedHidden} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Date(2026, 7, 20, 8, 0, 0, 0, time.UTC)
	if err := os.Chtimes(oldOrphan, now.Add(-2*time.Hour), now.Add(-2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(freshOrphan, now, now); err != nil {
		t.Fatal(err)
	}
	manager := newMovieImportUploadSessionStore(store, tasks.NewManager(), zap.NewNop())
	manager.now = func() time.Time { return now }
	manager.orphanGrace = time.Hour
	if err := manager.runJanitor(context.Background()); err != nil {
		t.Fatalf("runJanitor: %v", err)
	}
	if _, err := os.Stat(oldOrphan); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("old orphan still exists: %v", err)
	}
	for _, path := range []string{freshOrphan, invalidName, unrelatedHidden} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("janitor removed out-of-scope path %s: %v", path, err)
		}
	}
	audits, err := store.ListMovieImportUploadCleanupAudits(context.Background(), 10)
	if err != nil || len(audits) != 1 || audits[0].UploadID != "upload_0123456789abcdef" || audits[0].Reason != "orphan_staging" {
		t.Fatalf("audits=%+v err=%v", audits, err)
	}
}

func createMovieImportUploadForTest(t *testing.T, baseURL, relativePath string, size int64) contracts.MovieImportUploadDTO {
	t.Helper()
	body, err := json.Marshal(contracts.CreateMovieImportUploadRequest{Files: []contracts.MovieImportUploadFileManifest{{
		RelativePath: relativePath,
		Size:         size,
	}}})
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/import/movies/uploads", bytes.NewReader(body))
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
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("create upload status=%d body=%s", resp.StatusCode, data)
	}
	var upload contracts.MovieImportUploadDTO
	if err := json.NewDecoder(resp.Body).Decode(&upload); err != nil {
		t.Fatal(err)
	}
	return upload
}

func putMovieImportUploadChunkForTest(
	t *testing.T,
	baseURL string,
	uploadID string,
	fileID string,
	chunkIndex int,
	offset int64,
	content string,
) contracts.MovieImportUploadDTO {
	t.Helper()
	url := baseURL + "/api/import/movies/uploads/" + uploadID + "/files/" + fileID + "/chunks/" + strconv.Itoa(chunkIndex)
	req, err := http.NewRequest(http.MethodPut, url, strings.NewReader(content))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("X-Curated-Offset", formatInt64ForUploadTest(offset))
	req.Header.Set("X-Curated-Chunk-Size", formatInt64ForUploadTest(int64(len(content))))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("put chunk status=%d body=%s", resp.StatusCode, data)
	}
	var upload contracts.MovieImportUploadDTO
	if err := json.NewDecoder(resp.Body).Decode(&upload); err != nil {
		t.Fatal(err)
	}
	return upload
}

func getMovieImportUploadForTest(t *testing.T, baseURL, uploadID string) contracts.MovieImportUploadDTO {
	t.Helper()
	resp, err := http.Get(baseURL + "/api/import/movies/uploads/" + uploadID)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("get upload status=%d body=%s", resp.StatusCode, data)
	}
	var upload contracts.MovieImportUploadDTO
	if err := json.NewDecoder(resp.Body).Decode(&upload); err != nil {
		t.Fatal(err)
	}
	return upload
}

func commitMovieImportUploadForTest(t *testing.T, baseURL, uploadID string) contracts.TaskDTO {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/import/movies/uploads/"+uploadID+"/commit", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("commit upload status=%d body=%s", resp.StatusCode, data)
	}
	var task contracts.TaskDTO
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		t.Fatal(err)
	}
	return task
}

func formatInt64ForUploadTest(value int64) string {
	return strconv.FormatInt(value, 10)
}

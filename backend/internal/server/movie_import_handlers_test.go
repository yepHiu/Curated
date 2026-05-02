package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
	"curated-backend/internal/tasks"
)

type stubDefaultImportLibraryPathCtl struct {
	id  string
	err error
}

func (s *stubDefaultImportLibraryPathCtl) DefaultImportLibraryPathID() string {
	return s.id
}

func (s *stubDefaultImportLibraryPathCtl) SetDefaultImportLibraryPathID(id string) error {
	if s.err != nil {
		return s.err
	}
	s.id = id
	return nil
}

type recordingScanStarter struct {
	paths [][]string
}

func (s *recordingScanStarter) StartScan(ctx context.Context, paths []string) (contracts.TaskDTO, error) {
	_ = ctx
	s.paths = append(s.paths, append([]string(nil), paths...))
	return contracts.TaskDTO{
		TaskID:   "scan-after-import",
		Type:     "scan.library",
		Status:   contracts.TaskRunning,
		Progress: 0,
	}, nil
}

func newImportTestStore(t *testing.T, root string) *storage.SQLiteStore {
	t.Helper()
	store, err := storage.NewSQLiteStore(filepath.Join(root, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	return store
}

func multipartMoviesBody(t *testing.T, files map[string]string) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for name, content := range files {
		part, err := writer.CreateFormFile("files", name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(part, content); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return body, writer.FormDataContentType()
}

func TestHandleGetSettings_DefaultImportLibraryPathFromController(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	store := newImportTestStore(t, root)
	libRoot := filepath.Join(root, "library")
	if err := os.MkdirAll(libRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	path, err := store.AddLibraryPath(context.Background(), libRoot, "library")
	if err != nil {
		t.Fatal(err)
	}

	h := NewHandler(Deps{
		Cfg:                         config.Config{},
		Logger:                      zap.NewNop(),
		Store:                       store,
		DefaultImportLibraryPathCtl: &stubDefaultImportLibraryPathCtl{id: path.ID},
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/api/settings")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var dto contracts.SettingsDTO
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		t.Fatal(err)
	}
	if dto.DefaultImportLibraryPathID != path.ID {
		t.Fatalf("defaultImportLibraryPathId = %q, want %q", dto.DefaultImportLibraryPathID, path.ID)
	}
}

func TestHandlePatchSettings_DefaultImportLibraryPath(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	store := newImportTestStore(t, root)
	libRoot := filepath.Join(root, "library")
	if err := os.MkdirAll(libRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	path, err := store.AddLibraryPath(context.Background(), libRoot, "library")
	if err != nil {
		t.Fatal(err)
	}
	ctl := &stubDefaultImportLibraryPathCtl{}
	h := NewHandler(Deps{
		Cfg:                         config.Config{},
		Logger:                      zap.NewNop(),
		Store:                       store,
		DefaultImportLibraryPathCtl: ctl,
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodPatch, srv.URL+"/api/settings", bytes.NewBufferString(`{"defaultImportLibraryPathId":"`+path.ID+`"}`))
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
	if ctl.id != path.ID || dto.DefaultImportLibraryPathID != path.ID {
		t.Fatalf("default import path not saved: ctl=%q dto=%q", ctl.id, dto.DefaultImportLibraryPathID)
	}
}

func TestHandlePatchSettings_DefaultImportLibraryPathRejectsUnknownID(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	store := newImportTestStore(t, root)
	ctl := &stubDefaultImportLibraryPathCtl{}
	h := NewHandler(Deps{
		Cfg:                         config.Config{},
		Logger:                      zap.NewNop(),
		Store:                       store,
		DefaultImportLibraryPathCtl: ctl,
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodPatch, srv.URL+"/api/settings", bytes.NewBufferString(`{"defaultImportLibraryPathId":"missing"}`))
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
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d body=%s", resp.StatusCode, string(b))
	}
	if ctl.id != "" {
		t.Fatalf("default import path should not be saved, got %q", ctl.id)
	}
}

func TestHandleImportMovies_RejectsMissingDefaultTarget(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	store := newImportTestStore(t, root)
	tm := tasks.NewManager()
	h := NewHandler(Deps{
		Cfg:                         config.Config{},
		Logger:                      zap.NewNop(),
		Store:                       store,
		Tasks:                       tm,
		DefaultImportLibraryPathCtl: &stubDefaultImportLibraryPathCtl{},
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	body, contentType := multipartMoviesBody(t, map[string]string{"IMP-001.mp4": "video"})
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/import/movies", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", contentType)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d body=%s", resp.StatusCode, string(b))
	}
	var appErr contracts.AppError
	if err := json.NewDecoder(resp.Body).Decode(&appErr); err != nil {
		t.Fatal(err)
	}
	if appErr.Code != contracts.ErrorCodeImportTargetNotConfigured {
		t.Fatalf("code = %q, want %q", appErr.Code, contracts.ErrorCodeImportTargetNotConfigured)
	}
}

func TestHandleImportMovies_WritesUploadedFileToDefaultLibraryPath(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	store := newImportTestStore(t, root)
	libRoot := filepath.Join(root, "library")
	if err := os.MkdirAll(libRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	path, err := store.AddLibraryPath(context.Background(), libRoot, "library")
	if err != nil {
		t.Fatal(err)
	}
	tm := tasks.NewManager()
	scans := &recordingScanStarter{}
	h := NewHandler(Deps{
		Cfg:                         config.Config{},
		Logger:                      zap.NewNop(),
		Store:                       store,
		Tasks:                       tm,
		ScanStarter:                 scans,
		DefaultImportLibraryPathCtl: &stubDefaultImportLibraryPathCtl{id: path.ID},
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	body, contentType := multipartMoviesBody(t, map[string]string{"IMP-001.mp4": "fake-mp4"})
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/import/movies", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", contentType)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d body=%s", resp.StatusCode, string(b))
	}
	var task contracts.TaskDTO
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		t.Fatal(err)
	}
	if task.Type != contracts.TaskTypeImportMovies {
		t.Fatalf("task type = %q, want %q", task.Type, contracts.TaskTypeImportMovies)
	}
	if task.Status != contracts.TaskCompleted {
		t.Fatalf("task status = %q, want completed", task.Status)
	}
	got, err := os.ReadFile(filepath.Join(libRoot, "IMP-001.mp4"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "fake-mp4" {
		t.Fatalf("copied content = %q", string(got))
	}
	if len(scans.paths) != 1 || len(scans.paths[0]) != 1 || scans.paths[0][0] != libRoot {
		t.Fatalf("scan paths = %#v, want [[%q]]", scans.paths, libRoot)
	}
}

func TestHandleImportMovies_PartialFailureOnConflict(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	store := newImportTestStore(t, root)
	libRoot := filepath.Join(root, "library")
	if err := os.MkdirAll(libRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(libRoot, "IMP-EXISTS.mp4"), []byte("already"), 0o644); err != nil {
		t.Fatal(err)
	}
	path, err := store.AddLibraryPath(context.Background(), libRoot, "library")
	if err != nil {
		t.Fatal(err)
	}
	tm := tasks.NewManager()
	h := NewHandler(Deps{
		Cfg:                         config.Config{},
		Logger:                      zap.NewNop(),
		Store:                       store,
		Tasks:                       tm,
		ScanStarter:                 &recordingScanStarter{},
		DefaultImportLibraryPathCtl: &stubDefaultImportLibraryPathCtl{id: path.ID},
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	body, contentType := multipartMoviesBody(t, map[string]string{
		"IMP-OK.mp4":     "ok",
		"IMP-EXISTS.mp4": "new-content",
	})
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/import/movies", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", contentType)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d body=%s", resp.StatusCode, string(b))
	}
	var task contracts.TaskDTO
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		t.Fatal(err)
	}
	if task.Status != contracts.TaskPartialFailed {
		t.Fatalf("task status = %q, want partial_failed", task.Status)
	}
	if task.ErrorCode != contracts.ErrorCodeImportCopyFailed {
		t.Fatalf("error code = %q, want %q", task.ErrorCode, contracts.ErrorCodeImportCopyFailed)
	}
	if _, err := os.Stat(filepath.Join(libRoot, "IMP-OK.mp4")); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(filepath.Join(libRoot, "IMP-EXISTS.mp4")); err != nil || string(got) != "already" {
		t.Fatalf("conflict file overwritten or unreadable: content=%q err=%v", string(got), err)
	}
	if task.Metadata["completedFiles"] != float64(1) && task.Metadata["completedFiles"] != 1 {
		t.Fatalf("completedFiles metadata = %#v, want 1", task.Metadata["completedFiles"])
	}
}

func TestHandleImportMovieUploadSession_CommitsChunkToDefaultLibraryPath(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	store := newImportTestStore(t, root)
	libRoot := filepath.Join(root, "library")
	if err := os.MkdirAll(libRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	path, err := store.AddLibraryPath(context.Background(), libRoot, "library")
	if err != nil {
		t.Fatal(err)
	}
	tm := tasks.NewManager()
	scans := &recordingScanStarter{}
	h := NewHandler(Deps{
		Cfg:                         config.Config{},
		Logger:                      zap.NewNop(),
		Store:                       store,
		Tasks:                       tm,
		ScanStarter:                 scans,
		DefaultImportLibraryPathCtl: &stubDefaultImportLibraryPathCtl{id: path.ID},
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	createBody := bytes.NewBufferString(`{"files":[{"relativePath":"folder/IMP-CHUNK.mp4","size":8,"lastModified":1234}]}`)
	createReq, err := http.NewRequest(http.MethodPost, srv.URL+"/api/import/movies/uploads", createBody)
	if err != nil {
		t.Fatal(err)
	}
	createReq.Header.Set("Content-Type", "application/json")
	createResp, err := http.DefaultClient.Do(createReq)
	if err != nil {
		t.Fatal(err)
	}
	defer createResp.Body.Close()
	if createResp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(createResp.Body)
		t.Fatalf("create status = %d body=%s, want 201", createResp.StatusCode, string(b))
	}
	var created struct {
		UploadID string `json:"uploadId"`
		Files    []struct {
			FileID       string `json:"fileId"`
			RelativePath string `json:"relativePath"`
			Size         int64  `json:"size"`
		} `json:"files"`
		Task contracts.TaskDTO `json:"task"`
	}
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.UploadID == "" || len(created.Files) != 1 || created.Files[0].FileID == "" {
		t.Fatalf("created upload response missing ids: %#v", created)
	}
	if created.Task.Type != contracts.TaskTypeImportMovies {
		t.Fatalf("task type = %q, want %q", created.Task.Type, contracts.TaskTypeImportMovies)
	}

	chunkReq, err := http.NewRequest(
		http.MethodPut,
		srv.URL+"/api/import/movies/uploads/"+created.UploadID+"/files/"+created.Files[0].FileID+"/chunks/0",
		bytes.NewBufferString("fake-mp4"),
	)
	if err != nil {
		t.Fatal(err)
	}
	chunkReq.Header.Set("Content-Type", "application/octet-stream")
	chunkReq.Header.Set("X-Curated-Offset", "0")
	chunkReq.Header.Set("X-Curated-Chunk-Size", "8")
	chunkResp, err := http.DefaultClient.Do(chunkReq)
	if err != nil {
		t.Fatal(err)
	}
	defer chunkResp.Body.Close()
	if chunkResp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(chunkResp.Body)
		t.Fatalf("chunk status = %d body=%s, want 200", chunkResp.StatusCode, string(b))
	}

	statusResp, err := http.Get(srv.URL + "/api/import/movies/uploads/" + created.UploadID)
	if err != nil {
		t.Fatal(err)
	}
	defer statusResp.Body.Close()
	if statusResp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(statusResp.Body)
		t.Fatalf("status status = %d body=%s, want 200", statusResp.StatusCode, string(b))
	}
	var status struct {
		UploadID      string `json:"uploadId"`
		BytesReceived int64  `json:"bytesReceived"`
		Files         []struct {
			FileID        string `json:"fileId"`
			BytesReceived int64  `json:"bytesReceived"`
			Complete      bool   `json:"complete"`
		} `json:"files"`
	}
	if err := json.NewDecoder(statusResp.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	if status.UploadID != created.UploadID || status.BytesReceived != 8 || len(status.Files) != 1 || !status.Files[0].Complete {
		t.Fatalf("upload status = %#v, want 8 complete bytes", status)
	}

	commitReq, err := http.NewRequest(http.MethodPost, srv.URL+"/api/import/movies/uploads/"+created.UploadID+"/commit", nil)
	if err != nil {
		t.Fatal(err)
	}
	commitResp, err := http.DefaultClient.Do(commitReq)
	if err != nil {
		t.Fatal(err)
	}
	defer commitResp.Body.Close()
	if commitResp.StatusCode != http.StatusAccepted {
		b, _ := io.ReadAll(commitResp.Body)
		t.Fatalf("commit status = %d body=%s, want 202", commitResp.StatusCode, string(b))
	}
	var task contracts.TaskDTO
	if err := json.NewDecoder(commitResp.Body).Decode(&task); err != nil {
		t.Fatal(err)
	}
	if task.Status != contracts.TaskCompleted {
		t.Fatalf("commit task status = %q, want completed", task.Status)
	}
	got, err := os.ReadFile(filepath.Join(libRoot, "folder", "IMP-CHUNK.mp4"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "fake-mp4" {
		t.Fatalf("committed content = %q", string(got))
	}
	if _, err := os.Stat(filepath.Join(libRoot, ".curated-import", created.UploadID)); !os.IsNotExist(err) {
		t.Fatalf("staging upload dir still exists or stat failed unexpectedly: %v", err)
	}
	if len(scans.paths) != 1 || len(scans.paths[0]) != 1 || scans.paths[0][0] != libRoot {
		t.Fatalf("scan paths = %#v, want [[%q]]", scans.paths, libRoot)
	}
}

func TestHandleImportMovieUploadSession_CommitDoesNotOverwriteLateConflict(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	store := newImportTestStore(t, root)
	libRoot := filepath.Join(root, "library")
	if err := os.MkdirAll(libRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	path, err := store.AddLibraryPath(context.Background(), libRoot, "library")
	if err != nil {
		t.Fatal(err)
	}
	tm := tasks.NewManager()
	h := NewHandler(Deps{
		Cfg:                         config.Config{},
		Logger:                      zap.NewNop(),
		Store:                       store,
		Tasks:                       tm,
		DefaultImportLibraryPathCtl: &stubDefaultImportLibraryPathCtl{id: path.ID},
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	createReq, err := http.NewRequest(
		http.MethodPost,
		srv.URL+"/api/import/movies/uploads",
		bytes.NewBufferString(`{"files":[{"relativePath":"IMP-LATE.mp4","size":4}]}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	createReq.Header.Set("Content-Type", "application/json")
	createResp, err := http.DefaultClient.Do(createReq)
	if err != nil {
		t.Fatal(err)
	}
	defer createResp.Body.Close()
	if createResp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(createResp.Body)
		t.Fatalf("create status = %d body=%s, want 201", createResp.StatusCode, string(b))
	}
	var created struct {
		UploadID string `json:"uploadId"`
		Files    []struct {
			FileID string `json:"fileId"`
		} `json:"files"`
	}
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if len(created.Files) != 1 {
		t.Fatalf("created files = %#v, want one file", created.Files)
	}

	chunkReq, err := http.NewRequest(
		http.MethodPut,
		srv.URL+"/api/import/movies/uploads/"+created.UploadID+"/files/"+created.Files[0].FileID+"/chunks/0",
		bytes.NewBufferString("new!"),
	)
	if err != nil {
		t.Fatal(err)
	}
	chunkReq.Header.Set("Content-Type", "application/octet-stream")
	chunkReq.Header.Set("X-Curated-Offset", "0")
	chunkReq.Header.Set("X-Curated-Chunk-Size", "4")
	chunkResp, err := http.DefaultClient.Do(chunkReq)
	if err != nil {
		t.Fatal(err)
	}
	defer chunkResp.Body.Close()
	if chunkResp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(chunkResp.Body)
		t.Fatalf("chunk status = %d body=%s, want 200", chunkResp.StatusCode, string(b))
	}

	finalPath := filepath.Join(libRoot, "IMP-LATE.mp4")
	if err := os.WriteFile(finalPath, []byte("old!"), 0o644); err != nil {
		t.Fatal(err)
	}

	commitReq, err := http.NewRequest(http.MethodPost, srv.URL+"/api/import/movies/uploads/"+created.UploadID+"/commit", nil)
	if err != nil {
		t.Fatal(err)
	}
	commitResp, err := http.DefaultClient.Do(commitReq)
	if err != nil {
		t.Fatal(err)
	}
	defer commitResp.Body.Close()
	if commitResp.StatusCode != http.StatusConflict {
		b, _ := io.ReadAll(commitResp.Body)
		t.Fatalf("commit status = %d body=%s, want 409", commitResp.StatusCode, string(b))
	}
	got, err := os.ReadFile(finalPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "old!" {
		t.Fatalf("late conflict was overwritten with %q", string(got))
	}
}

func TestImportLocalMoviesRouteNotRegistered(t *testing.T) {
	t.Parallel()
	h := NewHandler(Deps{Logger: zap.NewNop()})
	req := httptest.NewRequest(http.MethodPost, "/api/import/movies/local-copy", bytes.NewBufferString(`{"items":[]}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d body=%s, want 404", rr.Code, rr.Body.String())
	}
}

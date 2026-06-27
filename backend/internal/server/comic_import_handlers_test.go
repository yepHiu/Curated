package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	"curated-backend/internal/tasks"
)

type recordingComicScanStarter struct {
	paths [][]contracts.ComicLibraryPathDTO
	err   error
}

func (s *recordingComicScanStarter) StartComicScan(ctx context.Context, paths []contracts.ComicLibraryPathDTO) (contracts.TaskDTO, error) {
	_ = ctx
	s.paths = append(s.paths, append([]contracts.ComicLibraryPathDTO(nil), paths...))
	if s.err != nil {
		return contracts.TaskDTO{}, s.err
	}
	return contracts.TaskDTO{
		TaskID:   "scan-comics-after-import",
		Type:     contracts.TaskTypeScanComics,
		Status:   contracts.TaskRunning,
		Progress: 0,
	}, nil
}

func multipartComicsBody(t *testing.T, files map[string]string) (*bytes.Buffer, string) {
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

func newComicImportTestServer(t *testing.T, defaultPathID string, storeRoot string, scanStarter ComicScanStarter) (*httptest.Server, *tasks.Manager) {
	t.Helper()
	store := newImportTestStore(t, storeRoot)
	tm := tasks.NewManager()
	h := NewHandler(Deps{
		Cfg:              config.Default(),
		Logger:           zap.NewNop(),
		Store:            store,
		Tasks:            tm,
		ComicSettingsCtl: &stubComicSettingsCtl{enabled: true, defaultPathID: defaultPathID},
		ComicScanStarter: scanStarter,
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)
	return srv, tm
}

func TestHandleComicImport_RejectsMissingDefaultTarget(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	srv, _ := newComicImportTestServer(t, "", root, &recordingComicScanStarter{})

	body, contentType := multipartComicsBody(t, map[string]string{"Book One.cbz": "archive"})
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/import/comics", body)
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
		t.Fatalf("status = %d body=%s, want 400", resp.StatusCode, string(b))
	}
	var appErr contracts.AppError
	if err := json.NewDecoder(resp.Body).Decode(&appErr); err != nil {
		t.Fatal(err)
	}
	if appErr.Code != contracts.ErrorCodeComicImportTargetMissing {
		t.Fatalf("code = %q, want %q", appErr.Code, contracts.ErrorCodeComicImportTargetMissing)
	}
}

func TestHandleComicImport_CopiesZipAndCbzToDefaultComicPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := newImportTestStore(t, root)
	comicRoot := filepath.Join(root, "comics")
	if err := os.MkdirAll(comicRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	comicPath, err := store.AddComicLibraryPath(context.Background(), comicRoot, "Comics")
	if err != nil {
		t.Fatal(err)
	}
	scans := &recordingComicScanStarter{}
	tm := tasks.NewManager()
	h := NewHandler(Deps{
		Cfg:              config.Default(),
		Logger:           zap.NewNop(),
		Store:            store,
		Tasks:            tm,
		ComicSettingsCtl: &stubComicSettingsCtl{enabled: true, defaultPathID: comicPath.ID},
		ComicScanStarter: scans,
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	body, contentType := multipartComicsBody(t, map[string]string{
		"Book One.zip": "zip-archive",
		"Book Two.cbz": "cbz-archive",
	})
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/import/comics", body)
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
		t.Fatalf("status = %d body=%s, want 202", resp.StatusCode, string(b))
	}
	var task contracts.TaskDTO
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		t.Fatal(err)
	}
	if task.Type != contracts.TaskTypeImportComics {
		t.Fatalf("task type = %q, want %q", task.Type, contracts.TaskTypeImportComics)
	}
	if task.Status != contracts.TaskCompleted {
		t.Fatalf("task status = %q, want completed", task.Status)
	}
	if got, err := os.ReadFile(filepath.Join(comicRoot, "Book One.zip")); err != nil || string(got) != "zip-archive" {
		t.Fatalf("zip copy = %q err=%v", string(got), err)
	}
	if got, err := os.ReadFile(filepath.Join(comicRoot, "Book Two.cbz")); err != nil || string(got) != "cbz-archive" {
		t.Fatalf("cbz copy = %q err=%v", string(got), err)
	}
	if len(scans.paths) != 1 || len(scans.paths[0]) != 1 || scans.paths[0][0].ID != comicPath.ID || scans.paths[0][0].Path != comicRoot {
		t.Fatalf("comic scan paths = %#v, want default comic path %q", scans.paths, comicRoot)
	}
	if task.Metadata["scanTaskId"] != "scan-comics-after-import" {
		t.Fatalf("scanTaskId metadata = %#v", task.Metadata["scanTaskId"])
	}
}

func TestHandleComicImport_RejectsUnsupportedArchiveType(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := newImportTestStore(t, root)
	comicRoot := filepath.Join(root, "comics")
	if err := os.MkdirAll(comicRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	comicPath, err := store.AddComicLibraryPath(context.Background(), comicRoot, "Comics")
	if err != nil {
		t.Fatal(err)
	}
	scans := &recordingComicScanStarter{}
	tm := tasks.NewManager()
	h := NewHandler(Deps{
		Cfg:              config.Default(),
		Logger:           zap.NewNop(),
		Store:            store,
		Tasks:            tm,
		ComicSettingsCtl: &stubComicSettingsCtl{enabled: true, defaultPathID: comicPath.ID},
		ComicScanStarter: scans,
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	body, contentType := multipartComicsBody(t, map[string]string{"Book.rar": "rar-content"})
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/import/comics", body)
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
		t.Fatalf("status = %d body=%s, want 202", resp.StatusCode, string(b))
	}
	var task contracts.TaskDTO
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		t.Fatal(err)
	}
	if task.Status != contracts.TaskFailed {
		t.Fatalf("task status = %q, want failed", task.Status)
	}
	if task.ErrorCode != contracts.ErrorCodeComicArchiveUnsupported {
		t.Fatalf("error code = %q, want %q", task.ErrorCode, contracts.ErrorCodeComicArchiveUnsupported)
	}
	if _, err := os.Stat(filepath.Join(comicRoot, "Book.rar")); !os.IsNotExist(err) {
		t.Fatalf("unsupported file was copied or stat failed unexpectedly: %v", err)
	}
	if len(scans.paths) != 0 {
		t.Fatalf("scan started for unsupported import: %#v", scans.paths)
	}
}

func TestHandleComicImport_SkipsConflictWithoutOverwriting(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := newImportTestStore(t, root)
	comicRoot := filepath.Join(root, "comics")
	if err := os.MkdirAll(comicRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(comicRoot, "Existing.cbz"), []byte("already"), 0o644); err != nil {
		t.Fatal(err)
	}
	comicPath, err := store.AddComicLibraryPath(context.Background(), comicRoot, "Comics")
	if err != nil {
		t.Fatal(err)
	}
	scans := &recordingComicScanStarter{}
	tm := tasks.NewManager()
	h := NewHandler(Deps{
		Cfg:              config.Default(),
		Logger:           zap.NewNop(),
		Store:            store,
		Tasks:            tm,
		ComicSettingsCtl: &stubComicSettingsCtl{enabled: true, defaultPathID: comicPath.ID},
		ComicScanStarter: scans,
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	body, contentType := multipartComicsBody(t, map[string]string{
		"Fresh.zip":    "fresh",
		"Existing.cbz": "new-content",
	})
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/import/comics", body)
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
		t.Fatalf("status = %d body=%s, want 202", resp.StatusCode, string(b))
	}
	var task contracts.TaskDTO
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		t.Fatal(err)
	}
	if task.Status != contracts.TaskPartialFailed {
		t.Fatalf("task status = %q, want partial_failed", task.Status)
	}
	if task.ErrorCode != contracts.ErrorCodeComicImportConflict {
		t.Fatalf("error code = %q, want %q", task.ErrorCode, contracts.ErrorCodeComicImportConflict)
	}
	if got, err := os.ReadFile(filepath.Join(comicRoot, "Existing.cbz")); err != nil || string(got) != "already" {
		t.Fatalf("conflict file overwritten or unreadable: content=%q err=%v", string(got), err)
	}
	if got, err := os.ReadFile(filepath.Join(comicRoot, "Fresh.zip")); err != nil || string(got) != "fresh" {
		t.Fatalf("fresh copy = %q err=%v", string(got), err)
	}
	if task.Metadata["completedFiles"] != float64(1) && task.Metadata["completedFiles"] != 1 {
		t.Fatalf("completedFiles metadata = %#v, want 1", task.Metadata["completedFiles"])
	}
}

func TestHandleComicImport_RecordsScanStartErrorWithoutDeletingCopiedArchive(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := newImportTestStore(t, root)
	comicRoot := filepath.Join(root, "comics")
	if err := os.MkdirAll(comicRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	comicPath, err := store.AddComicLibraryPath(context.Background(), comicRoot, "Comics")
	if err != nil {
		t.Fatal(err)
	}
	scans := &recordingComicScanStarter{err: errors.New("scan queue unavailable")}
	tm := tasks.NewManager()
	h := NewHandler(Deps{
		Cfg:              config.Default(),
		Logger:           zap.NewNop(),
		Store:            store,
		Tasks:            tm,
		ComicSettingsCtl: &stubComicSettingsCtl{enabled: true, defaultPathID: comicPath.ID},
		ComicScanStarter: scans,
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	body, contentType := multipartComicsBody(t, map[string]string{"Scan Fails.cbz": "archive"})
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/import/comics", body)
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
		t.Fatalf("status = %d body=%s, want 202", resp.StatusCode, string(b))
	}
	var task contracts.TaskDTO
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		t.Fatal(err)
	}
	if task.Status != contracts.TaskPartialFailed {
		t.Fatalf("task status = %q, want partial_failed", task.Status)
	}
	if task.ErrorCode != contracts.ErrorCodeImportScanFailed {
		t.Fatalf("error code = %q, want %q", task.ErrorCode, contracts.ErrorCodeImportScanFailed)
	}
	if task.Metadata["scanError"] != "scan queue unavailable" {
		t.Fatalf("scanError metadata = %#v", task.Metadata["scanError"])
	}
	if got, err := os.ReadFile(filepath.Join(comicRoot, "Scan Fails.cbz")); err != nil || string(got) != "archive" {
		t.Fatalf("copied archive deleted or changed: content=%q err=%v", string(got), err)
	}
}

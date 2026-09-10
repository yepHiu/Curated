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
	"strings"
	"testing"

	"go.uber.org/zap"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/tasks"
)

type recordingPhotoImportScanStarter struct {
	paths [][]contracts.PhotoLibraryPathDTO
	err   error
}

func (s *recordingPhotoImportScanStarter) StartPhotoScan(ctx context.Context, paths []contracts.PhotoLibraryPathDTO) (contracts.TaskDTO, error) {
	_ = ctx
	s.paths = append(s.paths, append([]contracts.PhotoLibraryPathDTO(nil), paths...))
	if s.err != nil {
		return contracts.TaskDTO{}, s.err
	}
	return contracts.TaskDTO{
		TaskID:   "scan-photos-after-import",
		Type:     contracts.TaskTypeScanPhotos,
		Status:   contracts.TaskRunning,
		Progress: 0,
	}, nil
}

func multipartPhotosBody(t *testing.T, files map[string]string) (*bytes.Buffer, string) {
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

func newPhotoImportTestServer(t *testing.T, defaultPathID string, storeRoot string, scanStarter PhotoScanStarter) (*httptest.Server, *tasks.Manager) {
	t.Helper()
	store := newImportTestStore(t, storeRoot)
	tm := tasks.NewManager()
	h := NewHandler(Deps{
		Cfg:              config.Default(),
		Logger:           zap.NewNop(),
		Store:            store,
		Tasks:            tm,
		PhotoSettingsCtl: &stubPhotoSettingsCtl{enabled: true, defaultPathID: defaultPathID},
		PhotoScanStarter: scanStarter,
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)
	return srv, tm
}

func TestHandlePhotoImport_RejectsMissingDefaultTarget(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	srv, _ := newPhotoImportTestServer(t, "", root, &recordingPhotoImportScanStarter{})

	body, contentType := multipartPhotosBody(t, map[string]string{"Book One.cbz": "archive"})
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/import/photos", body)
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
	if appErr.Code != contracts.ErrorCodePhotoImportTargetMissing {
		t.Fatalf("code = %q, want %q", appErr.Code, contracts.ErrorCodePhotoImportTargetMissing)
	}
}

func TestHandlePhotoImport_CopiesZipAndCbzToDefaultPhotoPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := newImportTestStore(t, root)
	photoRoot := filepath.Join(root, "photos")
	if err := os.MkdirAll(photoRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	photoPath, err := store.AddPhotoLibraryPath(context.Background(), photoRoot, "Photos")
	if err != nil {
		t.Fatal(err)
	}
	scans := &recordingPhotoImportScanStarter{}
	tm := tasks.NewManager()
	h := NewHandler(Deps{
		Cfg:              config.Default(),
		Logger:           zap.NewNop(),
		Store:            store,
		Tasks:            tm,
		PhotoSettingsCtl: &stubPhotoSettingsCtl{enabled: true, defaultPathID: photoPath.ID},
		PhotoScanStarter: scans,
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	body, contentType := multipartPhotosBody(t, map[string]string{
		"Book One.zip": "zip-archive",
		"Book Two.cbz": "cbz-archive",
	})
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/import/photos", body)
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
	if task.Type != contracts.TaskTypeImportPhotos {
		t.Fatalf("task type = %q, want %q", task.Type, contracts.TaskTypeImportPhotos)
	}
	if task.Status != contracts.TaskCompleted {
		t.Fatalf("task status = %q, want completed", task.Status)
	}
	if got, err := os.ReadFile(filepath.Join(photoRoot, "Book One.zip")); err != nil || string(got) != "zip-archive" {
		t.Fatalf("zip copy = %q err=%v", string(got), err)
	}
	if got, err := os.ReadFile(filepath.Join(photoRoot, "Book Two.cbz")); err != nil || string(got) != "cbz-archive" {
		t.Fatalf("cbz copy = %q err=%v", string(got), err)
	}
	if len(scans.paths) != 1 || len(scans.paths[0]) != 1 || scans.paths[0][0].ID != photoPath.ID || scans.paths[0][0].Path != photoRoot {
		t.Fatalf("photo scan paths = %#v, want default photo path %q", scans.paths, photoRoot)
	}
	if task.Metadata["scanTaskId"] != "scan-photos-after-import" {
		t.Fatalf("scanTaskId metadata = %#v", task.Metadata["scanTaskId"])
	}
}

func TestHandlePhotoImport_RejectsUnsupportedArchiveType(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := newImportTestStore(t, root)
	photoRoot := filepath.Join(root, "photos")
	if err := os.MkdirAll(photoRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	photoPath, err := store.AddPhotoLibraryPath(context.Background(), photoRoot, "Photos")
	if err != nil {
		t.Fatal(err)
	}
	scans := &recordingPhotoImportScanStarter{}
	tm := tasks.NewManager()
	h := NewHandler(Deps{
		Cfg:              config.Default(),
		Logger:           zap.NewNop(),
		Store:            store,
		Tasks:            tm,
		PhotoSettingsCtl: &stubPhotoSettingsCtl{enabled: true, defaultPathID: photoPath.ID},
		PhotoScanStarter: scans,
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	body, contentType := multipartPhotosBody(t, map[string]string{"Book.rar": "rar-content"})
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/import/photos", body)
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
	if task.ErrorCode != contracts.ErrorCodePhotoArchiveUnsupported {
		t.Fatalf("error code = %q, want %q", task.ErrorCode, contracts.ErrorCodePhotoArchiveUnsupported)
	}
	if _, err := os.Stat(filepath.Join(photoRoot, "Book.rar")); !os.IsNotExist(err) {
		t.Fatalf("unsupported file was copied or stat failed unexpectedly: %v", err)
	}
	if len(scans.paths) != 0 {
		t.Fatalf("scan started for unsupported import: %#v", scans.paths)
	}
}

func TestHandlePhotoImport_SkipsConflictWithoutOverwriting(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := newImportTestStore(t, root)
	photoRoot := filepath.Join(root, "photos")
	if err := os.MkdirAll(photoRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(photoRoot, "Existing.cbz"), []byte("already"), 0o644); err != nil {
		t.Fatal(err)
	}
	photoPath, err := store.AddPhotoLibraryPath(context.Background(), photoRoot, "Photos")
	if err != nil {
		t.Fatal(err)
	}
	scans := &recordingPhotoImportScanStarter{}
	tm := tasks.NewManager()
	h := NewHandler(Deps{
		Cfg:              config.Default(),
		Logger:           zap.NewNop(),
		Store:            store,
		Tasks:            tm,
		PhotoSettingsCtl: &stubPhotoSettingsCtl{enabled: true, defaultPathID: photoPath.ID},
		PhotoScanStarter: scans,
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	body, contentType := multipartPhotosBody(t, map[string]string{
		"Fresh.zip":    "fresh",
		"Existing.cbz": "new-content",
	})
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/import/photos", body)
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
	if task.ErrorCode != contracts.ErrorCodePhotoImportConflict {
		t.Fatalf("error code = %q, want %q", task.ErrorCode, contracts.ErrorCodePhotoImportConflict)
	}
	if got, err := os.ReadFile(filepath.Join(photoRoot, "Existing.cbz")); err != nil || string(got) != "already" {
		t.Fatalf("conflict file overwritten or unreadable: content=%q err=%v", string(got), err)
	}
	if got, err := os.ReadFile(filepath.Join(photoRoot, "Fresh.zip")); err != nil || string(got) != "fresh" {
		t.Fatalf("fresh copy = %q err=%v", string(got), err)
	}
	if task.Metadata["completedFiles"] != float64(1) && task.Metadata["completedFiles"] != 1 {
		t.Fatalf("completedFiles metadata = %#v, want 1", task.Metadata["completedFiles"])
	}
}

func TestHandlePhotoImport_RecordsScanStartErrorWithoutDeletingCopiedArchive(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := newImportTestStore(t, root)
	photoRoot := filepath.Join(root, "photos")
	if err := os.MkdirAll(photoRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	photoPath, err := store.AddPhotoLibraryPath(context.Background(), photoRoot, "Photos")
	if err != nil {
		t.Fatal(err)
	}
	scans := &recordingPhotoImportScanStarter{err: errors.New("scan queue unavailable")}
	tm := tasks.NewManager()
	h := NewHandler(Deps{
		Cfg:              config.Default(),
		Logger:           zap.NewNop(),
		Store:            store,
		Tasks:            tm,
		PhotoSettingsCtl: &stubPhotoSettingsCtl{enabled: true, defaultPathID: photoPath.ID},
		PhotoScanStarter: scans,
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	body, contentType := multipartPhotosBody(t, map[string]string{"Scan Fails.cbz": "archive"})
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/import/photos", body)
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
	if got, err := os.ReadFile(filepath.Join(photoRoot, "Scan Fails.cbz")); err != nil || string(got) != "archive" {
		t.Fatalf("copied archive deleted or changed: content=%q err=%v", string(got), err)
	}
}

// Regression: publication must not overwrite a destination created after preflight.
func TestPhotoImportPublicationPreservesConcurrentDestination(t *testing.T) {
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := root.WriteFile("upload.tmp", []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := root.WriteFile("book.cbz", []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := publishPhotoImport(root, "upload.tmp", "book.cbz"); !errors.Is(err, os.ErrExist) {
		t.Fatalf("want conflict, got %v", err)
	}
	data, _ := root.ReadFile("book.cbz")
	if string(data) != "original" {
		t.Fatal("existing archive replaced")
	}
	if _, err := copyPhotoImportPart(root, "../escape.cbz", strings.NewReader("no"), func(int64) {}); err == nil {
		t.Fatal("escaped target root")
	}
}

func TestHandlePhotoImportDisabledDoesNotCopy(t *testing.T) {
	root := t.TempDir()
	store := newImportTestStore(t, root)
	h := NewHandler(Deps{Cfg: config.Default(), Logger: zap.NewNop(), Store: store, Tasks: tasks.NewManager(), PhotoSettingsCtl: &stubPhotoSettingsCtl{enabled: false}})
	body, contentType := multipartPhotosBody(t, map[string]string{"book.cbz": "data"})
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/api/import/photos", body)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	h.Routes().ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), contracts.ErrorCodePhotoLibraryDisabled) {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
}

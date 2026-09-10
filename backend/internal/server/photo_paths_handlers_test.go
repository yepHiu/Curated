package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
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

type recordingPhotoWatchReloader struct {
	calls int
}

func (r *recordingPhotoWatchReloader) ReloadPhotoLibraryWatches(context.Context) error {
	r.calls++
	return nil
}

func TestHandleGetSettings_PhotoPathsFromIndependentStore(t *testing.T) {
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

	ctl := &stubPhotoSettingsCtl{
		enabled:       true,
		autoWatch:     true,
		defaultPathID: photoPath.ID,
		viewer: contracts.PhotoViewerSettingsDTO{
			Mode:      "scroll",
			Fit:       "width",
			Direction: "ltr",
		},
		cache: contracts.PhotoCacheSettingsDTO{MaxBytes: 5 * 1024 * 1024 * 1024},
	}
	h := NewHandler(Deps{
		Cfg:              config.Default(),
		Logger:           zap.NewNop(),
		Store:            store,
		PhotoSettingsCtl: ctl,
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
	if len(dto.PhotoLibraryPaths) != 1 || dto.PhotoLibraryPaths[0].ID != photoPath.ID {
		t.Fatalf("photoLibraryPaths = %#v, want path %q", dto.PhotoLibraryPaths, photoPath.ID)
	}
	if dto.DefaultPhotoImportLibraryPathID != photoPath.ID {
		t.Fatalf("defaultPhotoImportLibraryPathId = %q, want %q", dto.DefaultPhotoImportLibraryPathID, photoPath.ID)
	}
}

func TestHandleAddPhotoLibraryPath_CreatesPathAndReloadsPhotoWatcherOnly(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := newImportTestStore(t, root)
	photoRoot := filepath.Join(root, "photos")
	if err := os.MkdirAll(photoRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	photoReloader := &recordingPhotoWatchReloader{}
	comicReloader := &recordingComicWatchReloader{}
	h := NewHandler(Deps{
		Cfg:                       config.Default(),
		Logger:                    zap.NewNop(),
		Store:                     store,
		PhotoLibraryWatchReloader: photoReloader,
		ComicLibraryWatchReloader: comicReloader,
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	body := bytes.NewBufferString(`{"path":` + quoteJSON(photoRoot) + `,"title":"Photos"}`)
	resp, err := http.Post(srv.URL+"/api/library/photos/paths", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d body=%s", resp.StatusCode, string(b))
	}
	var dto contracts.AddPhotoLibraryPathResponse
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		t.Fatal(err)
	}
	if dto.ID == "" || dto.Path != photoRoot || dto.Title != "Photos" {
		t.Fatalf("created photo path = %#v", dto)
	}

	paths, err := store.ListPhotoLibraryPaths(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 1 || paths[0].ID != dto.ID {
		t.Fatalf("stored photo paths = %#v, want created %q", paths, dto.ID)
	}
	if photoReloader.calls != 1 {
		t.Fatalf("photo watch reload calls = %d, want 1", photoReloader.calls)
	}
	if comicReloader.calls != 0 {
		t.Fatalf("comic watch reload calls = %d, want 0", comicReloader.calls)
	}
}

func TestHandleAddPhotoLibraryPath_StartsInitialPhotoScanWhenEnabled(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := newImportTestStore(t, root)
	photoRoot := filepath.Join(root, "photos")
	if err := os.MkdirAll(photoRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	starter := &recordingPhotoScanStarter{}
	h := NewHandler(Deps{
		Cfg: config.Config{
			PhotoLibraryEnabled: true,
		},
		Logger:           zap.NewNop(),
		Store:            store,
		Tasks:            tasks.NewManager(),
		PhotoScanStarter: starter,
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	body := bytes.NewBufferString(`{"path":` + quoteJSON(photoRoot) + `,"title":"Photos"}`)
	resp, err := http.Post(srv.URL+"/api/library/photos/paths", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d body=%s", resp.StatusCode, string(b))
	}
	var dto contracts.AddPhotoLibraryPathResponse
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		t.Fatal(err)
	}
	if dto.ScanTask == nil || dto.ScanTask.Type != contracts.TaskTypeScanPhotos {
		t.Fatalf("scanTask = %#v, want scan.photos", dto.ScanTask)
	}
	if len(starter.paths) != 1 || len(starter.paths[0]) != 1 || starter.paths[0][0].ID != dto.ID {
		t.Fatalf("scan paths = %#v, want created photo path %q", starter.paths, dto.ID)
	}
}

func TestHandlePatchAndDeletePhotoLibraryPath(t *testing.T) {
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
	photoReloader := &recordingPhotoWatchReloader{}
	h := NewHandler(Deps{
		Cfg:                       config.Default(),
		Logger:                    zap.NewNop(),
		Store:                     store,
		PhotoLibraryWatchReloader: photoReloader,
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodPatch, srv.URL+"/api/library/photos/paths/"+photoPath.ID, bytes.NewBufferString(`{"title":"Photo Shelf"}`))
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
		t.Fatalf("patch status = %d body=%s", resp.StatusCode, string(b))
	}
	var updated contracts.PhotoLibraryPathDTO
	if err := json.NewDecoder(resp.Body).Decode(&updated); err != nil {
		t.Fatal(err)
	}
	if updated.Title != "Photo Shelf" {
		t.Fatalf("updated title = %q, want Photo Shelf", updated.Title)
	}

	req, err = http.NewRequest(http.MethodDelete, srv.URL+"/api/library/photos/paths/"+photoPath.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("delete status = %d body=%s", resp.StatusCode, string(b))
	}
	if photoReloader.calls != 1 {
		t.Fatalf("photo watch reload calls = %d, want 1", photoReloader.calls)
	}
}

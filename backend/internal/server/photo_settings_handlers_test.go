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
)

type stubPhotoSettingsCtl struct {
	enabled       bool
	autoWatch     bool
	defaultPathID string
	viewer        contracts.PhotoViewerSettingsDTO
	cache         contracts.PhotoCacheSettingsDTO
}

func (s *stubPhotoSettingsCtl) PhotoLibraryEnabled() bool {
	return s.enabled
}

func (s *stubPhotoSettingsCtl) SetPhotoLibraryEnabled(v bool) error {
	s.enabled = v
	return nil
}

func (s *stubPhotoSettingsCtl) AutoPhotoLibraryWatch() bool {
	return s.autoWatch
}

func (s *stubPhotoSettingsCtl) SetAutoPhotoLibraryWatch(v bool) error {
	s.autoWatch = v
	return nil
}

func (s *stubPhotoSettingsCtl) DefaultPhotoImportLibraryPathID() string {
	return s.defaultPathID
}

func (s *stubPhotoSettingsCtl) SetDefaultPhotoImportLibraryPathID(id string) error {
	s.defaultPathID = id
	return nil
}

func (s *stubPhotoSettingsCtl) PhotoViewerSettings() contracts.PhotoViewerSettingsDTO {
	return s.viewer
}

func (s *stubPhotoSettingsCtl) SetPhotoViewerSettings(v contracts.PhotoViewerSettingsDTO) error {
	s.viewer = v
	return nil
}

func (s *stubPhotoSettingsCtl) PhotoCacheSettings() contracts.PhotoCacheSettingsDTO {
	return s.cache
}

func (s *stubPhotoSettingsCtl) SetPhotoCacheSettings(v contracts.PhotoCacheSettingsDTO) error {
	s.cache = v
	return nil
}

func TestHandleGetSettings_PhotoFieldsFromController(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := newImportTestStore(t, root)
	ctl := &stubPhotoSettingsCtl{
		enabled:       true,
		autoWatch:     false,
		defaultPathID: "photo-lib-main",
		viewer: contracts.PhotoViewerSettingsDTO{
			Mode:      "scroll",
			Fit:       "width",
			Direction: "rtl",
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
	if !dto.PhotoLibraryEnabled {
		t.Fatal("expected photoLibraryEnabled from controller")
	}
	if dto.AutoPhotoLibraryWatch {
		t.Fatal("expected autoPhotoLibraryWatch false from controller")
	}
	if len(dto.PhotoLibraryPaths) != 0 {
		t.Fatalf("photoLibraryPaths = %#v, want empty array", dto.PhotoLibraryPaths)
	}
	if dto.DefaultPhotoImportLibraryPathID != "photo-lib-main" {
		t.Fatalf("defaultPhotoImportLibraryPathId = %q", dto.DefaultPhotoImportLibraryPathID)
	}
	if dto.PhotoViewer.Mode != "scroll" || dto.PhotoViewer.Fit != "width" || dto.PhotoViewer.Direction != "rtl" {
		t.Fatalf("photoViewer = %#v", dto.PhotoViewer)
	}
	if dto.PhotoCache.MaxBytes != 5*1024*1024*1024 {
		t.Fatalf("photoCache.maxBytes = %d", dto.PhotoCache.MaxBytes)
	}
}

func TestHandlePatchSettings_PhotoSettings(t *testing.T) {
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
	ctl := &stubPhotoSettingsCtl{}
	h := NewHandler(Deps{
		Cfg:              config.Default(),
		Logger:           zap.NewNop(),
		Store:            store,
		PhotoSettingsCtl: ctl,
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	body := `{"photoLibraryEnabled":true,"autoPhotoLibraryWatch":false,"defaultPhotoImportLibraryPathId":"` + photoPath.ID + `","photoViewer":{"mode":"scroll","fit":"width","direction":"rtl"},"photoCache":{"maxBytes":1073741824}}`
	req, err := http.NewRequest(http.MethodPatch, srv.URL+"/api/settings", bytes.NewBufferString(body))
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
	if !ctl.enabled || !dto.PhotoLibraryEnabled {
		t.Fatalf("photoLibraryEnabled not saved: ctl=%v dto=%v", ctl.enabled, dto.PhotoLibraryEnabled)
	}
	if ctl.autoWatch || dto.AutoPhotoLibraryWatch {
		t.Fatalf("autoPhotoLibraryWatch not saved: ctl=%v dto=%v", ctl.autoWatch, dto.AutoPhotoLibraryWatch)
	}
	if ctl.defaultPathID != photoPath.ID || dto.DefaultPhotoImportLibraryPathID != photoPath.ID {
		t.Fatalf("default photo path not saved: ctl=%q dto=%q", ctl.defaultPathID, dto.DefaultPhotoImportLibraryPathID)
	}
	if ctl.viewer.Mode != "scroll" || ctl.viewer.Fit != "width" || ctl.viewer.Direction != "rtl" {
		t.Fatalf("controller viewer = %#v", ctl.viewer)
	}
	if dto.PhotoCache.MaxBytes != 1073741824 || ctl.cache.MaxBytes != 1073741824 {
		t.Fatalf("photo cache not saved: ctl=%#v dto=%#v", ctl.cache, dto.PhotoCache)
	}
}

func TestHandlePatchSettings_PhotoLibraryRequiresPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := newImportTestStore(t, root)
	ctl := &stubPhotoSettingsCtl{}
	h := NewHandler(Deps{
		Cfg:              config.Default(),
		Logger:           zap.NewNop(),
		Store:            store,
		PhotoSettingsCtl: ctl,
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodPatch, srv.URL+"/api/settings", bytes.NewBufferString(`{"photoLibraryEnabled":true}`))
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
	var appErr contracts.AppError
	if err := json.NewDecoder(resp.Body).Decode(&appErr); err != nil {
		t.Fatal(err)
	}
	if appErr.Code != contracts.ErrorCodePhotoPathNotConfigured {
		t.Fatalf("error code = %q, want %q", appErr.Code, contracts.ErrorCodePhotoPathNotConfigured)
	}
	if ctl.enabled {
		t.Fatal("photo library should remain disabled when no photo path exists")
	}
}

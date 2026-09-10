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

type stubComicSettingsCtl struct {
	enabled       bool
	autoWatch     bool
	defaultPathID string
	reader        contracts.ComicReaderSettingsDTO
	cache         contracts.ComicCacheSettingsDTO
}

func (s *stubComicSettingsCtl) ComicLibraryEnabled() bool {
	return s.enabled
}

func (s *stubComicSettingsCtl) SetComicLibraryEnabled(v bool) error {
	s.enabled = v
	return nil
}

func (s *stubComicSettingsCtl) AutoComicLibraryWatch() bool {
	return s.autoWatch
}

func (s *stubComicSettingsCtl) SetAutoComicLibraryWatch(v bool) error {
	s.autoWatch = v
	return nil
}

func (s *stubComicSettingsCtl) DefaultComicImportLibraryPathID() string {
	return s.defaultPathID
}

func (s *stubComicSettingsCtl) SetDefaultComicImportLibraryPathID(id string) error {
	s.defaultPathID = id
	return nil
}

func (s *stubComicSettingsCtl) ComicReaderSettings() contracts.ComicReaderSettingsDTO {
	return s.reader
}

func (s *stubComicSettingsCtl) SetComicReaderSettings(v contracts.ComicReaderSettingsDTO) error {
	s.reader = v
	return nil
}

func (s *stubComicSettingsCtl) ComicCacheSettings() contracts.ComicCacheSettingsDTO {
	return s.cache
}

func (s *stubComicSettingsCtl) SetComicCacheSettings(v contracts.ComicCacheSettingsDTO) error {
	s.cache = v
	return nil
}

type recordingComicWatchReloader struct {
	calls int
}

func (r *recordingComicWatchReloader) ReloadComicLibraryWatches(context.Context) error {
	r.calls++
	return nil
}

func TestHandleGetSettings_ComicFieldsFromControllerAndPaths(t *testing.T) {
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

	ctl := &stubComicSettingsCtl{
		enabled:       true,
		autoWatch:     true,
		defaultPathID: comicPath.ID,
		reader: contracts.ComicReaderSettingsDTO{
			Mode:      "scroll",
			Fit:       "width",
			Direction: "rtl",
		},
		cache: contracts.ComicCacheSettingsDTO{MaxBytes: 5 * 1024 * 1024 * 1024},
	}
	h := NewHandler(Deps{
		Cfg:              config.Default(),
		Logger:           zap.NewNop(),
		Store:            store,
		ComicSettingsCtl: ctl,
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
	if !dto.ComicLibraryEnabled {
		t.Fatal("expected comicLibraryEnabled from controller")
	}
	if !dto.AutoComicLibraryWatch {
		t.Fatal("expected autoComicLibraryWatch from controller")
	}
	if len(dto.ComicLibraryPaths) != 1 || dto.ComicLibraryPaths[0].ID != comicPath.ID {
		t.Fatalf("comicLibraryPaths = %#v, want path %q", dto.ComicLibraryPaths, comicPath.ID)
	}
	if dto.DefaultComicImportLibraryPathID != comicPath.ID {
		t.Fatalf("defaultComicImportLibraryPathId = %q, want %q", dto.DefaultComicImportLibraryPathID, comicPath.ID)
	}
	if dto.ComicReader.Mode != "scroll" || dto.ComicReader.Fit != "width" || dto.ComicReader.Direction != "rtl" {
		t.Fatalf("comicReader = %#v", dto.ComicReader)
	}
	if dto.ComicCache.MaxBytes != 5*1024*1024*1024 {
		t.Fatalf("comicCache.maxBytes = %d", dto.ComicCache.MaxBytes)
	}
}

func TestHandleGetSettings_ComicPathsReturnsEmptyArray(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := newImportTestStore(t, root)
	h := NewHandler(Deps{
		Cfg:    config.Default(),
		Logger: zap.NewNop(),
		Store:  store,
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
	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		t.Fatal(err)
	}
	value, ok := raw["comicLibraryPaths"]
	if !ok {
		t.Fatal("comicLibraryPaths key missing")
	}
	paths, ok := value.([]any)
	if !ok {
		t.Fatalf("comicLibraryPaths = %#v, want array", value)
	}
	if len(paths) != 0 {
		t.Fatalf("comicLibraryPaths length = %d, want 0", len(paths))
	}
}

func TestHandleAddComicLibraryPath_CreatesPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := newImportTestStore(t, root)
	comicRoot := filepath.Join(root, "comics")
	if err := os.MkdirAll(comicRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	reloader := &recordingComicWatchReloader{}
	h := NewHandler(Deps{
		Cfg:                       config.Default(),
		Logger:                    zap.NewNop(),
		Store:                     store,
		ComicLibraryWatchReloader: reloader,
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	body := bytes.NewBufferString(`{"path":` + quoteJSON(comicRoot) + `,"title":"Manga"}`)
	resp, err := http.Post(srv.URL+"/api/library/comics/paths", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d body=%s", resp.StatusCode, string(b))
	}
	var dto contracts.AddComicLibraryPathResponse
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		t.Fatal(err)
	}
	if dto.ID == "" || dto.Path != comicRoot || dto.Title != "Manga" {
		t.Fatalf("created comic path = %#v", dto)
	}

	paths, err := store.ListComicLibraryPaths(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 1 || paths[0].ID != dto.ID {
		t.Fatalf("stored comic paths = %#v, want created %q", paths, dto.ID)
	}
	if reloader.calls != 1 {
		t.Fatalf("comic watch reload calls = %d, want 1", reloader.calls)
	}
}

func TestHandleDeleteComicLibraryPath_ReloadsComicWatcher(t *testing.T) {
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
	reloader := &recordingComicWatchReloader{}
	h := NewHandler(Deps{
		Cfg:                       config.Default(),
		Logger:                    zap.NewNop(),
		Store:                     store,
		ComicLibraryWatchReloader: reloader,
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodDelete, srv.URL+"/api/library/comics/paths/"+comicPath.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d body=%s", resp.StatusCode, string(b))
	}
	if reloader.calls != 1 {
		t.Fatalf("comic watch reload calls = %d, want 1", reloader.calls)
	}
}

// TestHandlePatchSettings_ComicBetaCanEnableBeforeAddingPath covers the enable-first Beta flow.
func TestHandlePatchSettings_ComicBetaCanEnableBeforeAddingPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := newImportTestStore(t, root)
	ctl := &stubComicSettingsCtl{}
	h := NewHandler(Deps{
		Cfg:              config.Default(),
		Logger:           zap.NewNop(),
		Store:            store,
		ComicSettingsCtl: ctl,
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodPatch, srv.URL+"/api/settings", bytes.NewBufferString(`{"comicLibraryEnabled":true}`))
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
	if !ctl.enabled {
		t.Fatal("comic beta should enable before configuring paths")
	}
}

func TestHandlePatchSettings_ComicSettings(t *testing.T) {
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
	ctl := &stubComicSettingsCtl{}
	h := NewHandler(Deps{
		Cfg:              config.Default(),
		Logger:           zap.NewNop(),
		Store:            store,
		ComicSettingsCtl: ctl,
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	body := `{"comicLibraryEnabled":true,"autoComicLibraryWatch":false,"defaultComicImportLibraryPathId":"` + comicPath.ID + `","comicReader":{"mode":"scroll","fit":"width","direction":"rtl"},"comicCache":{"maxBytes":1073741824}}`
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
	if !ctl.enabled || !dto.ComicLibraryEnabled {
		t.Fatalf("comicLibraryEnabled not saved: ctl=%v dto=%v", ctl.enabled, dto.ComicLibraryEnabled)
	}
	if ctl.autoWatch || dto.AutoComicLibraryWatch {
		t.Fatalf("autoComicLibraryWatch not saved: ctl=%v dto=%v", ctl.autoWatch, dto.AutoComicLibraryWatch)
	}
	if ctl.defaultPathID != comicPath.ID || dto.DefaultComicImportLibraryPathID != comicPath.ID {
		t.Fatalf("default comic path not saved: ctl=%q dto=%q", ctl.defaultPathID, dto.DefaultComicImportLibraryPathID)
	}
	if ctl.reader.Mode != "scroll" || ctl.reader.Fit != "width" || ctl.reader.Direction != "rtl" {
		t.Fatalf("controller reader = %#v", ctl.reader)
	}
	if dto.ComicCache.MaxBytes != 1073741824 || ctl.cache.MaxBytes != 1073741824 {
		t.Fatalf("comic cache not saved: ctl=%#v dto=%#v", ctl.cache, dto.ComicCache)
	}
}

func quoteJSON(value string) string {
	b, _ := json.Marshal(value)
	return string(b)
}

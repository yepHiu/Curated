package app

import (
	"archive/zip"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
	"curated-backend/internal/tasks"

	"go.uber.org/zap"
)

func TestComicSettingsPersistToLibraryConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "library-config.cfg")
	if err := os.WriteFile(path, []byte(`{"unknownKey":"preserve"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := &App{
		cfg:                 config.Default(),
		librarySettingsPath: path,
	}

	if err := a.SetComicLibraryEnabled(true); err != nil {
		t.Fatal(err)
	}
	if err := a.SetAutoComicLibraryWatch(false); err != nil {
		t.Fatal(err)
	}
	if err := a.SetDefaultComicImportLibraryPathID("comic-lib-main"); err != nil {
		t.Fatal(err)
	}
	if err := a.SetComicReaderSettings(contracts.ComicReaderSettingsDTO{
		Mode:      "scroll",
		Fit:       "width",
		Direction: "rtl",
	}); err != nil {
		t.Fatal(err)
	}
	if err := a.SetComicCacheSettings(contracts.ComicCacheSettingsDTO{MaxBytes: 5 * 1024 * 1024 * 1024}); err != nil {
		t.Fatal(err)
	}

	if !a.ComicLibraryEnabled() {
		t.Fatal("expected comic library enabled")
	}
	if a.AutoComicLibraryWatch() {
		t.Fatal("expected comic auto watch disabled")
	}
	if got := a.DefaultComicImportLibraryPathID(); got != "comic-lib-main" {
		t.Fatalf("DefaultComicImportLibraryPathID = %q", got)
	}
	if got := a.ComicReaderSettings(); got.Mode != "scroll" || got.Fit != "width" || got.Direction != "rtl" {
		t.Fatalf("ComicReaderSettings = %#v", got)
	}
	if got := a.ComicCacheSettings(); got.MaxBytes != 5*1024*1024*1024 {
		t.Fatalf("ComicCacheSettings.MaxBytes = %d", got.MaxBytes)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var persisted map[string]any
	if err := json.Unmarshal(raw, &persisted); err != nil {
		t.Fatal(err)
	}
	if persisted["unknownKey"] != "preserve" {
		t.Fatalf("unknownKey was not preserved: %#v", persisted)
	}
	if persisted["comicLibraryEnabled"] != true {
		t.Fatalf("comicLibraryEnabled not persisted: %#v", persisted["comicLibraryEnabled"])
	}
	if persisted["autoComicLibraryWatch"] != false {
		t.Fatalf("autoComicLibraryWatch not persisted: %#v", persisted["autoComicLibraryWatch"])
	}
	if persisted["defaultComicImportLibraryPathId"] != "comic-lib-main" {
		t.Fatalf("defaultComicImportLibraryPathId not persisted: %#v", persisted["defaultComicImportLibraryPathId"])
	}
	reader, ok := persisted["comicReader"].(map[string]any)
	if !ok || reader["mode"] != "scroll" || reader["fit"] != "width" || reader["direction"] != "rtl" {
		t.Fatalf("comicReader not persisted: %#v", persisted["comicReader"])
	}
	cache, ok := persisted["comicCache"].(map[string]any)
	if !ok || cache["maxBytes"] != float64(5*1024*1024*1024) {
		t.Fatalf("comicCache not persisted: %#v", persisted["comicCache"])
	}
}

func TestStartComicScanWithExtraMetadataIndexesArchive(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := storage.NewSQLiteStore(filepath.Join(root, "curated.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	comicRoot := filepath.Join(root, "comics")
	if err := os.MkdirAll(comicRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeAppTestComicZip(filepath.Join(comicRoot, "Watch Book.cbz"), map[string]string{
		"001.jpg": "page 1",
	}); err != nil {
		t.Fatal(err)
	}
	comicPath, err := store.AddComicLibraryPath(context.Background(), comicRoot, "Comics")
	if err != nil {
		t.Fatal(err)
	}
	a := &App{
		store:  store,
		tasks:  tasks.NewManager(),
		appCtx: context.Background(),
	}

	task, err := a.startComicScan(context.Background(), []contracts.ComicLibraryPathDTO{comicPath}, map[string]any{
		"trigger": "fsnotify",
		"paths":   []string{comicRoot},
	})
	if err != nil {
		t.Fatal(err)
	}
	if task.Type != contracts.TaskTypeScanComics {
		t.Fatalf("task type = %q, want %q", task.Type, contracts.TaskTypeScanComics)
	}
	if task.Metadata["trigger"] != "fsnotify" {
		t.Fatalf("trigger metadata = %#v", task.Metadata["trigger"])
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		got, ok := a.tasks.Get(task.TaskID)
		if ok && got.Status == "completed" {
			if got.Metadata["trigger"] != "fsnotify" {
				t.Fatalf("completed trigger metadata = %#v", got.Metadata["trigger"])
			}
			page, err := store.ListComicBooks(context.Background(), contracts.ListComicBooksRequest{Limit: 10})
			if err != nil {
				t.Fatal(err)
			}
			if page.Total != 1 || page.Items[0].Title != "Watch Book" {
				t.Fatalf("indexed comics = %#v", page.Items)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for comic scan completion")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestEnqueueComicLibraryWatchScanRootsStartsFsnotifyScan(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := storage.NewSQLiteStore(filepath.Join(root, "curated.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	comicRoot := filepath.Join(root, "comics")
	if err := os.MkdirAll(comicRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeAppTestComicZip(filepath.Join(comicRoot, "Queued Book.cbz"), map[string]string{
		"001.jpg": "page 1",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddComicLibraryPath(context.Background(), comicRoot, "Comics"); err != nil {
		t.Fatal(err)
	}
	a := &App{
		store:                 store,
		tasks:                 tasks.NewManager(),
		appCtx:                context.Background(),
		comicLibraryEnabled:   true,
		autoComicLibraryWatch: true,
		comicWatchScanPending: make(map[string]struct{}),
	}

	a.EnqueueComicLibraryWatchScanRoots([]string{comicRoot})

	deadline := time.Now().Add(2 * time.Second)
	for {
		for _, got := range a.tasks.ListRecentFinished(10) {
			if got.Type != contracts.TaskTypeScanComics {
				continue
			}
			if got.Status != contracts.TaskCompleted {
				t.Fatalf("comic watch task status = %q, want completed", got.Status)
			}
			if got.Metadata["trigger"] != "fsnotify" {
				t.Fatalf("trigger metadata = %#v, want fsnotify", got.Metadata["trigger"])
			}
			paths, ok := got.Metadata["paths"].([]string)
			if !ok || len(paths) != 1 || paths[0] != comicRoot {
				t.Fatalf("paths metadata = %#v, want [%q]", got.Metadata["paths"], comicRoot)
			}
			page, err := store.ListComicBooks(context.Background(), contracts.ListComicBooksRequest{Limit: 10})
			if err != nil {
				t.Fatal(err)
			}
			if page.Total != 1 || page.Items[0].Title != "Queued Book" {
				t.Fatalf("indexed comics = %#v", page.Items)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for comic watch scan")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestSetAutoComicLibraryWatchControlsWatcherLoop(t *testing.T) {
	t.Parallel()

	a := newComicWatchSettingsTestApp(t, false, true)
	if err := a.SetAutoComicLibraryWatch(true); err != nil {
		t.Fatal(err)
	}
	if !a.comicWatchLoopRunningForTest() {
		t.Fatal("expected comic watcher loop to run after enabling auto comic watch")
	}
	if err := a.SetAutoComicLibraryWatch(false); err != nil {
		t.Fatal(err)
	}
	if a.comicWatchLoopRunningForTest() {
		t.Fatal("expected comic watcher loop to stop after disabling auto comic watch")
	}
}

func TestSetComicLibraryEnabledControlsWatcherLoop(t *testing.T) {
	t.Parallel()

	a := newComicWatchSettingsTestApp(t, true, false)
	if err := a.SetComicLibraryEnabled(true); err != nil {
		t.Fatal(err)
	}
	if !a.comicWatchLoopRunningForTest() {
		t.Fatal("expected comic watcher loop to run after enabling comic library")
	}
	if err := a.SetComicLibraryEnabled(false); err != nil {
		t.Fatal(err)
	}
	if a.comicWatchLoopRunningForTest() {
		t.Fatal("expected comic watcher loop to stop after disabling comic library")
	}
}

func writeAppTestComicZip(path string, entries map[string]string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := zip.NewWriter(file)
	for name, content := range entries {
		w, err := writer.Create(name)
		if err != nil {
			_ = writer.Close()
			return err
		}
		if _, err := io.WriteString(w, content); err != nil {
			_ = writer.Close()
			return err
		}
	}
	return writer.Close()
}

func newComicWatchSettingsTestApp(t *testing.T, autoWatch bool, enabled bool) *App {
	t.Helper()

	root := t.TempDir()
	settingsPath := filepath.Join(root, "library-config.cfg")
	if err := os.WriteFile(settingsPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := storage.NewSQLiteStore(filepath.Join(root, "curated.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	comicRoot := filepath.Join(root, "comics")
	if err := os.MkdirAll(comicRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddComicLibraryPath(context.Background(), comicRoot, "Comics"); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	cfg.ComicLibraryEnabled = enabled
	cfg.AutoComicLibraryWatch = autoWatch
	a := &App{
		cfg:                   cfg,
		logger:                zap.NewNop(),
		store:                 store,
		librarySettingsPath:   settingsPath,
		appCtx:                context.Background(),
		comicLibraryEnabled:   enabled,
		autoComicLibraryWatch: autoWatch,
		comicWatchScanPending: make(map[string]struct{}),
	}
	t.Cleanup(a.StopComicLibraryWatchLoop)
	return a
}

func (a *App) comicWatchLoopRunningForTest() bool {
	a.comicWatchMu.Lock()
	defer a.comicWatchMu.Unlock()
	return a.comicWatchLoopCancel != nil
}

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

func TestPhotoSettingsPersistToLibraryConfig(t *testing.T) {
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

	if err := a.SetPhotoLibraryEnabled(true); err != nil {
		t.Fatal(err)
	}
	if err := a.SetAutoPhotoLibraryWatch(false); err != nil {
		t.Fatal(err)
	}
	if err := a.SetDefaultPhotoImportLibraryPathID("photo-lib-main"); err != nil {
		t.Fatal(err)
	}
	if err := a.SetPhotoViewerSettings(contracts.PhotoViewerSettingsDTO{
		Mode:      "scroll",
		Fit:       "width",
		Direction: "rtl",
	}); err != nil {
		t.Fatal(err)
	}
	if err := a.SetPhotoCacheSettings(contracts.PhotoCacheSettingsDTO{MaxBytes: 5 * 1024 * 1024 * 1024}); err != nil {
		t.Fatal(err)
	}

	if !a.PhotoLibraryEnabled() {
		t.Fatal("expected photo library enabled")
	}
	if a.AutoPhotoLibraryWatch() {
		t.Fatal("expected photo auto watch disabled")
	}
	if got := a.DefaultPhotoImportLibraryPathID(); got != "photo-lib-main" {
		t.Fatalf("DefaultPhotoImportLibraryPathID = %q", got)
	}
	if got := a.PhotoViewerSettings(); got.Mode != "scroll" || got.Fit != "width" || got.Direction != "rtl" {
		t.Fatalf("PhotoViewerSettings = %#v", got)
	}
	if got := a.PhotoCacheSettings(); got.MaxBytes != 5*1024*1024*1024 {
		t.Fatalf("PhotoCacheSettings.MaxBytes = %d", got.MaxBytes)
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
	if persisted["photoLibraryEnabled"] != true {
		t.Fatalf("photoLibraryEnabled not persisted: %#v", persisted["photoLibraryEnabled"])
	}
	if persisted["autoPhotoLibraryWatch"] != false {
		t.Fatalf("autoPhotoLibraryWatch not persisted: %#v", persisted["autoPhotoLibraryWatch"])
	}
	if persisted["defaultPhotoImportLibraryPathId"] != "photo-lib-main" {
		t.Fatalf("defaultPhotoImportLibraryPathId not persisted: %#v", persisted["defaultPhotoImportLibraryPathId"])
	}
	viewer, ok := persisted["photoViewer"].(map[string]any)
	if !ok || viewer["mode"] != "scroll" || viewer["fit"] != "width" || viewer["direction"] != "rtl" {
		t.Fatalf("photoViewer not persisted: %#v", persisted["photoViewer"])
	}
	cache, ok := persisted["photoCache"].(map[string]any)
	if !ok || cache["maxBytes"] != float64(5*1024*1024*1024) {
		t.Fatalf("photoCache not persisted: %#v", persisted["photoCache"])
	}
}

func TestStartPhotoScanWithExtraMetadataIndexesArchive(t *testing.T) {
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
	photoRoot := filepath.Join(root, "photos")
	if err := os.MkdirAll(photoRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeAppTestPhotoZip(filepath.Join(photoRoot, "Watch Photo.cbz"), map[string]string{
		"001.jpg": "page 1",
	}); err != nil {
		t.Fatal(err)
	}
	photoPath, err := store.AddPhotoLibraryPath(context.Background(), photoRoot, "Photos")
	if err != nil {
		t.Fatal(err)
	}
	a := &App{
		store:  store,
		tasks:  tasks.NewManager(),
		appCtx: context.Background(),
	}

	task, err := a.startPhotoScan(context.Background(), []contracts.PhotoLibraryPathDTO{photoPath}, map[string]any{
		"trigger": "fsnotify",
		"paths":   []string{photoRoot},
	})
	if err != nil {
		t.Fatal(err)
	}
	if task.Type != contracts.TaskTypeScanPhotos {
		t.Fatalf("task type = %q, want %q", task.Type, contracts.TaskTypeScanPhotos)
	}
	if task.Metadata["trigger"] != "fsnotify" {
		t.Fatalf("trigger metadata = %#v", task.Metadata["trigger"])
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		got, ok := a.tasks.Get(task.TaskID)
		if ok && got.Status == contracts.TaskCompleted {
			if got.Metadata["trigger"] != "fsnotify" {
				t.Fatalf("completed trigger metadata = %#v", got.Metadata["trigger"])
			}
			page, err := store.ListPhotoBooks(context.Background(), contracts.ListPhotoBooksRequest{Limit: 10})
			if err != nil {
				t.Fatal(err)
			}
			if page.Total != 1 || page.Items[0].Title != "Watch Photo" {
				t.Fatalf("indexed photo books = %#v", page.Items)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for photo scan completion")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestEnqueuePhotoLibraryWatchScanRootsStartsFsnotifyScan(t *testing.T) {
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
	photoRoot := filepath.Join(root, "photos")
	if err := os.MkdirAll(photoRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeAppTestPhotoZip(filepath.Join(photoRoot, "Queued Photo.cbz"), map[string]string{
		"001.jpg": "page 1",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddPhotoLibraryPath(context.Background(), photoRoot, "Photos"); err != nil {
		t.Fatal(err)
	}
	a := &App{
		store:                 store,
		tasks:                 tasks.NewManager(),
		appCtx:                context.Background(),
		photoLibraryEnabled:   true,
		autoPhotoLibraryWatch: true,
		photoWatchScanPending: make(map[string]struct{}),
	}

	a.EnqueuePhotoLibraryWatchScanRoots([]string{photoRoot})

	deadline := time.Now().Add(2 * time.Second)
	for {
		for _, got := range a.tasks.ListRecentFinished(10) {
			if got.Type != contracts.TaskTypeScanPhotos {
				continue
			}
			if got.Status != contracts.TaskCompleted {
				t.Fatalf("photo watch task status = %q, want completed", got.Status)
			}
			if got.Metadata["trigger"] != "fsnotify" {
				t.Fatalf("trigger metadata = %#v, want fsnotify", got.Metadata["trigger"])
			}
			paths, ok := got.Metadata["paths"].([]string)
			if !ok || len(paths) != 1 || paths[0] != photoRoot {
				t.Fatalf("paths metadata = %#v, want [%q]", got.Metadata["paths"], photoRoot)
			}
			page, err := store.ListPhotoBooks(context.Background(), contracts.ListPhotoBooksRequest{Limit: 10})
			if err != nil {
				t.Fatal(err)
			}
			if page.Total != 1 || page.Items[0].Title != "Queued Photo" {
				t.Fatalf("indexed photo books = %#v", page.Items)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for photo watch scan")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestSetAutoPhotoLibraryWatchControlsWatcherLoop(t *testing.T) {
	t.Parallel()

	a := newPhotoWatchSettingsTestApp(t, false, true)
	if err := a.SetAutoPhotoLibraryWatch(true); err != nil {
		t.Fatal(err)
	}
	if !a.photoWatchLoopRunningForTest() {
		t.Fatal("expected photo watcher loop to run after enabling auto photo watch")
	}
	if err := a.SetAutoPhotoLibraryWatch(false); err != nil {
		t.Fatal(err)
	}
	if a.photoWatchLoopRunningForTest() {
		t.Fatal("expected photo watcher loop to stop after disabling auto photo watch")
	}
}

func TestSetPhotoLibraryEnabledControlsWatcherLoop(t *testing.T) {
	t.Parallel()

	a := newPhotoWatchSettingsTestApp(t, true, false)
	if err := a.SetPhotoLibraryEnabled(true); err != nil {
		t.Fatal(err)
	}
	if !a.photoWatchLoopRunningForTest() {
		t.Fatal("expected photo watcher loop to run after enabling photo library")
	}
	if err := a.SetPhotoLibraryEnabled(false); err != nil {
		t.Fatal(err)
	}
	if a.photoWatchLoopRunningForTest() {
		t.Fatal("expected photo watcher loop to stop after disabling photo library")
	}
}

func writeAppTestPhotoZip(path string, entries map[string]string) error {
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

func newPhotoWatchSettingsTestApp(t *testing.T, autoWatch bool, enabled bool) *App {
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
	photoRoot := filepath.Join(root, "photos")
	if err := os.MkdirAll(photoRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddPhotoLibraryPath(context.Background(), photoRoot, "Photos"); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	cfg.PhotoLibraryEnabled = enabled
	cfg.AutoPhotoLibraryWatch = autoWatch
	a := &App{
		cfg:                   cfg,
		logger:                zap.NewNop(),
		store:                 store,
		librarySettingsPath:   settingsPath,
		appCtx:                context.Background(),
		photoLibraryEnabled:   enabled,
		autoPhotoLibraryWatch: autoWatch,
		photoWatchScanPending: make(map[string]struct{}),
	}
	t.Cleanup(a.StopPhotoLibraryWatchLoop)
	return a
}

func (a *App) photoWatchLoopRunningForTest() bool {
	a.photoWatchMu.Lock()
	defer a.photoWatchMu.Unlock()
	return a.photoWatchLoopCancel != nil
}

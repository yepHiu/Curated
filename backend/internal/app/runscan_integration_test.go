package app

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"go.uber.org/zap"

	"jav-shadcn/backend/internal/config"
	"jav-shadcn/backend/internal/contracts"
	"jav-shadcn/backend/internal/storage"
)

// Metatube 引擎在 DSN 为空时使用固定内存库；并行 test 会并发 DBAutoMigrate 导致冲突。
var integrationMetatubeMu sync.Mutex

// 集成测试：真实 SQLite + 临时磁盘目录 + 完整 App（含 Metatube 引擎初始化），
// 验证扩展导入标注、同片根去重、以及首次扫描后 pending 清零。

func decodeScanFileEvents(t *testing.T, buf *bytes.Buffer) (imported, updated, skipped []contracts.ScanFileResultDTO) {
	t.Helper()
	sc := bufio.NewScanner(buf)
	for sc.Scan() {
		line := sc.Bytes()
		var wrap struct {
			Kind    string          `json:"kind"`
			Type    string          `json:"type"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal(line, &wrap); err != nil {
			t.Fatalf("decode line: %v", err)
		}
		if wrap.Kind != "event" || len(wrap.Payload) == 0 {
			continue
		}
		var r contracts.ScanFileResultDTO
		if err := json.Unmarshal(wrap.Payload, &r); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		switch wrap.Type {
		case contracts.EventScanFileImported:
			imported = append(imported, r)
		case contracts.EventScanFileUpdated:
			updated = append(updated, r)
		case contracts.EventScanFileSkipped:
			skipped = append(skipped, r)
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scanner: %v", err)
	}
	return imported, updated, skipped
}

func newTestApp(t *testing.T, store *storage.SQLiteStore, cfg config.Config) *App {
	t.Helper()
	integrationMetatubeMu.Lock()
	defer integrationMetatubeMu.Unlock()
	ctx := context.Background()
	a, err := New(ctx, cfg, zap.NewNop(), store, "")
	if err != nil {
		t.Fatalf("app.New: %v", err)
	}
	return a
}

func startScanTask(a *App, store *storage.SQLiteStore, ctx context.Context, paths []string) string {
	task := a.tasks.Create("scan.library", map[string]any{"paths": paths})
	task = a.tasks.Start(task.TaskID, "integration test scan")
	_ = store.SaveTask(ctx, task)
	return task.TaskID
}

func TestIntegration_RunScan_ImportLayoutCuratedAndClearsPending(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	libRoot := filepath.Join(root, "media")
	if err := os.MkdirAll(filepath.Join(libRoot, "ABC-100"), 0o755); err != nil {
		t.Fatal(err)
	}
	video := filepath.Join(libRoot, "ABC-100", "ABC-100.mp4")
	if err := os.WriteFile(video, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := `{
  "schemaVersion": 1,
  "layout": "curated-movie-root-v1",
  "code": "ABC-100"
}`
	if err := os.WriteFile(filepath.Join(libRoot, "ABC-100", "Curated.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(root, "app.db")
	store, err := storage.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	lp, err := store.AddLibraryPath(ctx, libRoot, "media")
	if err != nil {
		t.Fatal(err)
	}
	if !lp.FirstLibraryScanPending {
		t.Fatal("new library path should have first_library_scan_pending")
	}

	cfg := config.Default()
	cfg.DatabasePath = dbPath
	cfg.CacheDir = filepath.Join(root, "cache")
	cfg.OrganizeLibrary = false
	cfg.ExtendedLibraryImport = true

	a := newTestApp(t, store, cfg)
	taskID := startScanTask(a, store, ctx, []string{libRoot})

	var buf bytes.Buffer
	a.runScan(ctx, &buf, taskID, []string{libRoot})

	imported, _, skipped := decodeScanFileEvents(t, &buf)
	if len(skipped) != 0 {
		t.Fatalf("unexpected skips: %+v", skipped)
	}
	if len(imported) != 1 {
		t.Fatalf("want 1 imported, got imported=%d updated=%d skipped=%d", len(imported), 0, len(skipped))
	}
	if imported[0].ImportLayout != "curated" {
		t.Fatalf("ImportLayout = %q want curated", imported[0].ImportLayout)
	}

	paths, err := store.ListLibraryPaths(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range paths {
		if p.ID == lp.ID && p.FirstLibraryScanPending {
			t.Fatal("first_library_scan_pending should be cleared after successful scan")
		}
	}
}

func TestIntegration_RunScan_ImportLayoutExternal(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	libRoot := filepath.Join(root, "lib")
	if err := os.MkdirAll(filepath.Join(libRoot, "XYZ-999"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(libRoot, "XYZ-999", "XYZ-999.mp4"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(root, "app.db")
	store, err := storage.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddLibraryPath(ctx, libRoot, "lib"); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	cfg.DatabasePath = dbPath
	cfg.CacheDir = filepath.Join(root, "cache")
	cfg.OrganizeLibrary = false
	cfg.ExtendedLibraryImport = true

	a := newTestApp(t, store, cfg)
	taskID := startScanTask(a, store, ctx, []string{libRoot})
	var buf bytes.Buffer
	a.runScan(ctx, &buf, taskID, []string{libRoot})

	imported, _, skipped := decodeScanFileEvents(t, &buf)
	if len(imported) != 1 || len(skipped) != 0 {
		t.Fatalf("imported=%d skipped=%d", len(imported), len(skipped))
	}
	if imported[0].ImportLayout != "external" {
		t.Fatalf("ImportLayout = %q want external", imported[0].ImportLayout)
	}
}

func TestIntegration_RunScan_ExtendedOff_NoImportLayout(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	libRoot := filepath.Join(root, "lib")
	dir := filepath.Join(libRoot, "MIDE-111")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "MIDE-111.mp4"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := `{"schemaVersion":1,"layout":"curated-movie-root-v1","code":"MIDE-111"}`
	if err := os.WriteFile(filepath.Join(dir, "Curated.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(root, "app.db")
	store, err := storage.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddLibraryPath(ctx, libRoot, "lib"); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	cfg.DatabasePath = dbPath
	cfg.CacheDir = filepath.Join(root, "cache")
	cfg.OrganizeLibrary = false
	cfg.ExtendedLibraryImport = false

	a := newTestApp(t, store, cfg)
	taskID := startScanTask(a, store, ctx, []string{libRoot})
	var buf bytes.Buffer
	a.runScan(ctx, &buf, taskID, []string{libRoot})

	imported, _, _ := decodeScanFileEvents(t, &buf)
	if len(imported) != 1 {
		t.Fatalf("want 1 imported, got %d", len(imported))
	}
	if imported[0].ImportLayout != "" {
		t.Fatalf("ImportLayout should be empty when extended import off, got %q", imported[0].ImportLayout)
	}
}

func TestIntegration_RunScan_DuplicateMovieRootSecondSkipped(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	libRoot := filepath.Join(root, "lib")
	dir := filepath.Join(libRoot, "SSIS-222")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SSIS-222.mp4"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "prefix.SSIS-222.suffix.mp4"), []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(root, "app.db")
	store, err := storage.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddLibraryPath(ctx, libRoot, "lib"); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	cfg.DatabasePath = dbPath
	cfg.CacheDir = filepath.Join(root, "cache")
	cfg.OrganizeLibrary = false
	cfg.ExtendedLibraryImport = true

	a := newTestApp(t, store, cfg)
	taskID := startScanTask(a, store, ctx, []string{libRoot})
	var buf bytes.Buffer
	a.runScan(ctx, &buf, taskID, []string{libRoot})

	imported, _, skipped := decodeScanFileEvents(t, &buf)
	if len(imported) != 1 {
		t.Fatalf("want 1 imported, got %d", len(imported))
	}
	if len(skipped) != 1 {
		t.Fatalf("want 1 skipped (duplicate), got %d", len(skipped))
	}
	if skipped[0].Reason != "duplicate_movie_root" {
		t.Fatalf("skip reason = %q", skipped[0].Reason)
	}
}

func TestIntegration_ClearFirstLibraryScanPendingAfterScan_Storage(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dbPath := filepath.Join(root, "t.db")
	store, err := storage.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	lib := filepath.Join(root, "L")
	if err := os.MkdirAll(lib, 0o755); err != nil {
		t.Fatal(err)
	}
	lp, err := store.AddLibraryPath(ctx, lib, "L")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ClearFirstLibraryScanPendingAfterScan(ctx, []string{filepath.Join(lib, "sub")}); err != nil {
		t.Fatal(err)
	}
	rows, err := store.ListLibraryPaths(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range rows {
		if p.ID == lp.ID && p.FirstLibraryScanPending {
			t.Fatal("expected pending cleared for scan root under library path")
		}
	}
}

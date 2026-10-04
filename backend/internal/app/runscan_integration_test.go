package app

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"go.uber.org/zap"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
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

func TestIntegration_RunScan_LegacyExtendedLibraryImportIgnoredAndClearsPending(t *testing.T) {
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
	settingsPath := filepath.Join(root, "library-config.cfg")
	if err := os.WriteFile(settingsPath, []byte(`{"extendedLibraryImport": true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := config.MergeLibrarySettingsFile(&cfg, settingsPath); err != nil {
		t.Fatal(err)
	}

	a := newTestApp(t, store, cfg)
	defer a.Close() // Drain background tasks before closing SQLite and deleting the fixture.
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
	if imported[0].ImportLayout != "" {
		t.Fatalf("ImportLayout should stay empty after legacy setting is ignored, got %q", imported[0].ImportLayout)
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

func TestIntegration_RunScan_LegacyExtendedLibraryImportIgnoredForExternalLayout(t *testing.T) {
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
	settingsPath := filepath.Join(root, "library-config.cfg")
	if err := os.WriteFile(settingsPath, []byte(`{"extendedLibraryImport": true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := config.MergeLibrarySettingsFile(&cfg, settingsPath); err != nil {
		t.Fatal(err)
	}

	a := newTestApp(t, store, cfg)
	defer a.Close() // Drain background tasks before closing SQLite and deleting the fixture.
	taskID := startScanTask(a, store, ctx, []string{libRoot})
	var buf bytes.Buffer
	a.runScan(ctx, &buf, taskID, []string{libRoot})

	imported, _, skipped := decodeScanFileEvents(t, &buf)
	if len(imported) != 1 || len(skipped) != 0 {
		t.Fatalf("imported=%d skipped=%d", len(imported), len(skipped))
	}
	if imported[0].ImportLayout != "" {
		t.Fatalf("ImportLayout should stay empty after legacy setting is ignored, got %q", imported[0].ImportLayout)
	}
}

func TestIntegration_RunScan_NoImportLayoutWithoutLegacySetting(t *testing.T) {
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

	a := newTestApp(t, store, cfg)
	defer a.Close() // Drain background tasks before closing SQLite and deleting the fixture.
	taskID := startScanTask(a, store, ctx, []string{libRoot})
	var buf bytes.Buffer
	a.runScan(ctx, &buf, taskID, []string{libRoot})

	imported, _, _ := decodeScanFileEvents(t, &buf)
	if len(imported) != 1 {
		t.Fatalf("want 1 imported, got %d", len(imported))
	}
	if imported[0].ImportLayout != "" {
		t.Fatalf("ImportLayout should be empty, got %q", imported[0].ImportLayout)
	}
}

// TestIntegration_RunScan_SameMovieRootKeepsEveryFile 验证本次浏览类别及文件归属的兼容行为。
func TestIntegration_RunScan_SameMovieRootKeepsEveryFile(t *testing.T) {
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

	a := newTestApp(t, store, cfg)
	defer a.Close() // Drain background tasks before closing SQLite and deleting the fixture.
	taskID := startScanTask(a, store, ctx, []string{libRoot})
	var buf bytes.Buffer
	a.runScan(ctx, &buf, taskID, []string{libRoot})

	imported, updated, skipped := decodeScanFileEvents(t, &buf)
	if len(imported) != 1 {
		t.Fatalf("want 1 imported, got %d", len(imported))
	}
	if len(skipped) != 0 || len(updated) != 1 {
		t.Fatalf("updated=%v skipped=%v", updated, skipped)
	}
	detail, err := store.GetMovieDetail(ctx, imported[0].MovieID)
	if err != nil || len(detail.Files) != 2 {
		t.Fatalf("files=%v err=%v", detail.Files, err)
	}
}

func TestIntegration_RunScan_MoreThan255MovieDirectories(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	libRoot := filepath.Join(root, "lib")
	const movieCount = 260
	for i := 1; i <= movieCount; i++ {
		code := "ABCD-" + fmt.Sprintf("%03d", i)
		dir := filepath.Join(libRoot, code)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, code+".mp4"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
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

	a := newTestApp(t, store, cfg)
	defer a.Close() // Drain background tasks before closing SQLite and deleting the fixture.
	taskID := startScanTask(a, store, ctx, []string{libRoot})
	var buf bytes.Buffer
	a.runScan(ctx, &buf, taskID, []string{libRoot})

	imported, _, skipped := decodeScanFileEvents(t, &buf)
	if len(skipped) != 0 {
		t.Fatalf("unexpected skipped files: got %d first=%+v", len(skipped), skipped[0])
	}
	if len(imported) != movieCount {
		t.Fatalf("imported=%d, want %d", len(imported), movieCount)
	}
	page, err := store.ListMovies(ctx, contracts.ListMoviesRequest{Limit: movieCount})
	if err != nil {
		t.Fatalf("list movies after scan: %v", err)
	}
	if page.Total != movieCount || len(page.Items) != movieCount {
		t.Fatalf("movie page total=%d items=%d, want %d", page.Total, len(page.Items), movieCount)
	}
}

func TestIntegration_RunScan_TrashedLocationDoesNotAbortLaterImports(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	libRoot := filepath.Join(root, "lib")
	if err := os.MkdirAll(filepath.Join(libRoot, "ABC-100"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(libRoot, "SIRO-5705"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(libRoot, "ABC-100", "ABC-100.mp4"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	siroPath := filepath.Join(libRoot, "SIRO-5705", "SIRO-5705.mp4")
	if err := os.WriteFile(siroPath, []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}
	sonePath := filepath.Join(libRoot, "SONE-305-C.mp4")
	if err := os.WriteFile(sonePath, []byte("c"), 0o644); err != nil {
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

	trashed, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
		Path:     siroPath,
		FileName: "SIRO-5705.mp4",
		Number:   "SIRO-5705",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.TrashMovie(ctx, trashed.MovieID); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	cfg.DatabasePath = dbPath
	cfg.CacheDir = filepath.Join(root, "cache")
	cfg.OrganizeLibrary = false

	a := newTestApp(t, store, cfg)
	defer a.Close() // Drain background tasks before closing SQLite and deleting the fixture.
	taskID := startScanTask(a, store, ctx, []string{libRoot})
	var buf bytes.Buffer
	a.runScan(ctx, &buf, taskID, []string{libRoot})

	imported, _, skipped := decodeScanFileEvents(t, &buf)
	if len(imported) != 2 {
		t.Fatalf("imported=%d skipped=%d imported=%+v skipped=%+v", len(imported), len(skipped), imported, skipped)
	}
	gotCodes := map[string]struct{}{}
	for _, item := range imported {
		gotCodes[item.Number] = struct{}{}
	}
	if _, ok := gotCodes["ABC-100"]; !ok {
		t.Fatalf("missing ABC-100 import: %+v", imported)
	}
	if _, ok := gotCodes["SONE-305"]; !ok {
		t.Fatalf("missing SONE-305 import: %+v", imported)
	}

	task, ok := a.tasks.Get(taskID)
	if !ok {
		t.Fatal("scan task missing")
	}
	if task.Status != "completed" {
		t.Fatalf("scan status=%q error=%s %s", task.Status, task.ErrorCode, task.ErrorMessage)
	}

	trashedStill, err := store.IsMovieTrashed(ctx, trashed.MovieID)
	if err != nil {
		t.Fatal(err)
	}
	if !trashedStill {
		t.Fatal("scan must not restore the trashed movie")
	}
}

// TestIntegration_ClearFirstLibraryScanPendingAfterScan_Storage 验证本次浏览类别及文件归属的兼容行为。
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

// TestIntegration_RunScan_MultipartOrganize confirms the complete scan path keeps each ordered part.
func TestIntegration_RunScan_MultipartOrganize(t *testing.T) {
	root := t.TempDir()
	libRoot := filepath.Join(root, "lib")
	if err := os.MkdirAll(libRoot, 0755); err != nil {
		t.Fatal(err)
	}
	for _, part := range []int{10, 2, 1} {
		if err := os.WriteFile(filepath.Join(libRoot, fmt.Sprintf("FC2-1234567_%d.mp4", part)), []byte("fixture"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	store, err := storage.NewSQLiteStore(filepath.Join(root, "app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddLibraryPath(ctx, libRoot, "lib"); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.DatabasePath = filepath.Join(root, "app.db")
	cfg.CacheDir = filepath.Join(root, "cache")
	cfg.OrganizeLibrary = true
	a := newTestApp(t, store, cfg)
	defer a.Close()
	taskID := startScanTask(a, store, ctx, []string{libRoot})
	var output bytes.Buffer
	a.runScan(ctx, &output, taskID, []string{libRoot})
	imported, updated, skipped := decodeScanFileEvents(t, &output)
	if len(imported) != 1 || len(updated) != 2 || len(skipped) != 0 {
		t.Fatalf("imported=%d updated=%d skipped=%v", len(imported), len(updated), skipped)
	}
	detail, err := store.GetMovieDetail(ctx, imported[0].MovieID)
	if err != nil || len(detail.Files) != 3 {
		t.Fatalf("files=%v err=%v", detail.Files, err)
	}
	for i, want := range []int{1, 2, 10} {
		if detail.Files[i].PartIndex != want {
			t.Fatalf("order=%v", detail.Files)
		}
		if _, err := os.Stat(detail.Files[i].Location); err != nil {
			t.Fatal(err)
		}
	}
}

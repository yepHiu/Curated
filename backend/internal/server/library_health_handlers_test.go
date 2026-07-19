package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

type libraryHealthStorageProvider struct {
	items []contracts.LibraryPathStorageStatusDTO
}

func (p *libraryHealthStorageProvider) ListLibraryPathStorageStatus(context.Context) (contracts.LibraryPathStorageStatusListDTO, error) {
	return contracts.LibraryPathStorageStatusListDTO{Items: append([]contracts.LibraryPathStorageStatusDTO(nil), p.items...)}, nil
}

func (p *libraryHealthStorageProvider) CheckLibraryPathStorageStatus(context.Context, []string) (contracts.LibraryPathStorageStatusListDTO, error) {
	return contracts.LibraryPathStorageStatusListDTO{Items: append([]contracts.LibraryPathStorageStatusDTO(nil), p.items...)}, nil
}

func (p *libraryHealthStorageProvider) RebindLibraryPathStorage(context.Context, string) (contracts.LibraryPathStorageStatusDTO, error) {
	return contracts.LibraryPathStorageStatusDTO{}, nil
}

func TestLibraryHealthScanReportsFilesMetadataAndScopedStagingWithoutMutation(t *testing.T) {
	root := t.TempDir()
	store, err := storage.NewSQLiteStore(filepath.Join(t.TempDir(), "health-api.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	validPath := filepath.Join(root, "HEALTH-OK.mp4")
	if err := os.WriteFile(validPath, []byte("video"), 0o644); err != nil {
		t.Fatal(err)
	}
	emptyPath := filepath.Join(root, "HEALTH-EMPTY.mp4")
	if err := os.WriteFile(emptyPath, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	missingPath := filepath.Join(root, "HEALTH-MISSING.mp4")
	for _, item := range []struct{ code, path string }{
		{"HEALTH-OK", validPath}, {"HEALTH-EMPTY", emptyPath}, {"HEALTH-MISSING", missingPath},
	} {
		if _, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
			TaskID: "scan-" + item.code, Path: item.path, FileName: filepath.Base(item.path), Number: item.code,
		}); err != nil {
			t.Fatal(err)
		}
	}
	orphanStaging := filepath.Join(root, movieImportUploadStagingDirName, "upload_0123456789abcdef")
	if err := os.MkdirAll(orphanStaging, 0o755); err != nil {
		t.Fatal(err)
	}

	provider := &libraryHealthStorageProvider{items: []contracts.LibraryPathStorageStatusDTO{{
		LibraryPathID: "library-health", Path: root, Title: "Health Library",
		Status: contracts.LibraryPathStorageStatusOnline, Message: "online", CanRescan: true, CanImport: true,
	}}}
	h := NewHandler(Deps{Cfg: config.Default(), Logger: zap.NewNop(), Store: store, LibraryPathStorageStatusProvider: provider})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	report := requestLibraryHealthReport(t, srv.URL+"/api/library/health/scan")
	if !report.Database.QuickCheckOK || !report.Database.ForeignKeyOK {
		t.Fatalf("database health = %#v", report.Database)
	}
	for _, category := range []string{"source_missing", "source_empty", "metadata_missing", "import_staging_residue"} {
		if report.Summary.CategoryCounts[category] == 0 {
			t.Fatalf("missing category %s in %#v", category, report.Summary.CategoryCounts)
		}
	}
	if _, err := os.Stat(orphanStaging); err != nil {
		t.Fatalf("read-only scan mutated staging directory: %v", err)
	}

	second := requestLibraryHealthReport(t, srv.URL+"/api/library/health/scan")
	firstIDs := findingIDsByCategory(report.Findings)
	secondIDs := findingIDsByCategory(second.Findings)
	for category, id := range firstIDs {
		if secondIDs[category] != id {
			t.Fatalf("finding id for %s changed: %q -> %q", category, id, secondIDs[category])
		}
	}

	limited := requestLibraryHealthReport(t, srv.URL+"/api/library/health/scan?findingLimit=1")
	if !limited.Truncated || len(limited.Findings) != 1 || limited.Summary.TotalFindings <= 1 {
		t.Fatalf("limited report = %#v", limited)
	}
}

func TestLibraryHealthScanDoesNotCallOfflineMovieMissing(t *testing.T) {
	root := filepath.Join(t.TempDir(), "offline-root")
	store, err := storage.NewSQLiteStore(filepath.Join(t.TempDir(), "health-offline.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	moviePath := filepath.Join(root, "OFFLINE-001.mp4")
	if _, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
		TaskID: "scan-offline", Path: moviePath, FileName: filepath.Base(moviePath), Number: "OFFLINE-001",
	}); err != nil {
		t.Fatal(err)
	}
	provider := &libraryHealthStorageProvider{items: []contracts.LibraryPathStorageStatusDTO{{
		LibraryPathID: "library-offline", Path: root, Title: "Offline Library",
		Status: contracts.LibraryPathStorageStatusOffline, Message: "storage is offline",
	}}}
	h := NewHandler(Deps{Cfg: config.Default(), Logger: zap.NewNop(), Store: store, LibraryPathStorageStatusProvider: provider})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	report := requestLibraryHealthReport(t, srv.URL+"/api/library/health/scan")
	if report.Summary.CategoryCounts["storage_unavailable"] != 1 {
		t.Fatalf("storage counts = %#v", report.Summary.CategoryCounts)
	}
	if report.Summary.CategoryCounts["source_missing"] != 0 {
		t.Fatalf("offline source was misreported missing: %#v", report.Summary.CategoryCounts)
	}
	if report.Summary.SkippedOfflineFiles != 1 {
		t.Fatalf("skipped offline files = %d, want 1", report.Summary.SkippedOfflineFiles)
	}
}

func requestLibraryHealthReport(t *testing.T, url string) contracts.LibraryHealthReportDTO {
	t.Helper()
	resp, err := http.Post(url, "application/json", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var report contracts.LibraryHealthReportDTO
	if err := json.NewDecoder(resp.Body).Decode(&report); err != nil {
		t.Fatal(err)
	}
	return report
}

func findingIDsByCategory(findings []contracts.LibraryHealthFindingDTO) map[string]string {
	out := make(map[string]string)
	for _, finding := range findings {
		if _, exists := out[finding.Category]; !exists {
			out[finding.Category] = finding.ID
		}
	}
	return out
}

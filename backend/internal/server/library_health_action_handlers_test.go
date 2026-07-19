package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
	"curated-backend/internal/tasks"
)

func TestConfirmedLibraryHealthActionCleansOnlyExactOrphanStaging(t *testing.T) {
	root := t.TempDir()
	store, err := storage.NewSQLiteStore(filepath.Join(t.TempDir(), "health-action.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddLibraryPath(ctx, root, "Health Actions"); err != nil {
		t.Fatal(err)
	}
	orphan := filepath.Join(root, movieImportUploadStagingDirName, "upload_0123456789abcdef")
	unrelated := filepath.Join(root, movieImportUploadStagingDirName, ".unrelated")
	if err := os.MkdirAll(orphan, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(unrelated, 0o755); err != nil {
		t.Fatal(err)
	}

	taskManager := tasks.NewManager()
	provider := &libraryHealthStorageProvider{items: []contracts.LibraryPathStorageStatusDTO{{
		LibraryPathID: "library-actions", Path: root, Title: "Health Actions",
		Status: contracts.LibraryPathStorageStatusOnline, Message: "online", CanRescan: true, CanImport: true,
	}}}
	h := NewHandler(Deps{
		RuntimeContext: context.Background(), Cfg: config.Default(), Logger: zap.NewNop(), Store: store,
		Tasks: taskManager, LibraryPathStorageStatusProvider: provider,
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	report := requestLibraryHealthReport(t, srv.URL+"/api/library/health/scan")
	var findingID string
	for _, finding := range report.Findings {
		if finding.Path == orphan && slicesContainsString(finding.RepairActions, "cleanup_import_staging") {
			findingID = finding.ID
			break
		}
	}
	if findingID == "" {
		t.Fatalf("orphan staging finding missing: %#v", report.Findings)
	}

	unconfirmed := map[string]any{
		"action": "cleanup_import_staging", "findingIds": []string{findingID}, "confirm": false,
	}
	body, _ := json.Marshal(unconfirmed)
	resp, err := http.Post(srv.URL+"/api/library/health/actions", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("unconfirmed status = %d", resp.StatusCode)
	}
	if _, err := os.Stat(orphan); err != nil {
		t.Fatalf("unconfirmed cleanup mutated orphan: %v", err)
	}

	confirmed := map[string]any{
		"action": "cleanup_import_staging", "findingIds": []string{findingID}, "confirm": true,
	}
	body, _ = json.Marshal(confirmed)
	resp, err = http.Post(srv.URL+"/api/library/health/actions", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusAccepted {
		_ = resp.Body.Close()
		t.Fatalf("confirmed status = %d", resp.StatusCode)
	}
	var task contracts.TaskDTO
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		_ = resp.Body.Close()
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	finished := waitForTask(t, srv.URL, task.TaskID)
	if finished.Status != contracts.TaskCompleted {
		t.Fatalf("cleanup task = %#v", finished)
	}
	if _, err := os.Stat(orphan); !os.IsNotExist(err) {
		t.Fatalf("orphan staging still exists: %v", err)
	}
	if _, err := os.Stat(unrelated); err != nil {
		t.Fatalf("unrelated staging was removed: %v", err)
	}
	audits, err := store.ListMovieImportUploadCleanupAudits(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(audits) != 1 || audits[0].Reason != "health_confirmed_orphan_staging" || audits[0].Outcome != "removed" {
		t.Fatalf("cleanup audits = %#v", audits)
	}

	body, _ = json.Marshal(confirmed)
	resp, err = http.Post(srv.URL+"/api/library/health/actions", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("stale finding status = %d, want 409", resp.StatusCode)
	}
}

func waitForTask(t *testing.T, baseURL, taskID string) contracts.TaskDTO {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(baseURL + "/api/tasks/" + taskID)
		if err != nil {
			t.Fatal(err)
		}
		var task contracts.TaskDTO
		decodeErr := json.NewDecoder(resp.Body).Decode(&task)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK || decodeErr != nil {
			t.Fatalf("task status=%d decode=%v", resp.StatusCode, decodeErr)
		}
		if isTerminalTaskStatus(task.Status) {
			return task
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("timed out waiting for task")
	return contracts.TaskDTO{}
}

func slicesContainsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

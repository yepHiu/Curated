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
	"time"

	"curated-backend/internal/backup"
	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

func TestBackupFileUploadRetainsOnlyVerifiedPackages(t *testing.T) {
	for _, tc := range []struct {
		name    string
		valid   bool
		failure error
	}{
		{name: "valid", valid: true}, {name: "invalid"}, {name: "unreadable", failure: errors.New("bad zip")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			directory := t.TempDir()
			provider := &backupProviderStub{preflightResult: contracts.BackupRestorePreflightDTO{Verification: contracts.BackupVerificationDTO{Valid: tc.valid}}, preflightErr: tc.failure}
			h := NewHandler(Deps{Cfg: config.Config{BackupDirectory: directory}, BackupProvider: provider})
			payload := &bytes.Buffer{}
			writer := multipart.NewWriter(payload)
			part, err := writer.CreateFormFile("file", "../../selected.curated-backup")
			if err != nil {
				t.Fatal(err)
			}
			part.Write([]byte("archive bytes"))
			writer.Close()
			req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/api/maintenance/backups/inspect-file", payload)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			req.RemoteAddr = "127.0.0.1:1234"
			recorder := httptest.NewRecorder()
			h.Routes().ServeHTTP(recorder, req)
			wantStatus := http.StatusOK
			if tc.failure != nil {
				wantStatus = http.StatusBadRequest
			}
			if recorder.Code != wantStatus {
				t.Fatalf("status %d: %s", recorder.Code, recorder.Body.String())
			}
			entries, err := os.ReadDir(filepath.Join(directory, "imports"))
			if err != nil {
				t.Fatal(err)
			}
			if !tc.valid {
				if len(entries) != 0 {
					t.Fatal("invalid upload was retained")
				}
				return
			}
			var result contracts.BackupFileInspectionDTO
			if err := json.NewDecoder(recorder.Body).Decode(&result); err != nil {
				t.Fatal(err)
			}
			if len(entries) != 1 || filepath.Dir(result.BackupPath) != filepath.Join(directory, "imports") || !strings.HasSuffix(result.BackupPath, ".curated-backup") {
				t.Fatalf("unexpected retained path: %s", result.BackupPath)
			}
			data, err := os.ReadFile(result.BackupPath)
			if err != nil || string(data) != "archive bytes" {
				t.Fatalf("retained content: %s %v", data, err)
			}
			if _, err := os.Stat(provider.preflightPath); !os.IsNotExist(err) {
				t.Fatal("temporary path was not removed")
			}
		})
	}
}

func TestLatestBackupUsesConfiguredFolderAndImports(t *testing.T) {
	directory := t.TempDir()
	if err := os.Mkdir(filepath.Join(directory, "imports"), 0700); err != nil {
		t.Fatal(err)
	}
	older := filepath.Join(directory, "older.curated-backup")
	newest := filepath.Join(directory, "imports", "newer.curated-backup")
	for _, path := range []string{older, newest, filepath.Join(directory, "ignored.tmp")} {
		if err := os.WriteFile(path, []byte("backup"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(older, old, old); err != nil {
		t.Fatal(err)
	}
	h := NewHandler(Deps{Cfg: config.Config{BackupDirectory: directory}})
	recorder := httptest.NewRecorder()
	h.handleLatestBackup(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	var result contracts.BackupFileSelectionDTO
	if err := json.NewDecoder(recorder.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.BackupPath != newest {
		t.Fatalf("latest = %q", result.BackupPath)
	}
}

func TestBackupFileRoutesRejectRemoteClients(t *testing.T) {
	h := NewHandler(Deps{})
	for _, tc := range []struct{ method, path string }{{http.MethodGet, "/api/maintenance/backups/latest"}, {http.MethodPost, "/api/maintenance/backups/inspect-file"}} {
		req := httptest.NewRequest(tc.method, "http://192.168.1.5"+tc.path, nil)
		req.RemoteAddr = "192.168.1.6:1234"
		recorder := httptest.NewRecorder()
		h.Routes().ServeHTTP(recorder, req)
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("remote %s: %d", tc.path, recorder.Code)
		}
		if isAuthPublicPath(tc.method, tc.path) {
			t.Fatalf("public route: %s", tc.path)
		}
	}
}

func TestBackupFileUploadRejectsEmptyAndAdditionalParts(t *testing.T) {
	for _, tc := range []struct {
		name, filename, contents string
		extra                    bool
	}{
		{name: "empty", filename: "empty.curated-backup"},
		{name: "extension", filename: "other.txt", contents: "data"},
		{name: "extra", filename: "one.curated-backup", contents: "data", extra: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			directory := t.TempDir()
			provider := &backupProviderStub{}
			h := NewHandler(Deps{Cfg: config.Config{BackupDirectory: directory}, BackupProvider: provider})
			payload := &bytes.Buffer{}
			writer := multipart.NewWriter(payload)
			part, err := writer.CreateFormFile("file", tc.filename)
			if err != nil {
				t.Fatal(err)
			}
			io.WriteString(part, tc.contents)
			if tc.extra {
				writer.WriteField("unexpected", "value")
			}
			writer.Close()
			req := httptest.NewRequest(http.MethodPost, "/", payload)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			recorder := httptest.NewRecorder()
			h.handleInspectBackupFile(recorder, req)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status %d", recorder.Code)
			}
			if provider.preflightPath != "" {
				t.Fatal("invalid upload reached verification")
			}
			entries, _ := os.ReadDir(filepath.Join(directory, "imports"))
			if len(entries) != 0 {
				t.Fatal("rejected upload left a file")
			}
		})
	}
}

// Exercise multipart transport against the actual ZIP/SQLite preflight code.
type realBackupInspectionProvider struct {
	backupProviderStub
	targetDatabase string
}

func (p *realBackupInspectionProvider) PreflightBackupRestore(ctx context.Context, path string) (contracts.BackupRestorePreflightDTO, error) {
	result, err := backup.PreflightRestore(ctx, backup.PreflightOptions{BackupPath: path, TargetDatabase: p.targetDatabase})
	if err != nil {
		return contracts.BackupRestorePreflightDTO{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return contracts.BackupRestorePreflightDTO{}, err
	}
	var dto contracts.BackupRestorePreflightDTO
	err = json.Unmarshal(encoded, &dto)
	return dto, err
}

func TestSelectedBackupFileRealPreflight(t *testing.T) {
	root := t.TempDir()
	store, err := storage.NewSQLiteStore(filepath.Join(root, "source.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	original := filepath.Join(root, "original.curated-backup")
	if _, err := backup.Create(context.Background(), backup.CreateOptions{Store: store, DestinationPath: original}); err != nil {
		t.Fatal(err)
	}
	source, err := os.Open(original)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "selected.curated-backup")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(part, source); err != nil {
		t.Fatal(err)
	}
	writer.Close()
	target := filepath.Join(root, "restored.db")
	provider := &realBackupInspectionProvider{targetDatabase: target}
	h := NewHandler(Deps{Cfg: config.Config{BackupDirectory: filepath.Join(root, "backups")}, BackupProvider: provider})
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	h.handleInspectBackupFile(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("status %d: %s", response.Code, response.Body.String())
	}
	var result contracts.BackupFileInspectionDTO
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if !result.Preflight.CanRestore || !result.Preflight.Verification.Valid || result.BackupPath == "" {
		t.Fatalf("preflight: %+v", result)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatal("inspection wrote the live database")
	}
	verification, err := backup.Verify(context.Background(), result.BackupPath)
	if err != nil || !verification.Valid {
		t.Fatalf("retained package invalid: %v", err)
	}
}

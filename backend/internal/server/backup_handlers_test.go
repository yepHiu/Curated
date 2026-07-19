package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"curated-backend/internal/contracts"
)

type backupProviderStub struct {
	createPath      string
	verifyPath      string
	preflightPath   string
	createResult    contracts.BackupManifestDTO
	verifyResult    contracts.BackupVerificationDTO
	preflightResult contracts.BackupRestorePreflightDTO
	createErr       error
	verifyErr       error
	preflightErr    error
}

func (s *backupProviderStub) CreateBackup(_ context.Context, destinationPath string) (contracts.BackupManifestDTO, error) {
	s.createPath = destinationPath
	return s.createResult, s.createErr
}

func (s *backupProviderStub) VerifyBackup(_ context.Context, backupPath string) (contracts.BackupVerificationDTO, error) {
	s.verifyPath = backupPath
	return s.verifyResult, s.verifyErr
}

func (s *backupProviderStub) PreflightBackupRestore(_ context.Context, backupPath string) (contracts.BackupRestorePreflightDTO, error) {
	s.preflightPath = backupPath
	return s.preflightResult, s.preflightErr
}

func TestBackupHandlersCreateVerifyAndPreflight(t *testing.T) {
	provider := &backupProviderStub{
		createResult:    contracts.BackupManifestDTO{Format: "curated-backup", FormatVersion: 1},
		verifyResult:    contracts.BackupVerificationDTO{Valid: true, Errors: []string{}, Warnings: []string{}},
		preflightResult: contracts.BackupRestorePreflightDTO{CanRestore: true, Errors: []string{}, Warnings: []string{}},
	}
	server := httptest.NewServer(NewHandler(Deps{BackupProvider: provider}).Routes())
	defer server.Close()
	backupPath := filepath.Join(t.TempDir(), "curated.curated-backup")

	createResponse := postJSONValue(t, server.URL+"/api/maintenance/backups", map[string]string{"destinationPath": "  " + backupPath + "  "})
	defer createResponse.Body.Close()
	if createResponse.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d", createResponse.StatusCode)
	}
	if provider.createPath != backupPath {
		t.Fatalf("create path = %q", provider.createPath)
	}

	verifyResponse := postJSONValue(t, server.URL+"/api/maintenance/backups/verify", map[string]string{"backupPath": "  " + backupPath + "  "})
	defer verifyResponse.Body.Close()
	if verifyResponse.StatusCode != http.StatusOK || provider.verifyPath == "" {
		t.Fatalf("verify status = %d path = %q", verifyResponse.StatusCode, provider.verifyPath)
	}

	preflightResponse := postJSONValue(t, server.URL+"/api/maintenance/backups/preflight", map[string]string{"backupPath": "  " + backupPath + "  "})
	defer preflightResponse.Body.Close()
	if preflightResponse.StatusCode != http.StatusOK || provider.preflightPath == "" {
		t.Fatalf("preflight status = %d path = %q", preflightResponse.StatusCode, provider.preflightPath)
	}

	for _, path := range []string{
		"/api/maintenance/backups",
		"/api/maintenance/backups/verify",
		"/api/maintenance/backups/preflight",
	} {
		if isAuthPublicPath(http.MethodPost, path) {
			t.Fatalf("backup route unexpectedly public: %s", path)
		}
	}
}

func TestBackupHandlersRejectInvalidAndConflictingRequests(t *testing.T) {
	provider := &backupProviderStub{createErr: contracts.ErrBackupDestinationExists}
	server := httptest.NewServer(NewHandler(Deps{BackupProvider: provider}).Routes())
	defer server.Close()

	invalid := postJSON(t, server.URL+"/api/maintenance/backups", `{"destinationPath":"","unexpected":true}`)
	defer invalid.Body.Close()
	if invalid.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid status = %d", invalid.StatusCode)
	}
	assertAPIErrorCode(t, invalid, contracts.ErrorCodeBackupInvalidRequest)

	backupPath := filepath.Join(t.TempDir(), "existing.curated-backup")
	conflict := postJSONValue(t, server.URL+"/api/maintenance/backups", map[string]string{"destinationPath": backupPath})
	defer conflict.Body.Close()
	if conflict.StatusCode != http.StatusConflict {
		t.Fatalf("conflict status = %d", conflict.StatusCode)
	}
	assertAPIErrorCode(t, conflict, contracts.ErrorCodeBackupConflict)

	provider.verifyErr = errors.New("unreadable")
	verify := postJSONValue(t, server.URL+"/api/maintenance/backups/verify", map[string]string{"backupPath": filepath.Join(t.TempDir(), "missing.curated-backup")})
	defer verify.Body.Close()
	if verify.StatusCode != http.StatusBadRequest {
		t.Fatalf("verify error status = %d", verify.StatusCode)
	}
	assertAPIErrorCode(t, verify, contracts.ErrorCodeBackupVerifyFailed)
}

func postJSON(t *testing.T, url, body string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	return response
}

func postJSONValue(t *testing.T, url string, value any) *http.Response {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	return postJSON(t, url, string(body))
}

func assertAPIErrorCode(t *testing.T, response *http.Response, want string) {
	t.Helper()
	var body contracts.AppError
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if body.Code != want {
		t.Fatalf("error code = %q, want %q", body.Code, want)
	}
}

package app

import (
	"encoding/json"
	"testing"
	"time"

	"curated-backend/internal/backup"
	"curated-backend/internal/storage"
)

func TestBackupDTOsEncodeRequiredCollectionsAsArrays(t *testing.T) {
	verification := backup.Verification{
		Valid:     true,
		CheckedAt: time.Date(2026, time.July, 25, 8, 20, 38, 0, time.UTC),
		Manifest: &backup.Manifest{
			Format:        backup.FormatName,
			FormatVersion: backup.FormatVersion,
			CreatedAt:     time.Date(2026, time.July, 25, 8, 19, 35, 0, time.UTC),
		},
		DatabaseIntegrity: storage.IntegrityReport{
			QuickCheck:           "ok",
			ForeignKeyViolations: 0,
		},
	}

	assertBackupJSONShape(t, backupVerificationDTO(verification), []string{
		"errors",
		"warnings",
	})
	assertBackupJSONShape(t, backupPreflightDTO(backup.RestorePreflight{
		CheckedAt:    verification.CheckedAt,
		Verification: verification,
	}), []string{
		"unsupportedMigrations",
		"errors",
		"warnings",
	})
}

func assertBackupJSONShape(t *testing.T, value any, arrayFields []string) {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal backup DTO: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatalf("decode backup DTO: %v", err)
	}
	for _, field := range arrayFields {
		if _, ok := payload[field].([]any); !ok {
			t.Fatalf("%s must encode as an array, payload = %s", field, encoded)
		}
	}

	verificationPayload := payload
	if nested, ok := payload["verification"].(map[string]any); ok {
		verificationPayload = nested
	}
	manifest, ok := verificationPayload["manifest"].(map[string]any)
	if !ok {
		t.Fatalf("manifest must encode as an object, payload = %s", encoded)
	}
	if _, ok := manifest["schemaMigrations"].([]any); !ok {
		t.Fatalf("schemaMigrations must encode as an array, payload = %s", encoded)
	}
	integrity, ok := verificationPayload["databaseIntegrity"].(map[string]any)
	if !ok {
		t.Fatalf("databaseIntegrity must encode as an object, payload = %s", encoded)
	}
	if integrity["quickCheck"] != "ok" {
		t.Fatalf("quickCheck = %#v, payload = %s", integrity["quickCheck"], encoded)
	}
	if _, exists := integrity["QuickCheck"]; exists {
		t.Fatalf("databaseIntegrity must use camelCase keys, payload = %s", encoded)
	}
}

package maintenance

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"curated-backend/internal/processlock"
	"curated-backend/internal/storage"
)

func TestRunCreatesAndVerifiesBackup(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	databasePath := filepath.Join(root, "curated.db")
	store, err := storage.NewSQLiteStore(databasePath)
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	configPath := filepath.Join(root, "library-config.cfg")
	if err := os.WriteFile(configPath, []byte("{}\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	backupPath := filepath.Join(root, "library.curated-backup")

	var createOutput bytes.Buffer
	if err := Run(ctx, Options{
		Action:            ActionBackupCreate,
		BackupPath:        backupPath,
		DatabasePath:      databasePath,
		LibraryConfigPath: configPath,
		AppVersion:        "test",
		AppChannel:        "dev",
	}, &createOutput); err != nil {
		t.Fatalf("Run(create): %v", err)
	}
	assertJSONAction(t, createOutput.Bytes(), ActionBackupCreate)

	var verifyOutput bytes.Buffer
	if err := Run(ctx, Options{
		Action:     ActionBackupVerify,
		BackupPath: backupPath,
	}, &verifyOutput); err != nil {
		t.Fatalf("Run(verify): %v", err)
	}
	assertJSONAction(t, verifyOutput.Bytes(), ActionBackupVerify)
	if !strings.Contains(verifyOutput.String(), `"valid": true`) {
		t.Fatalf("verification output = %s", verifyOutput.String())
	}
}

func TestRunRestoreRefusesActiveDatabaseLock(t *testing.T) {
	root := t.TempDir()
	databasePath := filepath.Join(root, "curated.db")
	lock, err := processlock.Acquire(databasePath + ".runtime.lock")
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	defer func() { _ = lock.Release() }()

	err = Run(context.Background(), Options{
		Action:         ActionBackupRestore,
		BackupPath:     filepath.Join(root, "unused.curated-backup"),
		DatabasePath:   databasePath,
		ConfirmRestore: true,
	}, &bytes.Buffer{})
	if err == nil || !errors.Is(err, processlock.ErrAlreadyLocked) || !strings.Contains(err.Error(), "fully exit") {
		t.Fatalf("Run(restore) error = %v", err)
	}
}

func assertJSONAction(t *testing.T, data []byte, want string) {
	t.Helper()
	var value struct {
		Action string `json:"action"`
	}
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("decode output %q: %v", data, err)
	}
	if value.Action != want {
		t.Fatalf("action = %q, want %q", value.Action, want)
	}
}

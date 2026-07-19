package maintenance

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"curated-backend/internal/backup"
	"curated-backend/internal/processlock"
	"curated-backend/internal/storage"
)

func TestRunPathMigrationPlanIsReadOnly(t *testing.T) {
	root := t.TempDir()
	databasePath := filepath.Join(root, "curated.db")
	sourceRoot := filepath.Join(root, "source")
	targetRoot := filepath.Join(root, "target")
	mustMakeDirectory(t, sourceRoot)
	mustMakeDirectory(t, targetRoot)
	seedMaintenanceLibraryPath(t, databasePath, sourceRoot)
	dropMaintenanceAuditSchema(t, databasePath)

	var output bytes.Buffer
	err := Run(context.Background(), Options{
		Action:       ActionPathMigratePlan,
		DatabasePath: databasePath,
		PathFrom:     sourceRoot,
		PathTo:       targetRoot,
	}, &output)
	if err != nil {
		t.Fatalf("Run(path-migrate-plan): %v\n%s", err, output.String())
	}
	var result struct {
		Action string                    `json:"action"`
		Plan   storage.PathMigrationPlan `json:"plan"`
	}
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatalf("decode output: %v\n%s", err, output.String())
	}
	if result.Action != ActionPathMigratePlan || !result.Plan.CanApply || result.Plan.AffectedRows != 1 {
		t.Fatalf("result=%+v", result)
	}
	assertMaintenanceAuditSchemaAbsent(t, databasePath)
	assertMaintenanceLibraryPath(t, databasePath, sourceRoot)
}

func TestRunPathMigrationPlanWritesBlockedJSON(t *testing.T) {
	root := t.TempDir()
	databasePath := filepath.Join(root, "curated.db")
	sourceRoot := filepath.Join(root, "source")
	targetRoot := filepath.Join(root, "missing-target")
	mustMakeDirectory(t, sourceRoot)
	seedMaintenanceLibraryPath(t, databasePath, sourceRoot)

	var output bytes.Buffer
	err := Run(context.Background(), Options{
		Action:       ActionPathMigratePlan,
		DatabasePath: databasePath,
		PathFrom:     sourceRoot,
		PathTo:       targetRoot,
	}, &output)
	if !errors.Is(err, ErrPathMigrationBlocked) {
		t.Fatalf("Run error=%v", err)
	}
	var result struct {
		Action string                    `json:"action"`
		Plan   storage.PathMigrationPlan `json:"plan"`
	}
	if decodeErr := json.Unmarshal(output.Bytes(), &result); decodeErr != nil {
		t.Fatalf("decode output: %v\n%s", decodeErr, output.String())
	}
	if result.Action != ActionPathMigratePlan || result.Plan.CanApply || result.Plan.MissingTargets != 1 || len(result.Plan.Errors) == 0 {
		t.Fatalf("result=%+v", result)
	}
	assertMaintenanceLibraryPath(t, databasePath, sourceRoot)
}

func TestRunPathMigrationApplyRequiresConfirmation(t *testing.T) {
	err := Run(context.Background(), Options{
		Action:               ActionPathMigrateApply,
		DatabasePath:         filepath.Join(t.TempDir(), "curated.db"),
		PathFrom:             `D:\Media`,
		PathTo:               `E:\Media`,
		ConfirmPathMigration: false,
	}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "-confirm-path-migration") {
		t.Fatalf("Run error=%v", err)
	}
}

func TestRunPathMigrationRefusesActiveDatabaseLock(t *testing.T) {
	root := t.TempDir()
	databasePath := filepath.Join(root, "curated.db")
	sourceRoot := filepath.Join(root, "source")
	targetRoot := filepath.Join(root, "target")
	mustMakeDirectory(t, sourceRoot)
	mustMakeDirectory(t, targetRoot)
	seedMaintenanceLibraryPath(t, databasePath, sourceRoot)

	lock, err := processlock.Acquire(databasePath + ".runtime.lock")
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	defer func() { _ = lock.Release() }()

	err = Run(context.Background(), Options{
		Action:       ActionPathMigratePlan,
		DatabasePath: databasePath,
		PathFrom:     sourceRoot,
		PathTo:       targetRoot,
	}, &bytes.Buffer{})
	if err == nil || !errors.Is(err, processlock.ErrAlreadyLocked) || !strings.Contains(err.Error(), "fully exit") {
		t.Fatalf("Run error=%v", err)
	}
}

func TestRunPathMigrationApplyCreatesVerifiedBackupAndAudit(t *testing.T) {
	root := t.TempDir()
	databasePath := filepath.Join(root, "curated.db")
	configPath := filepath.Join(root, "library-config.cfg")
	backupPath := filepath.Join(root, "before-migration.curated-backup")
	sourceRoot := filepath.Join(root, "source")
	targetRoot := filepath.Join(root, "target")
	mustMakeDirectory(t, sourceRoot)
	mustMakeDirectory(t, targetRoot)
	seedMaintenanceLibraryPath(t, databasePath, sourceRoot)
	if err := os.WriteFile(configPath, []byte("{}\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	appliedAt := time.Date(2026, 7, 20, 13, 0, 0, 0, time.UTC)

	var output bytes.Buffer
	err := Run(context.Background(), Options{
		Action:               ActionPathMigrateApply,
		BackupPath:           backupPath,
		DatabasePath:         databasePath,
		LibraryConfigPath:    configPath,
		AppVersion:           "test",
		AppChannel:           "dev",
		PathFrom:             sourceRoot,
		PathTo:               targetRoot,
		ConfirmPathMigration: true,
		Now:                  func() time.Time { return appliedAt },
	}, &output)
	if err != nil {
		t.Fatalf("Run(path-migrate-apply): %v\n%s", err, output.String())
	}
	var result struct {
		Action     string                           `json:"action"`
		BackupPath string                           `json:"backupPath"`
		Migration  storage.PathMigrationApplyResult `json:"migration"`
	}
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatalf("decode output: %v\n%s", err, output.String())
	}
	if result.Action != ActionPathMigrateApply || result.BackupPath != backupPath || !result.Migration.Applied || result.Migration.AuditID == "" {
		t.Fatalf("result=%+v", result)
	}
	verification, err := backup.Verify(context.Background(), backupPath)
	if err != nil || !verification.Valid {
		t.Fatalf("Verify backup valid=%v err=%v errors=%v", verification.Valid, err, verification.Errors)
	}
	assertMaintenanceLibraryPath(t, databasePath, targetRoot)
	assertMaintenanceAudit(t, databasePath, result.Migration.AuditID, backupPath)
	assertBackupLibraryPath(t, backupPath, sourceRoot)
}

func TestRunPathMigrationBackupFailureLeavesDatabaseUnchanged(t *testing.T) {
	root := t.TempDir()
	databasePath := filepath.Join(root, "curated.db")
	backupPath := filepath.Join(root, "existing.curated-backup")
	sourceRoot := filepath.Join(root, "source")
	targetRoot := filepath.Join(root, "target")
	mustMakeDirectory(t, sourceRoot)
	mustMakeDirectory(t, targetRoot)
	seedMaintenanceLibraryPath(t, databasePath, sourceRoot)
	if err := os.WriteFile(backupPath, []byte("do not overwrite"), 0o600); err != nil {
		t.Fatalf("write existing backup: %v", err)
	}

	err := Run(context.Background(), Options{
		Action:               ActionPathMigrateApply,
		BackupPath:           backupPath,
		DatabasePath:         databasePath,
		PathFrom:             sourceRoot,
		PathTo:               targetRoot,
		ConfirmPathMigration: true,
	}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "create pre-migration backup") {
		t.Fatalf("Run error=%v", err)
	}
	assertMaintenanceLibraryPath(t, databasePath, sourceRoot)
	data, readErr := os.ReadFile(backupPath)
	if readErr != nil || string(data) != "do not overwrite" {
		t.Fatalf("existing backup changed: data=%q err=%v", data, readErr)
	}
}

func seedMaintenanceLibraryPath(t *testing.T, databasePath, libraryPath string) {
	t.Helper()
	store, err := storage.NewSQLiteStore(databasePath)
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	if err := store.Migrate(context.Background()); err != nil {
		_ = store.Close()
		t.Fatalf("Migrate: %v", err)
	}
	if _, err := store.AddLibraryPath(context.Background(), libraryPath, "Library"); err != nil {
		_ = store.Close()
		t.Fatalf("AddLibraryPath: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func assertMaintenanceLibraryPath(t *testing.T, databasePath, want string) {
	t.Helper()
	store, err := storage.NewSQLiteStore(databasePath)
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer func() { _ = store.Close() }()
	paths, err := store.ListLibraryPaths(context.Background())
	if err != nil {
		t.Fatalf("ListLibraryPaths: %v", err)
	}
	if len(paths) != 1 || paths[0].Path != want {
		t.Fatalf("paths=%+v, want %q", paths, want)
	}
}

func dropMaintenanceAuditSchema(t *testing.T, databasePath string) {
	t.Helper()
	db := openMaintenanceDatabase(t, databasePath)
	defer func() { _ = db.Close() }()
	for _, statement := range []string{
		`DROP INDEX IF EXISTS idx_path_migration_audits_created_at`,
		`DROP TABLE IF EXISTS path_migration_audits`,
		`DELETE FROM schema_migrations WHERE name = '0028_path_migration_audits.sql'`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("drop audit schema: %v", err)
		}
	}
}

func assertMaintenanceAuditSchemaAbsent(t *testing.T, databasePath string) {
	t.Helper()
	db := openMaintenanceDatabase(t, databasePath)
	defer func() { _ = db.Close() }()
	var tableCount, migrationCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'path_migration_audits'`).Scan(&tableCount); err != nil {
		t.Fatalf("inspect audit table: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE name = '0028_path_migration_audits.sql'`).Scan(&migrationCount); err != nil {
		t.Fatalf("inspect audit migration: %v", err)
	}
	if tableCount != 0 || migrationCount != 0 {
		t.Fatalf("audit schema table=%d migration=%d", tableCount, migrationCount)
	}
}

func assertMaintenanceAudit(t *testing.T, databasePath, auditID, backupPath string) {
	t.Helper()
	db := openMaintenanceDatabase(t, databasePath)
	defer func() { _ = db.Close() }()
	var gotBackupPath string
	var affectedRows int
	if err := db.QueryRow(`SELECT backup_path, affected_rows FROM path_migration_audits WHERE id = ?`, auditID).Scan(&gotBackupPath, &affectedRows); err != nil {
		t.Fatalf("read audit: %v", err)
	}
	if gotBackupPath != backupPath || affectedRows != 1 {
		t.Fatalf("audit backup=%q affected=%d", gotBackupPath, affectedRows)
	}
}

func assertBackupLibraryPath(t *testing.T, backupPath, want string) {
	t.Helper()
	reader, err := zip.OpenReader(backupPath)
	if err != nil {
		t.Fatalf("open backup ZIP: %v", err)
	}
	defer func() { _ = reader.Close() }()
	databaseCopy := filepath.Join(t.TempDir(), "backup.db")
	found := false
	for _, file := range reader.File {
		if file.Name != "database/curated.db" {
			continue
		}
		input, err := file.Open()
		if err != nil {
			t.Fatalf("open backup database entry: %v", err)
		}
		output, err := os.OpenFile(databaseCopy, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			_ = input.Close()
			t.Fatalf("create backup database copy: %v", err)
		}
		_, copyErr := io.Copy(output, input)
		closeOutputErr := output.Close()
		closeInputErr := input.Close()
		if copyErr != nil || closeOutputErr != nil || closeInputErr != nil {
			t.Fatalf("copy backup database: copy=%v closeOutput=%v closeInput=%v", copyErr, closeOutputErr, closeInputErr)
		}
		found = true
		break
	}
	if !found {
		t.Fatal("backup database entry not found")
	}
	assertMaintenanceLibraryPath(t, databaseCopy, want)
}

func openMaintenanceDatabase(t *testing.T, databasePath string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", databasePath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		t.Fatalf("ping sqlite: %v", err)
	}
	return db
}

func mustMakeDirectory(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

package storage

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSQLiteStoreCreateConsistentBackup(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(root, "source.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if _, err := store.db.ExecContext(ctx, `INSERT INTO library_paths (id, path, title, created_at, updated_at) VALUES ('path-1', 'D:\\Media', 'Media', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`); err != nil {
		t.Fatalf("insert source row: %v", err)
	}

	destination := filepath.Join(root, "backup's", "curated.db")
	if err := store.CreateConsistentBackup(ctx, destination); err != nil {
		t.Fatalf("CreateConsistentBackup: %v", err)
	}
	info, err := os.Stat(destination)
	if err != nil {
		t.Fatalf("stat destination: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("backup database is empty")
	}

	backupStore, err := NewSQLiteStore(destination)
	if err != nil {
		t.Fatalf("open backup: %v", err)
	}
	defer func() { _ = backupStore.Close() }()
	var title string
	if err := backupStore.db.QueryRowContext(ctx, `SELECT title FROM library_paths WHERE id = 'path-1'`).Scan(&title); err != nil {
		t.Fatalf("read backup row: %v", err)
	}
	if title != "Media" {
		t.Fatalf("title = %q, want Media", title)
	}
	report, err := backupStore.CheckIntegrity(ctx)
	if err != nil {
		t.Fatalf("CheckIntegrity: %v", err)
	}
	if report.QuickCheck != "ok" || report.ForeignKeyViolations != 0 {
		t.Fatalf("unexpected integrity report: %+v", report)
	}

	sourceMigrations, err := store.AppliedMigrations(ctx)
	if err != nil {
		t.Fatalf("source AppliedMigrations: %v", err)
	}
	backupMigrations, err := backupStore.AppliedMigrations(ctx)
	if err != nil {
		t.Fatalf("backup AppliedMigrations: %v", err)
	}
	if !reflect.DeepEqual(backupMigrations, sourceMigrations) {
		t.Fatalf("backup migrations differ\n got: %v\nwant: %v", backupMigrations, sourceMigrations)
	}
}

func TestSQLiteStoreCreateConsistentBackupRefusesExistingDestination(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(root, "source.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer func() { _ = store.Close() }()
	destination := filepath.Join(root, "existing.db")
	if err := os.WriteFile(destination, []byte("keep me"), 0o600); err != nil {
		t.Fatalf("write existing destination: %v", err)
	}

	err = store.CreateConsistentBackup(ctx, destination)
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("CreateConsistentBackup error = %v, want already exists", err)
	}
	contents, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("read existing destination: %v", err)
	}
	if string(contents) != "keep me" {
		t.Fatalf("existing destination changed: %q", contents)
	}
}

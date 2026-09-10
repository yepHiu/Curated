package storage

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestPhotoLibraryMigrationCreatesTables(t *testing.T) {
	t.Parallel()

	store := openPhotoMigrationTestStore(t)
	assertPhotoTableExists(t, store.db, "photo_library_paths")
	assertPhotoTableExists(t, store.db, "photo_books")
	assertPhotoTableExists(t, store.db, "photo_pages")
	assertPhotoTableExists(t, store.db, "photo_book_tags")
}

func openPhotoMigrationTestStore(t *testing.T) *SQLiteStore {
	t.Helper()

	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "photo-migration.db"))
	if err != nil {
		t.Fatalf("new sqlite store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return store
}

func assertPhotoTableExists(t *testing.T, db *sql.DB, table string) {
	t.Helper()

	var name string
	err := db.QueryRow(
		`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`,
		table,
	).Scan(&name)
	if err != nil {
		t.Fatalf("table %q should exist after migrations: %v", table, err)
	}
	if name != table {
		t.Fatalf("sqlite_master returned table %q, want %q", name, table)
	}
}

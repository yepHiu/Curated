package storage

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestComicLibraryMigrationCreatesTables(t *testing.T) {
	t.Parallel()

	store := openComicMigrationTestStore(t)
	for _, table := range []string{
		"comic_library_paths",
		"comic_books",
		"comic_pages",
		"comic_tags",
		"comic_book_tags",
		"comic_reading_progress",
		"comic_reading_preferences",
		"comic_cache_entries",
	} {
		assertComicTableExists(t, store.db, table)
	}
}

func openComicMigrationTestStore(t *testing.T) *SQLiteStore {
	t.Helper()

	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "comic-migration.db"))
	if err != nil {
		t.Fatalf("new sqlite store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return store
}

func assertComicTableExists(t *testing.T, db *sql.DB, table string) {
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

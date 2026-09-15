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
		"comic_book_comments",
		"comic_cache_entries",
	} {
		assertComicTableExists(t, store.db, table)
	}
	assertComicColumnExists(t, store.db, "comic_books", "user_title")
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

// assertComicColumnExists 确认迁移后指定漫画表列存在。
func assertComicColumnExists(t *testing.T, db *sql.DB, table, column string) {
	t.Helper()
	rows, err := db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		t.Fatalf("pragma table_info %s: %v", table, err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			t.Fatalf("scan table_info: %v", err)
		}
		if name == column {
			return
		}
	}
	t.Fatalf("column %s.%s should exist after migrations", table, column)
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

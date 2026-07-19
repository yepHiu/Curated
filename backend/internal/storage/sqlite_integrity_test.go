package storage

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewSQLiteStoreEnablesForeignKeys(t *testing.T) {
	t.Parallel()

	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "foreign-keys.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	var enabled int
	if err := store.db.QueryRow(`PRAGMA foreign_keys`).Scan(&enabled); err != nil {
		t.Fatal(err)
	}
	if enabled != 1 {
		t.Fatalf("PRAGMA foreign_keys = %d, want 1", enabled)
	}
}

func TestOrphanMigrationQuarantinesEveryForeignKeyViolation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "legacy-orphans.db")
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Migrate(ctx); err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`DELETE FROM schema_migrations WHERE name = '0026_quarantine_orphaned_foreign_keys.sql'`); err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`PRAGMA foreign_keys = OFF`); err != nil {
		_ = store.Close()
		t.Fatal(err)
	}

	_, err = store.db.Exec(`
		INSERT INTO media_assets (id, movie_id, type, local_path)
		VALUES ('asset-orphan', 'movie-missing', 'cover', 'missing.jpg');
		INSERT INTO movie_actors (movie_id, actor_id) VALUES ('movie-missing', 9001);
		INSERT INTO movie_tags (movie_id, tag_id) VALUES ('movie-missing', 9002);
		INSERT INTO playback_progress (movie_id, position_sec, duration_sec, updated_at)
		VALUES ('movie-missing', 12.5, 100, '2026-07-19T00:00:00Z');
		INSERT INTO curated_frames (
			id, movie_id, title, code, actors_json, position_sec, captured_at, tags_json, image_blob
		) VALUES (
			'frame-orphan', 'movie-missing', 'Frame', 'TEST-001', '["Actor"]', 12.5,
			'2026-07-19T00:00:00Z', '["Tag"]', X'89504E47'
		);
		INSERT INTO library_played_movies (movie_id, first_played_at)
		VALUES ('movie-missing', '2026-07-19T00:00:00Z');
		INSERT INTO actor_user_tags (actor_id, tag) VALUES (9001, 'favorite');
		INSERT INTO library_movie_comments (movie_id, body, updated_at)
		VALUES ('movie-missing', 'preserve me', '2026-07-19T00:00:00Z');
		INSERT INTO actor_external_links (actor_id, url, created_at, updated_at)
		VALUES (9001, 'https://example.invalid/actor', '2026-07-19T00:00:00Z', '2026-07-19T00:00:00Z');
		INSERT INTO playback_daily_watch_time (day_key, movie_id, watched_sec, updated_at)
		VALUES ('2026-07-19', 'movie-missing', 42, '2026-07-19T00:00:00Z');
		INSERT INTO library_path_storage_bindings (
			library_path_id, root_path, identity_confidence, updated_at
		) VALUES ('path-missing', 'Z:\\missing', 'low', '2026-07-19T00:00:00Z');
	`)
	if err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	store, err = NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	var quarantineCount int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM data_integrity_quarantine`).Scan(&quarantineCount); err != nil {
		t.Fatal(err)
	}
	if quarantineCount != 11 {
		t.Fatalf("quarantine count = %d, want 11", quarantineCount)
	}

	expectedSources := []string{
		"actor_external_links",
		"actor_user_tags",
		"curated_frames",
		"library_movie_comments",
		"library_path_storage_bindings",
		"library_played_movies",
		"media_assets",
		"movie_actors",
		"movie_tags",
		"playback_daily_watch_time",
		"playback_progress",
	}
	rows, err := store.db.Query(`SELECT source_table FROM data_integrity_quarantine ORDER BY source_table`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	var sources []string
	for rows.Next() {
		var source string
		if err := rows.Scan(&source); err != nil {
			t.Fatal(err)
		}
		sources = append(sources, source)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(sources) != len(expectedSources) {
		t.Fatalf("quarantine sources = %v", sources)
	}
	for i := range expectedSources {
		if sources[i] != expectedSources[i] {
			t.Fatalf("quarantine sources = %v, want %v", sources, expectedSources)
		}
	}

	var imageHex string
	if err := store.db.QueryRow(`
		SELECT json_extract(payload_json, '$.image_blob_hex')
		FROM data_integrity_quarantine WHERE source_table = 'curated_frames'
	`).Scan(&imageHex); err != nil {
		t.Fatal(err)
	}
	if imageHex != "89504E47" {
		t.Fatalf("archived image hex = %q", imageHex)
	}

	foreignKeyRows, err := store.db.Query(`PRAGMA foreign_key_check`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = foreignKeyRows.Close() }()
	if foreignKeyRows.Next() {
		t.Fatal("PRAGMA foreign_key_check still reports a violation after orphan quarantine")
	}
	if err := foreignKeyRows.Err(); err != nil {
		t.Fatal(err)
	}
}

func TestMigrateRejectsAnUnrepairedForeignKeyViolation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "unrepaired-orphan.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`PRAGMA foreign_keys = OFF`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`
		INSERT INTO playback_progress (movie_id, position_sec, duration_sec, updated_at)
		VALUES ('unexpected-orphan', 1, 2, '2026-07-19T00:00:00Z')
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		t.Fatal(err)
	}

	err = store.Migrate(ctx)
	if err == nil {
		t.Fatal("Migrate() should reject an unrepaired foreign key violation")
	}
	if !strings.Contains(err.Error(), "table=playback_progress") || !strings.Contains(err.Error(), "parent=movies") {
		t.Fatalf("Migrate() error = %q, want violation details", err)
	}
}

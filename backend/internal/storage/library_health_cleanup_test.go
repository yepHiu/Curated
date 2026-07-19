package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestCleanupOrphanUserStateIsWhitelistedConditionalAndAudited(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "health-cleanup.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `PRAGMA foreign_keys = OFF`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `
		INSERT INTO playback_progress (movie_id, position_sec, duration_sec, updated_at)
		VALUES ('missing-movie', 10, 100, '2026-07-20T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `PRAGMA foreign_keys = ON`); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 20, 3, 0, 0, 0, time.UTC)
	removed, err := store.CleanupOrphanUserState(ctx, "cleanup-task", "health-finding", "playback_progress", "missing-movie", now)
	if err != nil || !removed {
		t.Fatalf("removed=%v err=%v", removed, err)
	}
	var count int
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM playback_progress WHERE movie_id = 'missing-movie'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("orphan count = %d", count)
	}
	audits, err := store.ListLibraryHealthCleanupAudits(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(audits) != 1 || audits[0].Outcome != "removed" || audits[0].EntityType != "playback_progress" {
		t.Fatalf("audits = %#v", audits)
	}

	removed, err = store.CleanupOrphanUserState(ctx, "cleanup-task-2", "health-finding-2", "movies", "missing-movie", now.Add(time.Minute))
	if err != nil || removed {
		t.Fatalf("non-whitelisted cleanup removed=%v err=%v", removed, err)
	}
	audits, err = store.ListLibraryHealthCleanupAudits(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(audits) != 2 || audits[0].Outcome != "skipped" {
		t.Fatalf("audits after skipped = %#v", audits)
	}
}

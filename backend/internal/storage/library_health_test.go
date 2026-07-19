package storage

import (
	"context"
	"path/filepath"
	"testing"

	"curated-backend/internal/contracts"
)

func TestInspectLibraryHealthCollectsIntegrityAndEntityFacts(t *testing.T) {
	root := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(root, "health.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	moviePath := filepath.Join(root, "HEALTH-001.mp4")
	if _, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
		TaskID: "scan-health", Path: moviePath, FileName: filepath.Base(moviePath), Number: "HEALTH-001",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `
		INSERT INTO media_assets (id, movie_id, type, source_url, local_path, last_http_status, last_error)
		VALUES ('asset-health', 'health-001', 'cover', '', ?, 404, 'not found')`, filepath.Join(root, "missing-cover.jpg")); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `INSERT INTO actors (name, avatar) VALUES ('Health Actor', '')`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `INSERT INTO movie_actors (movie_id, actor_id) SELECT 'health-001', id FROM actors WHERE name = 'Health Actor'`); err != nil {
		t.Fatal(err)
	}

	if _, err := store.db.ExecContext(ctx, `PRAGMA foreign_keys = OFF`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `
		INSERT INTO playback_progress (movie_id, position_sec, duration_sec, updated_at)
		VALUES ('missing-movie', 12, 120, '2026-07-20T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `PRAGMA foreign_keys = ON`); err != nil {
		t.Fatal(err)
	}

	snapshot, err := store.InspectLibraryHealth(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Movies) != 1 || snapshot.Movies[0].ID != "health-001" {
		t.Fatalf("movies = %#v", snapshot.Movies)
	}
	if len(snapshot.Assets) != 1 || snapshot.Assets[0].LastHTTPStatus != 404 {
		t.Fatalf("assets = %#v", snapshot.Assets)
	}
	if len(snapshot.Actors) != 1 || snapshot.Actors[0].Name != "Health Actor" {
		t.Fatalf("actors = %#v", snapshot.Actors)
	}
	if len(snapshot.Orphans) != 1 || snapshot.Orphans[0].TableName != "playback_progress" {
		t.Fatalf("orphans = %#v", snapshot.Orphans)
	}
	if len(snapshot.QuickCheckMessages) != 1 || snapshot.QuickCheckMessages[0] != "ok" {
		t.Fatalf("quick check = %#v", snapshot.QuickCheckMessages)
	}
	if len(snapshot.ForeignKeyViolations) != 1 || snapshot.ForeignKeyViolations[0].TableName != "playback_progress" {
		t.Fatalf("foreign key violations = %#v", snapshot.ForeignKeyViolations)
	}
}

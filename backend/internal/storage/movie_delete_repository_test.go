package storage

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"curated-backend/internal/contracts"
	"curated-backend/internal/scraper"
)

func TestDeleteMovie_NotFound(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(root, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	err = store.DeleteMovie(ctx, "missing-id", "")
	if !errors.Is(err, ErrMovieNotFound) {
		t.Fatalf("expected ErrMovieNotFound, got %v", err)
	}
}

func TestDeleteMovie_RemovesRowsAndFiles(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	videoPath := filepath.Join(root, "DEL-001.mp4")
	if err := os.WriteFile(videoPath, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	nfoPath := filepath.Join(root, "movie.nfo")
	if err := os.WriteFile(nfoPath, []byte("<movie/>"), 0o644); err != nil {
		t.Fatal(err)
	}
	coverPath := filepath.Join(root, "cover.jpg")
	if err := os.WriteFile(coverPath, []byte("jpg"), 0o644); err != nil {
		t.Fatal(err)
	}

	store, err := NewSQLiteStore(filepath.Join(root, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	outcome, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
		TaskID:   "task-1",
		Path:     videoPath,
		FileName: "DEL-001.mp4",
		Number:   "DEL-001",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveMovieMetadata(ctx, scraper.Metadata{
		MovieID:  outcome.MovieID,
		Number:   "DEL-001",
		Title:    "T",
		Summary:  "S",
		Studio:   "St",
		CoverURL: "http://x",
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateMediaAssetLocalPath(ctx, outcome.MovieID, "cover", "http://x", coverPath); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`
		INSERT INTO actors (name) VALUES ('Delete Test Actor');
		INSERT INTO tags (name, type) VALUES ('delete-test-tag', 'user');
		INSERT INTO movie_actors (movie_id, actor_id)
		SELECT ?, id FROM actors WHERE name = 'Delete Test Actor';
		INSERT INTO movie_tags (movie_id, tag_id)
		SELECT ?, id FROM tags WHERE name = 'delete-test-tag' AND type = 'user';
		INSERT INTO playback_progress (movie_id, position_sec, duration_sec, updated_at)
		VALUES (?, 12, 100, '2026-07-19T00:00:00Z');
		INSERT INTO curated_frames (
			id, movie_id, title, code, position_sec, captured_at, image_blob
		) VALUES ('delete-frame', ?, 'Frame', 'DEL-001', 12, '2026-07-19T00:00:00Z', X'89504E47');
		INSERT INTO library_played_movies (movie_id, first_played_at)
		VALUES (?, '2026-07-19T00:00:00Z');
		INSERT INTO library_movie_comments (movie_id, body, updated_at)
		VALUES (?, 'delete me', '2026-07-19T00:00:00Z');
		INSERT INTO playback_daily_watch_time (day_key, movie_id, watched_sec, updated_at)
		VALUES ('2026-07-19', ?, 20, '2026-07-19T00:00:00Z');
	`, outcome.MovieID, outcome.MovieID, outcome.MovieID, outcome.MovieID, outcome.MovieID, outcome.MovieID, outcome.MovieID); err != nil {
		t.Fatal(err)
	}

	cacheRoot := filepath.Join(root, "asset-cache")
	if err := store.DeleteMovie(ctx, outcome.MovieID, cacheRoot); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(videoPath); !os.IsNotExist(err) {
		t.Fatalf("video should be gone: %v", err)
	}
	if _, err := os.Stat(nfoPath); !os.IsNotExist(err) {
		t.Fatalf("nfo should be gone: %v", err)
	}
	if _, err := os.Stat(coverPath); !os.IsNotExist(err) {
		t.Fatalf("cover should be gone: %v", err)
	}

	_, err = store.GetMovieDetail(ctx, outcome.MovieID)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows after delete, got %v", err)
	}
	for _, table := range []string{
		"media_assets",
		"movie_actors",
		"movie_tags",
		"scan_items",
		"playback_progress",
		"curated_frames",
		"library_played_movies",
		"library_movie_comments",
		"playback_daily_watch_time",
	} {
		var count int
		query := `SELECT COUNT(*) FROM ` + table + ` WHERE movie_id = ?`
		if err := store.db.QueryRowContext(ctx, query, outcome.MovieID).Scan(&count); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if count != 0 {
			t.Fatalf("%s still has %d row(s) for deleted movie", table, count)
		}
	}
}

func TestDeleteMovie_RemovesAssetCacheDirectory(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	videoPath := filepath.Join(root, "CACHE-DEL.mp4")
	if err := os.WriteFile(videoPath, []byte("v"), 0o644); err != nil {
		t.Fatal(err)
	}
	cacheRoot := filepath.Join(root, "cache")
	movieCacheDir := filepath.Join(cacheRoot, "cache-del")
	if err := os.MkdirAll(movieCacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	orphan := filepath.Join(movieCacheDir, "orphan.jpg")
	if err := os.WriteFile(orphan, []byte("jpg"), 0o644); err != nil {
		t.Fatal(err)
	}

	store, err := NewSQLiteStore(filepath.Join(root, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	outcome, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
		TaskID:   "task-cache",
		Path:     videoPath,
		FileName: "CACHE-DEL.mp4",
		Number:   "CACHE-DEL",
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.MovieID != "cache-del" {
		t.Fatalf("unexpected movie id %q", outcome.MovieID)
	}

	if err := store.DeleteMovie(ctx, outcome.MovieID, cacheRoot); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(movieCacheDir); !os.IsNotExist(err) {
		t.Fatalf("asset cache dir should be removed: %v", err)
	}
}

func TestDeleteMovieRecordsOnly_KeepsFilesOnDisk(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	videoPath := filepath.Join(root, "REC-ONLY.mp4")
	if err := os.WriteFile(videoPath, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}

	store, err := NewSQLiteStore(filepath.Join(root, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	outcome, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
		TaskID:   "task-rec",
		Path:     videoPath,
		FileName: "REC-ONLY.mp4",
		Number:   "REC-ONLY",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveMovieMetadata(ctx, scraper.Metadata{
		MovieID: outcome.MovieID,
		Number:  "REC-ONLY",
		Title:   "T",
		Summary: "S",
		Studio:  "St",
	}); err != nil {
		t.Fatal(err)
	}

	if err := store.DeleteMovieRecordsOnly(ctx, outcome.MovieID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(videoPath); err != nil {
		t.Fatalf("video file should remain after records-only delete: %v", err)
	}
	_, err = store.GetMovieDetail(ctx, outcome.MovieID)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected movie row gone, got %v", err)
	}
}

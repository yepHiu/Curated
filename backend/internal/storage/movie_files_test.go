package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"curated-backend/internal/contracts"
)

// multipartStore isolates a migrated database and media roots from the user's library.
func multipartStore(t *testing.T) (*SQLiteStore, string) {
	t.Helper()
	root := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(root, "library.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { /* SQLite must close before the temporary directory is removed. */ _ = store.Close() })
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	return store, root
}

// addMultipartFile creates an indexed fixture with real bytes for source validation.
func addMultipartFile(t *testing.T, store *SQLiteStore, root, name, code string) string {
	t.Helper()
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, name)
	if err := os.WriteFile(path, []byte(name), 0644); err != nil {
		t.Fatal(err)
	}
	outcome, err := store.PersistScanMovie(context.Background(), contracts.ScanFileResultDTO{Number: code, Path: path, FileName: name})
	if err != nil {
		t.Fatal(err)
	}
	return outcome.MovieID
}

// TestMultipartScanProgressAndOwnership verifies stable parent/file identity and independent resume.
func TestMultipartScanProgressAndOwnership(t *testing.T) {
	store, root := multipartStore(t)
	ctx := context.Background()
	id := addMultipartFile(t, store, root, "FC2-1234567-CD10.mp4", "FC2-1234567")
	addMultipartFile(t, store, root, "FC2-1234567-CD2.mp4", "FC2-PPV-1234567")
	addMultipartFile(t, store, root, "FC2-1234567_1.mp4", "fc2_1234567")
	detail, err := store.GetMovieDetail(ctx, id)
	if err != nil || detail.FileCount != 3 || len(detail.Files) != 3 {
		t.Fatalf("detail=%+v err=%v", detail, err)
	}
	for i, want := range []int{1, 2, 10} {
		if detail.Files[i].PartIndex != want {
			t.Fatalf("files=%v", detail.Files)
		}
	}
	originalIDs := []string{detail.Files[0].ID, detail.Files[1].ID, detail.Files[2].ID}
	addMultipartFile(t, store, root, "FC2-1234567-CD2.mp4", "FC2-1234567")
	again, _ := store.GetMovieDetail(ctx, id)
	if len(again.Files) != 3 || again.Files[1].ID != originalIDs[1] {
		t.Fatal("rescan changed files")
	}
	for i, file := range detail.Files {
		selected := contracts.WithMovieFileSelection(ctx, file.ID)
		if err := store.UpsertPlaybackProgress(selected, id, float64(10+i), 100); err != nil {
			t.Fatal(err)
		}
	}
	for i, file := range detail.Files {
		row, err := store.GetPlaybackProgress(contracts.WithMovieFileSelection(ctx, file.ID), id)
		if err != nil || row == nil || row.PositionSec != float64(10+i) {
			t.Fatalf("file=%s progress=%v err=%v", file.ID, row, err)
		}
	}
	last, err := store.GetMoviePlaybackDetail(ctx, id)
	if err != nil || last.SelectedFileID != detail.Files[2].ID {
		t.Fatalf("last=%v err=%v", last.SelectedFileID, err)
	}
	other := addMultipartFile(t, store, root, "ABC-100.mp4", "ABC-100")
	if _, err := store.GetMoviePlaybackDetail(contracts.WithMovieFileSelection(ctx, detail.Files[0].ID), other); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("foreign file=%v", err)
	}
	if err := store.UpsertPlaybackProgress(contracts.WithMovieFileSelection(ctx, detail.Files[0].ID), other, 33, 100); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("foreign progress=%v", err)
	}
	renamed := filepath.Join(root, "FC2-1234567-CD1.mp4")
	if err := os.Rename(detail.Files[0].Location, renamed); err != nil {
		t.Fatal(err)
	}
	if err := store.RelocateMovieFile(ctx, detail.Files[0].Location, renamed); err != nil {
		t.Fatal(err)
	}
	updated, _ := store.GetMovieDetail(ctx, id)
	if updated.Files[0].ID != originalIDs[0] || updated.Files[0].Location != renamed {
		t.Fatalf("rename=%v", updated.Files)
	}
}

// TestMultipartRootRemovalPreservesRemainingWork tests database-only pruning across shared roots.
func TestMultipartRootRemovalPreservesRemainingWork(t *testing.T) {
	store, root := multipartStore(t)
	ctx := context.Background()
	firstRoot, secondRoot := filepath.Join(root, "one"), filepath.Join(root, "two")
	first, err := store.AddLibraryPath(ctx, firstRoot, "One")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddLibraryPath(ctx, secondRoot, "Two"); err != nil {
		t.Fatal(err)
	}
	id := addMultipartFile(t, store, firstRoot, "ABC-123-CD1.mp4", "ABC-123")
	addMultipartFile(t, store, secondRoot, "ABC-123-CD2.mp4", "ABC-123")
	pruned, err := store.DeleteLibraryPathAndPruneOrphanMovies(ctx, first.ID)
	if err != nil || pruned != 0 {
		t.Fatalf("pruned=%d err=%v", pruned, err)
	}
	detail, err := store.GetMovieDetail(ctx, id)
	if err != nil || len(detail.Files) != 1 || detail.Files[0].PartIndex != 2 {
		t.Fatalf("detail=%v err=%v", detail.Files, err)
	}
	if _, err := os.Stat(filepath.Join(firstRoot, "ABC-123-CD1.mp4")); err != nil {
		t.Fatal("removed source file", err)
	}
}

// TestMovieFilesMigrationBindsOriginalProgress verifies pre-upgrade data remains on its actual source.
func TestMovieFilesMigrationBindsOriginalProgress(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "legacy.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() >= "0064_movie_files.sql" {
			continue
		}
		sqlBytes, err := migrationFiles.ReadFile("migrations/" + entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := store.db.Exec(string(sqlBytes)); err != nil {
			t.Fatalf("%s: %v", entry.Name(), err)
		}
	}
	if _, err := store.db.Exec(`INSERT INTO movies(id,title,code,studio,summary,added_at,location,resolution,year) VALUES('legacy','Legacy','ABC-123','Studio','','2026','D:/ABC-123-CD3.mp4','mp4',2026)`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`INSERT INTO playback_progress(movie_id,position_sec,duration_sec,updated_at) VALUES('legacy',42,100,'2026')`); err != nil {
		t.Fatal(err)
	}
	migration, _ := migrationFiles.ReadFile("migrations/0064_movie_files.sql")
	if _, err := store.db.Exec(string(migration)); err != nil {
		t.Fatal(err)
	}
	detail, err := store.GetMoviePlaybackDetail(context.Background(), "legacy")
	if err != nil || detail.SelectedFileID != "legacy:primary" || detail.Files[0].PartIndex != 3 {
		t.Fatalf("detail=%+v err=%v", detail, err)
	}
	row, err := store.GetPlaybackProgress(contracts.WithMovieFileSelection(context.Background(), detail.SelectedFileID), "legacy")
	if err != nil || row == nil || row.PositionSec != 42 {
		t.Fatalf("row=%v err=%v", row, err)
	}
}

// TestMultipartFrameIdentity separates captures with identical times in different source files.
func TestMultipartFrameIdentity(t *testing.T) {
	store, root := multipartStore(t)
	ctx := context.Background()
	id := addMultipartFile(t, store, root, "ABC-123-CD1.mp4", "ABC-123")
	addMultipartFile(t, store, root, "ABC-123-CD2.mp4", "ABC-123")
	detail, _ := store.GetMovieDetail(ctx, id)
	for i, file := range detail.Files {
		meta := CuratedFrameMeta{ID: fmt.Sprintf("frame-%d", i), MovieID: id, FileID: file.ID, PositionSec: 12, CapturedAt: "2026", Actors: []string{}, Tags: []string{}}
		if err := store.InsertCuratedFrame(ctx, meta, []byte("png")); err != nil {
			t.Fatal(err)
		}
	}
	row, err := store.FindNearbyCuratedFrame(contracts.WithMovieFileSelection(ctx, detail.Files[1].ID), id, 12, 1)
	if err != nil || row == nil || row.ID != "frame-1" || row.FileID != detail.Files[1].ID {
		t.Fatalf("row=%v err=%v", row, err)
	}
	page, err := store.QueryCuratedFrames(ctx, CuratedFrameQuery{MovieID: id})
	if err != nil || len(page.Items) != 2 || page.Items[0].FileID == page.Items[1].FileID {
		t.Fatalf("page=%v err=%v", page, err)
	}
	meta := CuratedFrameMeta{ID: "frame-0", MovieID: id, FileID: detail.Files[1].ID, PositionSec: 12, CapturedAt: "2026"}
	if replay, err := store.MatchesCuratedFrameCapture(ctx, meta, []byte("png")); err != nil || replay {
		t.Fatalf("wrong-file replay=%v err=%v", replay, err)
	}
}

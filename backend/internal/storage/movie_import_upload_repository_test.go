package storage

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestMovieImportUploadRepositoryPersistsSessionsFilesAndChunks(t *testing.T) {
	ctx := context.Background()
	store := newMovieImportUploadRepositoryTestStore(t)
	defer func() { _ = store.Close() }()
	now := time.Date(2026, 7, 20, 1, 2, 3, 0, time.UTC)
	record := movieImportUploadRepositoryTestRecord(t, now)

	if err := store.CreateMovieImportUploadSession(ctx, record); err != nil {
		t.Fatalf("CreateMovieImportUploadSession: %v", err)
	}
	loaded, err := store.GetMovieImportUploadSession(ctx, record.UploadID)
	if err != nil {
		t.Fatalf("GetMovieImportUploadSession: %v", err)
	}
	if loaded.TaskID != record.TaskID || loaded.TotalBytes != 15 || loaded.BytesReceived != 0 || len(loaded.Files) != 2 {
		t.Fatalf("loaded=%+v", loaded)
	}
	if loaded.Files[0].Ordinal != 0 || loaded.Files[0].RelativePath != "folder/movie-a.mkv" || loaded.Files[0].State != MovieImportUploadFileStatePending {
		t.Fatalf("first file=%+v", loaded.Files[0])
	}

	first, err := store.RecordMovieImportUploadChunk(ctx, record.UploadID, "file-a", MovieImportUploadChunkRecord{
		ChunkIndex: 0,
		Offset:     0,
		Size:       4,
	}, now.Add(time.Minute), now.Add(25*time.Hour))
	if err != nil {
		t.Fatalf("RecordMovieImportUploadChunk(first): %v", err)
	}
	if first.Duplicate || first.FileBytesReceived != 4 || first.SessionBytesReceived != 4 {
		t.Fatalf("first=%+v", first)
	}

	duplicate, err := store.RecordMovieImportUploadChunk(ctx, record.UploadID, "file-a", MovieImportUploadChunkRecord{
		ChunkIndex: 0,
		Offset:     0,
		Size:       4,
	}, now.Add(2*time.Minute), now.Add(25*time.Hour))
	if err != nil {
		t.Fatalf("RecordMovieImportUploadChunk(duplicate): %v", err)
	}
	if !duplicate.Duplicate || duplicate.FileBytesReceived != 4 || duplicate.SessionBytesReceived != 4 {
		t.Fatalf("duplicate=%+v", duplicate)
	}

	_, err = store.RecordMovieImportUploadChunk(ctx, record.UploadID, "file-a", MovieImportUploadChunkRecord{
		ChunkIndex: 0,
		Offset:     4,
		Size:       4,
	}, now, now.Add(25*time.Hour))
	if !errors.Is(err, ErrMovieImportUploadChunkConflict) {
		t.Fatalf("same-index conflict error=%v", err)
	}
	_, err = store.RecordMovieImportUploadChunk(ctx, record.UploadID, "file-a", MovieImportUploadChunkRecord{
		ChunkIndex: 1,
		Offset:     2,
		Size:       4,
	}, now, now.Add(25*time.Hour))
	if !errors.Is(err, ErrMovieImportUploadChunkOverlap) {
		t.Fatalf("overlap error=%v", err)
	}

	second, err := store.RecordMovieImportUploadChunk(ctx, record.UploadID, "file-a", MovieImportUploadChunkRecord{
		ChunkIndex: 1,
		Offset:     4,
		Size:       6,
	}, now.Add(3*time.Minute), now.Add(25*time.Hour))
	if err != nil {
		t.Fatalf("RecordMovieImportUploadChunk(second): %v", err)
	}
	if second.FileBytesReceived != 10 || second.SessionBytesReceived != 10 {
		t.Fatalf("second=%+v", second)
	}

	loaded, err = store.GetMovieImportUploadSession(ctx, record.UploadID)
	if err != nil {
		t.Fatalf("GetMovieImportUploadSession(after chunks): %v", err)
	}
	if loaded.BytesReceived != 10 || loaded.Files[0].BytesReceived != 10 || len(loaded.Files[0].Chunks) != 2 {
		t.Fatalf("loaded after chunks=%+v", loaded)
	}
	if loaded.Files[0].Chunks[0].Offset != 0 || loaded.Files[0].Chunks[1].Offset != 4 {
		t.Fatalf("chunks=%+v", loaded.Files[0].Chunks)
	}
}

func TestMovieImportUploadRepositoryTransitionsAndCascades(t *testing.T) {
	ctx := context.Background()
	store := newMovieImportUploadRepositoryTestStore(t)
	defer func() { _ = store.Close() }()
	now := time.Date(2026, 7, 20, 2, 0, 0, 0, time.UTC)
	record := movieImportUploadRepositoryTestRecord(t, now)
	if err := store.CreateMovieImportUploadSession(ctx, record); err != nil {
		t.Fatalf("CreateMovieImportUploadSession: %v", err)
	}

	if err := store.TransitionMovieImportUploadState(ctx, record.UploadID,
		[]string{MovieImportUploadStateUploading}, MovieImportUploadStateCommitting,
		now.Add(time.Hour), formatMovieImportUploadTime(now.Add(25*time.Hour)), "", "", ""); err != nil {
		t.Fatalf("TransitionMovieImportUploadState(committing): %v", err)
	}
	_, err := store.RecordMovieImportUploadChunk(ctx, record.UploadID, "file-a", MovieImportUploadChunkRecord{
		ChunkIndex: 0, Offset: 0, Size: 1,
	}, now, now.Add(time.Hour))
	if !errors.Is(err, ErrMovieImportUploadStateConflict) {
		t.Fatalf("record in committing state error=%v", err)
	}
	if err := store.MarkMovieImportUploadFileCommitted(ctx, record.UploadID, "file-a", now.Add(2*time.Hour)); err != nil {
		t.Fatalf("MarkMovieImportUploadFileCommitted: %v", err)
	}
	if err := store.MarkMovieImportUploadFileCommitted(ctx, record.UploadID, "file-a", now.Add(2*time.Hour)); err != nil {
		t.Fatalf("MarkMovieImportUploadFileCommitted(idempotent): %v", err)
	}
	cleanupAfter := formatMovieImportUploadTime(now.Add(3 * time.Hour))
	if err := store.TransitionMovieImportUploadState(ctx, record.UploadID,
		[]string{MovieImportUploadStateCommitting}, MovieImportUploadStateCommitted,
		now.Add(2*time.Hour), "", cleanupAfter, "", ""); err != nil {
		t.Fatalf("TransitionMovieImportUploadState(committed): %v", err)
	}

	candidates, err := store.ListMovieImportUploadCleanupCandidates(ctx, now.Add(4*time.Hour))
	if err != nil {
		t.Fatalf("ListMovieImportUploadCleanupCandidates: %v", err)
	}
	if len(candidates) != 1 || candidates[0].UploadID != record.UploadID || candidates[0].State != MovieImportUploadStateCommitted {
		t.Fatalf("candidates=%+v", candidates)
	}

	auditID, err := store.CreateMovieImportUploadCleanupAudit(ctx, record.UploadID, "committed", candidates[0].State,
		record.StagingDir, "", now.Add(4*time.Hour))
	if err != nil {
		t.Fatalf("CreateMovieImportUploadCleanupAudit: %v", err)
	}
	if err := store.CompleteMovieImportUploadCleanupAudit(ctx, auditID, "removed", "staging removed", now.Add(4*time.Hour)); err != nil {
		t.Fatalf("CompleteMovieImportUploadCleanupAudit: %v", err)
	}
	if err := store.DeleteMovieImportUploadSession(ctx, record.UploadID); err != nil {
		t.Fatalf("DeleteMovieImportUploadSession: %v", err)
	}
	if _, err := store.GetMovieImportUploadSession(ctx, record.UploadID); !errors.Is(err, ErrMovieImportUploadNotFound) {
		t.Fatalf("Get deleted session error=%v", err)
	}
	var fileCount, chunkCount int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM movie_import_upload_files WHERE upload_id = ?`, record.UploadID).Scan(&fileCount); err != nil {
		t.Fatalf("count files: %v", err)
	}
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM movie_import_upload_chunks WHERE upload_id = ?`, record.UploadID).Scan(&chunkCount); err != nil {
		t.Fatalf("count chunks: %v", err)
	}
	if fileCount != 0 || chunkCount != 0 {
		t.Fatalf("cascade files=%d chunks=%d", fileCount, chunkCount)
	}
	audits, err := store.ListMovieImportUploadCleanupAudits(ctx, 10)
	if err != nil {
		t.Fatalf("ListMovieImportUploadCleanupAudits: %v", err)
	}
	if len(audits) != 1 || audits[0].Outcome != "removed" || audits[0].CompletedAt == "" {
		t.Fatalf("audits=%+v", audits)
	}
}

func TestMovieImportUploadRepositoryReconcilesDerivedCounters(t *testing.T) {
	ctx := context.Background()
	store := newMovieImportUploadRepositoryTestStore(t)
	defer func() { _ = store.Close() }()
	now := time.Date(2026, 7, 20, 3, 0, 0, 0, time.UTC)
	record := movieImportUploadRepositoryTestRecord(t, now)
	if err := store.CreateMovieImportUploadSession(ctx, record); err != nil {
		t.Fatalf("CreateMovieImportUploadSession: %v", err)
	}
	if err := store.ReconcileMovieImportUploadCounters(ctx, record.UploadID,
		map[string]int64{"file-a": 4, "file-b": 3}, 7, now.Add(time.Hour), now.Add(25*time.Hour)); err != nil {
		t.Fatalf("ReconcileMovieImportUploadCounters: %v", err)
	}
	loaded, err := store.GetMovieImportUploadSession(ctx, record.UploadID)
	if err != nil {
		t.Fatalf("GetMovieImportUploadSession: %v", err)
	}
	if loaded.BytesReceived != 7 || loaded.Files[0].BytesReceived != 4 || loaded.Files[1].BytesReceived != 3 {
		t.Fatalf("loaded=%+v", loaded)
	}
}

func TestMovieImportUploadExpirationDoesNotRaceWithChunkTTLRefresh(t *testing.T) {
	ctx := context.Background()
	store := newMovieImportUploadRepositoryTestStore(t)
	defer func() { _ = store.Close() }()
	now := time.Date(2026, 7, 20, 4, 0, 0, 0, time.UTC)
	record := movieImportUploadRepositoryTestRecord(t, now)
	record.ExpiresAt = formatMovieImportUploadTime(now.Add(-time.Minute))
	if err := store.CreateMovieImportUploadSession(ctx, record); err != nil {
		t.Fatalf("CreateMovieImportUploadSession: %v", err)
	}
	candidates, err := store.ListMovieImportUploadCleanupCandidates(ctx, now)
	if err != nil || len(candidates) != 1 {
		t.Fatalf("initial candidates=%+v err=%v", candidates, err)
	}
	if _, err := store.RecordMovieImportUploadChunk(ctx, record.UploadID, "file-a", MovieImportUploadChunkRecord{
		ChunkIndex: 0, Offset: 0, Size: 4,
	}, now, now.Add(24*time.Hour)); err != nil {
		t.Fatalf("RecordMovieImportUploadChunk: %v", err)
	}
	err = store.ExpireMovieImportUploadSession(ctx, record.UploadID, MovieImportUploadStateUploading,
		now, now.Add(time.Minute), "upload_expired", "expired")
	if !errors.Is(err, ErrMovieImportUploadNotDue) {
		t.Fatalf("ExpireMovieImportUploadSession error=%v", err)
	}
	loaded, err := store.GetMovieImportUploadSession(ctx, record.UploadID)
	if err != nil || loaded.State != MovieImportUploadStateUploading || loaded.BytesReceived != 4 {
		t.Fatalf("loaded=%+v err=%v", loaded, err)
	}
}

func TestMovieImportUploadRepositoryRejectsInvalidManifestTotals(t *testing.T) {
	store := newMovieImportUploadRepositoryTestStore(t)
	defer func() { _ = store.Close() }()
	record := movieImportUploadRepositoryTestRecord(t, time.Now())
	record.TotalBytes++
	if err := store.CreateMovieImportUploadSession(context.Background(), record); err == nil {
		t.Fatal("CreateMovieImportUploadSession unexpectedly succeeded")
	}
}

func newMovieImportUploadRepositoryTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "curated.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	if err := store.Migrate(context.Background()); err != nil {
		_ = store.Close()
		t.Fatalf("Migrate: %v", err)
	}
	return store
}

func movieImportUploadRepositoryTestRecord(t *testing.T, now time.Time) MovieImportUploadSessionRecord {
	t.Helper()
	root := t.TempDir()
	return MovieImportUploadSessionRecord{
		UploadID:            "upload_test",
		TaskID:              "import.movies-test",
		TargetLibraryPathID: "library-test",
		TargetRoot:          root,
		StagingDir:          filepath.Join(root, ".curated-import", "upload_test"),
		State:               MovieImportUploadStateUploading,
		ChunkSize:           4,
		TotalBytes:          15,
		CreatedAt:           formatMovieImportUploadTime(now),
		UpdatedAt:           formatMovieImportUploadTime(now),
		ExpiresAt:           formatMovieImportUploadTime(now.Add(24 * time.Hour)),
		Files: []MovieImportUploadFileRecord{
			{
				FileID:       "file-a",
				Ordinal:      0,
				RelativePath: "folder/movie-a.mkv",
				Size:         10,
				StagingPath:  filepath.Join(root, ".curated-import", "upload_test", "file-a.part"),
				FinalPath:    filepath.Join(root, "folder", "movie-a.mkv"),
			},
			{
				FileID:       "file-b",
				Ordinal:      1,
				RelativePath: "movie-b.mp4",
				Size:         5,
				StagingPath:  filepath.Join(root, ".curated-import", "upload_test", "file-b.part"),
				FinalPath:    filepath.Join(root, "movie-b.mp4"),
			},
		},
	}
}

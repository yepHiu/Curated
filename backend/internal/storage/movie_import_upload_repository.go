package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

var (
	ErrMovieImportUploadNotFound      = errors.New("movie import upload not found")
	ErrMovieImportUploadStateConflict = errors.New("movie import upload state conflict")
	ErrMovieImportUploadChunkConflict = errors.New("movie import upload chunk conflict")
	ErrMovieImportUploadChunkOverlap  = errors.New("movie import upload chunk overlaps an existing range")
	ErrMovieImportUploadNotDue        = errors.New("movie import upload is not due for cleanup")
)

const (
	MovieImportUploadStateUploading     = "uploading"
	MovieImportUploadStateCommitting    = "committing"
	MovieImportUploadStateCommitted     = "committed"
	MovieImportUploadStateAborted       = "aborted"
	MovieImportUploadStateExpired       = "expired"
	MovieImportUploadStateUnrecoverable = "unrecoverable"

	MovieImportUploadFileStatePending   = "pending"
	MovieImportUploadFileStateCommitted = "committed"
)

type MovieImportUploadChunkRecord struct {
	ChunkIndex int
	Offset     int64
	Size       int64
	CreatedAt  string
}

type MovieImportUploadFileRecord struct {
	FileID        string
	Ordinal       int
	RelativePath  string
	Size          int64
	StagingPath   string
	FinalPath     string
	BytesReceived int64
	State         string
	CommittedAt   string
	Chunks        []MovieImportUploadChunkRecord
}

type MovieImportUploadSessionRecord struct {
	UploadID            string
	TaskID              string
	TargetLibraryPathID string
	TargetRoot          string
	StagingDir          string
	State               string
	ChunkSize           int64
	TotalBytes          int64
	BytesReceived       int64
	CreatedAt           string
	UpdatedAt           string
	ExpiresAt           string
	CleanupAfter        string
	DiagnosticCode      string
	DiagnosticMessage   string
	Files               []MovieImportUploadFileRecord
}

type MovieImportUploadChunkWriteResult struct {
	Duplicate            bool
	FileBytesReceived    int64
	SessionBytesReceived int64
}

type MovieImportUploadCleanupCandidate struct {
	UploadID          string
	TargetRoot        string
	StagingDir        string
	State             string
	ExpiresAt         string
	CleanupAfter      string
	DiagnosticCode    string
	DiagnosticMessage string
}

type MovieImportUploadCleanupAudit struct {
	ID          int64
	UploadID    string
	Reason      string
	PriorState  string
	StagingDir  string
	Outcome     string
	Diagnostic  string
	CreatedAt   string
	CompletedAt string
}

func (s *SQLiteStore) CreateMovieImportUploadSession(ctx context.Context, record MovieImportUploadSessionRecord) error {
	if err := validateMovieImportUploadRecord(record); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin create movie import upload: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO movie_import_upload_sessions (
			upload_id, task_id, target_library_path_id, target_root, staging_dir,
			state, chunk_size, total_bytes, bytes_received, created_at, updated_at,
			expires_at, cleanup_after, diagnostic_code, diagnostic_message
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, record.UploadID, record.TaskID, record.TargetLibraryPathID, record.TargetRoot, record.StagingDir,
		record.State, record.ChunkSize, record.TotalBytes, record.BytesReceived,
		record.CreatedAt, record.UpdatedAt, record.ExpiresAt, record.CleanupAfter,
		record.DiagnosticCode, record.DiagnosticMessage); err != nil {
		return fmt.Errorf("insert movie import upload session: %w", err)
	}

	for _, file := range record.Files {
		state := strings.TrimSpace(file.State)
		if state == "" {
			state = MovieImportUploadFileStatePending
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO movie_import_upload_files (
				upload_id, file_id, ordinal, relative_path, size_bytes, staging_path,
				final_path, bytes_received, state, committed_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, record.UploadID, file.FileID, file.Ordinal, file.RelativePath, file.Size,
			file.StagingPath, file.FinalPath, file.BytesReceived, state, file.CommittedAt); err != nil {
			return fmt.Errorf("insert movie import upload file %s: %w", file.FileID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit movie import upload session: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetMovieImportUploadSession(ctx context.Context, uploadID string) (MovieImportUploadSessionRecord, error) {
	uploadID = strings.TrimSpace(uploadID)
	if uploadID == "" {
		return MovieImportUploadSessionRecord{}, ErrMovieImportUploadNotFound
	}
	records, err := s.loadMovieImportUploadSessions(ctx, `WHERE upload_id = ?`, uploadID)
	if err != nil {
		return MovieImportUploadSessionRecord{}, err
	}
	if len(records) == 0 {
		return MovieImportUploadSessionRecord{}, ErrMovieImportUploadNotFound
	}
	return records[0], nil
}

func (s *SQLiteStore) ListRecoverableMovieImportUploadSessions(ctx context.Context) ([]MovieImportUploadSessionRecord, error) {
	return s.loadMovieImportUploadSessions(ctx,
		`WHERE state IN (?, ?) ORDER BY created_at, upload_id`,
		MovieImportUploadStateUploading, MovieImportUploadStateCommitting)
}

func (s *SQLiteStore) loadMovieImportUploadSessions(ctx context.Context, where string, args ...any) ([]MovieImportUploadSessionRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT upload_id, task_id, target_library_path_id, target_root, staging_dir,
		       state, chunk_size, total_bytes, bytes_received, created_at, updated_at,
		       expires_at, cleanup_after, diagnostic_code, diagnostic_message
		FROM movie_import_upload_sessions
		`+where, args...)
	if err != nil {
		return nil, fmt.Errorf("query movie import upload sessions: %w", err)
	}
	var records []MovieImportUploadSessionRecord
	for rows.Next() {
		var record MovieImportUploadSessionRecord
		if err := rows.Scan(
			&record.UploadID, &record.TaskID, &record.TargetLibraryPathID, &record.TargetRoot,
			&record.StagingDir, &record.State, &record.ChunkSize, &record.TotalBytes,
			&record.BytesReceived, &record.CreatedAt, &record.UpdatedAt, &record.ExpiresAt,
			&record.CleanupAfter, &record.DiagnosticCode, &record.DiagnosticMessage,
		); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan movie import upload session: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, fmt.Errorf("iterate movie import upload sessions: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close movie import upload sessions: %w", err)
	}

	for index := range records {
		files, err := s.loadMovieImportUploadFiles(ctx, records[index].UploadID)
		if err != nil {
			return nil, err
		}
		records[index].Files = files
	}
	return records, nil
}

func (s *SQLiteStore) loadMovieImportUploadFiles(ctx context.Context, uploadID string) ([]MovieImportUploadFileRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT file_id, ordinal, relative_path, size_bytes, staging_path, final_path,
		       bytes_received, state, committed_at
		FROM movie_import_upload_files
		WHERE upload_id = ?
		ORDER BY ordinal, file_id
	`, uploadID)
	if err != nil {
		return nil, fmt.Errorf("query movie import upload files: %w", err)
	}
	var files []MovieImportUploadFileRecord
	for rows.Next() {
		var file MovieImportUploadFileRecord
		if err := rows.Scan(&file.FileID, &file.Ordinal, &file.RelativePath, &file.Size,
			&file.StagingPath, &file.FinalPath, &file.BytesReceived, &file.State, &file.CommittedAt); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan movie import upload file: %w", err)
		}
		files = append(files, file)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, fmt.Errorf("iterate movie import upload files: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close movie import upload files: %w", err)
	}

	for index := range files {
		chunks, err := s.loadMovieImportUploadChunks(ctx, uploadID, files[index].FileID)
		if err != nil {
			return nil, err
		}
		files[index].Chunks = chunks
	}
	return files, nil
}

func (s *SQLiteStore) loadMovieImportUploadChunks(ctx context.Context, uploadID, fileID string) ([]MovieImportUploadChunkRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT chunk_index, offset_bytes, size_bytes, created_at
		FROM movie_import_upload_chunks
		WHERE upload_id = ? AND file_id = ?
		ORDER BY offset_bytes, chunk_index
	`, uploadID, fileID)
	if err != nil {
		return nil, fmt.Errorf("query movie import upload chunks: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var chunks []MovieImportUploadChunkRecord
	for rows.Next() {
		var chunk MovieImportUploadChunkRecord
		if err := rows.Scan(&chunk.ChunkIndex, &chunk.Offset, &chunk.Size, &chunk.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan movie import upload chunk: %w", err)
		}
		chunks = append(chunks, chunk)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate movie import upload chunks: %w", err)
	}
	return chunks, nil
}

func (s *SQLiteStore) RecordMovieImportUploadChunk(
	ctx context.Context,
	uploadID string,
	fileID string,
	chunk MovieImportUploadChunkRecord,
	updatedAt time.Time,
	expiresAt time.Time,
) (MovieImportUploadChunkWriteResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return MovieImportUploadChunkWriteResult{}, fmt.Errorf("begin record movie import upload chunk: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var state string
	var totalBytes, sessionBytes int64
	if err := tx.QueryRowContext(ctx, `
		SELECT state, total_bytes, bytes_received
		FROM movie_import_upload_sessions WHERE upload_id = ?
	`, uploadID).Scan(&state, &totalBytes, &sessionBytes); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return MovieImportUploadChunkWriteResult{}, ErrMovieImportUploadNotFound
		}
		return MovieImportUploadChunkWriteResult{}, fmt.Errorf("load movie import upload for chunk: %w", err)
	}
	if state != MovieImportUploadStateUploading {
		return MovieImportUploadChunkWriteResult{}, fmt.Errorf("%w: state=%s", ErrMovieImportUploadStateConflict, state)
	}

	var fileSize, fileBytes int64
	if err := tx.QueryRowContext(ctx, `
		SELECT size_bytes, bytes_received
		FROM movie_import_upload_files WHERE upload_id = ? AND file_id = ?
	`, uploadID, fileID).Scan(&fileSize, &fileBytes); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return MovieImportUploadChunkWriteResult{}, ErrMovieImportUploadNotFound
		}
		return MovieImportUploadChunkWriteResult{}, fmt.Errorf("load movie import upload file for chunk: %w", err)
	}
	if chunk.ChunkIndex < 0 || chunk.Offset < 0 || chunk.Size <= 0 || chunk.Offset > fileSize-chunk.Size {
		return MovieImportUploadChunkWriteResult{}, fmt.Errorf("%w: chunk range is outside file bounds", ErrMovieImportUploadChunkConflict)
	}

	var existingOffset, existingSize int64
	err = tx.QueryRowContext(ctx, `
		SELECT offset_bytes, size_bytes FROM movie_import_upload_chunks
		WHERE upload_id = ? AND file_id = ? AND chunk_index = ?
	`, uploadID, fileID, chunk.ChunkIndex).Scan(&existingOffset, &existingSize)
	switch {
	case err == nil:
		if existingOffset != chunk.Offset || existingSize != chunk.Size {
			return MovieImportUploadChunkWriteResult{}, ErrMovieImportUploadChunkConflict
		}
		return MovieImportUploadChunkWriteResult{
			Duplicate:            true,
			FileBytesReceived:    fileBytes,
			SessionBytesReceived: sessionBytes,
		}, nil
	case !errors.Is(err, sql.ErrNoRows):
		return MovieImportUploadChunkWriteResult{}, fmt.Errorf("inspect existing movie import upload chunk: %w", err)
	}

	var overlap int
	if err := tx.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM movie_import_upload_chunks
			WHERE upload_id = ? AND file_id = ?
			  AND offset_bytes < ?
			  AND offset_bytes + size_bytes > ?
		)
	`, uploadID, fileID, chunk.Offset+chunk.Size, chunk.Offset).Scan(&overlap); err != nil {
		return MovieImportUploadChunkWriteResult{}, fmt.Errorf("check movie import upload chunk overlap: %w", err)
	}
	if overlap != 0 {
		return MovieImportUploadChunkWriteResult{}, ErrMovieImportUploadChunkOverlap
	}
	if fileBytes > fileSize-chunk.Size || sessionBytes > totalBytes-chunk.Size {
		return MovieImportUploadChunkWriteResult{}, fmt.Errorf("%w: persisted byte counters exceed manifest", ErrMovieImportUploadChunkConflict)
	}

	createdAt := strings.TrimSpace(chunk.CreatedAt)
	if createdAt == "" {
		createdAt = formatMovieImportUploadTime(updatedAt)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO movie_import_upload_chunks (
			upload_id, file_id, chunk_index, offset_bytes, size_bytes, created_at
		) VALUES (?, ?, ?, ?, ?, ?)
	`, uploadID, fileID, chunk.ChunkIndex, chunk.Offset, chunk.Size, createdAt); err != nil {
		return MovieImportUploadChunkWriteResult{}, fmt.Errorf("insert movie import upload chunk: %w", err)
	}
	fileBytes += chunk.Size
	sessionBytes += chunk.Size
	if _, err := tx.ExecContext(ctx, `
		UPDATE movie_import_upload_files SET bytes_received = ?
		WHERE upload_id = ? AND file_id = ?
	`, fileBytes, uploadID, fileID); err != nil {
		return MovieImportUploadChunkWriteResult{}, fmt.Errorf("update movie import upload file bytes: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE movie_import_upload_sessions
		SET bytes_received = ?, updated_at = ?, expires_at = ?, diagnostic_code = '', diagnostic_message = ''
		WHERE upload_id = ?
	`, sessionBytes, formatMovieImportUploadTime(updatedAt), formatMovieImportUploadTime(expiresAt), uploadID); err != nil {
		return MovieImportUploadChunkWriteResult{}, fmt.Errorf("update movie import upload session bytes: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return MovieImportUploadChunkWriteResult{}, fmt.Errorf("commit movie import upload chunk: %w", err)
	}
	return MovieImportUploadChunkWriteResult{
		FileBytesReceived:    fileBytes,
		SessionBytesReceived: sessionBytes,
	}, nil
}

func (s *SQLiteStore) TransitionMovieImportUploadState(
	ctx context.Context,
	uploadID string,
	expectedStates []string,
	newState string,
	updatedAt time.Time,
	expiresAt string,
	cleanupAfter string,
	diagnosticCode string,
	diagnosticMessage string,
) error {
	if strings.TrimSpace(uploadID) == "" || len(expectedStates) == 0 || strings.TrimSpace(newState) == "" {
		return errors.New("movie import upload transition requires id states and destination state")
	}
	placeholders := make([]string, len(expectedStates))
	args := make([]any, 0, 7+len(expectedStates))
	args = append(args, newState, formatMovieImportUploadTime(updatedAt), strings.TrimSpace(expiresAt),
		strings.TrimSpace(cleanupAfter), strings.TrimSpace(diagnosticCode), strings.TrimSpace(diagnosticMessage), uploadID)
	for index, state := range expectedStates {
		placeholders[index] = "?"
		args = append(args, state)
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE movie_import_upload_sessions
		SET state = ?, updated_at = ?, expires_at = ?, cleanup_after = ?,
		    diagnostic_code = ?, diagnostic_message = ?
		WHERE upload_id = ? AND state IN (`+strings.Join(placeholders, ",")+`)`, args...)
	if err != nil {
		return fmt.Errorf("transition movie import upload state: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read movie import upload transition result: %w", err)
	}
	if affected == 1 {
		return nil
	}
	var currentState string
	if err := s.db.QueryRowContext(ctx, `SELECT state FROM movie_import_upload_sessions WHERE upload_id = ?`, uploadID).Scan(&currentState); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrMovieImportUploadNotFound
		}
		return fmt.Errorf("inspect movie import upload transition conflict: %w", err)
	}
	return fmt.Errorf("%w: state=%s", ErrMovieImportUploadStateConflict, currentState)
}

func (s *SQLiteStore) ExpireMovieImportUploadSession(
	ctx context.Context,
	uploadID string,
	expectedState string,
	now time.Time,
	cleanupAfter time.Time,
	diagnosticCode string,
	diagnosticMessage string,
) error {
	nowValue := formatMovieImportUploadTime(now)
	result, err := s.db.ExecContext(ctx, `
		UPDATE movie_import_upload_sessions
		SET state = ?, updated_at = ?, expires_at = '', cleanup_after = ?,
		    diagnostic_code = ?, diagnostic_message = ?
		WHERE upload_id = ? AND state = ? AND julianday(expires_at) <= julianday(?)
	`, MovieImportUploadStateExpired, nowValue, formatMovieImportUploadTime(cleanupAfter),
		strings.TrimSpace(diagnosticCode), strings.TrimSpace(diagnosticMessage),
		uploadID, expectedState, nowValue)
	if err != nil {
		return fmt.Errorf("expire movie import upload session: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read movie import upload expiration result: %w", err)
	}
	if affected == 1 {
		return nil
	}
	var state, expiresAt string
	if err := s.db.QueryRowContext(ctx, `
		SELECT state, expires_at FROM movie_import_upload_sessions WHERE upload_id = ?
	`, uploadID).Scan(&state, &expiresAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrMovieImportUploadNotFound
		}
		return fmt.Errorf("inspect movie import upload expiration conflict: %w", err)
	}
	if state == expectedState {
		parsedExpiry, parseErr := time.Parse(time.RFC3339Nano, expiresAt)
		if parseErr == nil && parsedExpiry.After(now) {
			return ErrMovieImportUploadNotDue
		}
	}
	return fmt.Errorf("%w: state=%s", ErrMovieImportUploadStateConflict, state)
}

func (s *SQLiteStore) MarkMovieImportUploadFileCommitted(ctx context.Context, uploadID, fileID string, committedAt time.Time) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE movie_import_upload_files
		SET state = ?, committed_at = ?
		WHERE upload_id = ? AND file_id = ? AND state = ?
	`, MovieImportUploadFileStateCommitted, formatMovieImportUploadTime(committedAt), uploadID, fileID, MovieImportUploadFileStatePending)
	if err != nil {
		return fmt.Errorf("mark movie import upload file committed: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read movie import upload file commit result: %w", err)
	}
	if affected == 1 {
		return nil
	}
	var state string
	if err := s.db.QueryRowContext(ctx, `
		SELECT state FROM movie_import_upload_files WHERE upload_id = ? AND file_id = ?
	`, uploadID, fileID).Scan(&state); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrMovieImportUploadNotFound
		}
		return fmt.Errorf("inspect movie import upload file commit: %w", err)
	}
	if state == MovieImportUploadFileStateCommitted {
		return nil
	}
	return fmt.Errorf("%w: file state=%s", ErrMovieImportUploadStateConflict, state)
}

func (s *SQLiteStore) ReconcileMovieImportUploadCounters(
	ctx context.Context,
	uploadID string,
	fileBytes map[string]int64,
	sessionBytes int64,
	updatedAt time.Time,
	expiresAt time.Time,
) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin reconcile movie import upload: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for fileID, bytesReceived := range fileBytes {
		result, err := tx.ExecContext(ctx, `
			UPDATE movie_import_upload_files SET bytes_received = ?
			WHERE upload_id = ? AND file_id = ? AND ? BETWEEN 0 AND size_bytes
		`, bytesReceived, uploadID, fileID, bytesReceived)
		if err != nil {
			return fmt.Errorf("reconcile movie import upload file %s: %w", fileID, err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("read reconcile movie import upload file %s result: %w", fileID, err)
		}
		if affected != 1 {
			return fmt.Errorf("reconcile movie import upload file %s affected %d rows", fileID, affected)
		}
	}
	result, err := tx.ExecContext(ctx, `
		UPDATE movie_import_upload_sessions
		SET bytes_received = ?, updated_at = ?, expires_at = ?, diagnostic_code = '', diagnostic_message = ''
		WHERE upload_id = ? AND ? BETWEEN 0 AND total_bytes
	`, sessionBytes, formatMovieImportUploadTime(updatedAt), formatMovieImportUploadTime(expiresAt), uploadID, sessionBytes)
	if err != nil {
		return fmt.Errorf("reconcile movie import upload session: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read reconcile movie import upload session result: %w", err)
	}
	if affected != 1 {
		return fmt.Errorf("reconcile movie import upload session affected %d rows", affected)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit movie import upload reconciliation: %w", err)
	}
	return nil
}

func (s *SQLiteStore) ListMovieImportUploadCleanupCandidates(ctx context.Context, now time.Time) ([]MovieImportUploadCleanupCandidate, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT upload_id, target_root, staging_dir, state, expires_at, cleanup_after,
		       diagnostic_code, diagnostic_message
		FROM movie_import_upload_sessions
		WHERE state IN (?, ?, ?, ?)
		   OR (state = ? AND julianday(expires_at) <= julianday(?))
		   OR (state = ? AND julianday(expires_at) <= julianday(?))
		ORDER BY updated_at, upload_id
	`, MovieImportUploadStateCommitted, MovieImportUploadStateAborted,
		MovieImportUploadStateExpired, MovieImportUploadStateUnrecoverable,
		MovieImportUploadStateUploading, formatMovieImportUploadTime(now),
		MovieImportUploadStateCommitting, formatMovieImportUploadTime(now))
	if err != nil {
		return nil, fmt.Errorf("list movie import upload cleanup candidates: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var candidates []MovieImportUploadCleanupCandidate
	for rows.Next() {
		var candidate MovieImportUploadCleanupCandidate
		if err := rows.Scan(&candidate.UploadID, &candidate.TargetRoot, &candidate.StagingDir,
			&candidate.State, &candidate.ExpiresAt, &candidate.CleanupAfter,
			&candidate.DiagnosticCode, &candidate.DiagnosticMessage); err != nil {
			return nil, fmt.Errorf("scan movie import upload cleanup candidate: %w", err)
		}
		candidates = append(candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate movie import upload cleanup candidates: %w", err)
	}
	return candidates, nil
}

func (s *SQLiteStore) ListMovieImportUploadSessionIDs(ctx context.Context) (map[string]struct{}, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT upload_id FROM movie_import_upload_sessions`)
	if err != nil {
		return nil, fmt.Errorf("list movie import upload ids: %w", err)
	}
	defer func() { _ = rows.Close() }()
	ids := make(map[string]struct{})
	for rows.Next() {
		var uploadID string
		if err := rows.Scan(&uploadID); err != nil {
			return nil, fmt.Errorf("scan movie import upload id: %w", err)
		}
		ids[uploadID] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate movie import upload ids: %w", err)
	}
	return ids, nil
}

func (s *SQLiteStore) DeleteMovieImportUploadSession(ctx context.Context, uploadID string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM movie_import_upload_sessions WHERE upload_id = ?`, uploadID)
	if err != nil {
		return fmt.Errorf("delete movie import upload session: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read movie import upload delete result: %w", err)
	}
	if affected == 0 {
		return ErrMovieImportUploadNotFound
	}
	return nil
}

func (s *SQLiteStore) CreateMovieImportUploadCleanupAudit(
	ctx context.Context,
	uploadID string,
	reason string,
	priorState string,
	stagingDir string,
	diagnostic string,
	createdAt time.Time,
) (int64, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO movie_import_upload_cleanup_audits (
			upload_id, reason, prior_state, staging_dir, outcome, diagnostic, created_at
		) VALUES (?, ?, ?, ?, 'pending', ?, ?)
	`, uploadID, reason, priorState, stagingDir, diagnostic, formatMovieImportUploadTime(createdAt))
	if err != nil {
		return 0, fmt.Errorf("create movie import upload cleanup audit: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("read movie import upload cleanup audit id: %w", err)
	}
	return id, nil
}

func (s *SQLiteStore) CompleteMovieImportUploadCleanupAudit(
	ctx context.Context,
	auditID int64,
	outcome string,
	diagnostic string,
	completedAt time.Time,
) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE movie_import_upload_cleanup_audits
		SET outcome = ?, diagnostic = ?, completed_at = ?
		WHERE id = ? AND outcome = 'pending'
	`, outcome, diagnostic, formatMovieImportUploadTime(completedAt), auditID)
	if err != nil {
		return fmt.Errorf("complete movie import upload cleanup audit: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read movie import upload cleanup audit result: %w", err)
	}
	if affected != 1 {
		return fmt.Errorf("complete movie import upload cleanup audit affected %d rows", affected)
	}
	return nil
}

func (s *SQLiteStore) ListMovieImportUploadCleanupAudits(ctx context.Context, limit int) ([]MovieImportUploadCleanupAudit, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, upload_id, reason, prior_state, staging_dir, outcome, diagnostic, created_at, completed_at
		FROM movie_import_upload_cleanup_audits
		ORDER BY id DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list movie import upload cleanup audits: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var audits []MovieImportUploadCleanupAudit
	for rows.Next() {
		var audit MovieImportUploadCleanupAudit
		if err := rows.Scan(&audit.ID, &audit.UploadID, &audit.Reason, &audit.PriorState,
			&audit.StagingDir, &audit.Outcome, &audit.Diagnostic, &audit.CreatedAt, &audit.CompletedAt); err != nil {
			return nil, fmt.Errorf("scan movie import upload cleanup audit: %w", err)
		}
		audits = append(audits, audit)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate movie import upload cleanup audits: %w", err)
	}
	return audits, nil
}

func validateMovieImportUploadRecord(record MovieImportUploadSessionRecord) error {
	if strings.TrimSpace(record.UploadID) == "" || strings.TrimSpace(record.TaskID) == "" ||
		strings.TrimSpace(record.TargetLibraryPathID) == "" || strings.TrimSpace(record.TargetRoot) == "" ||
		strings.TrimSpace(record.StagingDir) == "" {
		return errors.New("movie import upload identity and paths are required")
	}
	if record.State == "" || record.ChunkSize <= 0 || record.TotalBytes <= 0 ||
		record.BytesReceived < 0 || record.BytesReceived > record.TotalBytes {
		return errors.New("movie import upload state and byte counts are invalid")
	}
	if len(record.Files) == 0 {
		return errors.New("movie import upload requires at least one file")
	}
	files := append([]MovieImportUploadFileRecord(nil), record.Files...)
	sort.Slice(files, func(i, j int) bool { return files[i].Ordinal < files[j].Ordinal })
	var total int64
	for index, file := range files {
		if file.Ordinal != index || strings.TrimSpace(file.FileID) == "" || strings.TrimSpace(file.RelativePath) == "" ||
			strings.TrimSpace(file.StagingPath) == "" || strings.TrimSpace(file.FinalPath) == "" || file.Size <= 0 ||
			file.BytesReceived < 0 || file.BytesReceived > file.Size {
			return fmt.Errorf("movie import upload file %d is invalid", index)
		}
		if total > record.TotalBytes-file.Size {
			return errors.New("movie import upload file sizes overflow total bytes")
		}
		total += file.Size
	}
	if total != record.TotalBytes {
		return fmt.Errorf("movie import upload total bytes %d does not match files %d", record.TotalBytes, total)
	}
	return nil
}

func formatMovieImportUploadTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

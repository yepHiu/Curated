package storage

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"curated-backend/internal/contracts"
	"curated-backend/internal/library/moviecode"
)

// ScanPersistOutcome describes the result of persisting a single scanned movie file.
type ScanPersistOutcome struct {
	MovieID string
	Status  string
	Reason  string
}

// SaveTask inserts or updates an async task record (scan, scrape, etc.) in scan_jobs.
func (s *SQLiteStore) SaveTask(ctx context.Context, task contracts.TaskDTO) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO scan_jobs (
			task_id, type, status, progress, message, error_code, error_message, created_at, started_at, finished_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(task_id) DO UPDATE SET
			type = excluded.type,
			status = excluded.status,
			progress = excluded.progress,
			message = excluded.message,
			error_code = excluded.error_code,
			error_message = excluded.error_message,
			created_at = excluded.created_at,
			started_at = excluded.started_at,
			finished_at = excluded.finished_at`,
		task.TaskID,
		task.Type,
		task.Status,
		task.Progress,
		task.Message,
		task.ErrorCode,
		task.ErrorMessage,
		task.CreatedAt,
		task.StartedAt,
		task.FinishedAt,
	)
	return err
}

// SaveScanItem inserts or updates a single file scan result row in scan_items.
func (s *SQLiteStore) SaveScanItem(ctx context.Context, result contracts.ScanFileResultDTO) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO scan_items (
			task_id, path, file_name, number, movie_id, status, reason, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(task_id, path) DO UPDATE SET
			file_name = excluded.file_name,
			number = excluded.number,
			movie_id = excluded.movie_id,
			status = excluded.status,
			reason = excluded.reason,
			updated_at = excluded.updated_at`,
		result.TaskID,
		result.Path,
		result.FileName,
		result.Number,
		result.MovieID,
		result.Status,
		result.Reason,
		nowUTC(),
		nowUTC(),
	)
	return err
}

// PersistScanMovie inserts a new movie or updates location for an existing one, deduplicating by code and path.
func (s *SQLiteStore) PersistScanMovie(ctx context.Context, result contracts.ScanFileResultDTO) (ScanPersistOutcome, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ScanPersistOutcome{}, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	located, pathErr := lookupScanMovie(ctx, tx, `location = ?`, result.Path)
	switch {
	case pathErr == nil:
		if err := tx.Commit(); err != nil {
			return ScanPersistOutcome{}, err
		}
		if movieRowIsTrashed(located.trashedAt) {
			reason := "trashed_path_indexed"
			if located.code == result.Number {
				reason = "trashed_already_indexed"
			}
			return ScanPersistOutcome{
				MovieID: located.id,
				Status:  "skipped",
				Reason:  reason,
			}, nil
		}
		if located.code == result.Number {
			return ScanPersistOutcome{
				MovieID: located.id,
				Status:  "skipped",
				Reason:  "already_indexed",
			}, nil
		}
		return ScanPersistOutcome{
			MovieID: located.id,
			Status:  "skipped",
			Reason:  "path_already_indexed",
		}, nil
	case !errors.Is(pathErr, sql.ErrNoRows):
		return ScanPersistOutcome{}, pathErr
	}

	matched, queryErr := lookupScanMovie(ctx, tx, `code = ?`, result.Number)
	switch {
	case errors.Is(queryErr, sql.ErrNoRows):
		movieID := moviecode.NormalizeForStorageID(result.Number)
		now := nowUTC()
		addedAt := time.Now().UTC().Format("2006-01-02")

		_, err = tx.ExecContext(
			ctx,
			`INSERT INTO movies (
				id, title, code, studio, summary, runtime_minutes, rating, is_favorite, added_at, location, resolution, year, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			movieID,
			result.Number,
			result.Number,
			"Unknown",
			"Metadata pending scrape.",
			0,
			0,
			0,
			addedAt,
			result.Path,
			strings.TrimPrefix(strings.ToLower(filepath.Ext(result.Path)), "."),
			0,
			now,
			now,
		)
		if isSQLiteUniqueConstraint(err) {
			if err := tx.Commit(); err != nil {
				return ScanPersistOutcome{}, err
			}
			return ScanPersistOutcome{
				MovieID: movieID,
				Status:  "skipped",
				Reason:  "unique_constraint",
			}, nil
		}
		if err != nil {
			return ScanPersistOutcome{}, err
		}

		if err := tx.Commit(); err != nil {
			return ScanPersistOutcome{}, err
		}
		return ScanPersistOutcome{
			MovieID: movieID,
			Status:  "imported",
		}, nil

	case queryErr != nil:
		return ScanPersistOutcome{}, queryErr
	}

	if movieRowIsTrashed(matched.trashedAt) {
		if err := tx.Commit(); err != nil {
			return ScanPersistOutcome{}, err
		}
		return ScanPersistOutcome{
			MovieID: matched.id,
			Status:  "skipped",
			Reason:  "trashed_code_indexed",
		}, nil
	}

	_, err = tx.ExecContext(
		ctx,
		`UPDATE movies SET location = ?, updated_at = ? WHERE id = ?`,
		result.Path,
		nowUTC(),
		matched.id,
	)
	if isSQLiteUniqueConstraint(err) {
		if err := tx.Commit(); err != nil {
			return ScanPersistOutcome{}, err
		}
		return ScanPersistOutcome{
			MovieID: matched.id,
			Status:  "skipped",
			Reason:  "unique_constraint",
		}, nil
	}
	if err != nil {
		return ScanPersistOutcome{}, err
	}

	if err := tx.Commit(); err != nil {
		return ScanPersistOutcome{}, err
	}
	return ScanPersistOutcome{
		MovieID: matched.id,
		Status:  "updated",
		Reason:  "path_refreshed",
	}, nil
}

type scanMovieLookup struct {
	id        string
	code      string
	location  string
	trashedAt string
}

func lookupScanMovie(ctx context.Context, tx *sql.Tx, where string, arg string) (scanMovieLookup, error) {
	var row scanMovieLookup
	err := tx.QueryRowContext(
		ctx,
		`SELECT id, code, location, IFNULL(trashed_at, '') FROM movies WHERE `+where+` LIMIT 1`,
		arg,
	).Scan(&row.id, &row.code, &row.location, &row.trashedAt)
	return row, err
}

func movieRowIsTrashed(trashedAt string) bool {
	return strings.TrimSpace(trashedAt) != ""
}

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}

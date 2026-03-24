package storage

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"jav-shadcn/backend/internal/contracts"
	"jav-shadcn/backend/internal/library/moviecode"
)

type ScanPersistOutcome struct {
	MovieID string
	Status  string
	Reason  string
}

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

func (s *SQLiteStore) PersistScanMovie(ctx context.Context, result contracts.ScanFileResultDTO) (ScanPersistOutcome, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ScanPersistOutcome{}, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var (
		movieID  string
		code     string
		location string
	)

	queryErr := tx.QueryRowContext(
		ctx,
		`SELECT id, code, location FROM movies WHERE code = ? OR location = ? LIMIT 1`,
		result.Number,
		result.Path,
	).Scan(&movieID, &code, &location)

	switch {
	case errors.Is(queryErr, sql.ErrNoRows):
		movieID = moviecode.NormalizeForStorageID(result.Number)
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

	if code == result.Number {
		if location == result.Path {
			if err := tx.Commit(); err != nil {
				return ScanPersistOutcome{}, err
			}
			return ScanPersistOutcome{
				MovieID: movieID,
				Status:  "skipped",
				Reason:  "already_indexed",
			}, nil
		}

		_, err = tx.ExecContext(
			ctx,
			`UPDATE movies SET location = ?, updated_at = ? WHERE id = ?`,
			result.Path,
			nowUTC(),
			movieID,
		)
		if err != nil {
			return ScanPersistOutcome{}, err
		}

		if err := tx.Commit(); err != nil {
			return ScanPersistOutcome{}, err
		}
		return ScanPersistOutcome{
			MovieID: movieID,
			Status:  "updated",
			Reason:  "path_refreshed",
		}, nil
	}

	if err := tx.Commit(); err != nil {
		return ScanPersistOutcome{}, err
	}
	return ScanPersistOutcome{
		MovieID: movieID,
		Status:  "skipped",
		Reason:  "path_already_indexed",
	}, nil
}

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}

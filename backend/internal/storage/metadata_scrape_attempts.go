package storage

import (
	"context"
	"strings"
	"time"
)

// MovieMetadataScrapeAttempt records the latest scrape outcome for one movie.
type MovieMetadataScrapeAttempt struct {
	MovieID       string
	TaskID        string
	Status        string
	ErrorCode     string
	ErrorCategory string
	ErrorMessage  string
	Provider      string
	StartedAt     string
	FinishedAt    string
	UpdatedAt     string
}

// StartMovieMetadataScrapeAttempt replaces the previous outcome with a running attempt.
func (s *SQLiteStore) StartMovieMetadataScrapeAttempt(ctx context.Context, movieID, taskID string, startedAt time.Time) error {
	now := startedAt.UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO movie_metadata_scrape_attempts (
			movie_id, task_id, status, started_at, updated_at
		) VALUES (?, ?, 'running', ?, ?)
		ON CONFLICT(movie_id) DO UPDATE SET
			task_id = excluded.task_id,
			status = 'running',
			error_code = '',
			error_category = '',
			error_message = '',
			provider = '',
			started_at = excluded.started_at,
			finished_at = '',
			updated_at = excluded.updated_at`,
		strings.TrimSpace(movieID), strings.TrimSpace(taskID), now, now)
	return err
}

// FinishMovieMetadataScrapeAttempt stores the terminal result only if taskID is still current.
func (s *SQLiteStore) FinishMovieMetadataScrapeAttempt(ctx context.Context, attempt MovieMetadataScrapeAttempt, finishedAt time.Time) error {
	status := strings.TrimSpace(attempt.Status)
	if status != "completed" && status != "failed" {
		status = "failed"
	}
	now := finishedAt.UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
		UPDATE movie_metadata_scrape_attempts
		SET status = ?, error_code = ?, error_category = ?, error_message = ?, provider = ?,
			finished_at = ?, updated_at = ?
		WHERE movie_id = ? AND task_id = ?`,
		status,
		strings.TrimSpace(attempt.ErrorCode),
		strings.TrimSpace(attempt.ErrorCategory),
		strings.TrimSpace(attempt.ErrorMessage),
		strings.TrimSpace(attempt.Provider),
		now, now,
		strings.TrimSpace(attempt.MovieID), strings.TrimSpace(attempt.TaskID))
	return err
}

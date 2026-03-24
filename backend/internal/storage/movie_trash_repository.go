package storage

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

// ErrMovieNotInTrash is returned when a permanent delete or restore requires the movie to be in trash.
var ErrMovieNotInTrash = errors.New("movie is not in trash")

// sqlMovieActiveClause matches rows not in the recycle bin (trashed_at empty or NULL).
const sqlMovieActiveClause = `(m.trashed_at IS NULL OR TRIM(m.trashed_at) = '')`

// sqlMovieTrashedClause matches rows in the recycle bin.
const sqlMovieTrashedClause = `(m.trashed_at IS NOT NULL AND TRIM(m.trashed_at) != '')`

// IsMovieTrashed reports whether the movie row exists and has a non-empty trashed_at.
func (s *SQLiteStore) IsMovieTrashed(ctx context.Context, movieID string) (bool, error) {
	movieID = strings.TrimSpace(movieID)
	if movieID == "" {
		return false, nil
	}
	var ts string
	err := s.db.QueryRowContext(ctx, `SELECT IFNULL(trashed_at, '') FROM movies WHERE id = ?`, movieID).Scan(&ts)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(ts) != "", nil
}

// TrashMovie sets trashed_at on an active movie. Idempotent: already trashed returns nil.
// Returns ErrMovieNotFound if no row exists.
func (s *SQLiteStore) TrashMovie(ctx context.Context, movieID string) error {
	movieID = strings.TrimSpace(movieID)
	if movieID == "" {
		return ErrMovieNotFound
	}
	ts := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx,
		`UPDATE movies SET trashed_at = ?, updated_at = ? WHERE id = ? AND (trashed_at IS NULL OR TRIM(trashed_at) = '')`,
		ts, ts, movieID,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	var existing string
	err = s.db.QueryRowContext(ctx, `SELECT IFNULL(trashed_at, '') FROM movies WHERE id = ?`, movieID).Scan(&existing)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrMovieNotFound
	}
	if err != nil {
		return err
	}
	if strings.TrimSpace(existing) != "" {
		return nil
	}
	return ErrMovieNotFound
}

// RestoreMovie clears trashed_at. Returns ErrMovieNotInTrash if not trashed, ErrMovieNotFound if no row.
func (s *SQLiteStore) RestoreMovie(ctx context.Context, movieID string) error {
	movieID = strings.TrimSpace(movieID)
	if movieID == "" {
		return ErrMovieNotFound
	}
	now := nowUTC()
	res, err := s.db.ExecContext(ctx,
		`UPDATE movies SET trashed_at = '', updated_at = ? WHERE id = ? AND trashed_at IS NOT NULL AND TRIM(trashed_at) != ''`,
		now, movieID,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	var one int
	err = s.db.QueryRowContext(ctx, `SELECT 1 FROM movies WHERE id = ? LIMIT 1`, movieID).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrMovieNotFound
	}
	if err != nil {
		return err
	}
	return ErrMovieNotInTrash
}

// DeleteMoviePermanently removes DB rows and on-disk files only if the movie is currently in trash.
func (s *SQLiteStore) DeleteMoviePermanently(ctx context.Context, movieID string, assetCacheRoot string) error {
	movieID = strings.TrimSpace(movieID)
	if movieID == "" {
		return ErrMovieNotFound
	}
	trashed, err := s.IsMovieTrashed(ctx, movieID)
	if err != nil {
		return err
	}
	if !trashed {
		return ErrMovieNotInTrash
	}
	return s.DeleteMovie(ctx, movieID, assetCacheRoot)
}

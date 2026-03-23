package storage

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"jav-shadcn/backend/internal/contracts"
)

// ErrMovieCommentTooLong is returned when comment body exceeds MaxMovieCommentRunes.
var ErrMovieCommentTooLong = errors.New("comment body too long")

// MovieRowExists reports whether a movies row exists for id.
func (s *SQLiteStore) MovieRowExists(ctx context.Context, movieID string) (bool, error) {
	movieID = strings.TrimSpace(movieID)
	if movieID == "" {
		return false, nil
	}
	var one int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM movies WHERE id = ? LIMIT 1`, movieID).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// GetMovieComment returns the saved note for the movie, or empty DTO when none exists.
// Caller should ensure the movie exists; this does not verify movies.id.
func (s *SQLiteStore) GetMovieComment(ctx context.Context, movieID string) (contracts.MovieCommentDTO, error) {
	movieID = strings.TrimSpace(movieID)
	if movieID == "" {
		return contracts.MovieCommentDTO{}, nil
	}
	var body, updatedAt string
	err := s.db.QueryRowContext(ctx,
		`SELECT body, updated_at FROM library_movie_comments WHERE movie_id = ?`, movieID,
	).Scan(&body, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return contracts.MovieCommentDTO{Body: "", UpdatedAt: ""}, nil
	}
	if err != nil {
		return contracts.MovieCommentDTO{}, err
	}
	return contracts.MovieCommentDTO{Body: body, UpdatedAt: updatedAt}, nil
}

// UpsertMovieComment replaces the note for a movie (must exist in movies). Body is trimmed; length validated by rune count.
func (s *SQLiteStore) UpsertMovieComment(ctx context.Context, movieID string, body string) (contracts.MovieCommentDTO, error) {
	movieID = strings.TrimSpace(movieID)
	if movieID == "" {
		return contracts.MovieCommentDTO{}, ErrMovieNotFound
	}
	body = strings.TrimSpace(body)
	if utf8.RuneCountInString(body) > contracts.MaxMovieCommentRunes {
		return contracts.MovieCommentDTO{}, ErrMovieCommentTooLong
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return contracts.MovieCommentDTO{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var one int
	switch err := tx.QueryRowContext(ctx, `SELECT 1 FROM movies WHERE id = ? LIMIT 1`, movieID).Scan(&one); {
	case errors.Is(err, sql.ErrNoRows):
		return contracts.MovieCommentDTO{}, ErrMovieNotFound
	case err != nil:
		return contracts.MovieCommentDTO{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO library_movie_comments (movie_id, body, updated_at) VALUES (?, ?, ?)
		ON CONFLICT(movie_id) DO UPDATE SET body = excluded.body, updated_at = excluded.updated_at`,
		movieID, body, now,
	)
	if err != nil {
		return contracts.MovieCommentDTO{}, err
	}
	if err := tx.Commit(); err != nil {
		return contracts.MovieCommentDTO{}, err
	}
	return contracts.MovieCommentDTO{Body: body, UpdatedAt: now}, nil
}

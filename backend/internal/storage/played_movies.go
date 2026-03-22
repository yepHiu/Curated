package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

func (s *SQLiteStore) ListPlayedMovieIDs(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT movie_id FROM library_played_movies ORDER BY first_played_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) RecordPlayedMovie(ctx context.Context, movieID string) error {
	if movieID == "" {
		return errors.New("empty movie id")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO library_played_movies (movie_id, first_played_at) VALUES (?, ?)
		ON CONFLICT(movie_id) DO NOTHING
	`, movieID, now)
	return err
}

func (s *SQLiteStore) CountPlayedMovies(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM library_played_movies`).Scan(&n)
	return n, err
}

// ErrPlayedMovieMovieNotFound is returned when the movie row does not exist.
var ErrPlayedMovieMovieNotFound = errors.New("movie not found")

func (s *SQLiteStore) RecordPlayedMovieIfMovieExists(ctx context.Context, movieID string) error {
	if movieID == "" {
		return errors.New("empty movie id")
	}
	var one int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM movies WHERE id = ? LIMIT 1`, movieID).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrPlayedMovieMovieNotFound
	}
	if err != nil {
		return err
	}
	return s.RecordPlayedMovie(ctx, movieID)
}

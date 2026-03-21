package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"jav-shadcn/backend/internal/contracts"
)

var (
	// ErrMovieNotFoundForPatch is returned when PATCH targets a missing movie id.
	ErrMovieNotFoundForPatch = errors.New("movie not found")
	// ErrInvalidUserRating is returned when user rating is outside 0..5.
	ErrInvalidUserRating = errors.New("user rating must be between 0 and 5")
)

func (s *SQLiteStore) PatchMovieUserPrefs(ctx context.Context, movieID string, patch contracts.PatchMovieInput) error {
	if movieID == "" {
		return ErrMovieNotFoundForPatch
	}

	var sets []string
	var args []any

	if patch.Favorite != nil {
		v := 0
		if *patch.Favorite {
			v = 1
		}
		sets = append(sets, "is_favorite = ?")
		args = append(args, v)
	}

	if patch.UserRatingSet {
		if patch.UserRatingClear {
			sets = append(sets, "user_rating = NULL")
		} else {
			if patch.UserRating < 0 || patch.UserRating > 5 {
				return fmt.Errorf("%w: got %v", ErrInvalidUserRating, patch.UserRating)
			}
			sets = append(sets, "user_rating = ?")
			args = append(args, patch.UserRating)
		}
	}

	if len(sets) == 0 {
		return nil
	}

	sets = append(sets, "updated_at = ?")
	args = append(args, nowUTC())
	args = append(args, movieID)

	q := "UPDATE movies SET " + strings.Join(sets, ", ") + " WHERE id = ?"
	res, err := s.db.ExecContext(ctx, q, args...)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrMovieNotFoundForPatch
	}
	return nil
}

func effectiveRating(metadata float64, user sql.NullFloat64) float64 {
	if user.Valid {
		return user.Float64
	}
	return metadata
}

func userRatingPtr(user sql.NullFloat64) *float64 {
	if !user.Valid {
		return nil
	}
	v := user.Float64
	return &v
}

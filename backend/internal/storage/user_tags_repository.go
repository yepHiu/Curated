package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

const tagTypeUser = "user"

// ErrInvalidUserTags is returned when user tag list fails validation (length/count).
var ErrInvalidUserTags = errors.New("invalid user tags")

const maxUserTagsPerMovie = 64
const maxUserTagRunes = 64

func normalizeUserTagsForPatch(raw []string) ([]string, error) {
	if raw == nil {
		raw = []string{}
	}
	if len(raw) > maxUserTagsPerMovie {
		return nil, fmt.Errorf("%w: at most %d tags per movie", ErrInvalidUserTags, maxUserTagsPerMovie)
	}
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, t := range raw {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if len([]rune(t)) > maxUserTagRunes {
			return nil, fmt.Errorf("%w: tag longer than %d characters", ErrInvalidUserTags, maxUserTagRunes)
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	if len(out) > maxUserTagsPerMovie {
		return nil, fmt.Errorf("%w: at most %d tags per movie", ErrInvalidUserTags, maxUserTagsPerMovie)
	}
	return out, nil
}

func ensureUserTag(ctx context.Context, tx *sql.Tx, name string) (int64, error) {
	_, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO tags (name, type) VALUES (?, ?)`, name, tagTypeUser)
	if err != nil {
		return 0, err
	}
	return lookupTagID(ctx, tx, name, tagTypeUser)
}

// replaceMovieUserTagsTx replaces all user-type tag links for a movie (full list). Verifies movie exists.
func replaceMovieUserTagsTx(ctx context.Context, tx *sql.Tx, movieID string, tags []string) error {
	var one int
	switch err := tx.QueryRowContext(ctx, `SELECT 1 FROM movies WHERE id = ?`, movieID).Scan(&one); {
	case errors.Is(err, sql.ErrNoRows):
		return ErrMovieNotFoundForPatch
	case err != nil:
		return err
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM movie_tags WHERE movie_id = ? AND tag_id IN (SELECT id FROM tags WHERE type = ?)`, movieID, tagTypeUser); err != nil {
		return err
	}

	for _, tag := range tags {
		tagID, err := ensureUserTag(ctx, tx, tag)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO movie_tags (movie_id, tag_id) VALUES (?, ?)`, movieID, tagID); err != nil {
			return err
		}
	}
	return nil
}

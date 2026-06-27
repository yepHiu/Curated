package storage

import (
	"context"
	"database/sql"
	"errors"

	"curated-backend/internal/contracts"
)

func (s *SQLiteStore) SaveComicReadingPreferences(ctx context.Context, comicID string, prefs contracts.ComicReadingPreferencesDTO) error {
	ts := prefs.UpdatedAt
	if ts == "" {
		ts = nowUTC()
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO comic_reading_preferences (comic_id, mode, fit, direction, updated_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(comic_id) DO UPDATE SET
		   mode = excluded.mode,
		   fit = excluded.fit,
		   direction = excluded.direction,
		   updated_at = excluded.updated_at`,
		comicID,
		prefs.Mode,
		prefs.Fit,
		prefs.Direction,
		ts,
	)
	return err
}

func (s *SQLiteStore) GetComicReadingPreferences(ctx context.Context, comicID string) (contracts.ComicReadingPreferencesDTO, error) {
	var dto contracts.ComicReadingPreferencesDTO
	err := s.db.QueryRowContext(ctx,
		`SELECT comic_id, mode, fit, direction, updated_at
		   FROM comic_reading_preferences
		  WHERE comic_id = ?`,
		comicID,
	).Scan(&dto.ComicID, &dto.Mode, &dto.Fit, &dto.Direction, &dto.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return contracts.ComicReadingPreferencesDTO{}, ErrComicBookNotFound
		}
		return contracts.ComicReadingPreferencesDTO{}, err
	}
	return dto, nil
}

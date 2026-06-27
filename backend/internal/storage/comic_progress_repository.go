package storage

import (
	"context"
	"database/sql"
	"errors"

	"curated-backend/internal/contracts"
)

func (s *SQLiteStore) SaveComicProgress(ctx context.Context, comicID string, pageIndex int, completed bool) error {
	ts := nowUTC()
	completedInt := 0
	readStatus := "reading"
	completedAt := ""
	if completed {
		completedInt = 1
		readStatus = "read"
		completedAt = ts
	} else if pageIndex <= 0 {
		readStatus = "unread"
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO comic_reading_progress (comic_id, current_page_index, completed, updated_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(comic_id) DO UPDATE SET
		   current_page_index = excluded.current_page_index,
		   completed = excluded.completed,
		   updated_at = excluded.updated_at`,
		comicID,
		pageIndex,
		completedInt,
		ts,
	)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`UPDATE comic_books
		    SET read_status = ?, last_read_at = ?, completed_at = CASE WHEN ? = 1 THEN ? ELSE completed_at END, updated_at = ?
		  WHERE id = ?`,
		readStatus,
		ts,
		completedInt,
		completedAt,
		ts,
		comicID,
	)
	return err
}

func (s *SQLiteStore) GetComicProgress(ctx context.Context, comicID string) (contracts.ComicReadingProgressDTO, error) {
	var dto contracts.ComicReadingProgressDTO
	var completedInt int
	err := s.db.QueryRowContext(ctx,
		`SELECT comic_id, current_page_index, completed, updated_at
		   FROM comic_reading_progress
		  WHERE comic_id = ?`,
		comicID,
	).Scan(&dto.ComicID, &dto.PageIndex, &completedInt, &dto.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return contracts.ComicReadingProgressDTO{}, ErrComicBookNotFound
		}
		return contracts.ComicReadingProgressDTO{}, err
	}
	dto.Completed = completedInt != 0
	return dto, nil
}

func (s *SQLiteStore) DeleteComicProgress(ctx context.Context, comicID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM comic_reading_progress WHERE comic_id = ?`, comicID)
	return err
}

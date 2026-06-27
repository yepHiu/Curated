package storage

import (
	"context"
	"database/sql"
)

func replaceComicTagsTx(ctx context.Context, tx *sql.Tx, comicID string, tags []string) error {
	var exists string
	if err := tx.QueryRowContext(ctx, `SELECT id FROM comic_books WHERE id = ?`, comicID).Scan(&exists); err != nil {
		if err == sql.ErrNoRows {
			return ErrComicBookNotFound
		}
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM comic_book_tags WHERE comic_id = ?`, comicID); err != nil {
		return err
	}
	ts := nowUTC()
	for _, tag := range tags {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO comic_tags (name, created_at) VALUES (?, ?)
			 ON CONFLICT(name) DO NOTHING`,
			tag,
			ts,
		); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO comic_book_tags (comic_id, tag, created_at) VALUES (?, ?, ?)
			 ON CONFLICT(comic_id, tag) DO NOTHING`,
			comicID,
			tag,
			ts,
		); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteStore) listComicTags(ctx context.Context, comicID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT tag FROM comic_book_tags WHERE comic_id = ? ORDER BY tag ASC`,
		comicID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]string, 0)
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		out = append(out, tag)
	}
	return out, rows.Err()
}

package storage

import (
	"context"
	"database/sql"
)

// replaceComicTagsTx 在事务内整表替换一本漫画的标签。
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

// listComicTags 读取单本漫画的全部标签，按名称升序。
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

// listComicTagsByIDs 一次取出本页漫画标签，按 comic_id 分组并保持 tag 升序。
func (s *SQLiteStore) listComicTagsByIDs(ctx context.Context, comicIDs []string) (map[string][]string, error) {
	out := make(map[string][]string, len(comicIDs))
	for _, id := range comicIDs {
		out[id] = []string{}
	}
	if len(comicIDs) == 0 {
		return out, nil
	}
	err := forEachInClauseBatch(comicIDs, func(batch []string) error {
		// 按最多 500 个 ID 一批查出标签行。
		query := `SELECT comic_id, tag FROM comic_book_tags WHERE comic_id IN (` + inClausePlaceholders(len(batch)) + `) ORDER BY comic_id ASC, tag ASC`
		rows, err := s.db.QueryContext(ctx, query, inClauseArgs(batch)...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id, tag string
			if err := rows.Scan(&id, &tag); err != nil {
				return err
			}
			out[id] = append(out[id], tag)
		}
		return rows.Err()
	})
	return out, err
}

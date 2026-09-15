package storage

import (
	"context"
	"database/sql"
)

func (s *SQLiteStore) ReplacePhotoTags(ctx context.Context, photoID string, raw []string) error {
	tags, err := NormalizeUserTagsForPatch(raw)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := replacePhotoTagsTx(ctx, tx, photoID, tags); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE photo_books SET updated_at = ? WHERE id = ?`, nowUTC(), photoID); err != nil {
		return err
	}
	return tx.Commit()
}

func replacePhotoTagsTx(ctx context.Context, tx *sql.Tx, photoID string, tags []string) error {
	var exists string
	if err := tx.QueryRowContext(ctx, `SELECT id FROM photo_books WHERE id = ?`, photoID).Scan(&exists); err != nil {
		if err == sql.ErrNoRows {
			return ErrPhotoBookNotFound
		}
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM photo_book_tags WHERE photo_id = ?`, photoID); err != nil {
		return err
	}
	ts := nowUTC()
	for _, tag := range tags {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO photo_tags (name, created_at) VALUES (?, ?)
			 ON CONFLICT(name) DO NOTHING`,
			tag,
			ts,
		); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO photo_book_tags (photo_id, tag, created_at) VALUES (?, ?, ?)
			 ON CONFLICT(photo_id, tag) DO NOTHING`,
			photoID,
			tag,
			ts,
		); err != nil {
			return err
		}
	}
	return nil
}

// listPhotoTags 读取单本写真的全部标签，按名称升序。
func (s *SQLiteStore) listPhotoTags(ctx context.Context, photoID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT tag FROM photo_book_tags WHERE photo_id = ? ORDER BY tag ASC`,
		photoID,
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

// listPhotoTagsByIDs 一次取出本页写真标签，按 photo_id 分组并保持 tag 升序。
func (s *SQLiteStore) listPhotoTagsByIDs(ctx context.Context, photoIDs []string) (map[string][]string, error) {
	out := make(map[string][]string, len(photoIDs))
	for _, id := range photoIDs {
		out[id] = []string{}
	}
	if len(photoIDs) == 0 {
		return out, nil
	}
	err := forEachInClauseBatch(photoIDs, func(batch []string) error {
		// 按最多 500 个 ID 一批查出标签行。
		query := `SELECT photo_id, tag FROM photo_book_tags WHERE photo_id IN (` + inClausePlaceholders(len(batch)) + `) ORDER BY photo_id ASC, tag ASC`
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

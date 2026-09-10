package storage

import (
	"context"
	"strings"

	"curated-backend/internal/contracts"
)

type PhotoPageInput struct {
	Index     int
	EntryPath string
	FileName  string
	ImageExt  string
	Width     int
	Height    int
	SizeBytes int64
}

func (s *SQLiteStore) ReplacePhotoPages(ctx context.Context, photoID string, pages []PhotoPageInput) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM photo_pages WHERE photo_id = ?`, photoID); err != nil {
		return err
	}
	ts := nowUTC()
	for _, page := range pages {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO photo_pages (photo_id, page_index, entry_path, file_name, image_ext, width, height, size_bytes, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			photoID,
			page.Index,
			strings.TrimSpace(page.EntryPath),
			strings.TrimSpace(page.FileName),
			strings.TrimSpace(page.ImageExt),
			page.Width,
			page.Height,
			page.SizeBytes,
			ts,
		); err != nil {
			return err
		}
	}
	_, err = tx.ExecContext(ctx, `UPDATE photo_books SET page_count = ?, updated_at = ? WHERE id = ?`, len(pages), ts, photoID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *SQLiteStore) ListPhotoPages(ctx context.Context, photoID string) ([]contracts.PhotoPageDTO, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT photo_id, page_index, entry_path, file_name, image_ext, width, height
		   FROM photo_pages
		  WHERE photo_id = ?
		  ORDER BY page_index ASC`,
		photoID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]contracts.PhotoPageDTO, 0)
	for rows.Next() {
		var page contracts.PhotoPageDTO
		if err := rows.Scan(&page.PhotoID, &page.Index, &page.EntryPath, &page.FileName, &page.ImageExt, &page.Width, &page.Height); err != nil {
			return nil, err
		}
		out = append(out, page)
	}
	return out, rows.Err()
}

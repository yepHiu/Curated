package storage

import (
	"context"
	"strings"
)

import "curated-backend/internal/contracts"

type ComicPageInput struct {
	Index     int
	EntryPath string
	FileName  string
	ImageExt  string
	Width     int
	Height    int
	SizeBytes int64
}

func (s *SQLiteStore) ReplaceComicPages(ctx context.Context, comicID string, pages []ComicPageInput) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM comic_pages WHERE comic_id = ?`, comicID); err != nil {
		return err
	}
	ts := nowUTC()
	for _, page := range pages {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO comic_pages (comic_id, page_index, entry_path, file_name, image_ext, width, height, size_bytes, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			comicID,
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
	_, err = tx.ExecContext(ctx, `UPDATE comic_books SET page_count = ?, updated_at = ? WHERE id = ?`, len(pages), ts, comicID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *SQLiteStore) ListComicPages(ctx context.Context, comicID string) ([]contracts.ComicPageDTO, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT comic_id, page_index, entry_path, file_name, image_ext, width, height
		   FROM comic_pages
		  WHERE comic_id = ?
		  ORDER BY page_index ASC`,
		comicID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]contracts.ComicPageDTO, 0)
	for rows.Next() {
		var page contracts.ComicPageDTO
		if err := rows.Scan(&page.ComicID, &page.Index, &page.EntryPath, &page.FileName, &page.ImageExt, &page.Width, &page.Height); err != nil {
			return nil, err
		}
		out = append(out, page)
	}
	return out, rows.Err()
}

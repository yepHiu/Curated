package storage

import (
	"context"
	"strings"

	"curated-backend/internal/contracts"
)

type ComicCacheEntryInput struct {
	CacheKey  string
	ComicID   string
	Kind      string
	PageIndex int
	Path      string
	SizeBytes int64
}

func (s *SQLiteStore) SaveComicCacheEntry(ctx context.Context, in ComicCacheEntryInput) error {
	ts := nowUTC()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO comic_cache_entries (cache_key, comic_id, kind, page_index, path, size_bytes, created_at, last_accessed_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(cache_key) DO UPDATE SET
		   comic_id = excluded.comic_id,
		   kind = excluded.kind,
		   page_index = excluded.page_index,
		   path = excluded.path,
		   size_bytes = excluded.size_bytes,
		   last_accessed_at = excluded.last_accessed_at`,
		strings.TrimSpace(in.CacheKey),
		strings.TrimSpace(in.ComicID),
		strings.TrimSpace(in.Kind),
		in.PageIndex,
		strings.TrimSpace(in.Path),
		in.SizeBytes,
		ts,
		ts,
	)
	return err
}

func (s *SQLiteStore) ListComicCacheEntries(ctx context.Context, limit int) ([]contracts.ComicCacheEntryDTO, error) {
	if limit <= 0 || limit > 10000 {
		limit = 1000
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT cache_key, comic_id, kind, page_index, path, size_bytes, created_at, last_accessed_at
		   FROM comic_cache_entries
		  ORDER BY last_accessed_at ASC
		  LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]contracts.ComicCacheEntryDTO, 0)
	for rows.Next() {
		var row contracts.ComicCacheEntryDTO
		if err := rows.Scan(&row.CacheKey, &row.ComicID, &row.Kind, &row.PageIndex, &row.Path, &row.SizeBytes, &row.CreatedAt, &row.LastAccessedAt); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) TouchComicCacheEntry(ctx context.Context, cacheKey string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE comic_cache_entries SET last_accessed_at = ? WHERE cache_key = ?`,
		nowUTC(),
		strings.TrimSpace(cacheKey),
	)
	return err
}

func (s *SQLiteStore) DeleteComicCacheEntry(ctx context.Context, cacheKey string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM comic_cache_entries WHERE cache_key = ?`, strings.TrimSpace(cacheKey))
	return err
}

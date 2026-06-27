package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"curated-backend/internal/contracts"
)

var (
	ErrComicLibraryPathDuplicate   = errors.New("comic library path already exists")
	ErrComicLibraryPathNotFound    = errors.New("comic library path not found")
	ErrComicLibraryPathNotAbsolute = errors.New("comic library path must be absolute")
)

func (s *SQLiteStore) AddComicLibraryPath(ctx context.Context, path, title string) (contracts.ComicLibraryPathDTO, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return contracts.ComicLibraryPathDTO{}, fmt.Errorf("path is required")
	}
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) {
		return contracts.ComicLibraryPathDTO{}, ErrComicLibraryPathNotAbsolute
	}
	title = strings.TrimSpace(title)
	if title == "" {
		title = filepath.Base(path)
		if title == "" || title == "." {
			title = path
		}
	}

	id := "comic-library-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	ts := nowUTC()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO comic_library_paths (id, path, title, created_at, updated_at, first_library_scan_pending)
		 VALUES (?, ?, ?, ?, ?, 1)`,
		id, path, title, ts, ts,
	)
	if err != nil {
		if isSQLiteUniqueConstraint(err) {
			return contracts.ComicLibraryPathDTO{}, ErrComicLibraryPathDuplicate
		}
		return contracts.ComicLibraryPathDTO{}, err
	}
	return contracts.ComicLibraryPathDTO{
		ID:                      id,
		Path:                    path,
		Title:                   title,
		FirstLibraryScanPending: true,
	}, nil
}

func (s *SQLiteStore) ListComicLibraryPaths(ctx context.Context) ([]contracts.ComicLibraryPathDTO, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, path, title, first_library_scan_pending FROM comic_library_paths ORDER BY path ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]contracts.ComicLibraryPathDTO, 0)
	for rows.Next() {
		var row contracts.ComicLibraryPathDTO
		var pending int
		if err := rows.Scan(&row.ID, &row.Path, &row.Title, &pending); err != nil {
			return nil, err
		}
		row.FirstLibraryScanPending = pending != 0
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) GetComicLibraryPath(ctx context.Context, id string) (contracts.ComicLibraryPathDTO, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return contracts.ComicLibraryPathDTO{}, ErrComicLibraryPathNotFound
	}
	var row contracts.ComicLibraryPathDTO
	var pending int
	err := s.db.QueryRowContext(ctx,
		`SELECT id, path, title, first_library_scan_pending FROM comic_library_paths WHERE id = ?`,
		id,
	).Scan(&row.ID, &row.Path, &row.Title, &pending)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return contracts.ComicLibraryPathDTO{}, ErrComicLibraryPathNotFound
		}
		return contracts.ComicLibraryPathDTO{}, err
	}
	row.FirstLibraryScanPending = pending != 0
	return row, nil
}

func (s *SQLiteStore) UpdateComicLibraryPathTitle(ctx context.Context, id, title string) (contracts.ComicLibraryPathDTO, error) {
	row, err := s.GetComicLibraryPath(ctx, id)
	if err != nil {
		return contracts.ComicLibraryPathDTO{}, err
	}
	title = strings.TrimSpace(title)
	if title == "" {
		title = filepath.Base(row.Path)
		if title == "" || title == "." {
			title = row.Path
		}
	}
	row.Title = title
	_, err = s.db.ExecContext(ctx,
		`UPDATE comic_library_paths SET title = ?, updated_at = ? WHERE id = ?`,
		title, nowUTC(), row.ID,
	)
	return row, err
}

func (s *SQLiteStore) DeleteComicLibraryPath(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return ErrComicLibraryPathNotFound
	}
	res, err := s.db.ExecContext(ctx, `DELETE FROM comic_library_paths WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrComicLibraryPathNotFound
	}
	return nil
}

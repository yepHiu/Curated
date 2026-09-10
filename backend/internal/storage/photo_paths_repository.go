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
	ErrPhotoLibraryPathDuplicate   = errors.New("photo library path already exists")
	ErrPhotoLibraryPathNotFound    = errors.New("photo library path not found")
	ErrPhotoLibraryPathNotAbsolute = errors.New("photo library path must be absolute")
)

func (s *SQLiteStore) AddPhotoLibraryPath(ctx context.Context, path, title string) (contracts.PhotoLibraryPathDTO, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return contracts.PhotoLibraryPathDTO{}, fmt.Errorf("path is required")
	}
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) {
		return contracts.PhotoLibraryPathDTO{}, ErrPhotoLibraryPathNotAbsolute
	}
	title = strings.TrimSpace(title)
	if title == "" {
		title = filepath.Base(path)
		if title == "" || title == "." {
			title = path
		}
	}

	id := "photo-library-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	ts := nowUTC()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO photo_library_paths (id, path, title, created_at, updated_at, first_library_scan_pending)
		 VALUES (?, ?, ?, ?, ?, 1)`,
		id, path, title, ts, ts,
	)
	if err != nil {
		if isSQLiteUniqueConstraint(err) {
			return contracts.PhotoLibraryPathDTO{}, ErrPhotoLibraryPathDuplicate
		}
		return contracts.PhotoLibraryPathDTO{}, err
	}
	return contracts.PhotoLibraryPathDTO{
		ID:                      id,
		Path:                    path,
		Title:                   title,
		FirstLibraryScanPending: true,
	}, nil
}

func (s *SQLiteStore) ListPhotoLibraryPaths(ctx context.Context) ([]contracts.PhotoLibraryPathDTO, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, path, title, first_library_scan_pending FROM photo_library_paths ORDER BY path ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]contracts.PhotoLibraryPathDTO, 0)
	for rows.Next() {
		var row contracts.PhotoLibraryPathDTO
		var pending int
		if err := rows.Scan(&row.ID, &row.Path, &row.Title, &pending); err != nil {
			return nil, err
		}
		row.FirstLibraryScanPending = pending != 0
		out = append(out, row)
	}
	return out, rows.Err()
}

// ListPhotoLibraryPathStrings returns photo path strings only, in the same order as ListPhotoLibraryPaths.
func (s *SQLiteStore) ListPhotoLibraryPathStrings(ctx context.Context) ([]string, error) {
	dtos, err := s.ListPhotoLibraryPaths(ctx)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(dtos))
	for _, d := range dtos {
		if strings.TrimSpace(d.Path) != "" {
			paths = append(paths, d.Path)
		}
	}
	return paths, nil
}

func (s *SQLiteStore) GetPhotoLibraryPath(ctx context.Context, id string) (contracts.PhotoLibraryPathDTO, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return contracts.PhotoLibraryPathDTO{}, ErrPhotoLibraryPathNotFound
	}
	var row contracts.PhotoLibraryPathDTO
	var pending int
	err := s.db.QueryRowContext(ctx,
		`SELECT id, path, title, first_library_scan_pending FROM photo_library_paths WHERE id = ?`,
		id,
	).Scan(&row.ID, &row.Path, &row.Title, &pending)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return contracts.PhotoLibraryPathDTO{}, ErrPhotoLibraryPathNotFound
		}
		return contracts.PhotoLibraryPathDTO{}, err
	}
	row.FirstLibraryScanPending = pending != 0
	return row, nil
}

func (s *SQLiteStore) UpdatePhotoLibraryPathTitle(ctx context.Context, id, title string) (contracts.PhotoLibraryPathDTO, error) {
	row, err := s.GetPhotoLibraryPath(ctx, id)
	if err != nil {
		return contracts.PhotoLibraryPathDTO{}, err
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
		`UPDATE photo_library_paths SET title = ?, updated_at = ? WHERE id = ?`,
		title, nowUTC(), row.ID,
	)
	return row, err
}

func (s *SQLiteStore) DeletePhotoLibraryPath(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return ErrPhotoLibraryPathNotFound
	}
	res, err := s.db.ExecContext(ctx, `DELETE FROM photo_library_paths WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrPhotoLibraryPathNotFound
	}
	return nil
}

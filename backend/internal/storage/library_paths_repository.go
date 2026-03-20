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

	"jav-shadcn/backend/internal/contracts"
)

// ErrLibraryPathDuplicate is returned when inserting a path that already exists.
var ErrLibraryPathDuplicate = errors.New("library path already exists")

// ErrLibraryPathNotFound is returned when deleting a non-existent id.
var ErrLibraryPathNotFound = errors.New("library path not found")

// ErrLibraryPathNotAbsolute is returned when the path is not absolute (API adds only).
var ErrLibraryPathNotAbsolute = errors.New("library path must be absolute")

// ListLibraryPaths returns all configured library paths ordered by path.
func (s *SQLiteStore) ListLibraryPaths(ctx context.Context) ([]contracts.LibraryPathDTO, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, path, title FROM library_paths ORDER BY path ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]contracts.LibraryPathDTO, 0)
	for rows.Next() {
		var row contracts.LibraryPathDTO
		if err := rows.Scan(&row.ID, &row.Path, &row.Title); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// ListLibraryPathStrings returns path strings only (for scanning), same order as ListLibraryPaths.
func (s *SQLiteStore) ListLibraryPathStrings(ctx context.Context) ([]string, error) {
	dtos, err := s.ListLibraryPaths(ctx)
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

// AddLibraryPath inserts a new library path. Path must be non-empty after trim; title defaults to base name of path.
func (s *SQLiteStore) AddLibraryPath(ctx context.Context, path, title string) (contracts.LibraryPathDTO, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return contracts.LibraryPathDTO{}, fmt.Errorf("path is required")
	}
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) {
		return contracts.LibraryPathDTO{}, ErrLibraryPathNotAbsolute
	}
	if title == "" {
		title = filepath.Base(path)
		if title == "" || title == "." {
			title = path
		}
	}

	id := "library-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	ts := nowUTC()

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO library_paths (id, path, title, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		id, path, title, ts, ts,
	)
	if err != nil {
		if isSQLiteUniqueConstraint(err) {
			return contracts.LibraryPathDTO{}, ErrLibraryPathDuplicate
		}
		return contracts.LibraryPathDTO{}, err
	}

	return contracts.LibraryPathDTO{ID: id, Path: path, Title: title}, nil
}

// UpdateLibraryPathTitle updates the display title for an existing row. Empty title after trim defaults to basename of path.
func (s *SQLiteStore) UpdateLibraryPathTitle(ctx context.Context, id, title string) (contracts.LibraryPathDTO, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return contracts.LibraryPathDTO{}, fmt.Errorf("id is required")
	}

	var path string
	err := s.db.QueryRowContext(ctx, `SELECT path FROM library_paths WHERE id = ?`, id).Scan(&path)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return contracts.LibraryPathDTO{}, ErrLibraryPathNotFound
		}
		return contracts.LibraryPathDTO{}, err
	}

	title = strings.TrimSpace(title)
	if title == "" {
		title = filepath.Base(path)
		if title == "" || title == "." {
			title = path
		}
	}

	ts := nowUTC()
	res, err := s.db.ExecContext(ctx,
		`UPDATE library_paths SET title = ?, updated_at = ? WHERE id = ?`,
		title, ts, id,
	)
	if err != nil {
		return contracts.LibraryPathDTO{}, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return contracts.LibraryPathDTO{}, err
	}
	if n == 0 {
		return contracts.LibraryPathDTO{}, ErrLibraryPathNotFound
	}

	return contracts.LibraryPathDTO{ID: id, Path: path, Title: title}, nil
}

// DeleteLibraryPath removes a row by id. ErrLibraryPathNotFound if no row deleted.
func (s *SQLiteStore) DeleteLibraryPath(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("id is required")
	}
	res, err := s.db.ExecContext(ctx, `DELETE FROM library_paths WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrLibraryPathNotFound
	}
	return nil
}

// SeedLibraryPathsIfEmpty inserts default paths when the table has no rows.
func (s *SQLiteStore) SeedLibraryPathsIfEmpty(ctx context.Context, paths []string) error {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM library_paths`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	index := 0
	for _, raw := range paths {
		p := strings.TrimSpace(raw)
		if p == "" {
			continue
		}
		p = filepath.Clean(p)
		if !filepath.IsAbs(p) {
			abs, err := filepath.Abs(p)
			if err != nil {
				continue
			}
			p = filepath.Clean(abs)
		}
		index++
		id := fmt.Sprintf("library-%d", index)
		title := fmt.Sprintf("Library path %d", index)
		ts := nowUTC()
		if _, err := s.db.ExecContext(ctx,
			`INSERT INTO library_paths (id, path, title, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
			id, p, title, ts, ts,
		); err != nil {
			if isSQLiteUniqueConstraint(err) {
				continue
			}
			return err
		}
	}
	return nil
}

func isSQLiteUniqueConstraint(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") && strings.Contains(msg, "constraint")
}

// --- helpers for tests / scan integration ---

// GetLibraryPathCount returns number of rows (for tests).
func (s *SQLiteStore) GetLibraryPathCount(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM library_paths`).Scan(&n)
	return n, err
}

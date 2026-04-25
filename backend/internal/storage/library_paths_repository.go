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

// ErrLibraryPathDuplicate is returned when inserting a path that already exists.
var ErrLibraryPathDuplicate = errors.New("library path already exists")

// ErrLibraryPathNotFound is returned when deleting a non-existent id.
var ErrLibraryPathNotFound = errors.New("library path not found")

// ErrLibraryPathNotAbsolute is returned when the path is not absolute (API adds only).
var ErrLibraryPathNotAbsolute = errors.New("library path must be absolute")

// ListLibraryPaths returns all configured library paths ordered by path.
func (s *SQLiteStore) ListLibraryPaths(ctx context.Context) ([]contracts.LibraryPathDTO, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, path, title, first_library_scan_pending FROM library_paths ORDER BY path ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]contracts.LibraryPathDTO, 0)
	for rows.Next() {
		var row contracts.LibraryPathDTO
		var pendingInt int
		if err := rows.Scan(&row.ID, &row.Path, &row.Title, &pendingInt); err != nil {
			return nil, err
		}
		row.FirstLibraryScanPending = pendingInt != 0
		out = append(out, row)
	}
	return out, rows.Err()
}

// GetLibraryPath returns a single configured library path row by id.
func (s *SQLiteStore) GetLibraryPath(ctx context.Context, id string) (contracts.LibraryPathDTO, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return contracts.LibraryPathDTO{}, fmt.Errorf("id is required")
	}

	var row contracts.LibraryPathDTO
	var pendingInt int
	err := s.db.QueryRowContext(ctx,
		`SELECT id, path, title, first_library_scan_pending FROM library_paths WHERE id = ?`,
		id,
	).Scan(&row.ID, &row.Path, &row.Title, &pendingInt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return contracts.LibraryPathDTO{}, ErrLibraryPathNotFound
		}
		return contracts.LibraryPathDTO{}, err
	}
	row.FirstLibraryScanPending = pendingInt != 0
	return row, nil
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

type libraryPathStringsQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func listLibraryPathStringsWithQuery(ctx context.Context, q libraryPathStringsQueryer) ([]string, error) {
	rows, err := q.QueryContext(ctx, `SELECT path FROM library_paths ORDER BY path ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	paths := make([]string, 0)
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		if strings.TrimSpace(path) != "" {
			paths = append(paths, path)
		}
	}
	return paths, rows.Err()
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
		`INSERT INTO library_paths (id, path, title, created_at, updated_at, first_library_scan_pending) VALUES (?, ?, ?, ?, ?, 1)`,
		id, path, title, ts, ts,
	)
	if err != nil {
		if isSQLiteUniqueConstraint(err) {
			return contracts.LibraryPathDTO{}, ErrLibraryPathDuplicate
		}
		return contracts.LibraryPathDTO{}, err
	}

	return contracts.LibraryPathDTO{ID: id, Path: path, Title: title, FirstLibraryScanPending: true}, nil
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

	var pendingInt int
	if err := s.db.QueryRowContext(ctx,
		`SELECT first_library_scan_pending FROM library_paths WHERE id = ?`, id,
	).Scan(&pendingInt); err != nil {
		return contracts.LibraryPathDTO{}, err
	}
	return contracts.LibraryPathDTO{ID: id, Path: path, Title: title, FirstLibraryScanPending: pendingInt != 0}, nil
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

// DeleteLibraryPathAndPruneOrphanMovies removes the library path row, then unbinds movie rows whose
// media path lies under the removed root but not under any remaining configured library root
// (so nested/overlapping roots still cover the same files). Only the database is updated; the
// configured library directory and any media files on disk are never deleted here.
func (s *SQLiteStore) DeleteLibraryPathAndPruneOrphanMovies(ctx context.Context, id string) (pruned int, err error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return 0, fmt.Errorf("id is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	var pathStr string
	err = tx.QueryRowContext(ctx, `SELECT path FROM library_paths WHERE id = ?`, id).Scan(&pathStr)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrLibraryPathNotFound
	}
	if err != nil {
		return 0, err
	}
	removedRoot := filepath.Clean(strings.TrimSpace(pathStr))
	if removedRoot == "" || removedRoot == "." {
		return 0, fmt.Errorf("invalid stored library path")
	}

	res, err := tx.ExecContext(ctx, `DELETE FROM library_paths WHERE id = ?`, id)
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	if n == 0 {
		return 0, ErrLibraryPathNotFound
	}

	remaining, err := listLibraryPathStringsWithQuery(ctx, tx)
	if err != nil {
		return 0, err
	}

	rows, err := tx.QueryContext(ctx,
		`SELECT id, location FROM movies WHERE TRIM(COALESCE(location, '')) != ''`)
	if err != nil {
		return 0, err
	}
	var toDelete []string
	for rows.Next() {
		var movieID, location string
		if err := rows.Scan(&movieID, &location); err != nil {
			_ = rows.Close()
			return pruned, err
		}
		loc := filepath.Clean(strings.TrimSpace(location))
		if loc == "" {
			continue
		}
		if !pathHasLibraryRoot(loc, removedRoot) {
			continue
		}
		stillCovered := false
		for _, rr := range remaining {
			r := filepath.Clean(strings.TrimSpace(rr))
			if r == "" || r == "." {
				continue
			}
			if pathHasLibraryRoot(loc, r) {
				stillCovered = true
				break
			}
		}
		if stillCovered {
			continue
		}
		toDelete = append(toDelete, movieID)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return pruned, err
	}
	if err := rows.Close(); err != nil {
		return pruned, err
	}

	for _, movieID := range toDelete {
		if _, err := deleteMovieDatabaseTx(ctx, tx, movieID); err != nil {
			if errors.Is(err, ErrMovieNotFound) {
				continue
			}
			return pruned, err
		}
		pruned++
	}
	if err := tx.Commit(); err != nil {
		return pruned, err
	}
	return pruned, nil
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
			`INSERT INTO library_paths (id, path, title, created_at, updated_at, first_library_scan_pending) VALUES (?, ?, ?, ?, ?, 0)`,
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

// ClearFirstLibraryScanPendingAfterScan sets first_library_scan_pending=0 for any library root that
// intersects the given scan roots (scan path is the root itself or a descendant of the configured path).
func (s *SQLiteStore) ClearFirstLibraryScanPendingAfterScan(ctx context.Context, scanRoots []string) error {
	if len(scanRoots) == 0 {
		return nil
	}
	rows, err := s.ListLibraryPaths(ctx)
	if err != nil {
		return err
	}
	ts := nowUTC()
	for _, row := range rows {
		if !row.FirstLibraryScanPending {
			continue
		}
		lib := filepath.Clean(strings.TrimSpace(row.Path))
		if lib == "" {
			continue
		}
		matched := false
		for _, sp := range scanRoots {
			sp = filepath.Clean(strings.TrimSpace(sp))
			if sp == "" {
				continue
			}
			rel, err := filepath.Rel(lib, sp)
			if err != nil {
				continue
			}
			if rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		if _, err := s.db.ExecContext(ctx,
			`UPDATE library_paths SET first_library_scan_pending = 0, updated_at = ? WHERE id = ?`,
			ts, row.ID,
		); err != nil {
			return err
		}
	}
	return nil
}

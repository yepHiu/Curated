package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// A removed root hides its books without discarding user metadata or source files.
// Rebinding to another configured root preserves nested/overlapping libraries.
func reconcileDetachedBooks(ctx context.Context, tx *sql.Tx, kind string) error {
	if kind != "comic" && kind != "photo" {
		return fmt.Errorf("invalid book kind")
	}
	rows, err := tx.QueryContext(ctx, `SELECT id, path FROM `+kind+`_library_paths ORDER BY LENGTH(path) DESC, id`)
	if err != nil {
		return err
	}
	type root struct{ id, path string }
	var roots []root
	for rows.Next() {
		var r root
		if err := rows.Scan(&r.id, &r.path); err != nil {
			rows.Close()
			return err
		}
		roots = append(roots, r)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	rows, err = tx.QueryContext(ctx, `SELECT id, location FROM `+kind+`_books WHERE library_path_id IS NULL`)
	if err != nil {
		return err
	}
	var books []root
	for rows.Next() {
		var b root
		if err := rows.Scan(&b.id, &b.path); err != nil {
			rows.Close()
			return err
		}
		books = append(books, b)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	for _, b := range books {
		var pathID any
		detached := 1
		for _, r := range roots {
			if pathHasLibraryRoot(b.path, r.path) {
				pathID, detached = r.id, 0
				break
			}
		}
		if _, err := tx.ExecContext(ctx, `UPDATE `+kind+`_books SET library_path_id=?, detached=? WHERE id=?`, pathID, detached, b.id); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteStore) deleteBookLibraryPath(ctx context.Context, kind, id string) error {
	var notFound error
	switch kind {
	case "comic":
		notFound = ErrComicLibraryPathNotFound
	case "photo":
		notFound = ErrPhotoLibraryPathNotFound
	default:
		return fmt.Errorf("invalid book kind")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return notFound
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var found string
	if err := tx.QueryRowContext(ctx, `SELECT id FROM `+kind+`_library_paths WHERE id=?`, id).Scan(&found); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return notFound
		}
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM `+kind+`_library_paths WHERE id=?`, id); err != nil {
		return err
	}
	if err := reconcileDetachedBooks(ctx, tx, kind); err != nil {
		return err
	}
	return tx.Commit()
}

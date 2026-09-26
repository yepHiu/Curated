// Package storage provides SQLite-based persistence for the Curated media library.
package storage

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	_ "github.com/glebarez/go-sqlite"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// SQLiteStore is the primary persistence store backed by a SQLite database.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore opens the SQLite database at path, creates parent directories, and verifies connectivity.
func NewSQLiteStore(path string) (*SQLiteStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	// Single writer to avoid SQLite busy/locked errors.
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable sqlite foreign keys: %w", err)
	}
	var foreignKeysEnabled int
	if err := db.QueryRow(`PRAGMA foreign_keys`).Scan(&foreignKeysEnabled); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("verify sqlite foreign keys: %w", err)
	}
	if foreignKeysEnabled != 1 {
		_ = db.Close()
		return nil, fmt.Errorf("verify sqlite foreign keys: pragma remained %d", foreignKeysEnabled)
	}

	return &SQLiteStore{db: db}, nil
}

// Close shuts down the underlying database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// Migrate runs embedded SQL migration files in name order, tracking applied migrations in schema_migrations.
func (s *SQLiteStore) Migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			name TEXT PRIMARY KEY,
			applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return err
	}

	var existingMigrations int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&existingMigrations); err != nil {
		return err
	}

	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		var applied string
		err := s.db.QueryRowContext(ctx, `SELECT name FROM schema_migrations WHERE name = ?`, entry.Name()).Scan(&applied)
		switch {
		case err == nil:
			continue
		case err != nil && err != sql.ErrNoRows:
			return err
		}

		statement, err := migrationFiles.ReadFile(filepath.ToSlash(filepath.Join("migrations", entry.Name())))
		if err != nil {
			return err
		}

		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}

		if _, err := tx.ExecContext(ctx, string(statement)); err != nil {
			_ = tx.Rollback()
			return err
		}
		if entry.Name() == "0056_library_paths_initialization.sql" && existingMigrations == 0 {
			if _, err := tx.ExecContext(ctx, `UPDATE library_paths_initialization SET initialized = 0 WHERE id = 1`); err != nil {
				_ = tx.Rollback()
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (name) VALUES (?)`, entry.Name()); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	if err := s.backfillActorNormalizedNames(ctx); err != nil {
		return fmt.Errorf("backfill normalized actor names: %w", err)
	}
	if err := s.backfillActorFeedbackNormalizedTargets(ctx); err != nil {
		return fmt.Errorf("backfill normalized actor feedback targets: %w", err)
	}

	return s.verifyForeignKeyIntegrity(ctx)
}

func (s *SQLiteStore) verifyForeignKeyIntegrity(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `PRAGMA foreign_key_check`)
	if err != nil {
		return fmt.Errorf("check sqlite foreign keys: %w", err)
	}
	defer func() { _ = rows.Close() }()

	if rows.Next() {
		var tableName string
		var rowID any
		var parentTable string
		var foreignKeyID int
		if err := rows.Scan(&tableName, &rowID, &parentTable, &foreignKeyID); err != nil {
			return fmt.Errorf("read sqlite foreign key violation: %w", err)
		}
		return fmt.Errorf(
			"sqlite foreign key violation: table=%s rowid=%v parent=%s foreign_key_id=%d",
			tableName,
			rowID,
			parentTable,
			foreignKeyID,
		)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("check sqlite foreign keys: %w", err)
	}
	return nil
}

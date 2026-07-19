package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// IntegrityReport summarizes SQLite's built-in integrity checks.
type IntegrityReport struct {
	QuickCheck           string
	ForeignKeyViolations int
}

// AvailableMigrations returns the migration names embedded in this binary.
func AvailableMigrations() ([]string, error) {
	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("read embedded migrations: %w", err)
	}
	migrations := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		migrations = append(migrations, entry.Name())
	}
	sort.Strings(migrations)
	return migrations, nil
}

// CreateConsistentBackup writes a transactionally consistent SQLite snapshot
// with VACUUM INTO. The destination must not already exist.
func (s *SQLiteStore) CreateConsistentBackup(ctx context.Context, destinationPath string) error {
	destinationPath = strings.TrimSpace(destinationPath)
	if destinationPath == "" {
		return errors.New("backup destination path is required")
	}
	if strings.IndexByte(destinationPath, 0) >= 0 {
		return errors.New("backup destination path contains a NUL byte")
	}

	absDestination, err := filepath.Abs(destinationPath)
	if err != nil {
		return fmt.Errorf("resolve backup destination: %w", err)
	}
	if _, err := os.Stat(absDestination); err == nil {
		return fmt.Errorf("backup destination already exists: %s", absDestination)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect backup destination: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(absDestination), 0o700); err != nil {
		return fmt.Errorf("create backup destination directory: %w", err)
	}

	// SQLite does not accept a bound parameter in every VACUUM INTO build.
	// Single-quote escaping keeps the filename a SQL string literal.
	quotedPath := strings.ReplaceAll(filepath.ToSlash(absDestination), "'", "''")
	if _, err := s.db.ExecContext(ctx, `VACUUM INTO '`+quotedPath+`'`); err != nil {
		_ = os.Remove(absDestination)
		return fmt.Errorf("create sqlite backup: %w", err)
	}
	if err := os.Chmod(absDestination, 0o600); err != nil {
		_ = os.Remove(absDestination)
		return fmt.Errorf("restrict sqlite backup permissions: %w", err)
	}
	file, err := os.OpenFile(absDestination, os.O_RDWR, 0)
	if err != nil {
		_ = os.Remove(absDestination)
		return fmt.Errorf("open sqlite backup for sync: %w", err)
	}
	syncErr := file.Sync()
	closeErr := file.Close()
	if syncErr != nil {
		_ = os.Remove(absDestination)
		return fmt.Errorf("sync sqlite backup: %w", syncErr)
	}
	if closeErr != nil {
		_ = os.Remove(absDestination)
		return fmt.Errorf("close sqlite backup: %w", closeErr)
	}
	return nil
}

// CheckIntegrity runs SQLite quick_check and foreign_key_check without
// mutating application data.
func (s *SQLiteStore) CheckIntegrity(ctx context.Context) (IntegrityReport, error) {
	var report IntegrityReport
	if err := s.db.QueryRowContext(ctx, `PRAGMA quick_check`).Scan(&report.QuickCheck); err != nil {
		return report, fmt.Errorf("run sqlite quick_check: %w", err)
	}
	if !strings.EqualFold(strings.TrimSpace(report.QuickCheck), "ok") {
		return report, fmt.Errorf("sqlite quick_check failed: %s", report.QuickCheck)
	}

	rows, err := s.db.QueryContext(ctx, `PRAGMA foreign_key_check`)
	if err != nil {
		return report, fmt.Errorf("run sqlite foreign_key_check: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var tableName string
		var rowID any
		var parentTable string
		var foreignKeyID int
		if err := rows.Scan(&tableName, &rowID, &parentTable, &foreignKeyID); err != nil {
			return report, fmt.Errorf("read sqlite foreign key violation: %w", err)
		}
		report.ForeignKeyViolations++
	}
	if err := rows.Err(); err != nil {
		return report, fmt.Errorf("read sqlite foreign_key_check: %w", err)
	}
	if report.ForeignKeyViolations > 0 {
		return report, fmt.Errorf("sqlite foreign_key_check found %d violation(s)", report.ForeignKeyViolations)
	}
	return report, nil
}

// AppliedMigrations returns applied migration names in stable order.
func (s *SQLiteStore) AppliedMigrations(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT name FROM schema_migrations ORDER BY name`)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("list schema migrations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var migrations []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("read schema migration: %w", err)
		}
		migrations = append(migrations, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate schema migrations: %w", err)
	}
	return migrations, nil
}

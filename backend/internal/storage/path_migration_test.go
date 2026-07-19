package storage

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"curated-backend/internal/pathmigration"
)

func TestPlanAndApplyPathMigrationCoverWhitelistedColumns(t *testing.T) {
	ctx := context.Background()
	store, sourceRoot, targetRoot := newPathMigrationTestStore(t)
	defer func() { _ = store.Close() }()

	paths := seedAllPathMigrationColumns(t, store, sourceRoot, targetRoot)
	mapping := mustPathMapping(t, sourceRoot, targetRoot)
	plan, err := store.PlanPathMigration(ctx, PathMigrationOptions{Mapping: mapping})
	if err != nil {
		t.Fatalf("PlanPathMigration: %v", err)
	}
	if !plan.CanApply || plan.AffectedRows != len(pathMigrationColumns) {
		t.Fatalf("plan canApply=%v affectedRows=%d errors=%v", plan.CanApply, plan.AffectedRows, plan.Errors)
	}
	if len(plan.Columns) != len(pathMigrationColumns) {
		t.Fatalf("columns = %d, want %d", len(plan.Columns), len(pathMigrationColumns))
	}
	for _, summary := range plan.Columns {
		if summary.AffectedRows != 1 {
			t.Fatalf("%s.%s affectedRows=%d", summary.Table, summary.Column, summary.AffectedRows)
		}
		if len(summary.Samples) != 1 {
			t.Fatalf("%s.%s samples=%d", summary.Table, summary.Column, len(summary.Samples))
		}
		if summary.Table == "library_path_storage_bindings" && summary.Samples[0].TargetStatus != "reset" {
			t.Fatalf("binding targetStatus=%q", summary.Samples[0].TargetStatus)
		}
	}

	appliedAt := time.Date(2026, 7, 20, 12, 34, 56, 0, time.UTC)
	result, err := store.ApplyPathMigration(ctx, PathMigrationOptions{
		Mapping:    mapping,
		BackupPath: filepath.Join(t.TempDir(), "verified.curated-backup"),
		Now:        func() time.Time { return appliedAt },
	})
	if err != nil {
		t.Fatalf("ApplyPathMigration: %v", err)
	}
	if !result.Applied || result.AppliedRows != len(pathMigrationColumns) || result.AuditID == "" {
		t.Fatalf("result = %+v", result)
	}
	if result.AppliedAt != appliedAt.Format(time.RFC3339Nano) {
		t.Fatalf("appliedAt=%q", result.AppliedAt)
	}

	assertStoredPath(t, store, `SELECT path FROM library_paths WHERE id = 'library-1'`, paths["library_paths"])
	assertStoredPath(t, store, `SELECT location FROM movies WHERE id = 'movie-1'`, paths["movies"])
	assertStoredPath(t, store, `SELECT path FROM scan_items WHERE task_id = 'scan-1'`, paths["scan_items"])
	assertStoredPath(t, store, `SELECT local_path FROM media_assets WHERE id = 'asset-1'`, paths["media_assets"])
	assertStoredPath(t, store, `SELECT avatar_local_path FROM actors WHERE name = 'Actor 1'`, paths["actors"])
	assertStoredPath(t, store, `SELECT downloaded_file_path FROM app_update_status WHERE status_key = 'default'`, paths["app_update_status"])

	var bindingCount int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM library_path_storage_bindings WHERE library_path_id = 'library-1'`).Scan(&bindingCount); err != nil {
		t.Fatalf("count bindings: %v", err)
	}
	if bindingCount != 0 {
		t.Fatalf("binding count=%d, want 0", bindingCount)
	}

	var auditCount, affectedRows int
	var auditBackupPath, summaryJSON string
	if err := store.db.QueryRow(`
		SELECT COUNT(*), affected_rows, backup_path, summary_json
		FROM path_migration_audits
		WHERE id = ?
	`, result.AuditID).Scan(&auditCount, &affectedRows, &auditBackupPath, &summaryJSON); err != nil {
		t.Fatalf("read audit: %v", err)
	}
	if auditCount != 1 || affectedRows != len(pathMigrationColumns) || auditBackupPath != result.BackupPath {
		t.Fatalf("audit count=%d affected=%d backup=%q", auditCount, affectedRows, auditBackupPath)
	}
	if !strings.Contains(summaryJSON, `"canApply":true`) {
		t.Fatalf("audit summary=%s", summaryJSON)
	}
	assertStoredPath(t, store, `SELECT path FROM library_paths WHERE id = 'unrelated'`, paths["unrelated"])
}

func TestPlanPathMigrationDetectsDestinationAndCrossPlatformConflicts(t *testing.T) {
	t.Run("host paths", func(t *testing.T) {
		store, sourceRoot, targetRoot := newPathMigrationTestStore(t)
		defer func() { _ = store.Close() }()
		mustExecPathMigration(t, store, `INSERT INTO library_paths (id, path, title) VALUES (?, ?, 'Source')`, "source", sourceRoot)
		mustExecPathMigration(t, store, `INSERT INTO library_paths (id, path, title) VALUES (?, ?, 'Target')`, "target", targetRoot)

		plan, err := store.PlanPathMigration(context.Background(), PathMigrationOptions{Mapping: mustPathMapping(t, sourceRoot, targetRoot)})
		if err != nil {
			t.Fatalf("PlanPathMigration: %v", err)
		}
		if plan.CanApply || len(plan.Conflicts) != 1 || plan.Conflicts[0].Table != "library_paths" {
			t.Fatalf("plan canApply=%v conflicts=%+v", plan.CanApply, plan.Conflicts)
		}
	})

	t.Run("Windows to Unix existing target", func(t *testing.T) {
		store := newMigratedPathStore(t)
		defer func() { _ = store.Close() }()
		mustExecPathMigration(t, store, `INSERT INTO library_paths (id, path, title) VALUES ('source', ?, 'Source')`, `D:\Media\Studio`)
		mustExecPathMigration(t, store, `INSERT INTO library_paths (id, path, title) VALUES ('target', ?, 'Target')`, `/srv/curated/media/Studio`)

		plan, err := store.PlanPathMigration(context.Background(), PathMigrationOptions{
			Mapping:      mustPathMapping(t, `D:\Media`, `/srv/curated/media`),
			AllowMissing: true,
		})
		if err != nil {
			t.Fatalf("PlanPathMigration: %v", err)
		}
		if plan.CanApply || len(plan.Conflicts) != 1 || plan.Conflicts[0].FinalPath != `/srv/curated/media/Studio` {
			t.Fatalf("plan canApply=%v conflicts=%+v errors=%v", plan.CanApply, plan.Conflicts, plan.Errors)
		}
	})
}

func TestPlanPathMigrationIgnoresUnrelatedSemanticDuplicates(t *testing.T) {
	store := newMigratedPathStore(t)
	defer func() { _ = store.Close() }()
	mustExecPathMigration(t, store, `INSERT INTO library_paths (id, path, title) VALUES ('source', ?, 'Source')`, `D:\Media\Movie`)
	mustExecPathMigration(t, store, `INSERT INTO library_paths (id, path, title) VALUES ('other-1', ?, 'Other 1')`, `Z:\Unrelated`)
	mustExecPathMigration(t, store, `INSERT INTO library_paths (id, path, title) VALUES ('other-2', ?, 'Other 2')`, `z:/unrelated`)

	plan, err := store.PlanPathMigration(context.Background(), PathMigrationOptions{
		Mapping:      mustPathMapping(t, `D:\Media`, `E:\Library`),
		AllowMissing: true,
	})
	if err != nil {
		t.Fatalf("PlanPathMigration: %v", err)
	}
	if !plan.CanApply || len(plan.Conflicts) != 0 {
		t.Fatalf("plan canApply=%v conflicts=%+v errors=%v", plan.CanApply, plan.Conflicts, plan.Errors)
	}
}

func TestPlanPathMigrationMissingTargetsRequireExplicitOverride(t *testing.T) {
	store, sourceRoot, targetRoot := newPathMigrationTestStore(t)
	defer func() { _ = store.Close() }()
	missingTarget := filepath.Join(targetRoot, "missing")
	mustExecPathMigration(t, store, `INSERT INTO library_paths (id, path, title) VALUES ('source', ?, 'Source')`, sourceRoot)
	mapping := mustPathMapping(t, sourceRoot, missingTarget)

	blocked, err := store.PlanPathMigration(context.Background(), PathMigrationOptions{Mapping: mapping})
	if err != nil {
		t.Fatalf("PlanPathMigration(blocked): %v", err)
	}
	if blocked.CanApply || blocked.MissingTargets != 1 {
		t.Fatalf("blocked plan=%+v", blocked)
	}
	allowed, err := store.PlanPathMigration(context.Background(), PathMigrationOptions{Mapping: mapping, AllowMissing: true})
	if err != nil {
		t.Fatalf("PlanPathMigration(allowed): %v", err)
	}
	if !allowed.CanApply || allowed.MissingTargets != 1 || len(allowed.Warnings) == 0 {
		t.Fatalf("allowed plan=%+v", allowed)
	}
}

func TestPlanPathMigrationWrongTargetTypeAlwaysBlocks(t *testing.T) {
	store, sourceRoot, targetRoot := newPathMigrationTestStore(t)
	defer func() { _ = store.Close() }()
	targetFile := filepath.Join(targetRoot, "not-a-directory")
	if err := os.WriteFile(targetFile, []byte("test"), 0o600); err != nil {
		t.Fatalf("write target file: %v", err)
	}
	mustExecPathMigration(t, store, `INSERT INTO library_paths (id, path, title) VALUES ('source', ?, 'Source')`, sourceRoot)

	plan, err := store.PlanPathMigration(context.Background(), PathMigrationOptions{
		Mapping:      mustPathMapping(t, sourceRoot, targetFile),
		AllowMissing: true,
	})
	if err != nil {
		t.Fatalf("PlanPathMigration: %v", err)
	}
	if plan.CanApply || plan.TargetErrors != 1 || len(plan.Errors) == 0 {
		t.Fatalf("plan=%+v", plan)
	}
}

func TestPlanPathMigrationBindingResetDoesNotRequireTarget(t *testing.T) {
	store, sourceRoot, targetRoot := newPathMigrationTestStore(t)
	defer func() { _ = store.Close() }()
	mustExecPathMigration(t, store, `INSERT INTO library_paths (id, path, title) VALUES ('library', ?, 'Library')`, filepath.Join(t.TempDir(), "unrelated"))
	mustExecPathMigration(t, store, `INSERT INTO library_path_storage_bindings (library_path_id, root_path, updated_at) VALUES ('library', ?, '2026-07-20T00:00:00Z')`, sourceRoot)

	plan, err := store.PlanPathMigration(context.Background(), PathMigrationOptions{
		Mapping: mustPathMapping(t, sourceRoot, filepath.Join(targetRoot, "missing")),
	})
	if err != nil {
		t.Fatalf("PlanPathMigration: %v", err)
	}
	if !plan.CanApply || plan.AffectedRows != 1 || plan.MissingTargets != 0 {
		t.Fatalf("plan=%+v", plan)
	}
}

func TestApplyPathMigrationRollsBackUpdatesWhenAuditWriteFails(t *testing.T) {
	store, sourceRoot, targetRoot := newPathMigrationTestStore(t)
	defer func() { _ = store.Close() }()
	mustExecPathMigration(t, store, `INSERT INTO library_paths (id, path, title) VALUES ('library', ?, 'Library')`, sourceRoot)
	mustExecPathMigration(t, store, `
		CREATE TRIGGER fail_path_migration_audit
		BEFORE INSERT ON path_migration_audits
		BEGIN
			SELECT RAISE(ABORT, 'forced audit failure');
		END
	`)

	result, err := store.ApplyPathMigration(context.Background(), PathMigrationOptions{
		Mapping:    mustPathMapping(t, sourceRoot, targetRoot),
		BackupPath: filepath.Join(t.TempDir(), "verified.curated-backup"),
	})
	if err == nil || !strings.Contains(err.Error(), "write path migration audit") {
		t.Fatalf("ApplyPathMigration result=%+v err=%v", result, err)
	}
	assertStoredPath(t, store, `SELECT path FROM library_paths WHERE id = 'library'`, sourceRoot)
	var auditCount int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM path_migration_audits`).Scan(&auditCount); err != nil {
		t.Fatalf("count audits: %v", err)
	}
	if auditCount != 0 {
		t.Fatalf("audit count=%d", auditCount)
	}
}

func TestApplyPathMigrationBlockedPlanRollsBackAuditSchema(t *testing.T) {
	store, sourceRoot, targetRoot := newPathMigrationTestStore(t)
	defer func() { _ = store.Close() }()
	dropPathMigrationAuditSchema(t, store)
	mustExecPathMigration(t, store, `INSERT INTO library_paths (id, path, title) VALUES ('source', ?, 'Source')`, sourceRoot)
	mustExecPathMigration(t, store, `INSERT INTO library_paths (id, path, title) VALUES ('target', ?, 'Target')`, targetRoot)

	_, err := store.ApplyPathMigration(context.Background(), PathMigrationOptions{
		Mapping:    mustPathMapping(t, sourceRoot, targetRoot),
		BackupPath: filepath.Join(t.TempDir(), "verified.curated-backup"),
	})
	if !errors.Is(err, ErrPathMigrationBlocked) {
		t.Fatalf("ApplyPathMigration error=%v", err)
	}
	assertPathMigrationAuditSchemaAbsent(t, store)
	assertStoredPath(t, store, `SELECT path FROM library_paths WHERE id = 'source'`, sourceRoot)
}

func newPathMigrationTestStore(t *testing.T) (*SQLiteStore, string, string) {
	t.Helper()
	root := t.TempDir()
	sourceRoot := filepath.Join(root, "source")
	targetRoot := filepath.Join(root, "target")
	if err := os.MkdirAll(sourceRoot, 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	if err := os.MkdirAll(targetRoot, 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	return newMigratedPathStore(t), sourceRoot, targetRoot
}

func newMigratedPathStore(t *testing.T) *SQLiteStore {
	t.Helper()
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "curated.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	if err := store.Migrate(context.Background()); err != nil {
		_ = store.Close()
		t.Fatalf("Migrate: %v", err)
	}
	return store
}

func seedAllPathMigrationColumns(t *testing.T, store *SQLiteStore, sourceRoot, targetRoot string) map[string]string {
	t.Helper()
	files := map[string]string{
		"movies":            "movie.mkv",
		"scan_items":        "scan.mkv",
		"media_assets":      "cover.jpg",
		"actors":            "actor.jpg",
		"app_update_status": "update.exe",
	}
	for _, relative := range files {
		if err := os.WriteFile(filepath.Join(targetRoot, relative), []byte("test"), 0o600); err != nil {
			t.Fatalf("write target %s: %v", relative, err)
		}
	}

	unrelatedPath := filepath.Join(t.TempDir(), "unrelated")
	mustExecPathMigration(t, store, `INSERT INTO library_paths (id, path, title) VALUES ('library-1', ?, 'Library')`, sourceRoot)
	mustExecPathMigration(t, store, `INSERT INTO library_paths (id, path, title) VALUES ('unrelated', ?, 'Unrelated')`, unrelatedPath)
	mustExecPathMigration(t, store, `
		INSERT INTO movies (id, title, code, studio, summary, added_at, location, resolution, year)
		VALUES ('movie-1', 'Movie', 'CODE-1', '', '', '2026-07-20T00:00:00Z', ?, '', 2026)
	`, filepath.Join(sourceRoot, files["movies"]))
	mustExecPathMigration(t, store, `
		INSERT INTO scan_items (task_id, path, file_name, status)
		VALUES ('scan-1', ?, 'scan.mkv', 'completed')
	`, filepath.Join(sourceRoot, files["scan_items"]))
	mustExecPathMigration(t, store, `
		INSERT INTO media_assets (id, movie_id, type, local_path)
		VALUES ('asset-1', 'movie-1', 'cover', ?)
	`, filepath.Join(sourceRoot, files["media_assets"]))
	mustExecPathMigration(t, store, `INSERT INTO actors (name, avatar, avatar_local_path) VALUES ('Actor 1', '', ?)`, filepath.Join(sourceRoot, files["actors"]))
	mustExecPathMigration(t, store, `
		INSERT INTO library_path_storage_bindings (library_path_id, root_path, updated_at)
		VALUES ('library-1', ?, '2026-07-20T00:00:00Z')
	`, sourceRoot)
	mustExecPathMigration(t, store, `
		INSERT INTO app_update_status (status_key, installed_version, status, checked_at, downloaded_file_path)
		VALUES ('default', '1.0.0', 'up-to-date', '2026-07-20T00:00:00Z', ?)
	`, filepath.Join(sourceRoot, files["app_update_status"]))

	paths := map[string]string{
		"library_paths":     targetRoot,
		"unrelated":         unrelatedPath,
		"movies":            filepath.Join(targetRoot, files["movies"]),
		"scan_items":        filepath.Join(targetRoot, files["scan_items"]),
		"media_assets":      filepath.Join(targetRoot, files["media_assets"]),
		"actors":            filepath.Join(targetRoot, files["actors"]),
		"app_update_status": filepath.Join(targetRoot, files["app_update_status"]),
	}
	return paths
}

func mustPathMapping(t *testing.T, from, to string) pathmigration.Mapping {
	t.Helper()
	mapping, err := pathmigration.NewMapping(from, to)
	if err != nil {
		t.Fatalf("NewMapping(%q, %q): %v", from, to, err)
	}
	return mapping
}

func mustExecPathMigration(t *testing.T, store *SQLiteStore, query string, args ...any) {
	t.Helper()
	if _, err := store.db.Exec(query, args...); err != nil {
		t.Fatalf("exec %q: %v", strings.TrimSpace(query), err)
	}
}

func assertStoredPath(t *testing.T, store *SQLiteStore, query, want string) {
	t.Helper()
	var got string
	if err := store.db.QueryRow(query).Scan(&got); err != nil {
		t.Fatalf("query path: %v", err)
	}
	if got != want {
		t.Fatalf("path=%q, want %q", got, want)
	}
}

func dropPathMigrationAuditSchema(t *testing.T, store *SQLiteStore) {
	t.Helper()
	mustExecPathMigration(t, store, `DROP INDEX IF EXISTS idx_path_migration_audits_created_at`)
	mustExecPathMigration(t, store, `DROP TABLE IF EXISTS path_migration_audits`)
	mustExecPathMigration(t, store, `DELETE FROM schema_migrations WHERE name = '0028_path_migration_audits.sql'`)
}

func assertPathMigrationAuditSchemaAbsent(t *testing.T, store *SQLiteStore) {
	t.Helper()
	var tableCount, migrationCount int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'path_migration_audits'`).Scan(&tableCount); err != nil {
		t.Fatalf("inspect audit table: %v", err)
	}
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE name = '0028_path_migration_audits.sql'`).Scan(&migrationCount); err != nil && !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("inspect audit migration: %v", err)
	}
	if tableCount != 0 || migrationCount != 0 {
		t.Fatalf("audit schema table=%d migration=%d", tableCount, migrationCount)
	}
}

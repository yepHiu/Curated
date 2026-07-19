package storage

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"curated-backend/internal/pathmigration"
)

const pathMigrationSampleLimit = 5

var ErrPathMigrationBlocked = errors.New("path migration plan is blocked")

type PathMigrationOptions struct {
	Mapping      pathmigration.Mapping
	AllowMissing bool
	BackupPath   string
	Now          func() time.Time
}

type PathMigrationChangeSample struct {
	RowID        int64  `json:"rowId"`
	OldPath      string `json:"oldPath"`
	NewPath      string `json:"newPath"`
	TargetStatus string `json:"targetStatus"`
}

type PathMigrationColumnSummary struct {
	Table              string                      `json:"table"`
	Column             string                      `json:"column"`
	TargetKind         string                      `json:"targetKind"`
	AffectedRows       int                         `json:"affectedRows"`
	EmptyRows          int                         `json:"emptyRows"`
	OutsidePrefixRows  int                         `json:"outsidePrefixRows"`
	InvalidStoredPaths int                         `json:"invalidStoredPaths"`
	MissingTargets     int                         `json:"missingTargets"`
	UncheckedTargets   int                         `json:"uncheckedTargets"`
	TargetErrors       int                         `json:"targetErrors"`
	Conflicts          int                         `json:"conflicts"`
	Samples            []PathMigrationChangeSample `json:"samples"`
}

type PathMigrationConflict struct {
	Table     string  `json:"table"`
	Column    string  `json:"column"`
	Scope     string  `json:"scope,omitempty"`
	FinalPath string  `json:"finalPath"`
	RowIDs    []int64 `json:"rowIds"`
}

type PathMigrationPlan struct {
	FromRoot           string                       `json:"fromRoot"`
	ToRoot             string                       `json:"toRoot"`
	SourceStyle        pathmigration.Style          `json:"sourceStyle"`
	TargetStyle        pathmigration.Style          `json:"targetStyle"`
	AllowMissing       bool                         `json:"allowMissing"`
	CanApply           bool                         `json:"canApply"`
	AffectedRows       int                          `json:"affectedRows"`
	MissingTargets     int                          `json:"missingTargets"`
	UncheckedTargets   int                          `json:"uncheckedTargets"`
	TargetErrors       int                          `json:"targetErrors"`
	InvalidStoredPaths int                          `json:"invalidStoredPaths"`
	Columns            []PathMigrationColumnSummary `json:"columns"`
	Conflicts          []PathMigrationConflict      `json:"conflicts"`
	Errors             []string                     `json:"errors"`
	Warnings           []string                     `json:"warnings"`
}

type PathMigrationApplyResult struct {
	Applied     bool              `json:"applied"`
	AppliedRows int               `json:"appliedRows"`
	AuditID     string            `json:"auditId"`
	AppliedAt   string            `json:"appliedAt"`
	BackupPath  string            `json:"backupPath"`
	Plan        PathMigrationPlan `json:"plan"`
}

type pathColumnSpec struct {
	table               string
	column              string
	targetKind          string
	unique              bool
	conflictScopeColumn string
	resetBinding        bool
}

var pathMigrationColumns = []pathColumnSpec{
	{table: "library_paths", column: "path", targetKind: "directory", unique: true},
	{table: "movies", column: "location", targetKind: "file", unique: true},
	{table: "scan_items", column: "path", targetKind: "file", unique: true, conflictScopeColumn: "task_id"},
	{table: "media_assets", column: "local_path", targetKind: "file"},
	{table: "actors", column: "avatar_local_path", targetKind: "file"},
	{table: "library_path_storage_bindings", column: "root_path", targetKind: "directory", resetBinding: true},
	{table: "app_update_status", column: "downloaded_file_path", targetKind: "file"},
}

type pathMigrationChange struct {
	specIndex    int
	rowID        int64
	scope        string
	oldPath      string
	newPath      string
	targetStatus string
}

type finalUniquePath struct {
	rowID   int64
	path    string
	changed bool
}

type pathMigrationQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func (s *SQLiteStore) PlanPathMigration(ctx context.Context, options PathMigrationOptions) (PathMigrationPlan, error) {
	plan, _, err := planPathMigration(ctx, s.db, options)
	return plan, err
}

func (s *SQLiteStore) ApplyPathMigration(ctx context.Context, options PathMigrationOptions) (PathMigrationApplyResult, error) {
	if strings.TrimSpace(options.BackupPath) == "" {
		return PathMigrationApplyResult{}, errors.New("verified pre-migration backup path is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return PathMigrationApplyResult{}, fmt.Errorf("begin path migration transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := ensurePathMigrationAuditSchema(ctx, tx); err != nil {
		return PathMigrationApplyResult{}, err
	}

	plan, changes, err := planPathMigration(ctx, tx, options)
	if err != nil {
		return PathMigrationApplyResult{}, err
	}
	result := PathMigrationApplyResult{Plan: plan, BackupPath: strings.TrimSpace(options.BackupPath)}
	if !plan.CanApply {
		return result, ErrPathMigrationBlocked
	}
	for _, change := range changes {
		spec := pathMigrationColumns[change.specIndex]
		var execution sql.Result
		if spec.resetBinding {
			execution, err = tx.ExecContext(ctx,
				`DELETE FROM library_path_storage_bindings WHERE rowid = ? AND root_path = ?`,
				change.rowID, change.oldPath)
		} else {
			statement := fmt.Sprintf(`UPDATE "%s" SET "%s" = ? WHERE rowid = ? AND "%s" = ?`, spec.table, spec.column, spec.column)
			execution, err = tx.ExecContext(ctx, statement, change.newPath, change.rowID, change.oldPath)
		}
		if err != nil {
			return result, fmt.Errorf("apply %s.%s rowid=%d: %w", spec.table, spec.column, change.rowID, err)
		}
		rowsAffected, rowsErr := execution.RowsAffected()
		if rowsErr != nil {
			return result, fmt.Errorf("read %s.%s affected rows: %w", spec.table, spec.column, rowsErr)
		}
		if rowsAffected != 1 {
			return result, fmt.Errorf("apply %s.%s rowid=%d affected %d rows", spec.table, spec.column, change.rowID, rowsAffected)
		}
	}
	if err := verifyQuickCheckQuery(ctx, tx); err != nil {
		return result, err
	}
	if err := verifyForeignKeysQuery(ctx, tx); err != nil {
		return result, err
	}
	auditID, err := newPathMigrationAuditID()
	if err != nil {
		return result, err
	}
	now := time.Now
	if options.Now != nil {
		now = options.Now
	}
	appliedAt := now().UTC().Format(time.RFC3339Nano)
	summaryJSON, err := json.Marshal(plan)
	if err != nil {
		return result, fmt.Errorf("encode path migration audit: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO path_migration_audits (
			id, created_at, from_root, to_root, source_style, target_style,
			affected_rows, missing_targets, unchecked_targets, allow_missing,
			backup_path, summary_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, auditID, appliedAt, plan.FromRoot, plan.ToRoot, plan.SourceStyle, plan.TargetStyle,
		plan.AffectedRows, plan.MissingTargets, plan.UncheckedTargets, boolToInt(options.AllowMissing),
		strings.TrimSpace(options.BackupPath), string(summaryJSON)); err != nil {
		return result, fmt.Errorf("write path migration audit: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return result, fmt.Errorf("commit path migration: %w", err)
	}
	result.Applied = true
	result.AppliedRows = len(changes)
	result.AuditID = auditID
	result.AppliedAt = appliedAt
	return result, nil
}

func planPathMigration(ctx context.Context, queryer pathMigrationQueryer, options PathMigrationOptions) (PathMigrationPlan, []pathMigrationChange, error) {
	mapping := options.Mapping
	plan := PathMigrationPlan{
		FromRoot:     mapping.FromRoot,
		ToRoot:       mapping.ToRoot,
		SourceStyle:  mapping.SourceStyle,
		TargetStyle:  mapping.TargetStyle,
		AllowMissing: options.AllowMissing,
		Columns:      make([]PathMigrationColumnSummary, len(pathMigrationColumns)),
		Conflicts:    []PathMigrationConflict{},
		Errors:       []string{},
		Warnings:     []string{},
	}
	changes := make([]pathMigrationChange, 0)
	finalUnique := make([]map[string][]finalUniquePath, len(pathMigrationColumns))
	targetStatuses := make(map[string]string)

	for specIndex, spec := range pathMigrationColumns {
		summary := PathMigrationColumnSummary{
			Table:      spec.table,
			Column:     spec.column,
			TargetKind: spec.targetKind,
			Samples:    []PathMigrationChangeSample{},
		}
		finalUnique[specIndex] = make(map[string][]finalUniquePath)
		selectColumns := fmt.Sprintf(`rowid, "%s"`, spec.column)
		if spec.conflictScopeColumn != "" {
			selectColumns += fmt.Sprintf(`, "%s"`, spec.conflictScopeColumn)
		}
		rows, err := queryer.QueryContext(ctx, fmt.Sprintf(`SELECT %s FROM "%s" ORDER BY rowid`, selectColumns, spec.table))
		if err != nil {
			return PathMigrationPlan{}, nil, fmt.Errorf("query path column %s.%s: %w", spec.table, spec.column, err)
		}
		for rows.Next() {
			var rowID int64
			var storedPath string
			var scope string
			if spec.conflictScopeColumn == "" {
				err = rows.Scan(&rowID, &storedPath)
			} else {
				err = rows.Scan(&rowID, &storedPath, &scope)
			}
			if err != nil {
				_ = rows.Close()
				return PathMigrationPlan{}, nil, fmt.Errorf("scan path column %s.%s: %w", spec.table, spec.column, err)
			}
			if storedPath == "" {
				summary.EmptyRows++
				continue
			}
			mapped, matched, mapErr := mapping.Map(storedPath)
			if mapErr != nil {
				summary.InvalidStoredPaths++
				if spec.unique {
					addFinalUniquePath(finalUnique[specIndex], scope, storedPath, mapping.TargetStyle, rowID, false)
				}
				continue
			}
			finalPath := storedPath
			if matched {
				finalPath = mapped
				status := "reset"
				if !spec.resetBinding {
					statusKey := spec.targetKind + "\x00" + mapped
					var found bool
					status, found = targetStatuses[statusKey]
					if !found {
						status = inspectMigrationTarget(mapped, spec.targetKind, mapping.TargetStyle)
						targetStatuses[statusKey] = status
					}
				}
				change := pathMigrationChange{
					specIndex:    specIndex,
					rowID:        rowID,
					scope:        scope,
					oldPath:      storedPath,
					newPath:      mapped,
					targetStatus: status,
				}
				changes = append(changes, change)
				summary.AffectedRows++
				switch status {
				case "missing":
					summary.MissingTargets++
				case "unchecked":
					summary.UncheckedTargets++
				case "error", "wrong-type":
					summary.TargetErrors++
				}
				if len(summary.Samples) < pathMigrationSampleLimit {
					summary.Samples = append(summary.Samples, PathMigrationChangeSample{
						RowID: rowID, OldPath: storedPath, NewPath: mapped, TargetStatus: status,
					})
				}
			} else {
				summary.OutsidePrefixRows++
			}
			if spec.unique {
				style := mapping.SourceStyle
				if matched {
					style = mapping.TargetStyle
				}
				addFinalUniquePath(finalUnique[specIndex], scope, finalPath, style, rowID, matched)
			}
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return PathMigrationPlan{}, nil, fmt.Errorf("iterate path column %s.%s: %w", spec.table, spec.column, err)
		}
		if err := rows.Close(); err != nil {
			return PathMigrationPlan{}, nil, fmt.Errorf("close path column %s.%s: %w", spec.table, spec.column, err)
		}
		plan.Columns[specIndex] = summary
	}

	for specIndex, groups := range finalUnique {
		spec := pathMigrationColumns[specIndex]
		for key, entries := range groups {
			if len(entries) < 2 {
				continue
			}
			involvesChange := false
			for _, entry := range entries {
				involvesChange = involvesChange || entry.changed
			}
			if !involvesChange {
				continue
			}
			scope, _, _ := strings.Cut(key, "\x00")
			rowIDs := make([]int64, 0, len(entries))
			for _, entry := range entries {
				rowIDs = append(rowIDs, entry.rowID)
			}
			plan.Conflicts = append(plan.Conflicts, PathMigrationConflict{
				Table: spec.table, Column: spec.column, Scope: scope, FinalPath: entries[0].path, RowIDs: rowIDs,
			})
			plan.Columns[specIndex].Conflicts++
		}
	}
	sort.Slice(plan.Conflicts, func(i, j int) bool {
		left, right := plan.Conflicts[i], plan.Conflicts[j]
		if left.Table != right.Table {
			return left.Table < right.Table
		}
		if left.Column != right.Column {
			return left.Column < right.Column
		}
		if left.Scope != right.Scope {
			return left.Scope < right.Scope
		}
		return left.FinalPath < right.FinalPath
	})

	for _, summary := range plan.Columns {
		plan.AffectedRows += summary.AffectedRows
		plan.MissingTargets += summary.MissingTargets
		plan.UncheckedTargets += summary.UncheckedTargets
		plan.TargetErrors += summary.TargetErrors
		plan.InvalidStoredPaths += summary.InvalidStoredPaths
	}
	if plan.AffectedRows == 0 {
		plan.Errors = append(plan.Errors, "no whitelisted path rows match the source prefix")
	}
	if len(plan.Conflicts) > 0 {
		plan.Errors = append(plan.Errors, fmt.Sprintf("%d destination path conflicts must be resolved", len(plan.Conflicts)))
	}
	if plan.TargetErrors > 0 {
		plan.Errors = append(plan.Errors, fmt.Sprintf("%d destination paths could not be validated", plan.TargetErrors))
	}
	if !options.AllowMissing && plan.MissingTargets > 0 {
		plan.Errors = append(plan.Errors, fmt.Sprintf("%d destination paths are missing; rerun only after moving files or explicitly allow missing paths", plan.MissingTargets))
	}
	if !options.AllowMissing && plan.UncheckedTargets > 0 {
		plan.Errors = append(plan.Errors, fmt.Sprintf("%d destination paths cannot be checked on this operating system", plan.UncheckedTargets))
	}
	if options.AllowMissing && plan.MissingTargets > 0 {
		plan.Warnings = append(plan.Warnings, fmt.Sprintf("allow-missing accepted %d missing destination paths", plan.MissingTargets))
	}
	if options.AllowMissing && plan.UncheckedTargets > 0 {
		plan.Warnings = append(plan.Warnings, fmt.Sprintf("allow-missing accepted %d unchecked cross-platform destination paths", plan.UncheckedTargets))
	}
	if plan.InvalidStoredPaths > 0 {
		plan.Warnings = append(plan.Warnings, fmt.Sprintf("%d non-empty whitelisted values are not valid source-style absolute paths and were left unchanged", plan.InvalidStoredPaths))
	}
	plan.CanApply = len(plan.Errors) == 0
	return plan, changes, nil
}

func addFinalUniquePath(group map[string][]finalUniquePath, scope, path string, style pathmigration.Style, rowID int64, changed bool) {
	normalized, err := pathmigration.Normalize(path, style)
	if err != nil {
		return
	}
	key := scope + "\x00" + string(style) + "\x00" + normalized
	group[key] = append(group[key], finalUniquePath{rowID: rowID, path: path, changed: changed})
}

func inspectMigrationTarget(targetPath, targetKind string, style pathmigration.Style) string {
	if !pathmigration.HostSupports(style) {
		return "unchecked"
	}
	info, err := os.Stat(targetPath)
	if errors.Is(err, os.ErrNotExist) {
		return "missing"
	}
	if err != nil {
		return "error"
	}
	if targetKind == "directory" && !info.IsDir() || targetKind == "file" && !info.Mode().IsRegular() {
		return "wrong-type"
	}
	return "exists"
}

func verifyForeignKeysQuery(ctx context.Context, queryer pathMigrationQueryer) error {
	rows, err := queryer.QueryContext(ctx, `PRAGMA foreign_key_check`)
	if err != nil {
		return fmt.Errorf("check path migration foreign keys: %w", err)
	}
	defer func() { _ = rows.Close() }()
	if rows.Next() {
		var tableName string
		var rowID any
		var parentTable string
		var foreignKeyID int
		if err := rows.Scan(&tableName, &rowID, &parentTable, &foreignKeyID); err != nil {
			return fmt.Errorf("read path migration foreign key violation: %w", err)
		}
		return fmt.Errorf("path migration foreign key violation: table=%s rowid=%v parent=%s foreign_key_id=%d", tableName, rowID, parentTable, foreignKeyID)
	}
	return rows.Err()
}

func verifyQuickCheckQuery(ctx context.Context, queryer pathMigrationQueryer) error {
	rows, err := queryer.QueryContext(ctx, `PRAGMA quick_check`)
	if err != nil {
		return fmt.Errorf("check path migration database integrity: %w", err)
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return fmt.Errorf("read path migration database integrity: %w", err)
		}
		return errors.New("path migration quick_check returned no result")
	}
	var result string
	if err := rows.Scan(&result); err != nil {
		return fmt.Errorf("read path migration database integrity: %w", err)
	}
	if !strings.EqualFold(strings.TrimSpace(result), "ok") {
		return fmt.Errorf("path migration quick_check failed: %s", result)
	}
	return nil
}

func ensurePathMigrationAuditSchema(ctx context.Context, tx *sql.Tx) error {
	statement, err := migrationFiles.ReadFile("migrations/0028_path_migration_audits.sql")
	if err != nil {
		return fmt.Errorf("read path migration audit migration: %w", err)
	}
	if _, err := tx.ExecContext(ctx, string(statement)); err != nil {
		return fmt.Errorf("create path migration audit schema: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO schema_migrations (name) VALUES ('0028_path_migration_audits.sql')`); err != nil {
		return fmt.Errorf("record path migration audit migration: %w", err)
	}
	return nil
}

func newPathMigrationAuditID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate path migration audit id: %w", err)
	}
	return "path-migration-" + hex.EncodeToString(bytes), nil
}

package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrLibraryHealthRepairNotFound = errors.New("library health repair not found")

type LibraryHealthRepairRun struct {
	RepairID       string
	TaskID         string
	Action         string
	CategoriesJSON string
	Status         string
	TotalItems     int
	CompletedItems int
	SucceededItems int
	FailedItems    int
	CreatedAt      string
	StartedAt      string
	FinishedAt     string
	Items          []LibraryHealthRepairItem
}

type LibraryHealthRepairItem struct {
	Ordinal      int
	FindingID    string
	Category     string
	MovieID      string
	Label        string
	Status       string
	ChildTaskID  string
	ErrorCode    string
	ErrorMessage string
	StartedAt    string
	FinishedAt   string
}

func (s *SQLiteStore) CreateLibraryHealthRepairRun(ctx context.Context, run LibraryHealthRepairRun) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO library_health_repair_runs (
			repair_id, task_id, action, categories_json, status, total_items, created_at
		) VALUES (?, ?, ?, ?, 'pending', ?, ?)`,
		run.RepairID, run.TaskID, run.Action, run.CategoriesJSON, len(run.Items), run.CreatedAt); err != nil {
		return err
	}
	for _, item := range run.Items {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO library_health_repair_items (
				repair_id, ordinal, finding_id, category, movie_id, label, status
			) VALUES (?, ?, ?, ?, ?, ?, 'pending')`,
			run.RepairID, item.Ordinal, item.FindingID, item.Category, item.MovieID, item.Label); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *SQLiteStore) StartLibraryHealthRepairRun(ctx context.Context, repairID string, startedAt time.Time) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE library_health_repair_runs SET status = 'running', started_at = ?
		WHERE repair_id = ? AND status = 'pending'`, startedAt.UTC().Format(time.RFC3339), repairID)
	if err != nil {
		return err
	}
	return requireOneLibraryHealthRepairRow(result, repairID)
}

func (s *SQLiteStore) QueueLibraryHealthRepairItem(ctx context.Context, repairID string, ordinal int, childTaskID string, startedAt time.Time) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE library_health_repair_items
		SET status = 'queued', child_task_id = ?, started_at = ?
		WHERE repair_id = ? AND ordinal = ? AND status = 'pending'`,
		strings.TrimSpace(childTaskID), startedAt.UTC().Format(time.RFC3339), repairID, ordinal)
	if err != nil {
		return err
	}
	return requireOneLibraryHealthRepairRow(result, repairID)
}

func (s *SQLiteStore) CompleteLibraryHealthRepairItem(
	ctx context.Context, repairID string, ordinal int, status, errorCode, errorMessage string, finishedAt time.Time,
) error {
	if status != "succeeded" && status != "failed" && status != "cancelled" {
		return fmt.Errorf("invalid repair item status %q", status)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.ExecContext(ctx, `
		UPDATE library_health_repair_items
		SET status = ?, error_code = ?, error_message = ?, finished_at = ?
		WHERE repair_id = ? AND ordinal = ? AND status IN ('pending', 'queued')`,
		status, strings.TrimSpace(errorCode), strings.TrimSpace(errorMessage), finishedAt.UTC().Format(time.RFC3339), repairID, ordinal)
	if err != nil {
		return err
	}
	if err := requireOneLibraryHealthRepairRow(result, repairID); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE library_health_repair_runs SET
			completed_items = (SELECT COUNT(*) FROM library_health_repair_items WHERE repair_id = ? AND status IN ('succeeded', 'failed', 'cancelled')),
			succeeded_items = (SELECT COUNT(*) FROM library_health_repair_items WHERE repair_id = ? AND status = 'succeeded'),
			failed_items = (SELECT COUNT(*) FROM library_health_repair_items WHERE repair_id = ? AND status IN ('failed', 'cancelled'))
		WHERE repair_id = ?`, repairID, repairID, repairID, repairID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *SQLiteStore) FinishLibraryHealthRepairRun(ctx context.Context, repairID, status string, finishedAt time.Time) error {
	if status != "completed" && status != "partial_failed" && status != "failed" && status != "cancelled" {
		return fmt.Errorf("invalid repair run status %q", status)
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE library_health_repair_runs SET status = ?, finished_at = ?
		WHERE repair_id = ? AND status IN ('pending', 'running')`,
		status, finishedAt.UTC().Format(time.RFC3339), repairID)
	if err != nil {
		return err
	}
	return requireOneLibraryHealthRepairRow(result, repairID)
}

func (s *SQLiteStore) InterruptLibraryHealthRepairRuns(ctx context.Context, now time.Time) error {
	timestamp := now.UTC().Format(time.RFC3339)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `
		UPDATE library_health_repair_items
		SET status = 'cancelled', error_code = 'HEALTH_REPAIR_INTERRUPTED',
			error_message = 'backend restarted before repair completed', finished_at = ?
		WHERE status IN ('pending', 'queued')
		  AND repair_id IN (SELECT repair_id FROM library_health_repair_runs WHERE status IN ('pending', 'running'))`, timestamp); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE library_health_repair_runs SET
			status = 'cancelled',
			completed_items = total_items,
			failed_items = total_items - succeeded_items,
			finished_at = ?
		WHERE status IN ('pending', 'running')`, timestamp); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *SQLiteStore) GetLibraryHealthRepairRun(ctx context.Context, repairID string) (LibraryHealthRepairRun, error) {
	var run LibraryHealthRepairRun
	err := s.db.QueryRowContext(ctx, `
		SELECT repair_id, task_id, action, categories_json, status, total_items,
			completed_items, succeeded_items, failed_items, created_at, started_at, finished_at
		FROM library_health_repair_runs WHERE repair_id = ?`, repairID).Scan(
		&run.RepairID, &run.TaskID, &run.Action, &run.CategoriesJSON, &run.Status, &run.TotalItems,
		&run.CompletedItems, &run.SucceededItems, &run.FailedItems, &run.CreatedAt, &run.StartedAt, &run.FinishedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return run, ErrLibraryHealthRepairNotFound
	}
	if err != nil {
		return run, err
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT ordinal, finding_id, category, movie_id, label, status, child_task_id,
			error_code, error_message, started_at, finished_at
		FROM library_health_repair_items WHERE repair_id = ? ORDER BY ordinal`, repairID)
	if err != nil {
		return run, err
	}
	defer rows.Close()
	for rows.Next() {
		var item LibraryHealthRepairItem
		if err := rows.Scan(&item.Ordinal, &item.FindingID, &item.Category, &item.MovieID, &item.Label,
			&item.Status, &item.ChildTaskID, &item.ErrorCode, &item.ErrorMessage, &item.StartedAt, &item.FinishedAt); err != nil {
			return run, err
		}
		run.Items = append(run.Items, item)
	}
	return run, rows.Err()
}

func requireOneLibraryHealthRepairRow(result sql.Result, repairID string) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return fmt.Errorf("%w: %s", ErrLibraryHealthRepairNotFound, repairID)
	}
	return nil
}

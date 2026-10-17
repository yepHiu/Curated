package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"curated-backend/internal/contracts"
)

func (s *SQLiteStore) CreateBookOrganization(ctx context.Context, id string, req contracts.BookOrganizationRequest) (string, error) {
	table, err := bookMetadataTable(req.Kind)
	if err != nil {
		return "", err
	}
	if len(req.BookIDs) < 1 || len(req.BookIDs) > 600 || strings.TrimSpace(req.RequestID) == "" || len(req.RequestID) > 128 {
		return "", fmt.Errorf("invalid book selection")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	var old, kind, locale string
	err = tx.QueryRowContext(ctx, `SELECT id,kind,locale FROM ai_book_organization_jobs WHERE request_id=?`, req.RequestID).Scan(&old, &kind, &locale)
	if err == nil {
		if kind != req.Kind || locale != req.Locale {
			return "", ErrAIWriteConflict
		}
		rows, e := tx.QueryContext(ctx, `SELECT book_id FROM ai_book_organization_items WHERE job_id=? ORDER BY position`, old)
		if e != nil {
			return "", e
		}
		ids := []string{}
		for rows.Next() {
			var value string
			if e = rows.Scan(&value); e != nil {
				rows.Close()
				return "", e
			}
			ids = append(ids, value)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return "", e
		}
		if len(ids) != len(req.BookIDs) {
			return "", ErrAIWriteConflict
		}
		for i := range ids {
			if ids[i] != req.BookIDs[i] {
				return "", ErrAIWriteConflict
			}
		}
		return old, nil
	}
	if err != sql.ErrNoRows {
		return "", err
	}
	now := nowUTC()
	if _, err = tx.ExecContext(ctx, `INSERT INTO ai_book_organization_jobs VALUES(?,?,?,?,?,?,?,?)`, id, req.RequestID, req.Kind, req.Locale, "queued", "queued", now, now); err != nil {
		return "", err
	}
	seen := map[string]bool{}
	for i, bookID := range req.BookIDs {
		if bookID == "" || seen[bookID] {
			return "", fmt.Errorf("invalid book selection")
		}
		seen[bookID] = true
		var title string
		if err = tx.QueryRowContext(ctx, `SELECT COALESCE(NULLIF(TRIM(user_title),''),title) FROM `+table+` WHERE id=?`, bookID).Scan(&title); err != nil {
			return "", err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO ai_book_organization_items(job_id,book_id,position,title) VALUES(?,?,?,?)`, id, bookID, i, title); err != nil {
			return "", err
		}
	}
	return id, tx.Commit()
}

func (s *SQLiteStore) GetBookOrganization(ctx context.Context, id string) (contracts.BookOrganizationJob, error) {
	var j contracts.BookOrganizationJob
	err := s.db.QueryRowContext(ctx, `SELECT id,kind,locale,status,stage,created_at,updated_at,
 (SELECT COUNT(*) FROM ai_book_organization_items WHERE job_id=j.id),
 (SELECT COUNT(*) FROM ai_book_organization_items WHERE job_id=j.id AND status<>'pending'),
 (SELECT COUNT(*) FROM ai_book_organization_items WHERE job_id=j.id AND status='ready'),
 (SELECT COUNT(*) FROM ai_book_organization_items WHERE job_id=j.id AND status='applied'),
 (SELECT COUNT(*) FROM ai_book_organization_items WHERE job_id=j.id AND status IN ('failed','conflict'))
 FROM ai_book_organization_jobs j WHERE id=?`, id).Scan(&j.ID, &j.Kind, &j.Locale, &j.Status, &j.Stage, &j.CreatedAt, &j.UpdatedAt, &j.Total, &j.Processed, &j.PendingReview, &j.Applied, &j.Failed)
	return j, err
}

func (s *SQLiteStore) ActiveBookOrganizations(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM ai_book_organization_jobs WHERE status IN ('queued','running') ORDER BY created_at,id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *SQLiteStore) BookOrganizationItems(ctx context.Context, id string, limit, offset int) ([]contracts.BookOrganizationItem, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.db.QueryContext(ctx, `SELECT book_id,title,status,revision,snapshot,metadata,reason FROM ai_book_organization_items WHERE job_id=? ORDER BY position LIMIT ? OFFSET ?`, id, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []contracts.BookOrganizationItem{}
	for rows.Next() {
		var item contracts.BookOrganizationItem
		var snapshot, metadata string
		if err = rows.Scan(&item.BookID, &item.Title, &item.Status, &item.Revision, &snapshot, &metadata, &item.Reason); err != nil {
			return nil, err
		}
		if snapshot != "" {
			if err = json.Unmarshal([]byte(snapshot), &item.Before); err != nil {
				return nil, err
			}
		}
		if err = json.Unmarshal([]byte(metadata), &item.Metadata); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *SQLiteStore) BookOrganizationItem(ctx context.Context, id, bookID string) (contracts.BookOrganizationItem, error) {
	// Resolve a bounded position rather than loading the entire job.
	var pos int
	if err := s.db.QueryRowContext(ctx, `SELECT position FROM ai_book_organization_items WHERE job_id=? AND book_id=?`, id, bookID).Scan(&pos); err != nil {
		return contracts.BookOrganizationItem{}, err
	}
	items, err := s.BookOrganizationItems(ctx, id, 1, pos)
	if err != nil {
		return contracts.BookOrganizationItem{}, err
	}
	if len(items) != 1 {
		return contracts.BookOrganizationItem{}, sql.ErrNoRows
	}
	return items[0], nil
}

func refreshBookOrganizationTx(ctx context.Context, tx *sql.Tx, id string) error {
	_, err := tx.ExecContext(ctx, `UPDATE ai_book_organization_jobs SET
 status=CASE WHEN status='cancelled' THEN status
 WHEN EXISTS(SELECT 1 FROM ai_book_organization_items WHERE job_id=? AND status='pending') THEN status
 WHEN EXISTS(SELECT 1 FROM ai_book_organization_items WHERE job_id=? AND status='ready') THEN 'needs_confirmation'
 WHEN EXISTS(SELECT 1 FROM ai_book_organization_items WHERE job_id=? AND status IN ('failed','conflict')) THEN 'partial_failed' ELSE 'completed' END,
 stage=CASE WHEN status='cancelled' THEN 'stopped'
 WHEN EXISTS(SELECT 1 FROM ai_book_organization_items WHERE job_id=? AND status='pending') THEN stage
 WHEN EXISTS(SELECT 1 FROM ai_book_organization_items WHERE job_id=? AND status='ready') THEN 'reviewing' ELSE 'done' END,
 updated_at=? WHERE id=?`, id, id, id, id, id, nowUTC(), id)
	return err
}

func (s *SQLiteStore) SaveBookOrganizationProposal(ctx context.Context, id, bookID, status, snapshot string, metadata contracts.BookMetadata, reason string) error {
	raw, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `UPDATE ai_book_organization_items SET status=?,snapshot=?,metadata=?,reason=?,revision=revision+1 WHERE job_id=? AND book_id=? AND status='pending' AND EXISTS(SELECT 1 FROM ai_book_organization_jobs WHERE id=? AND status IN ('queued','running'))`, status, snapshot, string(raw), reason, id, bookID, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return ErrAIWriteConflict
	}
	if err = refreshBookOrganizationTx(ctx, tx, id); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *SQLiteStore) SetBookOrganizationRunning(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `UPDATE ai_book_organization_jobs SET status='running',stage='recognizing',updated_at=? WHERE id=? AND status IN ('queued','running')`, nowUTC(), id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return ErrAIWriteConflict
	}
	return nil
}

func (s *SQLiteStore) ControlBookOrganization(ctx context.Context, id, operation string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var status string
	if err = tx.QueryRowContext(ctx, `SELECT status FROM ai_book_organization_jobs WHERE id=?`, id).Scan(&status); err != nil {
		return err
	}
	switch operation {
	case "cancel":
		if status != "running" && status != "queued" {
			return ErrAIWriteConflict
		}
		_, err = tx.ExecContext(ctx, `UPDATE ai_book_organization_jobs SET status='cancelled',stage='stopped',updated_at=? WHERE id=?`, nowUTC(), id)
	case "retry":
		if status == "running" || status == "queued" || status == "completed" {
			return ErrAIWriteConflict
		}
		_, err = tx.ExecContext(ctx, `UPDATE ai_book_organization_items SET status='pending',reason='',snapshot='',metadata='{}',revision=revision+1 WHERE job_id=? AND status IN ('failed','conflict')`, id)
		if err == nil {
			_, err = tx.ExecContext(ctx, `UPDATE ai_book_organization_jobs SET status='queued',stage='queued',updated_at=? WHERE id=?`, nowUTC(), id)
		}
		if err == nil {
			err = refreshBookOrganizationTx(ctx, tx, id)
		}
	default:
		return fmt.Errorf("invalid operation")
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}

type bookReviewContextKey struct{}
type bookReviewContext struct {
	JobID, BookID string
	Revision      int
}

func WithBookOrganizationReview(ctx context.Context, jobID, bookID string, revision int) context.Context {
	return context.WithValue(ctx, bookReviewContextKey{}, bookReviewContext{jobID, bookID, revision})
}

func finishBookReviewTx(ctx context.Context, tx *sql.Tx, id, bookID string, revision int, status, reason string) error {
	res, err := tx.ExecContext(ctx, `UPDATE ai_book_organization_items SET status=?,reason=? WHERE job_id=? AND book_id=? AND revision=? AND status='ready'`, status, reason, id, bookID, revision)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return ErrAIWriteConflict
	}
	return refreshBookOrganizationTx(ctx, tx, id)
}

func (s *SQLiteStore) FinishBookReview(ctx context.Context, id, bookID string, revision int, status, reason string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = finishBookReviewTx(ctx, tx, id, bookID, revision, status, reason); err != nil {
		return err
	}
	return tx.Commit()
}

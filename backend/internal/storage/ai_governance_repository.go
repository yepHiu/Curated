package storage

import (
	"context"
	"fmt"
	"time"

	"curated-backend/internal/contracts"
)

func (s *SQLiteStore) InsertAIRun(ctx context.Context, r contracts.AIRunDTO) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO ai_runs VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, r.ID, r.StartedAt, r.Channel, r.Action, r.SessionID, r.Provider, r.Model, r.PromptVersion, r.Status, r.ErrorCode, r.DurationMs, r.FirstTextMs, r.ModelCalls, r.UsageCalls, r.ToolCalls, r.PromptTokens, r.CompletionTokens, r.TotalTokens)
	return err
}

func aiReportWhere(q contracts.AIReportQuery, audit bool) (string, []any) {
	timeColumn := "started_at"
	statusColumn := "status"
	if audit {
		timeColumn = "created_at"
		statusColumn = "result"
	}
	where := timeColumn + " >= ?"
	args := []any{time.Now().UTC().AddDate(0, 0, -q.Days).Format(time.RFC3339Nano)}
	if q.Channel != "" {
		where += " AND channel = ?"
		args = append(args, q.Channel)
	}
	if q.Status != "" {
		if audit && q.Status == "failed" {
			where += " AND result IN ('error','rejected')"
		} else {
			where += " AND " + statusColumn + " = ?"
			args = append(args, q.Status)
		}
	}
	return where, args
}

func (s *SQLiteStore) GetAIReport(ctx context.Context, q contracts.AIReportQuery) (contracts.AIReportDTO, error) {
	out := contracts.AIReportDTO{Items: []contracts.AIRunDTO{}, Limit: q.Limit, Offset: q.Offset}
	where, args := aiReportWhere(q, false)
	v := &out.Summary
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(SUM(status='failed'),0), COALESCE(SUM(status='partial'),0), COALESCE(SUM(status='cancelled'),0), COALESCE(SUM(model_calls),0), COALESCE(SUM(usage_calls),0), COALESCE(SUM(tool_calls),0), COALESCE(SUM(prompt_tokens),0), COALESCE(SUM(completion_tokens),0), COALESCE(SUM(total_tokens),0), AVG(duration_ms), AVG(first_text_ms) FROM ai_runs WHERE `+where, args...).Scan(&v.Runs, &v.Failed, &v.Partial, &v.Cancelled, &v.ModelCalls, &v.UsageCalls, &v.ToolCalls, &v.PromptTokens, &v.CompletionTokens, &v.TotalTokens, &v.AvgDurationMs, &v.AvgFirstTextMs)
	if err != nil {
		return out, err
	}
	out.Total = v.Runs
	rows, err := s.db.QueryContext(ctx, `SELECT * FROM ai_runs WHERE `+where+` ORDER BY started_at DESC,id DESC LIMIT ? OFFSET ?`, append(args, q.Limit, q.Offset)...)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var r contracts.AIRunDTO
		if err := rows.Scan(&r.ID, &r.StartedAt, &r.Channel, &r.Action, &r.SessionID, &r.Provider, &r.Model, &r.PromptVersion, &r.Status, &r.ErrorCode, &r.DurationMs, &r.FirstTextMs, &r.ModelCalls, &r.UsageCalls, &r.ToolCalls, &r.PromptTokens, &r.CompletionTokens, &r.TotalTokens); err != nil {
			return out, err
		}
		out.Items = append(out.Items, r)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) ListAIAudit(ctx context.Context, q contracts.AIReportQuery) (contracts.AIAuditPageDTO, error) {
	out := contracts.AIAuditPageDTO{Items: []contracts.AIAuditDTO{}, Limit: q.Limit, Offset: q.Offset}
	where, args := aiReportWhere(q, true)
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM ai_tool_invocations WHERE `+where, args...).Scan(&out.Total); err != nil {
		return out, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,created_at,channel,session_id,tool_name,permission,result,error_code,duration_ms FROM ai_tool_invocations WHERE `+where+` ORDER BY created_at DESC,id DESC LIMIT ? OFFSET ?`, append(args, q.Limit, q.Offset)...)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var r contracts.AIAuditDTO
		if err := rows.Scan(&r.ID, &r.CreatedAt, &r.Channel, &r.SessionID, &r.Tool, &r.Permission, &r.Result, &r.ErrorCode, &r.DurationMs); err != nil {
			return out, err
		}
		out.Items = append(out.Items, r)
	}
	return out, rows.Err()
}

// Only expired metadata and orphaned action receipts are removed. Chat messages
// and their applied-state receipts remain managed by explicit session deletion.
func (s *SQLiteStore) CleanupAIRecords(ctx context.Context, before time.Time) (contracts.AICleanupDTO, error) {
	out := contracts.AICleanupDTO{}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return out, err
	}
	defer tx.Rollback()
	cutoff := before.UTC().Format(time.RFC3339Nano)
	for _, op := range []struct {
		query string
		count *int64
	}{
		{`DELETE FROM ai_runs WHERE started_at < ?`, &out.Runs},
		{`DELETE FROM ai_tool_invocations WHERE created_at < ?`, &out.Audit},
		{`DELETE FROM ai_apply_receipts WHERE created_at < ? AND NOT EXISTS (SELECT 1 FROM ai_chat_sessions WHERE id=ai_apply_receipts.session_id)`, &out.Receipts},
	} {
		result, err := tx.ExecContext(ctx, op.query, cutoff)
		if err != nil {
			return contracts.AICleanupDTO{}, fmt.Errorf("clean AI records: %w", err)
		}
		*op.count, _ = result.RowsAffected()
	}
	return out, tx.Commit()
}

package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"curated-backend/internal/contracts"
)

func TestAIReportPaginationUnknownUsageAndRetention(t *testing.T) {
	s, err := NewSQLiteStore(filepath.Join(t.TempDir(), "ai.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	if err = s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	old := now.AddDate(0, 0, -40).Format(time.RFC3339Nano)
	for _, r := range []contracts.AIRunDTO{
		{ID: "a", StartedAt: now.Format(time.RFC3339Nano), Channel: "chat", Status: "completed", ModelCalls: 2, UsageCalls: 1, TotalTokens: 20, DurationMs: 200},
		{ID: "b", StartedAt: now.Format(time.RFC3339Nano), Channel: "test", Status: "failed", ModelCalls: 1, DurationMs: 100},
		{ID: "old", StartedAt: old, Channel: "action", Status: "completed"},
	} {
		if err = s.InsertAIRun(ctx, r); err != nil {
			t.Fatal(err)
		}
	}
	q := contracts.AIReportQuery{Days: 30, Limit: 1}
	report, err := s.GetAIReport(ctx, q)
	if err != nil {
		t.Fatal(err)
	}
	if report.Total != 2 || len(report.Items) != 1 || report.Items[0].ID != "b" || report.Summary.UsageCalls != 1 || report.Summary.AvgFirstTextMs != nil {
		t.Fatalf("report %+v", report)
	}
	q.Offset = 1
	second, err := s.GetAIReport(ctx, q)
	if err != nil || second.Items[0].ID != "a" {
		t.Fatalf("pagination %+v %v", second, err)
	}
	q.Offset = 0
	q.Status = "failed"
	filtered, err := s.GetAIReport(ctx, q)
	if err != nil || filtered.Total != 1 || filtered.Items[0].ID != "b" {
		t.Fatal("filter failed")
	}
	session, err := s.CreateAIChatSession(ctx, "retain")
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{session.ID, "act_expired"} {
		_, err = s.db.ExecContext(ctx, `INSERT INTO ai_apply_receipts VALUES (?,?,?,?,?,?)`, id, id, "tool", "hash", "{}", old)
		if err != nil {
			t.Fatal(err)
		}
	}
	cleaned, err := s.CleanupAIRecords(ctx, now.AddDate(0, 0, -30))
	if err != nil || cleaned.Runs != 1 || cleaned.Receipts != 1 {
		t.Fatalf("cleanup %+v %v", cleaned, err)
	}
	var retained int
	if err = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM ai_apply_receipts WHERE session_id=?`, session.ID).Scan(&retained); err != nil || retained != 1 {
		t.Fatal("chat receipt removed")
	}
}

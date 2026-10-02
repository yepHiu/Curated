package storage

import (
	"context"
	"testing"

	"curated-backend/internal/contracts"
)

func TestTopicIssuesPersistExcludeDefaultAndClearOnlyAfterSuccess(t *testing.T) {
	s := newTopicTestStore(t)
	ctx := context.Background()
	startTopicTestJob(t, s, "bad")
	if err := s.SetTagOrganizationItem(ctx, "bad", "a", "failed", "SOURCE_TOO_LONG"); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateTagOrganization(ctx, "bad", "partial_failed", "finished", ""); err != nil {
		t.Fatal(err)
	}
	requireTopicStats(t, s, contracts.TagOrganizationStatsDTO{Total: 2, NeedsAttention: 1, Unorganized: 1})
	issues, err := s.TagOrganizationIssues(ctx, 1, 0)
	if err != nil || len(issues) != 1 || issues[0].MovieID != "a" {
		t.Fatalf("issues=%+v err=%v", issues, err)
	}
	page, _ := s.TagOrganizationIssues(ctx, 1, 1)
	if len(page) != 0 {
		t.Fatal("pagination repeated issue")
	}
	id, err := s.CreateRemainingTagOrganization(ctx, "remaining", "remaining", "manual", "en", "unorganized")
	if err != nil {
		t.Fatal(err)
	}
	items, _ := s.TagOrganizationItems(ctx, id, false, 100, 0)
	if len(items) != 1 || items[0].MovieID != "b" {
		t.Fatalf("default scope=%+v", items)
	}
	if err = s.UpdateTagOrganization(ctx, id, "cancelled", "stopped", ""); err != nil {
		t.Fatal(err)
	}
	id, err = s.CreateRemainingTagOrganization(ctx, "retry", "retry", "manual", "en", "issues")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.UpdateTagOrganization(ctx, id, "running", "applying", ""); err != nil {
		t.Fatal(err)
	}
	// Choosing retry is not proof of completion; queue remains until atomic success.
	requireTopicStats(t, s, contracts.TagOrganizationStatsDTO{Total: 2, NeedsAttention: 1, Unorganized: 1})
	input, _ := s.TopicMovieInput(ctx, "a")
	if err = s.CompleteUnresolvedTopic(ctx, id, input); err != nil {
		t.Fatal(err)
	}
	requireTopicStats(t, s, contracts.TagOrganizationStatsDTO{Total: 2, Organized: 1, Unresolved: 1, Unorganized: 1})
	issues, _ = s.TagOrganizationIssues(ctx, 25, 0)
	if len(issues) != 0 {
		t.Fatal("success did not clear issue")
	}
}

func TestTopicIssueCheckpointIsAtomicAndCancelledWritesCannotCreateIssues(t *testing.T) {
	s := newTopicTestStore(t)
	ctx := context.Background()
	startTopicTestJob(t, s, "bad")
	if _, err := s.db.Exec(`CREATE TRIGGER reject_issue BEFORE INSERT ON movie_topic_issues BEGIN SELECT RAISE(ABORT,'fixture'); END;`); err != nil {
		t.Fatal(err)
	}
	if err := s.SetTagOrganizationItem(ctx, "bad", "a", "failed", "SOURCE_TOO_LONG"); err == nil {
		t.Fatal("issue failure accepted")
	}
	job, _ := s.GetTagOrganization(ctx, "bad")
	if job.Processed != 0 {
		t.Fatal("partially committed failure")
	}
	if err := s.UpdateTagOrganization(ctx, "bad", "cancelled", "stopped", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.SetTagOrganizationItem(ctx, "bad", "a", "failed", "SOURCE_TOO_LONG"); err != nil {
		t.Fatal(err)
	}
	issues, _ := s.TagOrganizationIssues(ctx, 25, 0)
	if len(issues) != 0 {
		t.Fatal("cancelled job created issue")
	}
}

func TestTopicIssueMigrationUsesLatestResultAndConvertsLegacyOversized(t *testing.T) {
	s := newTopicTestStore(t)
	_, err := s.db.Exec(`
 INSERT INTO ai_tag_organization_jobs(id,request_id,status,trigger_reason,created_at,updated_at) VALUES
 ('old','old','partial_failed','manual','2026-10-01','2026-10-01'),('new','new','completed','manual','2026-10-02','2026-10-02');
 INSERT INTO ai_tag_organization_items(job_id,movie_id,status,reason) VALUES
 ('old','a','failed','BAD'),('new','a','succeeded',''),('new','b','unresolved','SOURCE_TOO_LONG');
 DROP TABLE movie_topic_issues;`)
	if err != nil {
		t.Fatal(err)
	}
	migration, _ := migrationFiles.ReadFile("migrations/0063_movie_topic_issues.sql")
	if _, err = s.db.Exec(string(migration)); err != nil {
		t.Fatal(err)
	}
	issues, err := s.TagOrganizationIssues(context.Background(), 25, 0)
	if err != nil || len(issues) != 1 || issues[0].MovieID != "b" {
		t.Fatalf("issues=%+v err=%v", issues, err)
	}
	job, _ := s.GetTagOrganization(context.Background(), "new")
	if job.Status != "partial_failed" || job.Failed != 1 || job.Unresolved != 0 {
		t.Fatalf("legacy=%+v", job)
	}
}

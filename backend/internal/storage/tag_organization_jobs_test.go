package storage

import (
	"context"
	"testing"
)

func TestCreateAllTagOrganization10000Snapshot(t *testing.T) {
	s := newTopicTestStore(t)
	ctx := context.Background()
	// One transaction seeds a large, isolated library without filesystem fixtures.
	_, err := s.db.Exec(`WITH RECURSIVE seq(n) AS (SELECT 1 UNION ALL SELECT n+1 FROM seq WHERE n<10000)
	INSERT INTO movies(id,title,code,studio,summary,added_at,location,resolution,year)
	SELECT 'large-'||n,'Title '||n,'CODE-'||n,'','Theme','2026-10-02','path-'||n,'',2026 FROM seq`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(`UPDATE movies SET trashed_at='2026-10-02' WHERE id IN ('a','b')`); err != nil {
		t.Fatal(err)
	}
	id, err := s.CreateAllTagOrganization(ctx, "large", "request", "manual", "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	job, err := s.GetTagOrganization(ctx, id)
	if err != nil || job.Total != 10000 || job.Status != "queued" {
		t.Fatalf("job=%+v err=%v", job, err)
	}
	// Restoring two movies changes the live library, but never the fixed job scope.
	if _, err = s.db.Exec(`UPDATE movies SET trashed_at='' WHERE id IN ('a','b')`); err != nil {
		t.Fatal(err)
	}
	repeated, err := s.CreateAllTagOrganization(ctx, "another-id", "request", "manual", "en")
	if err != nil || repeated != id {
		t.Fatalf("idempotency id=%s err=%v", repeated, err)
	}
	job, err = s.GetTagOrganization(ctx, id)
	if err != nil || job.Total != 10000 || job.Locale != "zh-CN" {
		t.Fatalf("snapshot changed: %+v err=%v", job, err)
	}
	if _, err = s.CreateAllTagOrganization(ctx, "concurrent", "concurrent", "manual", "zh-CN"); err == nil {
		t.Fatal("second active task accepted")
	}
}

func TestCreateAllTagOrganizationEmptyRollback(t *testing.T) {
	s := newTopicTestStore(t)
	ctx := context.Background()
	if _, err := s.db.Exec(`UPDATE movies SET trashed_at='2026-10-02'`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateAllTagOrganization(ctx, "empty", "empty", "manual", "zh-CN"); err == nil {
		t.Fatal("empty library accepted")
	}
	jobs, err := s.ListTagOrganizations(ctx, false)
	if err != nil || len(jobs) != 0 {
		t.Fatalf("empty job leaked: %+v err=%v", jobs, err)
	}
}

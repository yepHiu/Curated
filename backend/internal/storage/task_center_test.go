package storage

import (
	"context"
	"fmt"
	"testing"

	"curated-backend/internal/contracts"
)

func TestTaskCenterPersistentPaginationAndSummary(t *testing.T) {
	s := newTopicTestStore(t)
	ctx := context.Background()
	// More than the legacy fifty-record history, with equal timestamps.
	for i := 0; i < 55; i++ {
		id := fmt.Sprintf("job-%02d", i)
		if _, err := s.db.Exec(`INSERT INTO ai_tag_organization_jobs(id,request_id,status,trigger_reason,created_at,updated_at) VALUES(?,?,'completed','manual','2026-10-17T00:00:00Z','2026-10-17T00:00:00Z')`, id, id); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.SaveSubtitleRecord(ctx, SubtitleRecord{RequestID: "sub", Job: contracts.SubtitleJob{ID: "job-00", MovieID: "a", FileID: "a:primary", State: "running", Stage: "translating", CreatedAt: "2026-10-17T00:00:00Z", UpdatedAt: "2026-10-17T00:00:00Z", Completed: 7, Total: 10, Error: "private/path"}}); err != nil {
		t.Fatal(err)
	}
	summary, err := s.TaskCenterSummary(ctx)
	if err != nil || summary.Total != 56 || summary.Running != 1 || summary.History != 55 {
		t.Fatalf("%+v %v", summary, err)
	}
	seen := map[string]bool{}
	cursor := ""
	for {
		keys, next, err := s.TaskCenterKeys(ctx, TaskCenterFilter{Cursor: cursor, Limit: 9})
		if err != nil {
			t.Fatal(err)
		}
		for _, key := range keys {
			if seen[key] {
				t.Fatalf("duplicate %s", key)
			}
			seen[key] = true
		}
		if next == "" {
			break
		}
		cursor = next
	}
	if len(seen) != 56 {
		t.Fatalf("lost records: %d", len(seen))
	}
	v, err := s.TaskCenterTask(ctx, "subtitle:job-00")
	if err != nil || v.Unit != "cues" || v.Subtitle.Error != "SUBTITLE_TASK_FAILED" {
		t.Fatalf("%+v %v", v, err)
	}
	if err = s.DeleteTagOrganization(ctx, "job-00"); err != nil {
		t.Fatal(err)
	}
	summary, err = s.TaskCenterSummary(ctx)
	if err != nil || summary.Total != 55 {
		t.Fatalf("deleted row resurfaced %+v %v", summary, err)
	}
	keys, _, err := s.TaskCenterKeys(ctx, TaskCenterFilter{State: "active"})
	if err != nil || len(keys) != 1 || keys[0] != "subtitle:job-00" {
		t.Fatalf("%v %v", keys, err)
	}
	if _, _, err = s.TaskCenterKeys(ctx, TaskCenterFilter{Cursor: "broken"}); err != ErrTaskCenterFilter {
		t.Fatalf("bad cursor: %v", err)
	}
}

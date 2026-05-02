package tasks

import (
	"testing"
	"time"

	"curated-backend/internal/contracts"
)

func TestProgressWithMetadata_MergesPatch(t *testing.T) {
	t.Parallel()

	m := NewManager()
	task := m.Create("scan.library", map[string]any{"paths": []string{"/tmp"}})
	m.Start(task.TaskID, "running")

	_ = m.ProgressWithMetadata(task.TaskID, 50, "half", map[string]any{
		"scanTotal":     10,
		"scanProcessed": 5,
		"scanImported":  2,
	})

	got, ok := m.Get(task.TaskID)
	if !ok {
		t.Fatal("task missing")
	}
	if got.Progress != 50 || got.Message != "half" {
		t.Fatalf("progress/message: %+v", got)
	}
	if got.Metadata["scanTotal"] != 10 || got.Metadata["scanProcessed"] != 5 {
		t.Fatalf("metadata: %+v", got.Metadata)
	}
	if got.Metadata["paths"] == nil {
		t.Fatal("expected original metadata paths preserved")
	}
}

func TestPartialFail_PreservesProgressAndMergesMetadata(t *testing.T) {
	t.Parallel()

	m := NewManager()
	task := m.Create("import.movies", map[string]any{"totalFiles": 2})
	m.Start(task.TaskID, "copying")
	m.ProgressWithMetadata(task.TaskID, 50, "copied one", map[string]any{"completedFiles": 1})

	m.PartialFail(task.TaskID, "IMPORT_PARTIAL_FAILED", "1 file copied, 1 failed", map[string]any{
		"failedFiles": 1,
		"errorItems": []map[string]any{
			{"sourceName": "bad.mp4", "errorCode": "IMPORT_COPY_FAILED"},
		},
	})

	got, ok := m.Get(task.TaskID)
	if !ok {
		t.Fatal("task missing")
	}
	if got.Status != contracts.TaskPartialFailed {
		t.Fatalf("Status = %q, want %q", got.Status, contracts.TaskPartialFailed)
	}
	if got.ErrorCode != "IMPORT_PARTIAL_FAILED" || got.ErrorMessage != "1 file copied, 1 failed" {
		t.Fatalf("error fields: %+v", got)
	}
	if got.Progress != 50 {
		t.Fatalf("Progress = %d, want preserved 50", got.Progress)
	}
	if got.FinishedAt == "" {
		t.Fatal("expected FinishedAt set")
	}
	if got.Metadata["totalFiles"] != 2 || got.Metadata["completedFiles"] != 1 || got.Metadata["failedFiles"] != 1 {
		t.Fatalf("metadata = %+v", got.Metadata)
	}
	if _, ok := got.Metadata["errorItems"].([]map[string]any); !ok {
		t.Fatalf("errorItems type = %T", got.Metadata["errorItems"])
	}
}

func TestListRecentFinished_LimitAndDescendingOrder(t *testing.T) {
	t.Parallel()
	m := NewManager()
	prev := ""
	for i := 0; i < 4; i++ {
		x := m.Create("scan.library", nil)
		m.Start(x.TaskID, "s")
		m.Complete(x.TaskID, "done")
		time.Sleep(time.Millisecond)
		got := m.ListRecentFinished(1)
		if len(got) != 1 {
			t.Fatalf("iteration %d: limit 1 got %d", i, len(got))
		}
		if prev != "" && got[0].FinishedAt < prev {
			t.Fatalf("not descending: %q after %q", got[0].FinishedAt, prev)
		}
		prev = got[0].FinishedAt
	}
	if len(m.ListRecentFinished(2)) != 2 {
		t.Fatal("expected 2 recent")
	}
}

func TestListRecentFinished_SkipsRunning(t *testing.T) {
	t.Parallel()
	m := NewManager()
	r := m.Create("scan.library", nil)
	m.Start(r.TaskID, "run")
	c := m.Create("scrape.movie", nil)
	m.Complete(c.TaskID, "x")
	got := m.ListRecentFinished(10)
	if len(got) != 1 || got[0].TaskID != c.TaskID {
		t.Fatalf("got %+v", got)
	}
	if got[0].Status != contracts.TaskCompleted {
		t.Fatalf("status %q", got[0].Status)
	}
}

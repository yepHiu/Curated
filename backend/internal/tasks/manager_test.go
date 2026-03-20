package tasks

import "testing"

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

package subtitles

import (
	"context"
	"testing"
	"time"

	"curated-backend/internal/storage"
)

func TestLegacyRestartMovesSupersededFailureOutOfAttention(t *testing.T) {
	for _, scenario := range []string{"replacement", "blocked replacement", "ordinary creation", "different audio", "changed source", "ambiguous failure"} {
		t.Run(scenario, func(t *testing.T) {
			opt, old := fixture(t)
			ctx := context.Background()
			old.Job.State, old.Job.Error = "failed", "original failure"
			if scenario == "blocked replacement" {
				old.Job.State = "blocked"
			}
			old.Job.UpdatedAt = time.Now().Add(-time.Hour).UTC().Format(time.RFC3339Nano)
			if err := opt.Store.SaveSubtitleRecord(ctx, old); err != nil {
				t.Fatal(err)
			}
			next := old
			next.Job.ID, next.RequestID = newID(), "restart-"+newID()
			next.Job.CreatedAt = time.Now().UTC().Format(time.RFC3339Nano)
			next.Job.State, next.Job.Error = "queued", ""
			if scenario == "ordinary creation" {
				next.RequestID = "ordinary-request"
			}
			if scenario == "different audio" {
				next.AudioTrack++
			}
			if scenario == "changed source" {
				next.Revision += "changed"
			}
			if scenario == "ambiguous failure" {
				extra := old
				extra.Job.ID, extra.RequestID = newID(), "extra-request"
				if err := opt.Store.SaveSubtitleRecord(ctx, extra); err != nil {
					t.Fatal(err)
				}
			}
			if err := opt.Store.SaveSubtitleRecord(ctx, next); err != nil {
				t.Fatal(err)
			}
			s := &Service{opt: opt}
			for range 2 {
				if err := s.reconcileLegacyRestarts(ctx); err != nil {
					t.Fatal(err)
				}
			}
			got, err := opt.Store.SubtitleRecord(ctx, old.Job.ID)
			if err != nil || got.Job.State != old.Job.State || got.Job.Error != old.Job.Error || got.Job.UpdatedAt != old.Job.UpdatedAt {
				t.Fatalf("legacy history changed: %+v %v", got, err)
			}
			want := ""
			if scenario == "replacement" || scenario == "blocked replacement" {
				want = next.Job.ID
			}
			if got.Job.SupersededBy != want {
				t.Fatalf("supersededBy=%s want=%s", got.Job.SupersededBy, want)
			}
			if want != "" {
				summary, err := opt.Store.TaskCenterSummary(ctx)
				if err != nil || summary.Total != 2 || summary.Queued != 1 || summary.Attention != 0 || summary.History != 1 {
					t.Fatalf("wrong legacy summary: %+v %v", summary, err)
				}
				keys, _, err := opt.Store.TaskCenterKeys(ctx, storage.TaskCenterFilter{State: "attention"})
				if err != nil || len(keys) != 0 {
					t.Fatalf("superseded failure still needs attention: %v %v", keys, err)
				}
				keys, _, err = opt.Store.TaskCenterKeys(ctx, storage.TaskCenterFilter{State: "history"})
				if err != nil || len(keys) != 1 || keys[0] != "subtitle:"+old.Job.ID {
					t.Fatalf("lost legacy history: %v %v", keys, err)
				}
				if _, err := s.Action(ctx, old.Job.ID, "restart"); err == nil {
					t.Fatal("superseded failure can create competing work")
				}
			}
		})
	}
}

package subtitles

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"curated-backend/internal/agent/prompts"
	"curated-backend/internal/contracts"
	"curated-backend/internal/executil"
	"curated-backend/internal/llm"
	"curated-backend/internal/storage"
)

func TestRestartKeepsTaskIDAndRequeuesWithCurrentSettings(t *testing.T) {
	for _, checkpointExists := range []bool{true, false} {
		t.Run(map[bool]string{true: "with checkpoint", false: "expired checkpoint"}[checkpointExists], func(t *testing.T) {
			opt, r := fixture(t)
			ctx := context.Background()
			r.Job.State = "failed"
			r.Job.Error = "old failure"
			r.Job.ProviderError = "old provider error"
			r.Job.ExpiresAt = time.Now().Add(-time.Hour).UTC().Format(time.RFC3339Nano)
			r.Job.Completed = 1
			r.Job.Total = 2
			r.AudioTrack = 1
			r.PromptVersion = "subtitle-translation-v2"
			assets, err := publish(r.SourcePath, r.Job.FileID, r.Job.ID, []Cue{{ID: 1, End: 1000, Text: "原文", Translation: "旧译文"}})
			if err != nil {
				t.Fatal(err)
			}
			r.Job.Assets = assets
			if err := opt.Store.SaveSubtitleRecord(ctx, r); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(opt.Root, r.Job.ID, "checkpoint.json")
			if checkpointExists {
				if err := writeJSON(path, checkpoint{Version: 1, Duration: 10, Parts: []part{{Recognized: true, Cues: []Cue{{ID: 1, Translation: "旧译文"}}}}}); err != nil {
					t.Fatal(err)
				}
			}
			before, _ := os.ReadFile(path)
			opt.Config = func(model string) (RuntimeConfig, error) {
				if model != "" {
					t.Errorf("restart should use current settings, got %s", model)
				}
				return RuntimeConfig{Enabled: true, APIKey: "key", RecognitionModel: "qwen3-asr-flash-filetrans", DefaultModel: "configured-ai", Text: llm.ClientConfig{BaseURL: "http://new-model/v1", Model: "new-model"}}, nil
			}
			s := &Service{opt: opt}
			next, err := s.Action(ctx, r.Job.ID, "restart")
			if err != nil {
				t.Fatal(err)
			}
			if next.ID != r.Job.ID || next.State != "queued" || next.Stage != "preparing" || next.Completed != 0 || next.Total != 0 || next.Error != "" || next.ProviderError != "" || next.ExpiresAt != "" || next.CreatedAt != r.Job.CreatedAt || len(next.Assets) != len(assets) || next.TranslationModel != "configured-ai" || next.RecognitionModel != "qwen3-asr-flash-filetrans" {
				t.Fatalf("restart not fresh: %+v", next)
			}
			record, err := opt.Store.SubtitleRecord(ctx, next.ID)
			if err != nil || record.PromptVersion != prompts.SubtitleTranslationPrompt().Version || record.TextModel != "new-model" || record.AudioTrack != 1 || record.Job.FileID != r.Job.FileID {
				t.Fatalf("wrong restart configuration: %+v error=%v", record, err)
			}
			if record.AttemptID == "" || record.AttemptID == r.Job.ID {
				t.Fatal("restart did not isolate its checkpoint")
			}
			if _, err := os.Stat(s.cpPath(checkpointID(record))); !os.IsNotExist(err) {
				t.Fatal("restart reused checkpoint")
			}
			records, err := opt.Store.ListSubtitleRecords(ctx, "")
			if err != nil || len(records) != 1 || records[0].RequestID != r.RequestID {
				t.Fatalf("restart created another task: %+v error=%v", records, err)
			}
			summary, err := opt.Store.TaskCenterSummary(ctx)
			if err != nil || summary.Total != 1 || summary.Attention != 0 || summary.Queued != 1 {
				t.Fatalf("restarted task still needs attention: %+v error=%v", summary, err)
			}
			keys, _, err := opt.Store.TaskCenterKeys(ctx, storage.TaskCenterFilter{State: "attention"})
			if err != nil || len(keys) != 0 {
				t.Fatalf("restarted task remains in attention list: %v %v", keys, err)
			}
			keys, _, err = opt.Store.TaskCenterKeys(ctx, storage.TaskCenterFilter{State: "active"})
			if err != nil || len(keys) != 1 || keys[0] != "subtitle:"+r.Job.ID {
				t.Fatalf("active task ID changed: %v %v", keys, err)
			}
			after, _ := os.ReadFile(path)
			if string(before) != string(after) {
				t.Fatal("old checkpoint changed")
			}
			if _, err := s.AssetPath(ctx, assets[1].ID); err != nil {
				t.Fatalf("old output lost: %v", err)
			}
			// Double-click/repeated requests return the existing active job.
			again, err := s.Action(ctx, r.Job.ID, "restart")
			if err != nil || again.ID != next.ID {
				t.Fatalf("duplicate restart: %+v %v", again, err)
			}
			againRecord, _ := opt.Store.SubtitleRecord(ctx, again.ID)
			if againRecord.AttemptID != record.AttemptID {
				t.Fatal("repeated restart discarded active progress")
			}
		})
	}
}

func TestRestartRejectsCompletedAndUnavailableProvider(t *testing.T) {
	for _, state := range []string{"completed", "failed"} {
		t.Run(state, func(t *testing.T) {
			opt, r := fixture(t)
			r.Job.State = state
			if err := opt.Store.SaveSubtitleRecord(context.Background(), r); err != nil {
				t.Fatal(err)
			}
			opt.Config = func(string) (RuntimeConfig, error) { return RuntimeConfig{}, nil }
			s := &Service{opt: opt}
			if _, err := s.Action(context.Background(), r.Job.ID, "restart"); err == nil {
				t.Fatal("invalid restart accepted")
			}
			records, err := opt.Store.ListSubtitleRecords(context.Background(), "")
			if err != nil || len(records) != 1 || records[0].Job.State != state {
				t.Fatalf("failed restart mutated history: %+v %v", records, err)
			}
		})
	}
}

func TestRestartRunsRecognitionAndTranslationFromBeginning(t *testing.T) {
	command, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("FFmpeg not installed")
	}
	opt, r := fixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := executil.CommandContext(ctx, command, "-nostdin", "-v", "error", "-y", "-f", "lavfi", "-i", "sine=frequency=500:sample_rate=48000", "-t", "1", "-c:a", "aac", r.SourcePath).Run(); err != nil {
		t.Fatal(err)
	}
	r.Job.State = "failed"
	r.Revision, _ = revision(r.SourcePath)
	if err := writeJSON(filepath.Join(opt.Root, r.Job.ID, "checkpoint.json"), checkpoint{Version: 1, Duration: 10, Parts: []part{{Recognized: true, Cues: []Cue{{ID: 1, Translation: "old completed translation"}}}}}); err != nil {
		t.Fatal(err)
	}
	if err := opt.Store.SaveSubtitleRecord(ctx, r); err != nil {
		t.Fatal(err)
	}
	opt.FFmpeg = command
	oldConfig := opt.Config
	opt.Config = func(model string) (RuntimeConfig, error) {
		cfg, err := oldConfig(model)
		cfg.DefaultModel = "configured-ai"
		return cfg, err
	}
	translations := 0
	oldTranslate := opt.Translate
	opt.Translate = func(ctx context.Context, r storage.SubtitleRecord, cues []Cue) ([]Cue, error) {
		translations++
		return oldTranslate(ctx, r, cues)
	}
	s := &Service{opt: opt}
	next, err := s.Action(ctx, r.Job.ID, "restart")
	if err != nil {
		t.Fatal(err)
	}
	record, err := opt.Store.SubtitleRecord(ctx, next.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.execute(ctx, &record); err != nil {
		t.Fatal(err)
	}
	if opt.Provider.(*fakeProvider).calls != 3 || translations != 1 || record.Job.State != "completed" {
		t.Fatalf("restart did not execute full pipeline: %+v", record.Job)
	}
}

func TestRetryAndServerResumeUseOnlyTheRestartedAttempt(t *testing.T) {
	opt, r := fixture(t)
	ctx := context.Background()
	r.Job.State = "failed"
	if err := opt.Store.SaveSubtitleRecord(ctx, r); err != nil {
		t.Fatal(err)
	}
	s := &Service{opt: opt}
	if err := writeJSON(s.cpPath(r.Job.ID), checkpoint{Version: 1, Duration: 10, Parts: []part{{Recognized: true, Cues: []Cue{{ID: 1, Text: "old", Translation: "old"}}}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Action(ctx, r.Job.ID, "restart"); err != nil {
		t.Fatal(err)
	}
	r, _ = opt.Store.SubtitleRecord(ctx, r.Job.ID)
	r.Job.State = "failed"
	if err := opt.Store.SaveSubtitleRecord(ctx, r); err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(s.cpPath(checkpointID(r)), checkpoint{Version: 1, Duration: 10, Parts: []part{{Recognized: true, Cues: []Cue{{ID: 1, End: 1000, Text: "new", Translation: "new translation"}}}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Action(ctx, r.Job.ID, "retry"); err != nil {
		t.Fatal(err)
	}
	worker := New(ctx, opt)
	defer worker.Close()
	job := waitJob(t, worker, r.Job.ID, "completed")
	if !strings.Contains(job.Assets[1].ID, "-"+r.AttemptID+"-") {
		t.Fatal("subtitle output version did not distinguish the new attempt")
	}
	if opt.Provider.(*fakeProvider).calls != 0 {
		t.Fatal("server resume resubmitted a recognition request")
	}
	path, err := worker.AssetPath(ctx, job.Assets[1].ID)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(raw), "new translation") || strings.Contains(string(raw), "old") {
		t.Fatalf("resumed wrong checkpoint: %s %v", raw, err)
	}
}

func TestRestartDoesNotCreateCompetingTasks(t *testing.T) {
	for _, state := range []string{"queued", "running", "blocked", "cancelled"} {
		t.Run(state, func(t *testing.T) {
			opt, r := fixture(t)
			ctx := context.Background()
			r.Job.State = state
			if err := opt.Store.SaveSubtitleRecord(ctx, r); err != nil {
				t.Fatal(err)
			}
			s := &Service{opt: opt}
			job, err := s.Action(ctx, r.Job.ID, "restart")
			if err != nil || job.ID != r.Job.ID || (active(state) && job.State != state) || (!active(state) && job.State != "queued") {
				t.Fatalf("unexpected restart: %+v %v", job, err)
			}
		})
	}
	opt, r := fixture(t)
	ctx := context.Background()
	other := r
	other.Job.ID, other.RequestID = newID(), "other-request"
	r.Job.State = "failed"
	if err := opt.Store.SaveSubtitleRecord(ctx, r); err != nil {
		t.Fatal(err)
	}
	if err := opt.Store.SaveSubtitleRecord(ctx, other); err != nil {
		t.Fatal(err)
	}
	s := &Service{opt: opt}
	if _, err := s.Action(ctx, r.Job.ID, "restart"); err == nil {
		t.Fatal("accepted competing restart")
	}
	unchanged, _ := opt.Store.SubtitleRecord(ctx, r.Job.ID)
	if unchanged.Job.State != "failed" || unchanged.AttemptID != "" {
		t.Fatal("conflicting restart changed original task")
	}
}

func TestRestartValidationKeepsOriginalProgress(t *testing.T) {
	for _, scenario := range []string{"full queue", "source missing", "still stopping"} {
		t.Run(scenario, func(t *testing.T) {
			opt, r := fixture(t)
			ctx := context.Background()
			r.Job.State, r.Job.Completed = "failed", 7
			if err := opt.Store.SaveSubtitleRecord(ctx, r); err != nil {
				t.Fatal(err)
			}
			s := &Service{opt: opt}
			if err := writeJSON(s.cpPath(r.Job.ID), checkpoint{Version: 1, Duration: 10}); err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "full queue":
				for i := range 8 {
					path := filepath.Join(filepath.Dir(r.SourcePath), fmt.Sprintf("DEMO-%03d.mp4", i+2))
					if err := os.WriteFile(path, []byte("fixture"), 0600); err != nil {
						t.Fatal(err)
					}
					movie, err := opt.Store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{Number: fmt.Sprintf("DEMO-%03d", i+2), Path: path, FileName: filepath.Base(path)})
					if err != nil {
						t.Fatal(err)
					}
					detail, err := opt.Store.GetMoviePlaybackDetail(ctx, movie.MovieID)
					if err != nil {
						t.Fatal(err)
					}
					other := r
					other.Job.ID, other.RequestID = newID(), newID()
					other.Job.MovieID, other.Job.FileID, other.Job.State = movie.MovieID, detail.SelectedFileID, "queued"
					if err := opt.Store.SaveSubtitleRecord(ctx, other); err != nil {
						t.Fatal(err)
					}
				}
			case "source missing":
				if err := os.Remove(r.SourcePath); err != nil {
					t.Fatal(err)
				}
			case "still stopping":
				s.active = map[string]context.CancelFunc{r.Job.ID: func() {}}
			}
			if _, err := s.Action(ctx, r.Job.ID, "restart"); err == nil {
				t.Fatal("invalid restart accepted")
			}
			got, err := opt.Store.SubtitleRecord(ctx, r.Job.ID)
			if err != nil || got.Job.State != "failed" || got.Job.Completed != 7 || got.AttemptID != "" {
				t.Fatalf("rejected restart lost progress: %+v %v", got, err)
			}
			if _, err := os.Stat(s.cpPath(r.Job.ID)); err != nil {
				t.Fatal("rejected restart lost checkpoint")
			}
		})
	}
}

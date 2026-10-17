package subtitles

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"curated-backend/internal/agent/prompts"
	"curated-backend/internal/executil"
	"curated-backend/internal/llm"
	"curated-backend/internal/storage"
)

func TestRestartCreatesFreshJobWithCurrentSettingsAndKeepsHistory(t *testing.T) {
	for _, checkpointExists := range []bool{true, false} {
		t.Run(map[bool]string{true: "with checkpoint", false: "expired checkpoint"}[checkpointExists], func(t *testing.T) {
			opt, r := fixture(t)
			ctx := context.Background()
			r.Job.State = "failed"
			r.Job.Error = "old failure"
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
			if next.ID == r.Job.ID || next.State != "queued" || next.Stage != "preparing" || next.Completed != 0 || next.Total != 0 || len(next.Assets) != 0 || next.TranslationModel != "configured-ai" || next.RecognitionModel != "qwen3-asr-flash-filetrans" {
				t.Fatalf("restart not fresh: %+v", next)
			}
			record, err := opt.Store.SubtitleRecord(ctx, next.ID)
			if err != nil || record.PromptVersion != prompts.SubtitleTranslationPrompt().Version || record.TextModel != "new-model" || record.AudioTrack != 1 || record.Job.FileID != r.Job.FileID {
				t.Fatalf("wrong restart configuration: %+v error=%v", record, err)
			}
			if _, err := os.Stat(s.cpPath(next.ID)); !os.IsNotExist(err) {
				t.Fatal("restart reused checkpoint")
			}
			old, err := opt.Store.SubtitleRecord(ctx, r.Job.ID)
			if err != nil || old.Job.State != "failed" || old.Job.Completed != 1 || old.Job.Error != "old failure" {
				t.Fatalf("old history changed: %+v error=%v", old, err)
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
		})
	}
}

func TestRestartRejectsRunningCompletedAndUnavailableProvider(t *testing.T) {
	for _, state := range []string{"running", "completed", "failed"} {
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

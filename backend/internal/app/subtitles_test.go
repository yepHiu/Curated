package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/llm"
	"curated-backend/internal/storage"
	"curated-backend/internal/subtitles"
)

func TestSubtitleTranslationSendsAllCuesWithoutFixedOutputLimit(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var body struct {
			Messages  []llm.ChatMessage `json:"messages"`
			MaxTokens *int              `json:"max_tokens"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		if body.MaxTokens != nil || len(body.Messages) != 2 {
			t.Errorf("unexpected request: %+v", body)
			return
		}
		var cues []subtitles.Cue
		if err := json.Unmarshal([]byte(body.Messages[1].Content), &cues); err != nil || len(cues) != 180 {
			t.Errorf("request lost full dialogue: count=%d err=%v", len(cues), err)
			return
		}
		for i := range cues {
			cues[i].Translation = "你好"
		}
		content, _ := json.Marshal(cues)
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{
			"finish_reason": "stop", "message": map[string]string{"content": string(content)},
		}}})
	}))
	defer server.Close()
	a := &App{cfg: config.Config{AIProvider: config.AIProviderConfig{BaseURL: server.URL, Model: "fixture", ContextWindow: 131072}}}
	cfg, err := a.subtitleRuntime("configured-ai")
	if err != nil || cfg.Translation.ContextWindow != 131072 {
		t.Fatalf("runtime capacity=%+v err=%v", cfg.Translation, err)
	}
	cues := make([]subtitles.Cue, 180)
	for i := range cues {
		cues[i] = subtitles.Cue{ID: i + 1, Text: "こんにちは"}
	}
	r := storage.SubtitleRecord{TextBaseURL: server.URL, TextModel: "fixture", Job: contracts.SubtitleJob{ID: "fixture", TranslationModel: "configured-ai"}}
	result, err := a.translateSubtitleCues(context.Background(), r, cues)
	if err != nil || calls != 1 || len(result) != 180 || result[179].ID != 180 || result[179].Translation != "你好" {
		t.Fatalf("calls=%d result=%d err=%v", calls, len(result), err)
	}
}

func TestSubtitleTranslationPreservesTruncationCauseForCapacityFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"finish_reason":"length","message":{"content":"[{"}}]}`))
	}))
	defer server.Close()
	a := &App{cfg: config.Config{AIProvider: config.AIProviderConfig{BaseURL: server.URL, Model: "fixture"}}}
	r := storage.SubtitleRecord{TextBaseURL: server.URL, TextModel: "fixture", Job: contracts.SubtitleJob{TranslationModel: "configured-ai"}}
	_, err := a.translateSubtitleCues(context.Background(), r, []subtitles.Cue{{ID: 1, Text: "原文"}})
	if !errors.Is(err, llm.ErrOutputTruncated) {
		t.Fatalf("lost truncation cause: %v", err)
	}
}

func TestSubtitleUsageIncludesTranslationValidation(t *testing.T) {
	for _, test := range []struct {
		name, content, status, code string
		partial                     bool
	}{
		{"missing", `[{"id":1,"translation":"你好"}]`, "failed", "incomplete_translation", true},
		{"empty", `[]`, "failed", "incomplete_translation", true},
		{"duplicate", `[{"id":1,"translation":"你好"},{"id":1,"translation":"重复"}]`, "failed", "invalid_translation", false},
		{"unknown", `[{"id":99,"translation":"错误"}]`, "failed", "invalid_translation", false},
		{"blank", `[{"id":1,"translation":" "}]`, "failed", "invalid_translation", false},
		{"complete", `[{"id":2,"translation":"再见"},{"id":1,"translation":"你好"}]`, "completed", "", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintf(w, `{"choices":[{"finish_reason":"stop","message":{"content":%q}}],"usage":{"prompt_tokens":100,"completion_tokens":30,"total_tokens":130}}`, test.content)
			}))
			defer server.Close()
			a := governanceTestApp(t)
			a.cfg.AIProvider = config.AIProviderConfig{BaseURL: server.URL, Model: "fixture"}
			r := storage.SubtitleRecord{TextBaseURL: server.URL, TextModel: "fixture", Job: contracts.SubtitleJob{ID: "job", TranslationModel: "configured-ai"}}
			result, err := a.translateSubtitleCues(context.Background(), r, []subtitles.Cue{{ID: 1, Text: "原文"}, {ID: 2, Text: "原文"}})
			if (err != nil) != (test.status == "failed") {
				t.Fatalf("unexpected validation error: %v", err)
			}
			var incomplete *subtitles.IncompleteTranslationError
			if errors.As(err, &incomplete) != test.partial {
				t.Fatalf("wrong repair classification: %v", err)
			}
			if test.name == "missing" && (len(result) != 1 || incomplete.Expected != 2 || incomplete.Received != 1) {
				t.Fatalf("lost validated subset or counts: result=%+v error=%v", result, err)
			}
			report, err := a.GetAIReport(context.Background(), contracts.AIReportQuery{Days: 30, Limit: 10, Channel: "subtitle"})
			if err != nil || len(report.Items) != 1 {
				t.Fatalf("missing usage: report=%+v error=%v", report, err)
			}
			row := report.Items[0]
			if row.Status != test.status || row.ErrorCode != test.code || row.TotalTokens != 130 || row.ModelCalls != 1 || row.Action != "job" {
				t.Fatalf("usage excludes validation or token counts: %+v", row)
			}
		})
	}
}

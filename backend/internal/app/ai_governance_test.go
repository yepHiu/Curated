package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"curated-backend/internal/agent/core"
	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/llm"
	"curated-backend/internal/storage"
)

func enabledAITestConfig() config.Config {
	v := config.DefaultAIGovernance()
	v.Enabled = true
	return config.Config{AIGovernance: &v}
}
func governanceTestApp(t *testing.T) *App {
	t.Helper()
	store, err := storage.NewSQLiteStore(filepath.Join(t.TempDir(), "ai.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	if err = store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	return &App{store: store, cfg: enabledAITestConfig(), librarySettingsPath: filepath.Join(t.TempDir(), "library-config.cfg")}
}
func TestAIGovernancePersistenceAndEnforcement(t *testing.T) {
	a := governanceTestApp(t)
	ctx := context.Background()
	settings := a.AIGovernanceSettings()
	settings.ReadOnly = true
	settings.Privacy = "minimal"
	settings.StepLimit = 3
	if err := a.SetAIGovernanceSettings(settings); err != nil {
		t.Fatal(err)
	}
	reloaded := config.Config{}
	if err := config.MergeLibrarySettingsFile(&reloaded, a.librarySettingsPath); err != nil {
		t.Fatal(err)
	}
	if reloaded.AIGovernance == nil || !reloaded.AIGovernance.ReadOnly {
		t.Fatal("not persisted")
	}
	if a.aiProjection("http://127.0.0.1/v1") != core.SanitizeMinimal {
		t.Fatal("privacy not applied")
	}
	preview := a.ensureAgentGateway().Invoke(ctx, core.Call{Name: core.SaveMovieCommentName, SessionID: "test", Args: json.RawMessage(`{"movieId":"m1","body":"private"}`)})
	if preview.OK || preview.Error.Code != "AI_TOOL_PERMISSION_DENIED" {
		t.Fatalf("read-only bypass: %+v", preview)
	}
	if _, err := a.ApplyAITool(ctx, contracts.AIToolApplyRequest{Name: core.SaveMovieCommentName}); err == nil {
		t.Fatal("apply allowed")
	}
	rows, err := a.ListAIAudit(ctx, contracts.AIReportQuery{Days: 30, Limit: 25})
	if err != nil || rows.Total != 2 {
		t.Fatalf("missing rejected audit %+v %v", rows, err)
	}
	settings.RetentionDays = 1
	if err := a.SetAIGovernanceSettings(settings); err == nil {
		t.Fatal("invalid retention accepted")
	}
}

func TestAIChatUsagePersistenceAndDisabledBackend(t *testing.T) {
	a := governanceTestApp(t)
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"hello\"}}]}\n\ndata: {\"choices\":[],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":2,\"total_tokens\":12}}\n\ndata: [DONE]\n\n")
	}))
	defer server.Close()
	a.cfg.AIProvider = config.AIProviderConfig{BaseURL: server.URL, Model: "fixture"}
	req := contracts.AIChatRequest{Messages: []contracts.AIChatMessage{{Role: "user", Content: "hello"}}}
	if err := a.StreamAIChat(context.Background(), req, nil); err != nil {
		t.Fatal(err)
	}
	report, err := a.GetAIReport(context.Background(), contracts.AIReportQuery{Days: 30, Limit: 25})
	if err != nil {
		t.Fatal(err)
	}
	if report.Total != 1 || report.Summary.TotalTokens != 12 || report.Items[0].FirstTextMs == nil || report.Items[0].Status != "completed" {
		t.Fatalf("report %+v", report)
	}
	value := a.AIGovernanceSettings()
	value.Enabled = false
	if err := a.SetAIGovernanceSettings(value); err != nil {
		t.Fatal(err)
	}
	if err := a.StreamAIChat(context.Background(), req, nil); err == nil {
		t.Fatal("disabled request succeeded")
	}
	if calls != 1 {
		t.Fatal("disabled request reached model")
	}
}

func TestAIObservationsAggregateMissingUsageAndCancellation(t *testing.T) {
	a := governanceTestApp(t)
	ctx, r, finish := a.beginAIRun(context.Background(), "chat", "")
	r.observe(llm.Observation{Usage: &llm.Usage{PromptTokens: 5, CompletionTokens: 2, TotalTokens: 7}})
	r.observe(llm.Observation{})
	value := a.AIGovernanceSettings()
	value.Enabled = false
	if err := a.SetAIGovernanceSettings(value); err != nil {
		t.Fatal(err)
	}
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("policy change did not cancel")
	}
	finish(nil)
	report, err := a.GetAIReport(context.Background(), contracts.AIReportQuery{Days: 30, Limit: 25})
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.ModelCalls != 2 || report.Summary.UsageCalls != 1 || report.Summary.TotalTokens != 7 || report.Summary.Cancelled != 1 {
		t.Fatalf("wrong partial usage %+v", report)
	}
}

func TestAIConfirmationRateLimitSpansActionSessions(t *testing.T) {
	a := governanceTestApp(t)
	a.cfg.AIGovernance.WritePerMinute = 1
	a.cfg.AIGovernance.StepLimit = 1 // Preview uses the step; human apply still works.
	ctx := context.Background()
	for i, session := range []string{"act_one", "act_two"} {
		args := mustJSON(map[string]any{"name": session, "filters": map[string]any{"schemaVersion": 1, "mode": "library"}})
		preview := a.ensureAgentGateway().Invoke(ctx, core.Call{Name: core.CreateSavedViewName, Channel: core.ChannelAction, SessionID: session, Args: args})
		if !preview.OK || preview.ConfirmToken == "" {
			t.Fatalf("preview %+v", preview)
		}
		_, err := a.ApplyAITool(ctx, contracts.AIToolApplyRequest{Name: core.CreateSavedViewName, SessionID: session, ConfirmToken: preview.ConfirmToken, Arguments: preview.ConfirmArgs})
		if i == 0 && err != nil {
			t.Fatal(err)
		}
		if i == 1 && (err == nil || aiRunError(err) != "AI_RATE_LIMITED") {
			t.Fatalf("cross-session limit bypass: %v", err)
		}
	}
}

func TestAIActionAndProviderFailuresAreMeasuredWithoutBodies(t *testing.T) {
	a := governanceTestApp(t)
	ctx := context.Background()
	movie, err := a.store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{TaskID: "t", Path: "D:/fixture/T-001.mp4", FileName: "T-001.mp4", Number: "T-001"})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"choices":[{"message":{"content":"Polished note"}}],"usage":{"prompt_tokens":10,"completion_tokens":3,"total_tokens":13}}`)
	}))
	defer server.Close()
	a.cfg.AIProvider = config.AIProviderConfig{BaseURL: server.URL, Model: "fixture"}
	preview, err := a.RunAIAction(ctx, "polish_comment", contracts.AIActionRequest{MovieID: movie.MovieID, Body: "private text"})
	if err != nil || preview.ConfirmToken == "" {
		t.Fatalf("preview %+v %v", preview, err)
	}
	report, err := a.GetAIReport(ctx, contracts.AIReportQuery{Days: 30, Limit: 25, Channel: "action"})
	if err != nil || report.Total != 1 || report.Items[0].TotalTokens != 13 || report.Items[0].ToolCalls != 1 || report.Items[0].FirstTextMs != nil {
		t.Fatalf("action report %+v %v", report, err)
	}
	failed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
		fmt.Fprint(w, "private provider response")
	}))
	defer failed.Close()
	result := a.TestAIProvider(ctx, &contracts.AIProviderSettingsDTO{BaseURL: failed.URL, Model: "fixture"})
	if result.OK {
		t.Fatal("failed provider accepted")
	}
	failures, err := a.GetAIReport(ctx, contracts.AIReportQuery{Days: 30, Limit: 25, Channel: "test", Status: "failed"})
	if err != nil || failures.Total != 1 || failures.Items[0].ErrorCode != "rate_limit" || failures.Items[0].UsageCalls != 0 {
		t.Fatalf("failure report %+v %v", failures, err)
	}
	encoded, _ := json.Marshal(failures)
	if strings.Contains(string(encoded), "private") || strings.Contains(string(encoded), failed.URL) {
		t.Fatal("raw provider data leaked into metrics")
	}
}

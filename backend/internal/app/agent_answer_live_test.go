package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"curated-backend/internal/agent/core"
	agenttools "curated-backend/internal/agent/tools"
	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
)

// Opt-in paid compatibility test: configuration only, never the real library
// or metadata providers. All source records and history are temporary.
func TestAILiveAnswerReferences(t *testing.T) {
	path := os.Getenv("CURATED_AI_EVAL_SETTINGS")
	if path == "" {
		t.Skip("set CURATED_AI_EVAL_SETTINGS to opt into paid synthetic compatibility test")
	}
	if !filepath.IsAbs(path) {
		t.Fatal("settings path must be absolute")
	}
	loaded := config.Config{}
	if config.MergeLibrarySettingsFile(&loaded, path) != nil {
		t.Fatal("could not load provider settings")
	}
	if loaded.AIProvider.BaseURL == "" || loaded.AIProvider.Model == "" {
		t.Fatal("provider not configured")
	}
	a := governanceTestApp(t)
	a.cfg.AIProvider = loaded.AIProvider
	a.cfg.Proxy = loaded.Proxy
	reg := core.NewRegistry()
	gw := core.NewGateway(reg, nil, nil, nil)
	if err := agenttools.RegisterQueryTools(reg, a); err != nil {
		t.Fatal(err)
	}
	if err := agenttools.RegisterPresentTools(reg, gw.MovieRefs()); err != nil {
		t.Fatal(err)
	}
	// 为当前隔离用例提供可控响应或收集结果，验证发布与取消边界。
	a.agentRT.once.Do(func() { a.agentRT.gateway = gw })
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()
	row, err := a.store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{TaskID: "synthetic", Path: "D:/synthetic/SYN-001.mp4", FileName: "SYN-001.mp4", Number: "SYN-001"})
	if err != nil {
		t.Fatal(err)
	}
	if err := a.store.PatchMovieUserPrefs(ctx, row.MovieID, contracts.PatchMovieInput{UserTitleSet: true, UserTitle: "Synthetic reference record"}); err != nil {
		t.Fatal(err)
	}
	var snapshot *contracts.AIAnswerEvidenceDTO
	var sessionID string
	var rawThinking bool
	// 收集用户实际可见的事件，检查草稿不会通过旁路发布。
	err = a.StreamAIChat(ctx, contracts.AIChatRequest{Locale: "zh-CN", Messages: []contracts.AIChatMessage{{Role: "user", Content: "查询本地 SYN-001，展示它的番号和标题。必须先查询，再用 submit_answer 提交本轮 answerRefs 中的记录；不需要外部查询。"}}}, func(e contracts.AIChatSSEEvent) {
		sessionID = e.SessionID
		if e.Type == "thinking_delta" {
			rawThinking = true
		}
		if e.Type == "message_done" {
			snapshot = e.AnswerEvidence
		}
	})
	if err != nil {
		t.Fatal("synthetic model request failed")
	}
	if rawThinking || snapshot == nil || len(snapshot.Items) != 1 || snapshot.Items[0].Fields["code"] != "SYN-001" || snapshot.Items[0].Fields["title"] != "Synthetic reference record" {
		t.Fatal("model did not publish the expected structured snapshot")
	}
	detail, err := a.GetAIChatSession(ctx, sessionID)
	if err != nil || len(detail.Messages) < 2 {
		t.Fatal("published history missing")
	}
	if detail.Messages[len(detail.Messages)-1].Events[len(detail.Messages[len(detail.Messages)-1].Events)-1].AnswerEvidence == nil {
		t.Fatal("snapshot not persisted")
	}
	report, err := a.GetAIReport(ctx, contracts.AIReportQuery{Days: 30, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("synthetic structured answer and history passed; calls=%d total_tokens=%d", report.Items[0].ModelCalls, report.Summary.TotalTokens)
}

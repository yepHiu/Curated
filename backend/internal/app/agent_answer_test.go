package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
)

// TestRejectedModelDraftNeverEntersChatHistory 验证 Rejected Model Draft Never Enters Chat History 的行为与失败边界，使用隔离测试数据。
func TestRejectedModelDraftNeverEntersChatHistory(t *testing.T) {
	a := governanceTestApp(t)
	// 提供受控的本地 SSE 响应，验证真实传输链路而不访问外部服务。
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"reasoning_content\":\"FAKE-999\",\"content\":\"FAKE-\"}}]}\n\ndata: {\"choices\":[{\"delta\":{\"content\":\"999\"}}]}\n\ndata: [DONE]\n\n")
	}))
	defer server.Close()
	a.cfg.AIProvider = config.AIProviderConfig{BaseURL: server.URL, Model: "synthetic"}
	var events []contracts.AIChatSSEEvent
	ctx := context.Background()
	// 收集用户实际可见的事件，检查草稿不会通过旁路发布。
	err := a.StreamAIChat(ctx, contracts.AIChatRequest{Locale: "en", Messages: []contracts.AIChatMessage{{Role: "user", Content: "recommend something"}}}, func(e contracts.AIChatSSEEvent) { events = append(events, e) })
	if err != nil {
		t.Fatal(err)
	}
	detail, err := a.GetAIChatSession(ctx, events[0].SessionID)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(detail)
	if strings.Contains(string(raw), "FAKE-") {
		t.Fatal("unverified draft persisted", string(raw))
	}
	last := detail.Messages[len(detail.Messages)-1]
	if last.Role != "assistant" || !strings.Contains(last.Content, "could not verify") || last.Events[len(last.Events)-1].Outcome.ReasonCode != "answer_rejected" {
		t.Fatal(last)
	}
}

// TestPublishedAnswerSnapshotSurvivesHistoryWithoutNewAuthority 验证 Published Answer Snapshot Survives History Without New Authority 的行为与失败边界，使用隔离测试数据。
func TestPublishedAnswerSnapshotSurvivesHistoryWithoutNewAuthority(t *testing.T) {
	a := governanceTestApp(t)
	ctx := context.Background()
	session, err := a.store.CreateAIChatSession(ctx, "snapshot")
	if err != nil {
		t.Fatal(err)
	}
	evidence := &contracts.AIAnswerEvidenceDTO{Version: 1, Items: []contracts.AIAnswerEvidenceItemDTO{{RefID: "expired_ref", Source: "provider", Tool: "search_provider_titles", RetrievedAt: "2026-09-09T00:00:00Z", Fields: map[string]any{"code": "TEST-101", "title": "Recorded"}}}}
	if err := a.persistAIChatTurn(ctx, session.ID, "Published record", []contracts.AIChatSSEEvent{{Type: "message_done", Outcome: &contracts.AIChatOutcomeDTO{Status: "completed"}, AnswerEvidence: evidence}}); err != nil {
		t.Fatal(err)
	}
	detail, err := a.GetAIChatSession(ctx, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	got := detail.Messages[0].Events[0].AnswerEvidence
	if got == nil || got.Items[0].Fields["title"] != "Recorded" {
		t.Fatal(got)
	}
	history, err := a.llmHistoryForSession(ctx, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(history)
	if strings.Contains(string(raw), "expired_ref") {
		t.Fatal("old reference projected into model context")
	}
}

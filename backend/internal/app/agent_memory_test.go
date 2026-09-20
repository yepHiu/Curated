package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/llm"
	"curated-backend/internal/storage"
)

type checkpointStreamer func(context.Context, llm.TurnRequest) (llm.AssistantTurn, error)

func (f checkpointStreamer) StreamTurn(ctx context.Context, req llm.TurnRequest, _ func(string)) (llm.AssistantTurn, error) {
	return f(ctx, req)
}

func TestPersistentMemoryBackfillsOldGoalsAndReusesCheckpoint(t *testing.T) {
	ctx := context.Background()
	store, err := storage.NewSQLiteStore(filepath.Join(t.TempDir(), "memory.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	session, err := store.CreateAIChatSession(ctx, "long conversation")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 101; i++ {
		role := "user"
		if i%2 == 1 {
			role = "assistant"
		}
		content := fmt.Sprintf("message %d", i)
		if i == 0 {
			content = "ORIGINAL GOAL: save no changes without asking"
		}
		if _, err := store.AppendAIChatMessage(ctx, session.ID, role, content, "", ""); err != nil {
			t.Fatal(err)
		}
	}
	calls := 0
	streamer := checkpointStreamer(func(_ context.Context, req llm.TurnRequest) (llm.AssistantTurn, error) {
		calls++
		if !strings.Contains(req.Messages[1].Content, "ORIGINAL GOAL") {
			t.Fatal("lost original goal before 80-message window")
		}
		return llm.AssistantTurn{Content: "ORIGINAL GOAL: save no changes without asking; pending request."}, nil
	})
	a := &App{store: store}
	var events []contracts.AIChatSSEEvent
	history, err := a.prepareAIHistory(ctx, session.ID, streamer, func(ev contracts.AIChatSSEEvent) { events = append(events, ev) })
	if err != nil || len(history) > 25 || !strings.Contains(history[0].Content, "ORIGINAL GOAL") || history[len(history)-1].Content != "message 100" {
		t.Fatalf("history=%+v err=%v", history, err)
	}
	seq, summary, err := store.AIContextCheckpoint(ctx, session.ID)
	if err != nil || seq != 78 || summary == "" || len(events) != 2 || events[1].Context.Phase != "ready" {
		t.Fatalf("checkpoint=%d %q events=%+v err=%v", seq, summary, events, err)
	}
	// A fresh App instance has no in-memory checkpoint; SQLite is the source.
	a = &App{store: store}
	previousCalls := calls
	if _, err := a.prepareAIHistory(ctx, session.ID, streamer, func(contracts.AIChatSSEEvent) {}); err != nil || calls != previousCalls {
		t.Fatalf("checkpoint not reused: %v", err)
	}
	if err := store.SaveAIContextCheckpoint(ctx, session.ID, seq-1, "stale"); err != nil {
		t.Fatal(err)
	}
	_, saved, _ := store.AIContextCheckpoint(ctx, session.ID)
	if saved != summary {
		t.Fatal("stale checkpoint replaced newer state")
	}
	rows, _ := store.ListAIChatContext(ctx, session.ID, 200)
	if len(rows) != 101 || !strings.Contains(rows[0].Content, "ORIGINAL GOAL") {
		t.Fatal("transcript modified")
	}
	if err := store.DeleteAIChatSession(ctx, session.ID); err != nil {
		t.Fatal(err)
	}
	seq, _, err = store.AIContextCheckpoint(ctx, session.ID)
	if err != nil || seq != 0 {
		t.Fatal("checkpoint survived session deletion")
	}
}

func TestMemoryFailureDoesNotAdvanceCheckpoint(t *testing.T) {
	ctx := context.Background()
	store, err := storage.NewSQLiteStore(filepath.Join(t.TempDir(), "memory.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	session, _ := store.CreateAIChatSession(ctx, "long")
	for i := 0; i < 30; i++ {
		if _, err := store.AppendAIChatMessage(ctx, session.ID, "user", fmt.Sprint(i), "", ""); err != nil {
			t.Fatal(err)
		}
	}
	a := &App{store: store}
	var phases []string
	history, err := a.prepareAIHistory(ctx, session.ID, checkpointStreamer(func(context.Context, llm.TurnRequest) (llm.AssistantTurn, error) {
		return llm.AssistantTurn{}, errors.New("offline")
	}), func(ev contracts.AIChatSSEEvent) { phases = append(phases, ev.Context.Phase) })
	seq, summary, _ := store.AIContextCheckpoint(ctx, session.ID)
	if err != nil || seq != 0 || summary != "" || history[len(history)-1].Content != "29" || phases[len(phases)-1] != "limited" {
		t.Fatalf("failed checkpoint committed: %d %q %v", seq, summary, phases)
	}
}

func TestSessionGateCancellationAndIsolation(t *testing.T) {
	a := &App{}
	release, err := a.acquireAIChat(context.Background(), "a")
	if err != nil {
		t.Fatal(err)
	}
	other, err := a.acquireAIChat(context.Background(), "b")
	if err != nil {
		t.Fatal(err)
	}
	other()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := a.acquireAIChat(ctx, "a"); !errors.Is(err, context.Canceled) {
		t.Fatalf("queued cancellation: %v", err)
	}
	release()
	if len(a.agentRT.chatGates) != 0 {
		t.Fatal("leaked session gates")
	}
}

func TestChatMemoryLifecycleKeepsSingleStartAndPrivateSummary(t *testing.T) {
	a := governanceTestApp(t)
	ctx := context.Background()
	session, err := a.store.CreateAIChatSession(ctx, "long chat")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 60; i++ {
		if _, err := a.store.AppendAIChatMessage(ctx, session.ID, "user", fmt.Sprintf("old question %d", i), "", ""); err != nil {
			t.Fatal(err)
		}
	}
	summaryCalls, answerCalls := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			ToolChoice string            `json:"tool_choice"`
			Messages   []llm.ChatMessage `json:"messages"`
			MaxTokens  int               `json:"max_tokens"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		content := "Continued the task."
		if body.MaxTokens != 32768 {
			t.Errorf("configured output reserve not passed to provider: %d", body.MaxTokens)
		}
		if body.ToolChoice == "none" {
			summaryCalls++
			content = "PRIVATE_CHECKPOINT: goal and remaining work"
		} else {
			answerCalls++
			if len(body.Messages) <= 26 {
				t.Error("configured history still uses the default 24-message window")
			}
			if len(body.Messages) < 2 || !strings.Contains(body.Messages[1].Content, "PRIVATE_CHECKPOINT") {
				t.Error("checkpoint missing from task request")
			}
		}
		payload, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"delta": map[string]string{"content": content}}}})
		fmt.Fprintf(w, "data: %s\n\ndata: [DONE]\n\n", payload)
	}))
	defer server.Close()
	a.cfg.AIProvider = config.AIProviderConfig{BaseURL: server.URL, Model: "synthetic", ContextWindow: 131072}
	var events []contracts.AIChatSSEEvent
	err = a.StreamAIChat(ctx, contracts.AIChatRequest{SessionID: session.ID, Messages: []contracts.AIChatMessage{{Role: "user", Content: "continue"}}}, func(ev contracts.AIChatSSEEvent) { events = append(events, ev) })
	if err != nil {
		t.Fatal(err)
	}
	starts, statuses := 0, 0
	for i, ev := range events {
		if ev.Seq != i+1 || ev.SessionID != session.ID || ev.MessageID == "" {
			t.Fatalf("invalid event identity: %+v", ev)
		}
		if ev.Type == "message_start" {
			starts++
		}
		if ev.Type == "context_status" {
			statuses++
		}
	}
	if starts != 1 || statuses != 2 || summaryCalls != 1 || answerCalls != 1 {
		t.Fatalf("starts=%d statuses=%d calls=%d/%d", starts, statuses, summaryCalls, answerCalls)
	}
	raw, _ := json.Marshal(events)
	if strings.Contains(string(raw), "PRIVATE_CHECKPOINT") {
		t.Fatal("private summary leaked via SSE")
	}
	detail, err := a.GetAIChatSession(ctx, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ = json.Marshal(detail)
	if strings.Contains(string(raw), "PRIVATE_CHECKPOINT") {
		t.Fatal("private summary leaked into visible transcript")
	}
}

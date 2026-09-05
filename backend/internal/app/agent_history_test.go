package app

import (
	"context"
	"path/filepath"
	"testing"

	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

func TestCancelledAIChatPersistsPartialReplyAndEvidence(t *testing.T) {
	store, err := storage.NewSQLiteStore(filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	session, err := store.CreateAIChatSession(ctx, "cancel")
	if err != nil {
		t.Fatal(err)
	}
	requestCtx, cancel := context.WithCancel(ctx)
	cancel()
	ok := false
	a := &App{store: store}
	if err := a.persistAIChatTurn(requestCtx, session.ID, "partial reply", []contracts.AIChatSSEEvent{
		{Type: "tool_call_result", Name: "search_movies", OK: &ok, Summary: "provider failed"},
	}); err != nil {
		t.Fatal(err)
	}
	detail, err := a.GetAIChatSession(ctx, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Messages) != 1 {
		t.Fatalf("messages: %+v", detail.Messages)
	}
	msg := detail.Messages[0]
	if msg.Content != "partial reply" || len(msg.Events) != 2 || *msg.Events[0].OK || msg.Events[1].Outcome.Status != "cancelled" {
		t.Fatalf("lost turn state: %+v", msg)
	}
	modelHistory, err := a.llmHistoryForSession(ctx, session.ID)
	if err != nil || len(modelHistory) != 1 || modelHistory[0].Content != "partial reply" {
		t.Fatalf("model history: %+v, %v", modelHistory, err)
	}
}

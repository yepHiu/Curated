package storage

import (
	"context"
	"path/filepath"
	"testing"
)

func TestAIChatSessionPersistence(t *testing.T) {
	t.Parallel()
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	session, err := store.CreateAIChatSession(ctx, "今晚看什么")
	if err != nil {
		t.Fatal(err)
	}
	if session.ID == "" || session.Title != "今晚看什么" {
		t.Fatalf("session = %+v", session)
	}
	if _, err := store.AppendAIChatMessage(ctx, session.ID, "user", "有没有未看的？", "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AppendAIChatMessage(ctx, session.ID, "assistant", "有 3 部", "", ""); err != nil {
		t.Fatal(err)
	}
	msgs, err := store.ListAIChatMessages(ctx, session.ID, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 || msgs[0].Seq != 1 || msgs[1].Role != "assistant" {
		t.Fatalf("messages = %+v", msgs)
	}
	if err := store.InsertAIToolInvocation(ctx, "chat", session.ID, "search_movies", "read", `{"q":"x"}`, "ok", "", 12); err != nil {
		t.Fatal(err)
	}
	listed, err := store.ListAIChatSessions(ctx, 10)
	if err != nil || len(listed) != 1 {
		t.Fatalf("list = %+v err=%v", listed, err)
	}
	if err := store.DeleteAIChatSession(ctx, session.ID); err != nil {
		t.Fatal(err)
	}
	listed, err = store.ListAIChatSessions(ctx, 10)
	if err != nil || len(listed) != 0 {
		t.Fatalf("after delete = %+v err=%v", listed, err)
	}
}

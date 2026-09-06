package storage

import (
	"context"
	"errors"
	"fmt"
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

func TestAIChatRecentContextSurvivesToolHeavyHistory(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	session, err := store.CreateAIChatSession(ctx, "long session")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 90; i++ {
		for _, role := range []string{"user", "tool", "tool", "assistant"} {
			if _, err := store.AppendAIChatMessage(ctx, session.ID, role, fmt.Sprintf("%s-%d", role, i), "", ""); err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err := store.AppendAIChatMessage(ctx, session.ID, "user", "latest question", "", ""); err != nil {
		t.Fatal(err)
	}
	history, err := store.ListAIChatMessages(ctx, session.ID, 80)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 80 || history[0].Seq != 282 || history[79].Content != "latest question" {
		t.Fatalf("wrong recent page: %+v", history)
	}
	window, err := store.ListAIChatContext(ctx, session.ID, 80)
	if err != nil {
		t.Fatal(err)
	}
	if len(window) != 80 || window[0].Content != "assistant-50" || window[79].Content != "latest question" {
		t.Fatalf("wrong model window: %+v", window)
	}
	for i, row := range window {
		if row.Role == "tool" || (i > 0 && row.Seq <= window[i-1].Seq) {
			t.Fatalf("invalid context order/role: %+v", row)
		}
	}
	page, cursor, err := store.ListAIChatMessagePage(ctx, session.ID, "")
	if err != nil || len(page) != 80 || cursor == "" {
		t.Fatalf("first page: %d %s %v", len(page), cursor, err)
	}
	seen := map[string]bool{}
	for _, item := range page {
		seen[item.ID] = true
	}
	// New messages cannot move a keyset cursor or create duplicate old rows.
	if _, err := store.AppendAIChatMessage(ctx, session.ID, "user", "arrived after page one", "", ""); err != nil {
		t.Fatal(err)
	}
	for cursor != "" {
		page, cursor, err = store.ListAIChatMessagePage(ctx, session.ID, cursor)
		if err != nil {
			t.Fatal(err)
		}
		for _, item := range page {
			if seen[item.ID] {
				t.Fatalf("duplicate %s", item.ID)
			}
			seen[item.ID] = true
		}
	}
	if len(seen) != 361 {
		t.Fatalf("lost messages: %d", len(seen))
	}
	if _, _, err := store.ListAIChatMessagePage(ctx, session.ID, "invalid"); !errors.Is(err, ErrInvalidAIChatCursor) {
		t.Fatalf("invalid cursor: %v", err)
	}
}

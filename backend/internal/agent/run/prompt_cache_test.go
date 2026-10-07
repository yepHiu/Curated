package run

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"curated-backend/internal/agent/prompts"
	"curated-backend/internal/contracts"
	"curated-backend/internal/llm"
)

func TestPageChangesPreserveRulesAndHistoryPrefix(t *testing.T) {
	history := []llm.ChatMessage{{Role: "user", Content: "old question"}, {Role: "assistant", Content: "old answer"}, {Role: "user", Content: "this one?"}}
	a := buildMessages(history, &contracts.AIChatContext{MovieID: "movie-a"}, "zh-CN")
	b := buildMessages(history, &contracts.AIChatContext{MovieID: "movie-b"}, "zh-CN")
	if len(a) != 5 || !reflect.DeepEqual(a[:3], b[:3]) || !reflect.DeepEqual(a[1:3], history[:2]) {
		t.Fatalf("page invalidates rules/history prefix: %+v / %+v", a, b)
	}
	if a[3].Role != "system" || !strings.Contains(a[3].Content, "movieId=movie-a") || !strings.Contains(b[3].Content, "movieId=movie-b") || !reflect.DeepEqual(a[4], history[2]) {
		t.Fatal("latest page/input missing")
	}
	cleared := buildMessages(history, nil, "zh-CN")
	if len(cleared) != 4 || !reflect.DeepEqual(cleared[:3], a[:3]) || !reflect.DeepEqual(cleared[3], history[2]) {
		t.Fatal("clearing context leaked a previous selection")
	}
}

func TestHistoryOmissionDoesNotChangeBasePrompt(t *testing.T) {
	history := []llm.ChatMessage{{Role: "user", Content: "old"}, {Role: "assistant", Content: "answer"}, {Role: "user", Content: "latest"}}
	out := buildMessagesWithBudget(history, nil, "en", ContextBudget{History: 20, Messages: 1})
	if len(out) != 3 || out[0].Content != prompts.SystemPrompt("en") || !strings.Contains(out[1].Content, prompts.HistoryOmittedPrompt()) || out[2].Content != "latest" {
		t.Fatalf("omission changed reusable base or lost latest input: %+v", out)
	}
}

func TestCompactionPreservesSeparatePageContext(t *testing.T) {
	history := []llm.ChatMessage{{Role: "user", Content: strings.Repeat("old ", 8000)}, {Role: "assistant", Content: "old answer"}, {Role: "user", Content: "this one?"}}
	messages := buildMessagesWithBudget(history, &contracts.AIChatContext{PhotoID: "photo-1", Route: "photo-detail", MovieID: "wrong-movie"}, "en", ContextBudget{History: 100000, Messages: 24})
	l := &Loop{streamer: &llm.ScriptedStreamer{Turns: []llm.AssistantTurn{{Content: "old task summary"}}}}
	out, changed, _ := l.compactWorkingContext(context.Background(), messages, nil)
	if !changed || out[0].Content != messages[0].Content || out[1].Role != "system" || !strings.Contains(out[1].Content, "photoId=photo-1") || strings.Contains(out[1].Content, "wrong-movie") || out[len(out)-1].Content != "this one?" {
		t.Fatalf("compaction lost rules or current page context: %+v", out)
	}
}

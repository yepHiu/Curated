package run

import (
	"curated-backend/internal/agent/core"
	"curated-backend/internal/llm"
	"strings"
	"testing"
)

func TestHistoryBudgetPreservesLatestInputWithoutSplittingUnicode(t *testing.T) {
	history := []llm.ChatMessage{}
	for i := 0; i < 30; i++ {
		history = append(history, llm.ChatMessage{Role: "user", Content: strings.Repeat("旧", 3000)}, llm.ChatMessage{Role: "assistant", Content: strings.Repeat("答", 3000)})
	}
	history = append(history, llm.ChatMessage{Role: "user", Content: "latest question"})
	window, omitted := boundedHistory(history)
	if !omitted || window[len(window)-1].Content != "latest question" || window[0].Role != "user" {
		t.Fatalf("window: %+v", window)
	}
	size := 0
	for _, message := range window {
		size += len(message.Content) + 8
	}
	if size > historyTextBudget {
		t.Fatalf("unbounded history: %d", size)
	}
	large := strings.Repeat("新", historyTextBudget)
	window, _ = boundedHistory(append(history, llm.ChatMessage{Role: "user", Content: large}))
	if len(window) != 1 || window[0].Content != large {
		t.Fatal("latest input was silently truncated")
	}
	if estimatedRequestTokens(window, nil) <= requestTokenEstimateBudget {
		t.Fatal("oversized input not detected")
	}
}

func TestOversizedRequestStopsBeforeProviderCall(t *testing.T) {
	streamer := &llm.ScriptedStreamer{Turns: []llm.AssistantTurn{{Content: "should not be called"}}}
	loop := NewLoop(core.NewGateway(core.NewRegistry(), nil, nil, nil), streamer, core.SanitizeFull, "en")
	events := collectEvents(t, loop, []llm.ChatMessage{{Role: "user", Content: strings.Repeat("a", requestTokenEstimateBudget+1)}})
	last := events[len(events)-1]
	if last.Outcome == nil || last.Outcome.Status != "needs_input" {
		t.Fatalf("budget not enforced: %+v", last)
	}
	for _, event := range events {
		if event.Type == "text_delta" {
			t.Fatal("provider was called")
		}
	}
}

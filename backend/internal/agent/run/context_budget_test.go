package run

import (
	"context"
	"curated-backend/internal/agent/core"
	"curated-backend/internal/llm"
	"strings"
	"testing"
)

func TestConfiguredContextChangesAdmissionAndOutputReserve(t *testing.T) {
	for _, window := range []int{32768, 65536, 204800, 1000000, 2097152} {
		budget := BudgetForContext(window)
		if budget.Input+budget.Output+window/20 != window || budget.Trigger >= budget.Input || budget.SummaryInput+2048 > budget.Input {
			t.Fatalf("invalid budget %+v for %d", budget, window)
		}
	}
	for _, window := range []int{32768, 204800} {
		calls := 0
		streamer := memoryStreamer(func(_ context.Context, req llm.TurnRequest) (llm.AssistantTurn, error) {
			calls++
			if req.MaxTokens != BudgetForContext(window).Output {
				t.Fatal("output reserve not sent")
			}
			return llm.AssistantTurn{Content: "Completed."}, nil
		})
		loop := NewLoop(core.NewGateway(core.NewRegistry(), nil, nil, nil), streamer, core.SanitizeFull, "en").WithContextWindow(window)
		events := collectEvents(t, loop, []llm.ChatMessage{{Role: "user", Content: strings.Repeat("x", 70000)}})
		if window == 32768 && (calls != 0 || events[len(events)-1].Outcome.ReasonCode != "context_input_too_large") {
			t.Fatal("small context not enforced")
		}
		if window == 204800 && (calls != 1 || events[len(events)-1].Outcome.Status != "completed") {
			t.Fatal("large context still using fixed 64KiB limit")
		}
	}
}

func TestConfiguredHistoryScalesWithModelCapacity(t *testing.T) {
	var history []llm.ChatMessage
	for i := 0; i < 100; i++ {
		history = append(history, llm.ChatMessage{Role: "user", Content: strings.Repeat("字", 1000)})
	}
	large := RecentHistory(history, 1000000)
	small := RecentHistory(history, 32768)
	if len(large) <= 24 || len(small) >= len(large) || small[len(small)-1].Content != history[len(history)-1].Content {
		t.Fatalf("small=%d large=%d", len(small), len(large))
	}
}

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
	if estimatedRequestBytes(window, nil) <= (64 * 1024) {
		t.Fatal("oversized input not detected")
	}
}

func TestOversizedRequestStopsBeforeProviderCall(t *testing.T) {
	streamer := &llm.ScriptedStreamer{Turns: []llm.AssistantTurn{{Content: "should not be called"}}}
	loop := NewLoop(core.NewGateway(core.NewRegistry(), nil, nil, nil), streamer, core.SanitizeFull, "en")
	events := collectEvents(t, loop, []llm.ChatMessage{{Role: "user", Content: strings.Repeat("a", (64*1024)+1)}})
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

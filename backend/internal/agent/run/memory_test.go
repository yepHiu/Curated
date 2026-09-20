package run

import (
	"context"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"curated-backend/internal/agent/core"
	"curated-backend/internal/llm"
)

type memoryStreamer func(context.Context, llm.TurnRequest) (llm.AssistantTurn, error)

func (f memoryStreamer) StreamTurn(ctx context.Context, req llm.TurnRequest, _ func(string)) (llm.AssistantTurn, error) {
	return f(ctx, req)
}

func TestCompactionContinuesAfterLargeToolResult(t *testing.T) {
	reg := core.NewRegistry()
	if err := reg.Register(core.ToolDefinition{Name: "large_read", Permission: core.PermissionRead, Domain: core.DomainQuery, ParamsSchema: objectSchema(nil), Handler: func(context.Context, core.Call) (core.Result, error) {
		return core.Result{OK: true, Data: map[string]any{"text": strings.Repeat("evidence ", 12000)}}, nil
	}}); err != nil {
		t.Fatal(err)
	}
	calls, summaries := 0, 0
	streamer := memoryStreamer(func(ctx context.Context, req llm.TurnRequest) (llm.AssistantTurn, error) {
		if req.ToolChoice == "none" {
			summaries++
			if len(req.Tools) != 0 || estimatedRequestBytes(req.Messages, nil) > 32*1024 {
				t.Fatal("unbounded or executable summary")
			}
			return llm.AssistantTurn{Content: "Goal: explain results. Completed: large_read. Next: answer."}, nil
		}
		calls++
		if calls == 1 {
			return llm.AssistantTurn{ToolCalls: []llm.ToolCall{{ID: "large", Function: llm.ToolCallFunction{Name: "large_read", Arguments: `{}`}}}}, nil
		}
		if estimatedRequestBytes(req.Messages, req.Tools) > (64 * 1024) {
			t.Fatal("request still over budget")
		}
		if req.Messages[len(req.Messages)-1].Content != "explain results" {
			t.Fatal("latest input changed")
		}
		return llm.AssistantTurn{Content: "Finished the requested explanation."}, nil
	})
	loop := NewLoop(core.NewGateway(reg, nil, nil, nil), streamer, core.SanitizeFull, "en")
	events := collectEvents(t, loop, []llm.ChatMessage{{Role: "user", Content: "explain results"}})
	if calls != 2 || summaries != 1 || events[len(events)-1].Outcome.Status != "completed" {
		t.Fatalf("calls=%d summaries=%d events=%+v", calls, summaries, events)
	}
}

func TestCompactionPreservesLatestToolBatchAndSystemRules(t *testing.T) {
	messages := []llm.ChatMessage{{Role: "system", Content: "rules"}, {Role: "user", Content: strings.Repeat("old", 10000)}, {Role: "assistant", Content: "old answer"}, {Role: "user", Content: "当前完整输入🙂"},
		{Role: "assistant", ToolCalls: []llm.ToolCall{{ID: "a", Function: llm.ToolCallFunction{Name: "read", Arguments: `{}`}}, {ID: "b", Function: llm.ToolCallFunction{Name: "read", Arguments: `{}`}}}},
		{Role: "tool", ToolCallID: "a", Content: `{"ok":true}`}, {Role: "tool", ToolCallID: "b", Content: `{"ok":false}`}, {Role: "system", Content: "repair: no more writes"}}
	loop := &Loop{streamer: &llm.ScriptedStreamer{Turns: []llm.AssistantTurn{{Content: "Goal and constraints"}}}}
	out, changed, _ := loop.compactWorkingContext(context.Background(), messages, nil)
	if !changed || len(out) != 7 || out[0].Content != "rules" || out[1].Content != "repair: no more writes" || out[3].Content != "当前完整输入🙂" || len(out[4].ToolCalls) != 2 || out[5].ToolCallID != "a" || out[6].ToolCallID != "b" {
		t.Fatalf("broken conversation: %+v", out)
	}
	if !strings.Contains(messages[1].Content, "oldold") {
		t.Fatal("mutated original transcript")
	}
}

func TestCompactionFailureFallbackAndCancellation(t *testing.T) {
	messages := []llm.ChatMessage{{Role: "system", Content: "rules"}, {Role: "user", Content: strings.Repeat("old", 20000)}, {Role: "user", Content: "latest"}}
	loop := &Loop{streamer: memoryStreamer(func(context.Context, llm.TurnRequest) (llm.AssistantTurn, error) {
		return llm.AssistantTurn{}, errors.New("offline")
	})}
	out, changed, degraded := loop.compactWorkingContext(context.Background(), messages, nil)
	if !changed || !degraded || out[len(out)-1].Content != "latest" || estimatedRequestBytes(out, nil) > 16*1024 {
		t.Fatal("fallback failed")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, changed, _ = loop.compactWorkingContext(ctx, messages, nil)
	if changed {
		t.Fatal("cancelled compaction committed")
	}
	text := MemoryExcerpt(strings.Repeat("字🙂", 100), 91)
	if !utf8.ValidString(text) || !strings.Contains(text, "omitted") {
		t.Fatal("invalid Unicode excerpt")
	}
}

func TestProviderOverflowRetriesOnceWithoutReplayingTools(t *testing.T) {
	calls, summaries := 0, 0
	loop := NewLoop(core.NewGateway(core.NewRegistry(), nil, nil, nil), memoryStreamer(func(_ context.Context, req llm.TurnRequest) (llm.AssistantTurn, error) {
		if req.ToolChoice == "none" {
			summaries++
			return llm.AssistantTurn{Content: "previous goal"}, nil
		}
		calls++
		return llm.AssistantTurn{}, &llm.HTTPError{Status: 400, Detail: "context_length_exceeded"}
	}), core.SanitizeFull, "en")
	events := collectEvents(t, loop, []llm.ChatMessage{{Role: "user", Content: strings.Repeat("older context ", 800)}, {Role: "assistant", Content: "ok"}, {Role: "user", Content: "continue"}})
	if calls != 2 || summaries != 1 || events[len(events)-1].Outcome.ReasonCode != "context_input_too_large" {
		t.Fatalf("unbounded retries: %d/%d %+v", calls, summaries, events)
	}
}

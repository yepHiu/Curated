package run

import (
	"context"
	"fmt"
	"testing"

	"curated-backend/internal/agent/core"
	"curated-backend/internal/llm"
)

func TestLoopUnlimitedContinuesBeyondLegacyLimit(t *testing.T) {
	reg := core.NewRegistry()
	hits := 0
	if err := reg.Register(core.ToolDefinition{
		Name: "get_ping", Description: "ping", ParamsSchema: objectSchema(nil),
		Permission: core.PermissionRead, Domain: core.DomainQuery,
		Handler: func(context.Context, core.Call) (core.Result, error) {
			hits++
			return core.Result{OK: true, Data: hits}, nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	turns := make([]llm.AssistantTurn, 0, 21)
	for i := 0; i < 20; i++ {
		turns = append(turns, llm.AssistantTurn{ToolCalls: []llm.ToolCall{{
			ID: fmt.Sprintf("call_%d", i), Function: llm.ToolCallFunction{Name: "get_ping", Arguments: `{}`},
		}}})
	}
	turns = append(turns, llm.AssistantTurn{Content: "Finished all twenty checks."})
	loop := NewLoop(core.NewGateway(reg, nil, nil, nil), &llm.ScriptedStreamer{Turns: turns}, core.SanitizeFull, "en")
	events := collectEvents(t, loop, []llm.ChatMessage{{Role: "user", Content: "Run twenty checks"}})
	if hits != 20 {
		t.Fatalf("executed %d tools, want 20", hits)
	}
	if outcome := events[len(events)-1].Outcome; outcome == nil || outcome.Status != "completed" {
		t.Fatalf("outcome = %+v", outcome)
	}
}

type limitStreamer struct {
	t     *testing.T
	calls int
}

func (s *limitStreamer) StreamTurn(_ context.Context, _ llm.TurnRequest, _ func(string)) (llm.AssistantTurn, error) {
	s.calls++
	if s.calls > 1 {
		s.t.Fatal("limit must stop before sending a batch with unanswered tool calls to the model")
	}
	return llm.AssistantTurn{ToolCalls: []llm.ToolCall{
		{ID: "one", Function: llm.ToolCallFunction{Name: "get_ping", Arguments: `{}`}},
		{ID: "two", Function: llm.ToolCallFunction{Name: "get_ping", Arguments: `{}`}},
	}}, nil
}

func TestLoopLimitStopsMidBatchWithExplicitPartialOutcome(t *testing.T) {
	reg := core.NewRegistry()
	hits := 0
	if err := reg.Register(core.ToolDefinition{
		Name: "get_ping", Description: "ping", ParamsSchema: objectSchema(nil),
		Permission: core.PermissionRead, Domain: core.DomainQuery,
		Handler: func(context.Context, core.Call) (core.Result, error) {
			hits++
			return core.Result{OK: true}, nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	streamer := &limitStreamer{t: t}
	loop := NewLoop(core.NewGateway(reg, nil, nil, func() core.Settings { return core.Settings{StepLimit: 1} }), streamer, core.SanitizeFull, "en")
	events := collectEvents(t, loop, []llm.ChatMessage{{Role: "user", Content: "Run two checks"}})
	if hits != 1 || streamer.calls != 1 {
		t.Fatalf("tool calls %d; model calls %d", hits, streamer.calls)
	}
	if outcome := events[len(events)-1].Outcome; outcome == nil || outcome.Status != "partial" || outcome.ReasonCode != "tool_step_limit" {
		t.Fatalf("outcome = %+v", outcome)
	}
}

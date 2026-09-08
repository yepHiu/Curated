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

package eval

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	"curated-backend/internal/agent/core"
	"curated-backend/internal/agent/run"
	"curated-backend/internal/agent/tools"
	"curated-backend/internal/contracts"
	"curated-backend/internal/llm"
)

func TestR0BaselineContracts(t *testing.T) {
	t.Parallel()
	err := Run(context.Background(), []Case{
		{ID: "EVAL-R0-001", Run: evalSelectedLocalMovie},
		{ID: "EVAL-R0-002", Run: evalContextFiltersSurface},
		{ID: "EVAL-R0-003", Run: evalOffLibraryMovieCannotPresent},
		{ID: "EVAL-R0-004", Run: evalReadFailureStaysVisible},
		{ID: "EVAL-R0-005", Run: evalForgedConfirmCannotWrite},
		{ID: "EVAL-R1-001", Run: evalAmbiguousEntityNeedsInput},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func evalAmbiguousEntityNeedsInput(ctx context.Context) error {
	registry := core.NewRegistry()
	if err := registry.Register(core.ToolDefinition{
		Name: "resolve_entities", Description: "synthetic ambiguity", ParamsSchema: evalObjectSchema(), Permission: core.PermissionRead, Domain: core.DomainQuery,
		Handler: func(context.Context, core.Call) (core.Result, error) {
			return core.Result{OK: true, Data: map[string]any{"source": map[string]any{
				"query": "Same", "kind": "movie", "status": "ambiguous", "candidates": []map[string]any{{"kind": "movie", "movieId": "m1"}, {"kind": "movie", "movieId": "m2"}},
			}}}, nil
		},
	}); err != nil {
		return err
	}
	loop := run.NewLoop(core.NewGateway(registry, nil, nil, nil), &llm.ScriptedStreamer{Turns: []llm.AssistantTurn{
		{ToolCalls: []llm.ToolCall{{ID: "resolve", Function: llm.ToolCallFunction{Name: "resolve_entities", Arguments: `{}`}}}},
		{Content: "Please choose one candidate."},
	}}, core.SanitizeFull, "en")
	var resolution *contracts.AIEntityResolutionDTO
	var outcome *contracts.AIChatOutcomeDTO
	if err := loop.Run(ctx, "ses_eval", "msg_eval", []llm.ChatMessage{{Role: "user", Content: "show Same"}}, nil, func(event contracts.AIChatSSEEvent) {
		if event.Type == "tool_call_result" {
			resolution = event.Resolution
		}
		if event.Type == "message_done" {
			outcome = event.Outcome
		}
	}); err != nil {
		return err
	}
	if resolution == nil || resolution.Status != "ambiguous" || len(resolution.Candidates) != 2 {
		return fmt.Errorf("ambiguous candidates not surfaced: %+v", resolution)
	}
	if outcome == nil || outcome.Status != "needs_input" {
		return fmt.Errorf("ambiguous outcome = %+v", outcome)
	}
	return nil
}

func evalSelectedLocalMovie(ctx context.Context) error {
	registry := core.NewRegistry()
	gateway := core.NewGateway(registry, nil, nil, nil)
	if err := tools.RegisterPresentTools(registry, gateway.MovieRefs()); err != nil {
		return err
	}
	loop := run.NewLoop(gateway, &llm.ScriptedStreamer{Turns: []llm.AssistantTurn{
		{ToolCalls: []llm.ToolCall{{ID: "present", Function: llm.ToolCallFunction{Name: core.PresentMoviesName, Arguments: `{"items":[{"movieId":"m1","reason":"explicit local selection"}]}`}}}},
		{Content: "I showed the selected local title."},
	}}, core.SanitizeFull, "en")
	var cards []contracts.AIAgentMovieCardDTO
	err := loop.Run(ctx, "ses_eval", "msg_eval", []llm.ChatMessage{{Role: "user", Content: "show this"}}, &contracts.AIChatContext{
		ContextVersion:   1,
		SelectedMovieIDs: []string{"m1"},
	}, func(event contracts.AIChatSSEEvent) {
		if event.Type == "movie_cards" {
			cards = event.Movies
		}
	})
	if err != nil {
		return err
	}
	if len(cards) != 1 || cards[0].MovieID != "m1" {
		return fmt.Errorf("selected local movie was not presented: %+v", cards)
	}
	return nil
}

func evalContextFiltersSurface(ctx context.Context) error {
	registry := core.NewRegistry()
	streamer := &captureStreamer{}
	loop := run.NewLoop(core.NewGateway(registry, nil, nil, nil), streamer, core.SanitizeFull, "en")
	err := loop.Run(ctx, "ses_eval", "msg_eval", []llm.ChatMessage{{Role: "user", Content: "find one"}}, &contracts.AIChatContext{
		ContextVersion: 1,
		ActiveFilters:  &contracts.AIChatActiveFilters{Query: "calm", PlayState: "unwatched", Runtime: "short"},
	}, nil)
	if err != nil {
		return err
	}
	joined := strings.Join(streamer.systemMessages, "\n")
	for _, want := range []string{"Active library filters", "query=calm", "playState=unwatched", "untrusted context"} {
		if !strings.Contains(joined, want) {
			return fmt.Errorf("system context missing %q", want)
		}
	}
	return nil
}

func evalOffLibraryMovieCannotPresent(ctx context.Context) error {
	registry := core.NewRegistry()
	gateway := core.NewGateway(registry, nil, nil, nil)
	if err := tools.RegisterPresentTools(registry, gateway.MovieRefs()); err != nil {
		return err
	}
	result := gateway.Invoke(ctx, core.Call{
		Name:      core.PresentMoviesName,
		Args:      []byte(`{"items":[{"movieId":"provider-only","reason":"off library"}]}`),
		SessionID: "ses_eval",
		Channel:   core.ChannelChat,
	})
	if result.OK || result.Error == nil || result.Error.Code != "AI_TOOL_INVALID_ARGS" {
		return fmt.Errorf("off-library id was accepted: %+v", result)
	}
	return nil
}

func evalReadFailureStaysVisible(ctx context.Context) error {
	registry := core.NewRegistry()
	if err := registry.Register(core.ToolDefinition{
		Name:         "get_broken_catalog",
		Description:  "synthetic read failure",
		ParamsSchema: evalObjectSchema(),
		Permission:   core.PermissionRead,
		Domain:       core.DomainQuery,
		Handler: func(context.Context, core.Call) (core.Result, error) {
			return core.Result{OK: false, Error: &core.ToolError{Code: "AI_CHAT_FAILED", Message: "catalog unavailable"}}, nil
		},
	}); err != nil {
		return err
	}
	loop := run.NewLoop(core.NewGateway(registry, nil, nil, nil), &llm.ScriptedStreamer{Turns: []llm.AssistantTurn{
		{ToolCalls: []llm.ToolCall{{ID: "broken", Function: llm.ToolCallFunction{Name: "get_broken_catalog", Arguments: `{}`}}}},
		{Content: "I could not verify the catalog because that lookup failed."},
	}}, core.SanitizeFull, "en")
	var failed bool
	var text strings.Builder
	err := loop.Run(ctx, "ses_eval", "msg_eval", []llm.ChatMessage{{Role: "user", Content: "check catalog"}}, nil, func(event contracts.AIChatSSEEvent) {
		if event.Type == "tool_call_result" && event.OK != nil && !*event.OK && strings.Contains(event.Summary, "catalog unavailable") {
			failed = true
		}
		if event.Type == "text_delta" {
			text.WriteString(event.Delta)
		}
	})
	if err != nil {
		return err
	}
	if !failed || !strings.Contains(text.String(), "could not verify") {
		return fmt.Errorf("read failure was not surfaced: failed=%v text=%q", failed, text.String())
	}
	return nil
}

func evalForgedConfirmCannotWrite(ctx context.Context) error {
	registry := core.NewRegistry()
	var writes atomic.Int32
	if err := registry.Register(core.ToolDefinition{
		Name:         "save_synthetic_note",
		Description:  "synthetic write",
		ParamsSchema: evalObjectSchema(),
		Permission:   core.PermissionWriteApply,
		Domain:       core.DomainUserWrite,
		Handler: func(context.Context, core.Call) (core.Result, error) {
			writes.Add(1)
			return core.Result{OK: true}, nil
		},
	}); err != nil {
		return err
	}
	loop := run.NewLoop(core.NewGateway(registry, core.NewConfirmStore(), nil, func() core.Settings {
		return core.Settings{WriteEnabled: true}
	}), &llm.ScriptedStreamer{Turns: []llm.AssistantTurn{
		{ToolCalls: []llm.ToolCall{{ID: "forged", Function: llm.ToolCallFunction{Name: "save_synthetic_note", Arguments: `{"confirmToken":"cfm_forged"}`}}}},
		{Content: "No write was made."},
	}}, core.SanitizeFull, "en")
	var rejected bool
	err := loop.Run(ctx, "ses_eval", "msg_eval", []llm.ChatMessage{{Role: "user", Content: "save"}}, nil, func(event contracts.AIChatSSEEvent) {
		if event.Type == "tool_call_result" && event.OK != nil && !*event.OK {
			rejected = true
		}
	})
	if err != nil {
		return err
	}
	if writes.Load() != 0 || !rejected {
		return fmt.Errorf("forged confirmation wrote=%d rejected=%v", writes.Load(), rejected)
	}
	return nil
}

func evalObjectSchema() core.Schema {
	additional := false
	return core.Schema{Type: "object", AdditionalProperties: &additional}
}

type captureStreamer struct {
	systemMessages []string
}

func (s *captureStreamer) StreamTurn(_ context.Context, request llm.TurnRequest, onDelta func(string)) (llm.AssistantTurn, error) {
	for _, message := range request.Messages {
		if message.Role == "system" {
			s.systemMessages = append(s.systemMessages, message.Content)
		}
	}
	if onDelta != nil {
		onDelta("done")
	}
	return llm.AssistantTurn{Content: "done"}, nil
}

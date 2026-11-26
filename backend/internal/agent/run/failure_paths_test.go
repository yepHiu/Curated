package run

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"curated-backend/internal/agent/core"
	"curated-backend/internal/agent/tools"
	"curated-backend/internal/contracts"
	"curated-backend/internal/llm"
)

// failingStreamer 用假模型返回指定错误，不访问网络。
type failingStreamer struct {
	err error
}

// StreamTurn 直接返回预设错误，用来覆盖超时、配置和坏流。
func (s failingStreamer) StreamTurn(context.Context, llm.TurnRequest, func(string)) (llm.AssistantTurn, error) {
	return llm.AssistantTurn{}, s.err
}

// TestLoopClassifiesProviderFailuresWithoutLeakingProviderText 确认超时、缺配置、鉴权、限流和坏流都有稳定原因码。
func TestLoopClassifiesProviderFailuresWithoutLeakingProviderText(t *testing.T) {
	t.Parallel()
	secret := "sk-live-secret-must-not-appear"
	cases := []struct {
		name string
		err  error
		code string
	}{
		{name: "timeout", err: context.DeadlineExceeded, code: "provider_timeout"},
		{name: "missing model", err: fmt.Errorf("%w: model is required", llm.ErrInvalidConfig), code: "provider_unconfigured"},
		{name: "rejected key", err: &llm.HTTPError{Status: 401, Detail: "invalid api key " + secret}, code: "provider_authentication"},
		{name: "rate limit", err: &llm.HTTPError{Status: 429, Detail: "slow down " + secret}, code: "provider_rate_limited"},
		{name: "broken stream", err: errors.New("invalid provider stream: " + secret), code: "provider_stream_invalid"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			loop := NewLoop(core.NewGateway(core.NewRegistry(), nil, nil, nil), failingStreamer{err: tc.err}, core.SanitizeFull, "zh-CN")
			var events []contracts.AIChatSSEEvent
			err := loop.Run(context.Background(), "ses_fail", "msg_fail", []llm.ChatMessage{{Role: "user", Content: "今晚看什么"}}, nil, func(ev contracts.AIChatSSEEvent) {
				events = append(events, ev)
			})
			if err == nil {
				t.Fatal("expected model failure")
			}
			if len(events) == 0 || events[len(events)-1].Outcome == nil {
				t.Fatalf("events = %+v", events)
			}
			outcome := events[len(events)-1].Outcome
			if outcome.Status != "failed" || !outcome.Retryable || outcome.ReasonCode != tc.code {
				t.Fatalf("outcome = %+v", outcome)
			}
			if outcome.Reason == "" || strings.Contains(outcome.Reason, secret) {
				t.Fatalf("reason leaked provider text: %q", outcome.Reason)
			}
		})
	}
}

// TestLoopEmptyAnswerUsesStableReason 确认空回答不会被当成成功。
func TestLoopEmptyAnswerUsesStableReason(t *testing.T) {
	t.Parallel()
	loop := NewLoop(core.NewGateway(core.NewRegistry(), nil, nil, nil), &llm.ScriptedStreamer{Turns: []llm.AssistantTurn{{}}}, core.SanitizeFull, "zh-CN")
	var events []contracts.AIChatSSEEvent
	if err := loop.Run(context.Background(), "ses_empty", "msg_empty", []llm.ChatMessage{{Role: "user", Content: "推荐"}}, nil, func(ev contracts.AIChatSSEEvent) {
		events = append(events, ev)
	}); err != nil {
		t.Fatal(err)
	}
	outcome := events[len(events)-1].Outcome
	if outcome == nil || outcome.Status != "failed" || outcome.ReasonCode != "provider_empty" {
		t.Fatalf("outcome = %+v", outcome)
	}
}

// TestPickerCannotApplyAnUnregisteredWrite 确认选片会话不能调用未注册的写入工具。
func TestPickerCannotApplyAnUnregisteredWrite(t *testing.T) {
	t.Parallel()
	reg := core.NewRegistry()
	if err := tools.RegisterQueryTools(reg, stubQueryForLoop{movie: contracts.MovieListItemDTO{ID: "m1", Title: "Hello", Code: "ABC-123"}}); err != nil {
		t.Fatal(err)
	}
	write := &commentWriteStub{exists: true, body: "original"}
	loop := NewLoop(core.NewGateway(reg, core.NewConfirmStore(), nil, func() core.Settings { return core.Settings{} }), &llm.ScriptedStreamer{Turns: []llm.AssistantTurn{
		{ToolCalls: []llm.ToolCall{{ID: "call_write", Function: llm.ToolCallFunction{Name: core.SaveMovieCommentName, Arguments: `{"movieId":"m1","body":"overwrite"}`}}}},
		{Content: "只展示检索结果"},
	}}, core.SanitizeFull, "zh-CN")
	var events []contracts.AIChatSSEEvent
	if err := loop.Run(context.Background(), "ses_pick", "msg_pick", []llm.ChatMessage{{Role: "user", Content: "今晚看什么"}}, nil, func(ev contracts.AIChatSSEEvent) {
		events = append(events, ev)
	}); err != nil {
		t.Fatal(err)
	}
	if write.writes != 0 {
		t.Fatalf("picker wrote %d times", write.writes)
	}
	var sawRejectedTool bool
	for _, ev := range events {
		if ev.Type == "tool_call_result" && ev.Name == core.SaveMovieCommentName && ev.OK != nil && !*ev.OK {
			sawRejectedTool = true
		}
	}
	if !sawRejectedTool {
		t.Fatalf("missing rejected write tool: %+v", events)
	}
	outcome := events[len(events)-1].Outcome
	if outcome == nil || outcome.Status != "partial" {
		t.Fatalf("outcome = %+v", outcome)
	}
}

package llm

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientConfigValidate(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		cfg     ClientConfig
		wantErr bool
	}{
		{"valid", ClientConfig{BaseURL: "http://127.0.0.1:11434/v1", Model: "qwen3"}, false},
		{"missing model", ClientConfig{BaseURL: "http://127.0.0.1:11434/v1"}, true},
		{"missing base url", ClientConfig{Model: "qwen3"}, true},
		{"bad scheme", ClientConfig{BaseURL: "ftp://example.com", Model: "qwen3"}, true},
		{"no host", ClientConfig{BaseURL: "http://", Model: "qwen3"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.cfg.Validate()
			if tc.wantErr && err == nil {
				t.Fatalf("Validate() = nil, want error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("Validate() = %v, want nil", err)
			}
		})
	}
}

func TestStreamChatAccumulatesDeltasAndIgnoresHeartbeats(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("Authorization = %q, want bearer test-key", got)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(": heartbeat\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"你好\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"，世界\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	t.Cleanup(server.Close)

	var deltas []string
	full, err := NewClient(ClientConfig{
		BaseURL: server.URL + "/v1",
		APIKey:  "test-key",
		Model:   "test-model",
	}, server.Client()).StreamChat(context.Background(), []ChatMessage{
		{Role: "user", Content: "hi"},
	}, func(delta string) { deltas = append(deltas, delta) })
	if err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(deltas, "|"), "你好|，世界"; got != want {
		t.Fatalf("deltas = %q, want %q", got, want)
	}
	if full != "你好，世界" {
		t.Fatalf("full = %q, want 你好，世界", full)
	}
}

func TestStreamTurnForwardsReasoningContent(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"reasoning_content\":\"先查库\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"有结果\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	t.Cleanup(server.Close)

	var thinking []string
	var deltas []string
	turn, err := NewClient(ClientConfig{BaseURL: server.URL, Model: "m"}, server.Client()).
		StreamTurn(context.Background(), TurnRequest{
			Messages: []ChatMessage{{Role: "user", Content: "hi"}},
			OnThinking: func(delta string) {
				thinking = append(thinking, delta)
			},
		}, func(delta string) { deltas = append(deltas, delta) })
	if err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(thinking, ""), "先查库"; got != want {
		t.Fatalf("thinking = %q, want %q", got, want)
	}
	if turn.Content != "有结果" || strings.Join(deltas, "") != "有结果" {
		t.Fatalf("content = %q deltas=%q", turn.Content, deltas)
	}
}

func TestStreamChatSurfacesProviderErrorBody(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid api key"}}`))
	}))
	t.Cleanup(server.Close)

	_, err := NewClient(ClientConfig{BaseURL: server.URL, Model: "m"}, server.Client()).
		StreamChat(context.Background(), []ChatMessage{{Role: "user", Content: "hi"}}, nil)
	if err == nil {
		t.Fatal("StreamChat() = nil error, want provider error")
	}
	if !strings.Contains(err.Error(), "401") || !strings.Contains(err.Error(), "invalid api key") {
		t.Fatalf("StreamChat() error = %v, want HTTP status and body snippet", err)
	}
}

func TestCompleteReturnsFirstChoiceContent(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"pong"}}]}`))
	}))
	t.Cleanup(server.Close)

	got, err := NewClient(ClientConfig{BaseURL: server.URL, Model: "m"}, server.Client()).
		Complete(context.Background(), []ChatMessage{{Role: "user", Content: "ping"}}, 16)
	if err != nil {
		t.Fatal(err)
	}
	if got != "pong" {
		t.Fatalf("Complete() = %q, want pong", got)
	}
}

func TestCompleteOmitsMaxTokensWhenUnlimited(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		if strings.Contains(string(raw), "max_tokens") {
			t.Errorf("request should omit max_tokens, got %s", raw)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	t.Cleanup(server.Close)

	got, err := NewClient(ClientConfig{BaseURL: server.URL, Model: "m"}, server.Client()).
		Complete(context.Background(), []ChatMessage{{Role: "user", Content: "ping"}}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got != "ok" {
		t.Fatalf("Complete() = %q, want ok", got)
	}
}

func TestCompleteReadsMultipartContentAndStripsThink(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":[{"type":"text","text":"<think>先想</think>译名"}]}}]}`))
	}))
	t.Cleanup(server.Close)

	got, err := NewClient(ClientConfig{BaseURL: server.URL, Model: "m"}, server.Client()).
		Complete(context.Background(), []ChatMessage{{Role: "user", Content: "ping"}}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got != "译名" {
		t.Fatalf("Complete() = %q, want 译名", got)
	}
}

func TestCompleteUsesReasoningAfterThinkTags(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"","reasoning_content":"<think>内部推理</think>\n中文标题"}}]}`))
	}))
	t.Cleanup(server.Close)

	got, err := NewClient(ClientConfig{BaseURL: server.URL, Model: "m"}, server.Client()).
		Complete(context.Background(), []ChatMessage{{Role: "user", Content: "ping"}}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got != "中文标题" {
		t.Fatalf("Complete() = %q, want 中文标题", got)
	}
}

func TestCompleteEmptyReasoningOnlyIsError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"finish_reason":"stop","message":{"content":"","reasoning_content":"只是在想题目"}}]}`))
	}))
	t.Cleanup(server.Close)

	_, err := NewClient(ClientConfig{BaseURL: server.URL, Model: "m"}, server.Client()).
		Complete(context.Background(), []ChatMessage{{Role: "user", Content: "ping"}}, 0)
	if err == nil || !strings.Contains(err.Error(), "reasoning only") {
		t.Fatalf("Complete() error = %v, want reasoning only", err)
	}
}

func TestStreamChatRejectsInvalidConfigBeforeRequest(t *testing.T) {
	t.Parallel()
	_, err := NewClient(ClientConfig{BaseURL: "", Model: ""}, http.DefaultClient).
		StreamChat(context.Background(), []ChatMessage{{Role: "user", Content: "hi"}}, nil)
	if err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("StreamChat() = %v, want invalid config error", err)
	}
}

func TestStreamTurnAccumulatesToolCallDeltas(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_1\",\"type\":\"function\",\"function\":{\"name\":\"search_movies\",\"arguments\":\"{\\\"q\\\"\"}}]}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\":\\\"ABC\\\"}\"}}]}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	t.Cleanup(server.Close)

	turn, err := NewClient(ClientConfig{BaseURL: server.URL, Model: "m"}, server.Client()).
		StreamTurn(context.Background(), TurnRequest{
			Messages: []ChatMessage{{Role: "user", Content: "find ABC"}},
			Tools:    []ToolSpec{{Name: "search_movies", Parameters: map[string]any{"type": "object"}}},
		}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(turn.ToolCalls) != 1 {
		t.Fatalf("tool calls = %+v", turn.ToolCalls)
	}
	if turn.ToolCalls[0].Name() != "search_movies" || turn.ToolCalls[0].Args() != `{"q":"ABC"}` {
		t.Fatalf("call = %+v", turn.ToolCalls[0])
	}
}

// Package llm provides a minimal OpenAI-compatible chat completions client with
// SSE streaming for the experimental Curated agent. E1 scope: plain chat only,
// no tool calls. When tool calling lands (charter B5 full), evaluate adopting the
// official openai-go SDK instead of extending this file.
package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// ChatMessage is one OpenAI-compatible chat message (system | user | assistant | tool).
type ChatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// ToolSpec is one function tool advertised to the model.
type ToolSpec struct {
	Name        string
	Description string
	Parameters  map[string]any
}

// ToolCall is one model-requested function invocation.
type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type,omitempty"`
	Function ToolCallFunction `json:"function"`
}

type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

func (c ToolCall) Name() string { return c.Function.Name }
func (c ToolCall) Args() string { return c.Function.Arguments }

// AssistantTurn is one model output: text and/or tool calls.
type AssistantTurn struct {
	Content   string
	ToolCalls []ToolCall
}

// TurnRequest is one chat-completions round, optionally with tools.
type TurnRequest struct {
	Messages   []ChatMessage
	Tools      []ToolSpec
	ToolChoice string
	MaxTokens  int
	OnThinking func(string)
}

// ClientConfig describes one OpenAI-compatible endpoint.
type ClientConfig struct {
	// BaseURL is the API root, e.g. http://127.0.0.1:11434/v1.
	BaseURL string
	// APIKey is sent as a bearer token; empty for local servers without auth.
	APIKey string
	// Model is the chat completions model name.
	Model string
}

// ErrInvalidConfig is returned when BaseURL or Model is missing or malformed.
var ErrInvalidConfig = errors.New("llm provider config invalid")

// Validate reports whether cfg has a usable BaseURL and Model.
func (c ClientConfig) Validate() error {
	if strings.TrimSpace(c.Model) == "" {
		return fmt.Errorf("%w: model is required", ErrInvalidConfig)
	}
	raw := strings.TrimSpace(c.BaseURL)
	if raw == "" {
		return fmt.Errorf("%w: baseUrl is required", ErrInvalidConfig)
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("%w: baseUrl must be an http(s) URL", ErrInvalidConfig)
	}
	return nil
}

// Client talks to one OpenAI-compatible endpoint.
type Client struct {
	cfg        ClientConfig
	httpClient *http.Client
}

// NewClient builds a client; httpClient must be non-nil (use a proxy-aware client
// when outbound proxy is configured). The client timeout governs whole requests,
// so streaming callers should pass a client without a timeout and rely on ctx.
func NewClient(cfg ClientConfig, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &Client{cfg: cfg, httpClient: httpClient}
}

type chatCompletionRequest struct {
	Model      string        `json:"model"`
	Messages   []ChatMessage `json:"messages"`
	Stream     bool          `json:"stream,omitempty"`
	MaxTokens  int           `json:"max_tokens,omitempty"`
	Tools      []openAITool  `json:"tools,omitempty"`
	ToolChoice any           `json:"tool_choice,omitempty"`
	StreamOpts *streamOpts   `json:"stream_options,omitempty"`
}

type openAITool struct {
	Type     string             `json:"type"`
	Function openAIToolFunction `json:"function"`
}

type openAIToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

type streamOpts struct {
	IncludeUsage bool `json:"include_usage"`
}

type chatCompletionResponse struct {
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Message      struct {
			Content   string     `json:"content"`
			ToolCalls []ToolCall `json:"tool_calls"`
		} `json:"message"`
		Delta struct {
			Content          string          `json:"content"`
			ReasoningContent string          `json:"reasoning_content"`
			ToolCalls        []toolCallDelta `json:"tool_calls"`
		} `json:"delta"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

type toolCallDelta struct {
	Index    int    `json:"index"`
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

func (c *Client) endpoint() string {
	return strings.TrimRight(strings.TrimSpace(c.cfg.BaseURL), "/") + "/chat/completions"
}

func (c *Client) newRequest(ctx context.Context, body chatCompletionRequest) (*http.Request, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint(), bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", body.accept())
	if strings.TrimSpace(c.cfg.APIKey) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(c.cfg.APIKey))
	}
	return req, nil
}

func (r chatCompletionRequest) accept() string {
	if r.Stream {
		return "text/event-stream"
	}
	return "application/json"
}

func (c *Client) do(ctx context.Context, body chatCompletionRequest) (*http.Response, error) {
	if err := c.cfg.Validate(); err != nil {
		return nil, err
	}
	req, err := c.newRequest(ctx, body)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		return nil, fmt.Errorf("provider returned HTTP %d: %s", resp.StatusCode, errorSnippet(resp.Body))
	}
	return resp, nil
}

// errorSnippet reads a bounded slice of an error response body for diagnostics.
func errorSnippet(r io.Reader) string {
	b, err := io.ReadAll(io.LimitReader(r, 8*1024))
	if err != nil || len(b) == 0 {
		return "no response body"
	}
	return strings.TrimSpace(string(b))
}

// Complete performs a non-streaming chat completion and returns the first
// choice content. Used by the provider connectivity test.
func (c *Client) Complete(ctx context.Context, messages []ChatMessage, maxTokens int) (string, error) {
	body := chatCompletionRequest{
		Model:     strings.TrimSpace(c.cfg.Model),
		Messages:  messages,
		MaxTokens: maxTokens,
	}
	resp, err := c.do(ctx, body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return "", err
	}
	var parsed chatCompletionResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("invalid provider response: %w", err)
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("provider error: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("provider returned no choices")
	}
	return parsed.Choices[0].Message.Content, nil
}

// StreamChat performs a streaming chat completion. Each content delta is passed
// to onDelta (never called concurrently); the accumulated full text is returned.
func (c *Client) StreamChat(ctx context.Context, messages []ChatMessage, onDelta func(string)) (string, error) {
	turn, err := c.StreamTurn(ctx, TurnRequest{Messages: messages}, onDelta)
	return turn.Content, err
}

// StreamTurn streams one model turn, accumulating text deltas and tool-call fragments.
func (c *Client) StreamTurn(ctx context.Context, req TurnRequest, onDelta func(string)) (AssistantTurn, error) {
	body := chatCompletionRequest{
		Model:     strings.TrimSpace(c.cfg.Model),
		Messages:  req.Messages,
		Stream:    true,
		MaxTokens: req.MaxTokens,
		Tools:     encodeTools(req.Tools),
	}
	if choice := strings.TrimSpace(req.ToolChoice); choice != "" {
		body.ToolChoice = choice
	}
	resp, err := c.do(ctx, body)
	if err != nil {
		return AssistantTurn{}, err
	}
	defer resp.Body.Close()

	var full strings.Builder
	acc := map[int]*ToolCall{}
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var dataLines []string

	processEvent := func() error {
		if len(dataLines) == 0 {
			return nil
		}
		data := strings.TrimSpace(strings.Join(dataLines, "\n"))
		dataLines = dataLines[:0]
		if data == "" || data == "[DONE]" {
			return nil
		}
		var chunk chatCompletionResponse
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return nil
		}
		if chunk.Error != nil {
			return fmt.Errorf("provider error: %s", chunk.Error.Message)
		}
		if len(chunk.Choices) == 0 {
			return nil
		}
		choice := chunk.Choices[0]
		if thinking := choice.Delta.ReasoningContent; thinking != "" && req.OnThinking != nil {
			req.OnThinking(thinking)
		}
		if delta := choice.Delta.Content; delta != "" {
			full.WriteString(delta)
			if onDelta != nil {
				onDelta(delta)
			}
		}
		for _, part := range choice.Delta.ToolCalls {
			slot, ok := acc[part.Index]
			if !ok {
				slot = &ToolCall{Type: "function"}
				acc[part.Index] = slot
			}
			if part.ID != "" {
				slot.ID = part.ID
			}
			if part.Type != "" {
				slot.Type = part.Type
			}
			if part.Function.Name != "" {
				slot.Function.Name = part.Function.Name
			}
			if part.Function.Arguments != "" {
				slot.Function.Arguments += part.Function.Arguments
			}
		}
		if len(choice.Message.ToolCalls) > 0 && len(acc) == 0 {
			for i, call := range choice.Message.ToolCalls {
				copied := call
				if copied.Type == "" {
					copied.Type = "function"
				}
				acc[i] = &copied
			}
		}
		return nil
	}

	for scanner.Scan() {
		if ctx.Err() != nil {
			return assembleTurn(full.String(), acc), ctx.Err()
		}
		line := scanner.Text()
		switch {
		case line == "":
			if err := processEvent(); err != nil {
				return assembleTurn(full.String(), acc), err
			}
		case strings.HasPrefix(line, ":"):
		case strings.HasPrefix(line, "data:"):
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	if err := scanner.Err(); err != nil {
		return assembleTurn(full.String(), acc), err
	}
	if err := processEvent(); err != nil {
		return assembleTurn(full.String(), acc), err
	}
	return assembleTurn(full.String(), acc), nil
}

func encodeTools(specs []ToolSpec) []openAITool {
	if len(specs) == 0 {
		return nil
	}
	out := make([]openAITool, 0, len(specs))
	for _, spec := range specs {
		out = append(out, openAITool{
			Type: "function",
			Function: openAIToolFunction{
				Name:        spec.Name,
				Description: spec.Description,
				Parameters:  spec.Parameters,
			},
		})
	}
	return out
}

func assembleTurn(content string, acc map[int]*ToolCall) AssistantTurn {
	if len(acc) == 0 {
		return AssistantTurn{Content: content}
	}
	indexes := make([]int, 0, len(acc))
	for i := range acc {
		indexes = append(indexes, i)
	}
	for i := 0; i < len(indexes); i++ {
		for j := i + 1; j < len(indexes); j++ {
			if indexes[j] < indexes[i] {
				indexes[i], indexes[j] = indexes[j], indexes[i]
			}
		}
	}
	calls := make([]ToolCall, 0, len(indexes))
	for _, i := range indexes {
		call := *acc[i]
		if call.Type == "" {
			call.Type = "function"
		}
		calls = append(calls, call)
	}
	return AssistantTurn{Content: content, ToolCalls: calls}
}

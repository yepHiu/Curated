package llm

import (
	"context"
	"errors"
	"io"
	"net"
	"time"
)

// Usage contains measured provider counts, never a tokenizer estimate.
// A nil Usage means the provider omitted or returned incomplete counts.
type Usage struct {
	PromptTokens     int64 `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`
	TotalTokens      int64 `json:"total_tokens"`
}

type Observation struct {
	DurationMs  int64
	FirstTextMs *int64 // nil for non-streaming calls or no visible text
	Usage       *Usage
	ErrorCode   string
}

type HTTPError struct {
	Status int
	Detail string
}

func (e *HTTPError) Error() string { return e.Detail }

// ErrorCategory never includes provider response bodies, prompts or URLs.
func ErrorCategory(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	if errors.Is(err, context.Canceled) {
		return "cancelled"
	}
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return "stream_interrupted"
	}
	if errors.Is(err, ErrInvalidConfig) {
		return "configuration"
	}
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		switch httpErr.Status {
		case 401, 403:
			return "authentication"
		case 429:
			return "rate_limit"
		default:
			return "provider_http"
		}
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return "timeout"
		}
		return "network"
	}
	return "invalid_response"
}

func (c *Client) observe(start time.Time, o Observation, err error) {
	if c.Observe == nil {
		return
	}
	o.DurationMs = time.Since(start).Milliseconds()
	o.ErrorCode = ErrorCategory(err)
	c.Observe(o)
}

type wireUsage struct {
	Prompt     *int64 `json:"prompt_tokens"`
	Completion *int64 `json:"completion_tokens"`
	Total      *int64 `json:"total_tokens"`
}

func (u *wireUsage) measured() *Usage {
	if u == nil || u.Prompt == nil || u.Completion == nil || u.Total == nil {
		return nil
	}
	if *u.Prompt < 0 || *u.Completion < 0 || *u.Total < 0 {
		return nil
	}
	return &Usage{*u.Prompt, *u.Completion, *u.Total}
}

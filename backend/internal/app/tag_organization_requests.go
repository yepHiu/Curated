package app

import (
	"context"
	"errors"
	"io"
	"net"
	"time"

	"curated-backend/internal/agent/prompts"
	"curated-backend/internal/llm"
)

// topicComplete retries only transient transport failures. Each attempt retains its
// own timeout and usage observation; cancelling the job also cancels the backoff.
func (a *App) topicComplete(ctx context.Context, prompt prompts.Definition, data any) (string, error) {
	for attempt := 0; ; attempt++ {
		raw, err := a.topicCompleteAttempt(ctx, prompt, data)
		if err == nil || ctx.Err() != nil || attempt >= 2 || !retryableTopicRequest(err) {
			return raw, err
		}
		timer := time.NewTimer(time.Second * time.Duration(1<<attempt))
		select {
		case <-ctx.Done():
			timer.Stop()
			return "", ctx.Err()
		case <-timer.C:
		}
	}
}

func retryableTopicRequest(err error) bool {
	var httpErr *llm.HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.Status == 408 || httpErr.Status == 429 || httpErr.Status >= 500 && httpErr.Status <= 599
	}
	var netErr net.Error
	return errors.Is(err, context.DeadlineExceeded) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) || errors.As(err, &netErr)
}

// Invalid batch output is isolated by reducing the batch, without relaxing evidence checks.
func splittableTopicClassification(err error) bool {
	switch topicOrganizationErrorCode(err) {
	case "AI_CONTEXT_TOO_LARGE", "AI_ORGANIZATION_OUTPUT_LIMIT", "AI_ORGANIZATION_INVALID_JSON", "AI_ORGANIZATION_INVALID_EVIDENCE":
		return true
	}
	return false
}

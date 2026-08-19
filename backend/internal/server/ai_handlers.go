package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/llm"
)

// Experimental agent endpoints (charter E1): provider connectivity test plus a
// plain streaming chat with no tool calls. All routes stay behind the shared
// /api auth-lock middleware like every other endpoint.

// aiChatMessageLimits bound POST /api/ai/chat input so a runaway client cannot
// balloon provider costs in one request.
const (
	aiChatMaxMessages     = 50
	aiChatMaxContentRunes = 256 * 1024
	aiChatMaxMessageRunes = 64 * 1024
)

// handleAIProviderTest reports whether the configured (or drafted) provider
// answers a minimal chat completion; failures return HTTP 200 with ok=false so
// the settings UI can surface the message (same contract as proxy pings).
func (h *Handler) handleAIProviderTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.aiChatProvider == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "ai provider runtime not available")
		return
	}

	var req contracts.AIProviderTestRequest
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid request body")
			return
		}
	}

	var override *contracts.AIProviderSettingsDTO
	if req.Provider != nil {
		override = &contracts.AIProviderSettingsDTO{
			Kind:    config.NormalizeAIProviderKind(req.Provider.Kind),
			BaseURL: strings.TrimSpace(req.Provider.BaseURL),
			APIKey:  req.Provider.APIKey,
			Model:   strings.TrimSpace(req.Provider.Model),
		}
	}

	resp := h.aiChatProvider.TestAIProvider(r.Context(), override)
	writeJSON(w, http.StatusOK, resp)
}

// handleAIChat streams one plain chat completion as SSE events:
// message_start -> text_delta* -> message_done, or a terminal error event.
func (h *Handler) handleAIChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.aiChatProvider == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "ai chat runtime not available")
		return
	}

	var req contracts.AIChatRequest
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid request body")
			return
		}
	}
	if err := validateAIChatMessages(req.Messages); err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, err.Error())
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "event streaming is not supported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	if err := writeSSEJSON(w, "message_start", map[string]any{"type": "message_start"}); err != nil {
		return
	}
	flusher.Flush()

	err := h.aiChatProvider.StreamAIChat(r.Context(), req.Messages, func(delta string) {
		_ = writeSSEJSON(w, "text_delta", map[string]any{"type": "text_delta", "delta": delta})
		flusher.Flush()
	})
	if err != nil {
		code := contracts.ErrorCodeAIChatFailed
		if errors.Is(err, llm.ErrInvalidConfig) {
			code = contracts.ErrorCodeAIProviderUnavailable
		}
		_ = writeSSEJSON(w, "error", map[string]any{
			"type":    "error",
			"code":    code,
			"message": err.Error(),
		})
		flusher.Flush()
		return
	}
	_ = writeSSEJSON(w, "message_done", map[string]any{"type": "message_done"})
	flusher.Flush()
}

func validateAIChatMessages(messages []contracts.AIChatMessage) error {
	if len(messages) == 0 {
		return fmt.Errorf("messages must not be empty")
	}
	if len(messages) > aiChatMaxMessages {
		return fmt.Errorf("messages exceed the limit of %d", aiChatMaxMessages)
	}
	total := 0
	hasUser := false
	for i, m := range messages {
		switch strings.TrimSpace(m.Role) {
		case "system", "user", "assistant":
		default:
			return fmt.Errorf("messages[%d]: unsupported role %q", i, m.Role)
		}
		contentRunes := len([]rune(m.Content))
		if strings.TrimSpace(m.Content) == "" {
			return fmt.Errorf("messages[%d]: content must not be empty", i)
		}
		if contentRunes > aiChatMaxMessageRunes {
			return fmt.Errorf("messages[%d]: content exceeds %d runes", i, aiChatMaxMessageRunes)
		}
		total += contentRunes
		if m.Role == "user" {
			hasUser = true
		}
	}
	if !hasUser {
		return fmt.Errorf("messages must include at least one user message")
	}
	if total > aiChatMaxContentRunes {
		return fmt.Errorf("messages total content exceeds %d runes", aiChatMaxContentRunes)
	}
	return nil
}

package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/llm"
)

// Experimental agent endpoints (charter E2): provider test, agent loop SSE,
// and persisted chat sessions. All routes stay behind the shared /api auth-lock
// middleware like every other endpoint.

// aiChatMessageLimits bound POST /api/ai/chat input so a runaway client cannot
// balloon provider costs in one request.
const (
	aiChatMaxMessages     = 50
	aiChatMaxContentRunes = 256 * 1024
	aiChatMaxMessageRunes = 64 * 1024
	aiChatContextVersion  = 1
	aiChatMaxContextRefs  = 8
	aiChatMaxContextRunes = 200
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

// handleAIChat streams one agent turn as SSE events:
// message_start -> (text_delta | tool_call_started | tool_call_result | movie_cards)* -> message_done,
// or a terminal error event.
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
	if err := normalizeAIChatContext(req.Context); err != nil {
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

	err := h.aiChatProvider.StreamAIChat(r.Context(), req, func(ev contracts.AIChatSSEEvent) {
		if strings.TrimSpace(ev.Type) == "" {
			return
		}
		_ = writeSSEJSON(w, ev.Type, ev)
		flusher.Flush()
	})
	if err != nil {
		code := contracts.ErrorCodeAIChatFailed
		if errors.Is(err, llm.ErrInvalidConfig) {
			code = contracts.ErrorCodeAIProviderUnavailable
		}
		if errors.Is(err, sql.ErrNoRows) {
			code = contracts.ErrorCodeNotFound
		}
		_ = writeSSEJSON(w, "error", map[string]any{
			"type":    "error",
			"code":    code,
			"message": err.Error(),
		})
		flusher.Flush()
	}
}

func (h *Handler) handleListAIChatSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.aiChatProvider == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "ai chat runtime not available")
		return
	}
	dto, err := h.aiChatProvider.ListAIChatSessions(r.Context())
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) handleCreateAIChatSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.aiChatProvider == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "ai chat runtime not available")
		return
	}
	var body struct {
		Title string `json:"title"`
	}
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid request body")
			return
		}
	}
	dto, err := h.aiChatProvider.CreateAIChatSession(r.Context(), body.Title)
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, dto)
}

func (h *Handler) handleGetAIChatSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.aiChatProvider == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "ai chat runtime not available")
		return
	}
	id := strings.TrimSpace(r.PathValue("sessionId"))
	if id == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "sessionId is required")
		return
	}
	dto, err := h.aiChatProvider.GetAIChatSession(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "ai chat session not found")
			return
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) handleDeleteAIChatSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.aiChatProvider == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "ai chat runtime not available")
		return
	}
	id := strings.TrimSpace(r.PathValue("sessionId"))
	if id == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "sessionId is required")
		return
	}
	if err := h.aiChatProvider.DeleteAIChatSession(r.Context(), id); err != nil {
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "ai chat session not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleAIAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.aiChatProvider == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "ai action runtime not available")
		return
	}
	name := strings.TrimSpace(r.PathValue("name"))
	if name == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "action name is required")
		return
	}
	var req contracts.AIActionRequest
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid request body")
			return
		}
	}
	dto, err := h.aiChatProvider.RunAIAction(r.Context(), name, req)
	if err != nil {
		writeAIActionError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) handleAIConfirm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.aiChatProvider == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "ai action runtime not available")
		return
	}
	var req contracts.AIToolApplyRequest
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid request body")
			return
		}
	}
	dto, err := h.aiChatProvider.ApplyAITool(r.Context(), req)
	if err != nil {
		writeAIActionError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func writeAIActionError(w http.ResponseWriter, err error) {
	if errors.Is(err, llm.ErrInvalidConfig) {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeAIProviderUnavailable, err.Error())
		return
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "unknown action"):
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, msg)
	case strings.Contains(msg, "movie not found"):
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, msg)
	case strings.Contains(msg, "confirm token"):
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeAIConfirmExpired, msg)
	case strings.Contains(msg, "required") || strings.Contains(msg, "empty") || strings.Contains(msg, "too long"):
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, msg)
	default:
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeAIChatFailed, msg)
	}
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

// normalizeAIChatContext bounds the browser-supplied page projection before it
// can reach the system prompt. Context is helpful only when it is explicit and
// small; it is never accepted as arbitrary page text or an executable filter.
func normalizeAIChatContext(page *contracts.AIChatContext) error {
	if page == nil {
		return nil
	}
	if page.ContextVersion < 0 || page.ContextVersion > aiChatContextVersion {
		return fmt.Errorf("unsupported contextVersion")
	}
	usesV1Fields := len(page.SelectedMovieIDs) > 0 || len(page.SelectedActors) > 0 || page.ActiveFilters != nil
	if usesV1Fields && page.ContextVersion != aiChatContextVersion {
		return fmt.Errorf("contextVersion 1 is required for selected entities or activeFilters")
	}
	if err := normalizeAIChatContextString(&page.Route, 80, "context.route"); err != nil {
		return err
	}
	if err := normalizeAIChatContextString(&page.MovieID, 128, "context.movieId"); err != nil {
		return err
	}
	if err := normalizeAIChatContextString(&page.ActorName, 160, "context.actorName"); err != nil {
		return err
	}
	if err := normalizeAIChatContextString(&page.Query, aiChatMaxContextRunes, "context.query"); err != nil {
		return err
	}
	if len(page.Mentions) > aiChatMaxContextRefs {
		return fmt.Errorf("context.mentions exceed the limit of %d", aiChatMaxContextRefs)
	}
	for i := range page.Mentions {
		mention := &page.Mentions[i]
		mention.Kind = strings.ToLower(strings.TrimSpace(mention.Kind))
		if mention.Kind != "movie" && mention.Kind != "actor" && mention.Kind != "tag" {
			return fmt.Errorf("context.mentions[%d].kind is invalid", i)
		}
		if err := normalizeAIChatContextString(&mention.ID, 160, fmt.Sprintf("context.mentions[%d].id", i)); err != nil {
			return err
		}
		if err := normalizeAIChatContextString(&mention.Label, 80, fmt.Sprintf("context.mentions[%d].label", i)); err != nil {
			return err
		}
		if mention.ID == "" && mention.Label == "" {
			return fmt.Errorf("context.mentions[%d] requires an id or label", i)
		}
	}
	var err error
	if page.SelectedMovieIDs, err = normalizeAIChatContextRefs(page.SelectedMovieIDs, 128, "context.selectedMovieIds"); err != nil {
		return err
	}
	if page.SelectedActors, err = normalizeAIChatContextRefs(page.SelectedActors, 160, "context.selectedActors"); err != nil {
		return err
	}
	if filters := page.ActiveFilters; filters != nil {
		if err := normalizeAIChatContextString(&filters.Query, aiChatMaxContextRunes, "context.activeFilters.query"); err != nil {
			return err
		}
		if err := normalizeAIChatContextString(&filters.Tag, aiChatMaxContextRunes, "context.activeFilters.tag"); err != nil {
			return err
		}
		if err := normalizeAIChatContextString(&filters.Actor, aiChatMaxContextRunes, "context.activeFilters.actor"); err != nil {
			return err
		}
		if err := normalizeAIChatContextString(&filters.PlayState, 32, "context.activeFilters.playState"); err != nil {
			return err
		}
		if err := normalizeAIChatContextString(&filters.Runtime, 32, "context.activeFilters.runtime"); err != nil {
			return err
		}
		if filters.PlayState != "" && filters.PlayState != "all" && filters.PlayState != "unwatched" && filters.PlayState != "in-progress" && filters.PlayState != "completed" {
			return fmt.Errorf("context.activeFilters.playState is invalid")
		}
		if filters.Runtime != "" && filters.Runtime != "short" && filters.Runtime != "standard" && filters.Runtime != "long" {
			return fmt.Errorf("context.activeFilters.runtime is invalid")
		}
		if filters.Query == "" && filters.Tag == "" && filters.Actor == "" && filters.PlayState == "" && filters.Runtime == "" {
			page.ActiveFilters = nil
		}
	}
	return nil
}

func normalizeAIChatContextRefs(values []string, maxRunes int, field string) ([]string, error) {
	if len(values) > aiChatMaxContextRefs {
		return nil, fmt.Errorf("%s exceed the limit of %d", field, aiChatMaxContextRefs)
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for i, value := range values {
		if err := normalizeAIChatContextString(&value, maxRunes, fmt.Sprintf("%s[%d]", field, i)); err != nil {
			return nil, err
		}
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out, nil
}

func normalizeAIChatContextString(value *string, maxRunes int, field string) error {
	if value == nil {
		return nil
	}
	*value = strings.TrimSpace(*value)
	if utf8.RuneCountInString(*value) > maxRunes {
		return fmt.Errorf("%s exceeds %d runes", field, maxRunes)
	}
	return nil
}

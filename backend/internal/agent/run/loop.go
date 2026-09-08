package run

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"curated-backend/internal/agent/core"
	"curated-backend/internal/agent/prompts"
	"curated-backend/internal/contracts"
	"curated-backend/internal/llm"
)

const maxRecentMessages = 24

type Emitter func(contracts.AIChatSSEEvent)

type Loop struct {
	gateway  *core.Gateway
	streamer llm.Streamer
	sanitize string
	locale   string
}

func NewLoop(gateway *core.Gateway, streamer llm.Streamer, sanitize, locale string) *Loop {
	if sanitize == "" {
		sanitize = core.SanitizeFull
	}
	return &Loop{gateway: gateway, streamer: streamer, sanitize: sanitize, locale: locale}
}

func (l *Loop) Run(ctx context.Context, sessionID, messageID string, history []llm.ChatMessage, page *contracts.AIChatContext, emit Emitter) error {
	if l == nil || l.gateway == nil || l.streamer == nil {
		return fmt.Errorf("agent loop is not configured")
	}
	if emit == nil {
		emit = func(contracts.AIChatSSEEvent) {}
	}
	l.gateway.ResetSessionSteps(sessionID)
	l.gateway.ResetMovieRefs(sessionID)
	l.gateway.ResetActorRefs(sessionID)
	l.gateway.ResetSourceURLs(sessionID)
	seedTurnEntities(l.gateway, sessionID, page)
	seq := 0
	nextSeq := func() int {
		seq++
		return seq
	}
	emit(contracts.AIChatSSEEvent{Type: "message_start", SessionID: sessionID, MessageID: messageID, Seq: nextSeq()})

	messages := buildMessages(history, page, l.locale)
	tools := l.toolSpecs()
	steps := 0
	stepLimit := l.gateway.StepLimit()
	hadFailure := false
	hadTruncation := false
	needsInput := false
	emitDone := func(status, reason string, retryable bool, reasonCode string) {
		emit(contracts.AIChatSSEEvent{
			Type: "message_done", SessionID: sessionID, MessageID: messageID, Seq: nextSeq(),
			Outcome: &contracts.AIChatOutcomeDTO{Status: status, Reason: reason, Retryable: retryable, ReasonCode: reasonCode},
		})
	}

	for {
		if ctx.Err() != nil {
			emitDone("cancelled", "The user cancelled this request before it finished.", false, "")
			return nil
		}
		if stepLimit > 0 && steps >= stepLimit {
			// End before another model call: the last batch may contain unexecuted
			// calls, which must not be sent back as an incomplete tool conversation.
			emitDone("partial", fmt.Sprintf("The configured limit of %d tool calls was reached. Completed results are preserved; the task is not finished.", stepLimit), true, "tool_step_limit")
			return nil
		}
		if estimatedRequestTokens(messages, tools) > requestTokenEstimateBudget {
			status := "needs_input"
			if steps > 0 {
				status = "partial"
			}
			emitDone(status, "The context budget was reached. Narrow the request or start a new conversation; completed results remain available.", false, "")
			return nil
		}
		turn, err := l.streamer.StreamTurn(ctx, llm.TurnRequest{
			Messages: messages,
			Tools:    tools,
			OnThinking: func(delta string) {
				if delta == "" {
					return
				}
				emit(contracts.AIChatSSEEvent{
					Type: "thinking_delta", SessionID: sessionID, MessageID: messageID, Seq: nextSeq(), Delta: delta,
				})
			},
		}, func(delta string) {
			if delta == "" {
				return
			}
			emit(contracts.AIChatSSEEvent{
				Type: "text_delta", SessionID: sessionID, MessageID: messageID, Seq: nextSeq(), Delta: delta,
			})
		})
		if err != nil {
			if ctx.Err() != nil {
				emitDone("cancelled", "The user cancelled this request before it finished.", false, "")
				return nil
			}
			emitDone("failed", "The model response could not be completed.", true, "")
			return err
		}
		if len(turn.ToolCalls) == 0 {
			if strings.TrimSpace(turn.Content) == "" {
				emitDone("failed", "The model returned no answer.", true, "")
			} else if needsInput {
				emitDone("needs_input", "A local entity needs the user's selection or a more specific name.", false, "")
			} else if hadFailure || hadTruncation {
				reason := "Some requested evidence could not be fully retrieved."
				if hadFailure {
					reason = "One or more retrievals failed; the answer contains only confirmed results."
				}
				emitDone("partial", reason, hadFailure, "")
			} else {
				emitDone("completed", "", false, "")
			}
			return nil
		}

		assistant := llm.ChatMessage{Role: "assistant", Content: turn.Content, ToolCalls: turn.ToolCalls}
		messages = append(messages, assistant)
		for _, call := range turn.ToolCalls {
			steps++
			toolCallID := call.ID
			if toolCallID == "" {
				toolCallID = fmt.Sprintf("call_%d", steps)
			}
			emit(contracts.AIChatSSEEvent{
				Type: "tool_call_started", SessionID: sessionID, MessageID: messageID, Seq: nextSeq(),
				ToolCallID: toolCallID, Name: call.Name(),
			})
			token, args := extractConfirmToken(nonzeroJSON(call.Args()))
			result := l.gateway.Invoke(ctx, core.Call{
				Name:       call.Name(),
				Args:       args,
				SessionID:  sessionID,
				Channel:    core.ChannelChat,
				ConfirmTok: token,
				Sanitize:   l.sanitize,
			})
			if result.OK {
				l.gateway.RememberMovieRefs(sessionID, core.ExtractMovieRefs(result))
				l.gateway.RememberActorNames(sessionID, core.ExtractActorNames(result))
				l.gateway.RememberSourceURLs(sessionID, core.ExtractSourceURLs(result))
			}
			if !result.OK {
				hadFailure = true
			}
			if result.Truncated {
				hadTruncation = true
			}
			resolution := entityResolutionFromResult(call.Name(), result)
			if resolution != nil {
				if resolution.Status == "matched" {
					for _, candidate := range resolution.Candidates {
						if candidate.MovieID != "" {
							l.gateway.RememberMovieRefs(sessionID, []core.MovieRef{{ID: candidate.MovieID, Title: candidate.Title, Code: candidate.Code}})
						}
						if candidate.ActorName != "" {
							l.gateway.RememberActorNames(sessionID, []string{candidate.ActorName})
						}
					}
				} else {
					needsInput = true
				}
			}
			summary := toolSummary(call.Name(), result)
			ok := result.OK
			movies := presentMovieCards(call.Name(), result)
			providerRows := providerTitleRows(call.Name(), result)
			emit(contracts.AIChatSSEEvent{
				Type: "tool_call_result", SessionID: sessionID, MessageID: messageID, Seq: nextSeq(),
				ToolCallID: toolCallID, Name: call.Name(), OK: &ok, Summary: summary, Truncated: result.Truncated,
				Movies: movies, ProviderRows: providerRows, Resolution: resolution, Evidence: evidenceForTool(call.Name(), result),
			})
			if len(movies) > 0 {
				emit(contracts.AIChatSSEEvent{
					Type: "movie_cards", SessionID: sessionID, MessageID: messageID, Seq: nextSeq(),
					ToolCallID: toolCallID, Name: call.Name(), Movies: movies,
				})
			}
			if result.ConfirmToken != "" {
				confirmArgs := result.ConfirmArgs
				if len(confirmArgs) == 0 {
					confirmArgs = args
				}
				emit(contracts.AIChatSSEEvent{
					Type: "confirm_required", SessionID: sessionID, MessageID: messageID, Seq: nextSeq(),
					ToolCallID: toolCallID, Name: call.Name(),
					ConfirmToken: result.ConfirmToken, ExpiresAt: result.ExpiresAt,
					Changes:   confirmChanges(result.Changes),
					Arguments: confirmArgs,
					Summary:   summary,
					OK:        &ok,
				})
				emitDone("needs_confirmation", "A write preview is ready. Confirm it to save the changes.", false, "confirmation_required")
				return nil
			}
			payload, _ := json.Marshal(result)
			messages = append(messages, llm.ChatMessage{
				Role:       "tool",
				ToolCallID: toolCallID,
				Content:    wrapToolContent(payload),
			})
			if stepLimit > 0 && steps >= stepLimit {
				break
			}
		}
	}
}

func providerTitleRows(name string, result core.Result) []contracts.AIAgentProviderTitleDTO {
	if name != core.SearchProviderTitlesName || !result.OK || result.Data == nil {
		return nil
	}
	raw, err := json.Marshal(result.Data)
	if err != nil {
		return nil
	}
	var envelope struct {
		Source struct {
			Items []contracts.AIAgentProviderTitleDTO `json:"items"`
		} `json:"source"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil
	}
	return envelope.Source.Items
}

func entityResolutionFromResult(name string, result core.Result) *contracts.AIEntityResolutionDTO {
	if name != "resolve_entities" || !result.OK || result.Data == nil {
		return nil
	}
	raw, err := json.Marshal(result.Data)
	if err != nil {
		return nil
	}
	var envelope struct {
		Source contracts.AIEntityResolutionDTO `json:"source"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil || envelope.Source.Status == "" {
		return nil
	}
	return &envelope.Source
}

func evidenceForTool(name string, result core.Result) *contracts.AIEvidenceDTO {
	evidence := &contracts.AIEvidenceDTO{
		Source:      evidenceSource(name),
		RetrievedAt: time.Now().UTC().Format(time.RFC3339),
		Truncated:   result.Truncated,
		NextCursor:  result.NextCursor,
	}
	if result.Error != nil {
		evidence.Failed = true
		evidence.ErrorCode = result.Error.Code
	}
	if result.Data != nil {
		evidence.Filters = evidenceFilters(result.Data)
	}
	return evidence
}

func evidenceSource(name string) string {
	switch name {
	case core.SearchProviderTitlesName:
		return "provider"
	case core.GetSourcePageName:
		return "source_page"
	default:
		return "local"
	}
}

func evidenceFilters(data any) map[string]string {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil
	}
	var envelope struct {
		Source map[string]any `json:"source"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil || envelope.Source == nil {
		return nil
	}
	query, ok := envelope.Source["query"].(map[string]any)
	if !ok || len(query) == 0 {
		return nil
	}
	out := make(map[string]string, len(query))
	for key, value := range query {
		switch v := value.(type) {
		case string:
			if strings.TrimSpace(v) != "" {
				out[key] = strings.TrimSpace(v)
			}
		case float64:
			out[key] = fmt.Sprintf("%v", v)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (l *Loop) toolSpecs() []llm.ToolSpec {
	defs := l.gateway.Registry().ListByPermission(map[string]bool{
		core.PermissionRead:         true,
		core.PermissionWritePreview: true,
	})
	out := make([]llm.ToolSpec, 0, len(defs))
	for _, def := range defs {
		out = append(out, llm.ToolSpec{
			Name:        def.Name,
			Description: def.Description,
			Parameters:  def.ParamsSchema.JSONSchemaMap(),
		})
	}
	return out
}

func buildMessages(history []llm.ChatMessage, page *contracts.AIChatContext, locale string) []llm.ChatMessage {
	trimmed, omitted := boundedHistory(history)
	out := make([]llm.ChatMessage, 0, len(trimmed)+1)
	out = append(out, llm.ChatMessage{Role: "system", Content: prompts.SystemPrompt(locale, page)})
	if omitted {
		out[0].Content += "\nEarlier conversation messages were omitted to fit the context budget. Do not assume missing facts or permissions; ask for clarification when needed."
	}
	out = append(out, trimmed...)
	return out
}

func nonzeroJSON(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return "{}"
	}
	return raw
}

func extractConfirmToken(raw string) (string, json.RawMessage) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", json.RawMessage(`{}`)
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(trimmed), &obj); err != nil || obj == nil {
		return "", json.RawMessage(trimmed)
	}
	token := ""
	if v, ok := obj["confirmToken"].(string); ok {
		token = strings.TrimSpace(v)
		delete(obj, "confirmToken")
	}
	encoded, err := json.Marshal(obj)
	if err != nil {
		return token, json.RawMessage(trimmed)
	}
	return token, encoded
}

func wrapToolContent(payload []byte) string {
	return "<source>\n" + string(payload) + "\n</source>"
}

func toolSummary(name string, result core.Result) string {
	if result.Error != nil {
		return result.Error.Message
	}
	if !result.OK {
		return "failed"
	}
	raw, err := json.Marshal(result.Data)
	if err != nil {
		return name + " ok"
	}
	text := string(raw)
	if utf8.RuneCountInString(text) > 180 {
		runes := []rune(text)
		text = string(runes[:180]) + "…"
	}
	if result.Truncated {
		return name + " truncated: " + text
	}
	return name + ": " + text
}

func presentMovieCards(name string, result core.Result) []contracts.AIAgentMovieCardDTO {
	if name != core.PresentMoviesName || !result.OK {
		return nil
	}
	refs := core.ExtractMovieRefs(result)
	if len(refs) == 0 {
		return nil
	}
	out := make([]contracts.AIAgentMovieCardDTO, 0, len(refs))
	for _, ref := range refs {
		out = append(out, contracts.AIAgentMovieCardDTO{
			MovieID:  ref.ID,
			Title:    ref.Title,
			Code:     ref.Code,
			Actors:   ref.Actors,
			CoverURL: ref.CoverURL,
			ThumbURL: ref.ThumbURL,
			Reason:   ref.Reason,
		})
	}
	return out
}

func confirmChanges(changes []core.Change) []contracts.AIConfirmChangeDTO {
	if len(changes) == 0 {
		return nil
	}
	out := make([]contracts.AIConfirmChangeDTO, 0, len(changes))
	for _, change := range changes {
		out = append(out, contracts.AIConfirmChangeDTO{
			Path:   change.Path,
			Before: change.Before,
			After:  change.After,
		})
	}
	return out
}

func seedTurnEntities(gateway *core.Gateway, sessionID string, page *contracts.AIChatContext) {
	if gateway == nil || page == nil {
		return
	}
	if id := strings.TrimSpace(page.MovieID); id != "" {
		gateway.RememberMovieRefs(sessionID, []core.MovieRef{{ID: id}})
	}
	if name := strings.TrimSpace(page.ActorName); name != "" {
		gateway.RememberActorNames(sessionID, []string{name})
	}
	for _, rawID := range page.SelectedMovieIDs {
		if id := strings.TrimSpace(rawID); id != "" {
			gateway.RememberMovieRefs(sessionID, []core.MovieRef{{ID: id}})
		}
	}
	gateway.RememberActorNames(sessionID, page.SelectedActors)
	for _, mention := range page.Mentions {
		kind := strings.ToLower(strings.TrimSpace(mention.Kind))
		switch kind {
		case "movie":
			if id := strings.TrimSpace(mention.ID); id != "" {
				gateway.RememberMovieRefs(sessionID, []core.MovieRef{{ID: id}})
			}
		case "actor":
			names := make([]string, 0, 2)
			if label := strings.TrimSpace(mention.Label); label != "" {
				names = append(names, label)
			}
			if id := strings.TrimSpace(mention.ID); id != "" {
				names = append(names, id)
			}
			gateway.RememberActorNames(sessionID, names)
		}
	}
}

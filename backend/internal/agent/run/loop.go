package run

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		choice := ""
		if steps >= stepLimit {
			messages = append(messages, llm.ChatMessage{Role: "system", Content: prompts.StepLimitNudge()})
			choice = "none"
		}
		turn, err := l.streamer.StreamTurn(ctx, llm.TurnRequest{
			Messages:   messages,
			Tools:      tools,
			ToolChoice: choice,
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
			return err
		}
		if len(turn.ToolCalls) == 0 {
			emit(contracts.AIChatSSEEvent{Type: "message_done", SessionID: sessionID, MessageID: messageID, Seq: nextSeq()})
			return nil
		}
		if steps >= stepLimit {
			emit(contracts.AIChatSSEEvent{
				Type: "text_delta", SessionID: sessionID, MessageID: messageID, Seq: nextSeq(),
				Delta: "\n\n已达到本轮工具步数上限，未能继续查询。",
			})
			emit(contracts.AIChatSSEEvent{Type: "message_done", SessionID: sessionID, MessageID: messageID, Seq: nextSeq()})
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
			}
			summary := toolSummary(call.Name(), result)
			ok := result.OK
			movies := presentMovieCards(call.Name(), result)
			emit(contracts.AIChatSSEEvent{
				Type: "tool_call_result", SessionID: sessionID, MessageID: messageID, Seq: nextSeq(),
				ToolCallID: toolCallID, Name: call.Name(), OK: &ok, Summary: summary, Truncated: result.Truncated,
				Movies: movies,
			})
			if len(movies) > 0 {
				emit(contracts.AIChatSSEEvent{
					Type: "movie_cards", SessionID: sessionID, MessageID: messageID, Seq: nextSeq(),
					ToolCallID: toolCallID, Name: call.Name(), Movies: movies,
				})
			}
			if result.ConfirmToken != "" {
				emit(contracts.AIChatSSEEvent{
					Type: "confirm_required", SessionID: sessionID, MessageID: messageID, Seq: nextSeq(),
					ToolCallID: toolCallID, Name: call.Name(),
					ConfirmToken: result.ConfirmToken, ExpiresAt: result.ExpiresAt,
					Changes:   confirmChanges(result.Changes),
					Arguments: args,
					Summary:   summary,
					OK:        &ok,
				})
				emit(contracts.AIChatSSEEvent{Type: "message_done", SessionID: sessionID, MessageID: messageID, Seq: nextSeq()})
				return nil
			}
			payload, _ := json.Marshal(result)
			messages = append(messages, llm.ChatMessage{
				Role:       "tool",
				ToolCallID: toolCallID,
				Content:    wrapToolContent(payload),
			})
			if steps >= stepLimit {
				break
			}
		}
	}
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
	trimmed := history
	if len(trimmed) > maxRecentMessages {
		trimmed = trimmed[len(trimmed)-maxRecentMessages:]
	}
	out := make([]llm.ChatMessage, 0, len(trimmed)+1)
	out = append(out, llm.ChatMessage{Role: "system", Content: prompts.SystemPrompt(locale, page)})
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

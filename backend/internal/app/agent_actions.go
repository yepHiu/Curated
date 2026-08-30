package app

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

type commentCompleter interface {
	Complete(ctx context.Context, messages []llm.ChatMessage, maxTokens int) (string, error)
}

// RunAIAction runs a single-shot action preset (E3 write-preview or read-only narrative).
func (a *App) RunAIAction(ctx context.Context, name string, req contracts.AIActionRequest) (contracts.AIActionPreviewDTO, error) {
	if !prompts.KnownAction(name) {
		return contracts.AIActionPreviewDTO{}, fmt.Errorf("unknown action")
	}
	cfg, err := normalizeAIProviderConfig(a.currentAIProviderConfig())
	if err != nil {
		return contracts.AIActionPreviewDTO{}, fmt.Errorf("%w: %v", llm.ErrInvalidConfig, err)
	}
	if strings.TrimSpace(cfg.BaseURL) == "" || strings.TrimSpace(cfg.Model) == "" {
		return contracts.AIActionPreviewDTO{}, ErrAIProviderNotConfigured
	}
	client, err := newAIHTTPClient(a.currentProxyConfig(), 0)
	if err != nil {
		return contracts.AIActionPreviewDTO{}, err
	}
	completer := llm.NewClient(llm.ClientConfig{
		BaseURL: cfg.BaseURL,
		APIKey:  cfg.APIKey,
		Model:   cfg.Model,
	}, client)
	switch {
	case prompts.IsCommentAction(name):
		return a.runCommentAction(ctx, completer, name, req)
	case prompts.IsDisplayAction(name):
		return a.runDisplayAction(ctx, completer, name, req)
	default:
		return a.runInsightsNarrative(ctx, completer, req)
	}
}

func (a *App) runCommentAction(ctx context.Context, completer commentCompleter, name string, req contracts.AIActionRequest) (contracts.AIActionPreviewDTO, error) {
	movieID := strings.TrimSpace(req.MovieID)
	if movieID == "" {
		return contracts.AIActionPreviewDTO{}, fmt.Errorf("movieId is required")
	}
	exists, err := a.MovieExists(ctx, movieID)
	if err != nil {
		return contracts.AIActionPreviewDTO{}, err
	}
	if !exists {
		return contracts.AIActionPreviewDTO{}, fmt.Errorf("movie not found")
	}
	original := strings.TrimSpace(req.Body)
	if original == "" {
		current, err := a.GetMovieComment(ctx, movieID)
		if err != nil {
			return contracts.AIActionPreviewDTO{}, err
		}
		original = current.Body
	}
	if original == "" {
		return contracts.AIActionPreviewDTO{}, fmt.Errorf("comment is empty")
	}
	if utf8.RuneCountInString(original) > contracts.MaxMovieCommentRunes {
		return contracts.AIActionPreviewDTO{}, fmt.Errorf("comment body too long")
	}
	proposed, err := completer.Complete(ctx, []llm.ChatMessage{
		{Role: "system", Content: prompts.CommentActionPrompt(original)},
		{Role: "user", Content: prompts.FormatCommentActionUser(original)},
	}, 0)
	if err != nil {
		return contracts.AIActionPreviewDTO{}, err
	}
	proposed = sanitizeActionText(proposed)
	if proposed == "" {
		return contracts.AIActionPreviewDTO{}, fmt.Errorf("provider returned empty text")
	}
	if utf8.RuneCountInString(proposed) > contracts.MaxMovieCommentRunes {
		runes := []rune(proposed)
		proposed = string(runes[:contracts.MaxMovieCommentRunes])
	}

	args, err := json.Marshal(map[string]string{"movieId": movieID, "body": proposed})
	if err != nil {
		return contracts.AIActionPreviewDTO{}, err
	}
	sessionID := newAgentID("act_")
	result := a.ensureAgentGateway().Invoke(ctx, core.Call{
		Name:      core.SaveMovieCommentName,
		Args:      args,
		SessionID: sessionID,
		Channel:   core.ChannelAction,
		Sanitize:  core.SanitizeFull,
	})
	if result.Error != nil {
		return contracts.AIActionPreviewDTO{}, fmt.Errorf("%s", result.Error.Message)
	}
	preview := contracts.AIActionPreviewDTO{
		Action:       name,
		Name:         core.SaveMovieCommentName,
		SessionID:    sessionID,
		OriginalText: original,
		ProposedText: proposed,
		Changes:      confirmChangeDTOs(result.Changes),
		ConfirmToken: result.ConfirmToken,
		ExpiresAt:    result.ExpiresAt,
		Arguments:    args,
		Noop:         result.ConfirmToken == "",
	}
	return preview, nil
}

// ApplyAITool consumes a confirm token and applies the previewed write.
func (a *App) ApplyAITool(ctx context.Context, req contracts.AIToolApplyRequest) (contracts.AIToolApplyDTO, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return contracts.AIToolApplyDTO{}, fmt.Errorf("name is required")
	}
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return contracts.AIToolApplyDTO{}, fmt.Errorf("sessionId is required")
	}
	if strings.TrimSpace(req.ConfirmToken) == "" {
		return contracts.AIToolApplyDTO{}, fmt.Errorf("confirmToken is required")
	}
	args := req.Arguments
	if len(args) == 0 {
		args = json.RawMessage(`{}`)
	}
	result := a.ensureAgentGateway().Invoke(ctx, core.Call{
		Name:       name,
		Args:       args,
		SessionID:  sessionID,
		Channel:    core.ChannelAction,
		ConfirmTok: strings.TrimSpace(req.ConfirmToken),
		Sanitize:   core.SanitizeFull,
	})
	if result.Error != nil {
		return contracts.AIToolApplyDTO{}, fmt.Errorf("%s", result.Error.Message)
	}
	return contracts.AIToolApplyDTO{OK: true, Name: name, Data: result.Data}, nil
}

func sanitizeActionText(raw string) string {
	text := strings.TrimSpace(raw)
	text = strings.TrimPrefix(text, "```")
	if strings.Contains(text, "\n") {
		lines := strings.Split(text, "\n")
		if len(lines) > 1 && (strings.HasPrefix(lines[0], "text") || strings.HasPrefix(lines[0], "markdown")) {
			text = strings.Join(lines[1:], "\n")
		}
	}
	text = strings.TrimSuffix(strings.TrimSpace(text), "```")
	return strings.TrimSpace(text)
}

func clipUTF8Bytes(value string, max int) string {
	if max <= 0 || len(value) <= max {
		return value
	}
	value = value[:max]
	for len(value) > 0 && !utf8.ValidString(value) {
		value = value[:len(value)-1]
	}
	return value
}

func confirmChangeDTOs(changes []core.Change) []contracts.AIConfirmChangeDTO {
	if len(changes) == 0 {
		return nil
	}
	out := make([]contracts.AIConfirmChangeDTO, 0, len(changes))
	for _, change := range changes {
		out = append(out, contracts.AIConfirmChangeDTO{Path: change.Path, Before: change.Before, After: change.After})
	}
	return out
}

func (a *App) runDisplayAction(ctx context.Context, completer commentCompleter, name string, req contracts.AIActionRequest) (contracts.AIActionPreviewDTO, error) {
	movieID := strings.TrimSpace(req.MovieID)
	if movieID == "" {
		return contracts.AIActionPreviewDTO{}, fmt.Errorf("movieId is required")
	}
	detail, err := a.GetMovieDetail(ctx, movieID)
	if err != nil {
		return contracts.AIActionPreviewDTO{}, err
	}
	var original string
	var system string
	var userKind string
	locale := strings.TrimSpace(req.TargetLocale)
	if locale == "" {
		locale = strings.TrimSpace(req.Locale)
	}
	draft := strings.TrimSpace(req.Body)
	switch name {
	case prompts.ActionTranslateSummary:
		original = draft
		if original == "" {
			original = strings.TrimSpace(detail.Summary)
		}
		if original == "" {
			return contracts.AIActionPreviewDTO{}, fmt.Errorf("summary is empty")
		}
		system = prompts.TranslateSummaryPrompt(original, locale)
		userKind = "Synopsis"
	case prompts.ActionTranslateTitle:
		original = draft
		if original == "" {
			original = strings.TrimSpace(detail.Title)
		}
		if original == "" {
			return contracts.AIActionPreviewDTO{}, fmt.Errorf("title is empty")
		}
		system = prompts.TranslateTitlePrompt(original, locale)
		userKind = "Title"
	default:
		return contracts.AIActionPreviewDTO{}, fmt.Errorf("unknown action")
	}
	proposed, err := completer.Complete(ctx, []llm.ChatMessage{
		{Role: "system", Content: system},
		{Role: "user", Content: prompts.FormatPlainUser(userKind, original)},
	}, 0)
	if err != nil {
		return contracts.AIActionPreviewDTO{}, err
	}
	proposed = sanitizeActionText(proposed)
	if proposed == "" {
		return contracts.AIActionPreviewDTO{}, fmt.Errorf("provider returned empty text")
	}
	argsMap := map[string]string{"movieId": movieID}
	if name == prompts.ActionTranslateSummary {
		proposed = clipUTF8Bytes(proposed, core.MaxMovieSummaryBytes)
		argsMap["userSummary"] = proposed
	} else {
		argsMap["userTitle"] = proposed
	}
	args, err := json.Marshal(argsMap)
	if err != nil {
		return contracts.AIActionPreviewDTO{}, err
	}
	sessionID := newAgentID("act_")
	result := a.ensureAgentGateway().Invoke(ctx, core.Call{
		Name:      core.UpdateMovieDisplayOverridesName,
		Args:      args,
		SessionID: sessionID,
		Channel:   core.ChannelAction,
		Sanitize:  core.SanitizeFull,
	})
	if result.Error != nil {
		return contracts.AIActionPreviewDTO{}, fmt.Errorf("%s", result.Error.Message)
	}
	return contracts.AIActionPreviewDTO{
		Action:       name,
		Name:         core.UpdateMovieDisplayOverridesName,
		SessionID:    sessionID,
		OriginalText: original,
		ProposedText: proposed,
		Changes:      confirmChangeDTOs(result.Changes),
		ConfirmToken: result.ConfirmToken,
		ExpiresAt:    result.ExpiresAt,
		Arguments:    args,
		Noop:         result.ConfirmToken == "",
	}, nil
}

func (a *App) runInsightsNarrative(ctx context.Context, completer commentCompleter, req contracts.AIActionRequest) (contracts.AIActionPreviewDTO, error) {
	rangeValue := strings.TrimSpace(req.Range)
	if rangeValue == "" {
		rangeValue = "30d"
	}
	timezone := strings.TrimSpace(req.Timezone)
	if timezone == "" {
		timezone = "UTC"
	}
	sessionID := newAgentID("act_")
	payload, err := a.collectInsightsActionPayload(ctx, sessionID, rangeValue, timezone)
	if err != nil {
		return contracts.AIActionPreviewDTO{}, err
	}
	locale := strings.TrimSpace(req.Locale)
	proposed, err := completer.Complete(ctx, []llm.ChatMessage{
		{Role: "system", Content: prompts.InsightsNarrativePrompt(locale, payload)},
		{Role: "user", Content: prompts.FormatPlainUser("Insights", rangeValue)},
	}, 0)
	if err != nil {
		return contracts.AIActionPreviewDTO{}, err
	}
	proposed = sanitizeActionText(proposed)
	if proposed == "" {
		return contracts.AIActionPreviewDTO{}, fmt.Errorf("provider returned empty text")
	}
	return contracts.AIActionPreviewDTO{
		Action:       prompts.ActionInsightsNarrative,
		Name:         "insights_narrative",
		SessionID:    sessionID,
		ProposedText: proposed,
		Noop:         true,
	}, nil
}

func (a *App) collectInsightsActionPayload(ctx context.Context, sessionID, rangeValue, timezone string) (string, error) {
	gw := a.ensureAgentGateway()
	overview := gw.Invoke(ctx, core.Call{
		Name:      "get_insights_overview",
		Args:      mustJSON(map[string]string{"range": rangeValue, "timezone": timezone}),
		SessionID: sessionID,
		Channel:   core.ChannelAction,
		Sanitize:  core.SanitizeFull,
	})
	if overview.Error != nil {
		return "", fmt.Errorf("%s", overview.Error.Message)
	}
	breakdowns := map[string]any{}
	for _, dim := range []string{"actor", "studio", "tag"} {
		result := gw.Invoke(ctx, core.Call{
			Name: "get_insights_breakdown",
			Args: mustJSON(map[string]any{
				"range":     rangeValue,
				"timezone":  timezone,
				"dimension": dim,
				"limit":     10,
			}),
			SessionID: sessionID,
			Channel:   core.ChannelAction,
			Sanitize:  core.SanitizeFull,
		})
		if result.Error != nil {
			return "", fmt.Errorf("%s", result.Error.Message)
		}
		breakdowns[dim] = result.Data
	}
	encoded, err := json.Marshal(map[string]any{
		"overview":   overview.Data,
		"breakdowns": breakdowns,
	})
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func mustJSON(v any) json.RawMessage {
	raw, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return raw
}

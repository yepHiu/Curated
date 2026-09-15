package app

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
	"curated-backend/internal/storage"
)

type commentCompleter interface {
	Complete(ctx context.Context, messages []llm.ChatMessage, maxTokens int) (string, error)
}

// RunAIAction runs a single-shot action preset (E3 write-preview or read-only narrative).
func (a *App) RunAIAction(ctx context.Context, name string, req contracts.AIActionRequest) (preview contracts.AIActionPreviewDTO, retErr error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	ctx, observation, finish := a.beginAIRun(ctx, "action", name)
	defer func() {
		if preview.SessionID != "" {
			observation.row.SessionID = preview.SessionID
		}
		finish(retErr)
	}()
	if !prompts.KnownAction(name) {
		return contracts.AIActionPreviewDTO{}, fmt.Errorf("unknown action")
	}
	if err := a.aiPermission(name != prompts.ActionInsightsNarrative); err != nil {
		return preview, err
	}
	cfg, err := normalizeAIProviderConfig(a.currentAIProviderConfig())
	if err != nil {
		return contracts.AIActionPreviewDTO{}, fmt.Errorf("%w: %v", llm.ErrInvalidConfig, err)
	}
	if strings.TrimSpace(cfg.BaseURL) == "" || strings.TrimSpace(cfg.Model) == "" {
		return contracts.AIActionPreviewDTO{}, ErrAIProviderNotConfigured
	}
	observation.row.Model = cfg.Model
	client, err := newAIHTTPClient(a.currentProxyConfig(), 0)
	if err != nil {
		return contracts.AIActionPreviewDTO{}, err
	}
	completer := llm.NewClient(llm.ClientConfig{
		BaseURL: cfg.BaseURL,
		APIKey:  cfg.APIKey,
		Model:   cfg.Model,
	}, client)
	completer.Observe = observation.observe
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
	comicID := strings.TrimSpace(req.ComicID)
	photoID := strings.TrimSpace(req.PhotoID)
	filled := 0
	if movieID != "" {
		filled++
	}
	if comicID != "" {
		filled++
	}
	if photoID != "" {
		filled++
	}
	if filled != 1 {
		return contracts.AIActionPreviewDTO{}, fmt.Errorf("movieId, comicId, or photoId is required")
	}

	original := strings.TrimSpace(req.Body)
	toolName := core.SaveMovieCommentName
	argKey := "movieId"
	entityID := movieID
	limit := contracts.MaxMovieCommentRunes
	switch {
	case movieID != "":
		exists, err := a.MovieExists(ctx, movieID)
		if err != nil {
			return contracts.AIActionPreviewDTO{}, err
		}
		if !exists {
			return contracts.AIActionPreviewDTO{}, fmt.Errorf("movie not found")
		}
		if original == "" {
			current, err := a.GetMovieComment(ctx, movieID)
			if err != nil {
				return contracts.AIActionPreviewDTO{}, err
			}
			original = current.Body
		}
	case comicID != "":
		toolName = core.SaveComicCommentName
		argKey = "comicId"
		entityID = comicID
		limit = contracts.MaxBookCommentRunes
		if _, err := a.GetComicBookDetail(ctx, comicID); err != nil {
			return contracts.AIActionPreviewDTO{}, err
		}
		if original == "" {
			current, err := a.GetComicComment(ctx, comicID)
			if err != nil {
				return contracts.AIActionPreviewDTO{}, err
			}
			original = current.Body
		}
	default:
		toolName = core.SavePhotoCommentName
		argKey = "photoId"
		entityID = photoID
		limit = contracts.MaxBookCommentRunes
		if _, err := a.GetPhotoBookDetail(ctx, photoID); err != nil {
			return contracts.AIActionPreviewDTO{}, err
		}
		if original == "" {
			current, err := a.GetPhotoComment(ctx, photoID)
			if err != nil {
				return contracts.AIActionPreviewDTO{}, err
			}
			original = current.Body
		}
	}
	if original == "" {
		return contracts.AIActionPreviewDTO{}, fmt.Errorf("comment is empty")
	}
	if utf8.RuneCountInString(original) > limit {
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
	if utf8.RuneCountInString(proposed) > limit {
		runes := []rune(proposed)
		proposed = string(runes[:limit])
	}

	args, err := json.Marshal(map[string]string{argKey: entityID, "body": proposed})
	if err != nil {
		return contracts.AIActionPreviewDTO{}, err
	}
	sessionID := newAgentID("act_")
	result := a.ensureAgentGateway().Invoke(ctx, core.Call{
		Name:      toolName,
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
		Name:         toolName,
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
	if err := a.aiPermission(true); err != nil {
		a.auditAIDeniedApply(ctx, req, err)
		return contracts.AIToolApplyDTO{}, err
	}
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
	gw := a.ensureAgentGateway()
	if def, ok := gw.Registry().Get(name); ok && def.NormalizeArgs != nil {
		var err error
		args, err = def.NormalizeArgs(args)
		if err != nil {
			return contracts.AIToolApplyDTO{}, err
		}
	}
	// Serialize only confirmations; reads and model generation remain concurrent.
	a.agentRT.applyMu.Lock()
	defer a.agentRT.applyMu.Unlock()
	if err := a.aiPermission(true); err != nil {
		a.auditAIDeniedApply(ctx, req, err)
		return contracts.AIToolApplyDTO{}, err
	}
	if err := ctx.Err(); err != nil {
		return contracts.AIToolApplyDTO{}, err
	}
	key := storage.NewAIApplyReceiptKey(strings.TrimSpace(req.ConfirmToken), sessionID, name, core.HashArgs(args))
	if a.store != nil {
		receipt, found, err := a.store.GetAIApplyReceipt(ctx, key)
		if err != nil {
			return contracts.AIToolApplyDTO{}, err
		}
		if found {
			receipt.Replayed = true
			return receipt, nil
		}
	}
	// An accepted confirmation finishes within a deadline even if its response
	// connection disappears; the write and its receipt commit atomically.
	applyCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), core.WriteTimeout)
	defer cancel()
	applyCtx = storage.WithAIApplyReceipt(applyCtx, key)
	result := gw.Invoke(applyCtx, core.Call{
		Name:       name,
		Args:       args,
		SessionID:  sessionID,
		Channel:    core.ChannelAction,
		ConfirmTok: strings.TrimSpace(req.ConfirmToken),
		Sanitize:   core.SanitizeFull,
	})
	if a.store != nil {
		receipt, found, err := a.store.GetAIApplyReceipt(applyCtx, key)
		if err != nil {
			return contracts.AIToolApplyDTO{}, err
		}
		if found {
			return receipt, nil
		}
	}
	if result.Error != nil {
		return contracts.AIToolApplyDTO{}, result.Error
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

// runDisplayAction 把翻译 Action 分到影片简介/标题或书库展示标题。
func (a *App) runDisplayAction(ctx context.Context, completer commentCompleter, name string, req contracts.AIActionRequest) (contracts.AIActionPreviewDTO, error) {
	if name == prompts.ActionTranslateSummary {
		return a.runMovieDisplayAction(ctx, completer, name, req)
	}
	if name != prompts.ActionTranslateTitle {
		return contracts.AIActionPreviewDTO{}, fmt.Errorf("unknown action")
	}
	movieID := strings.TrimSpace(req.MovieID)
	comicID := strings.TrimSpace(req.ComicID)
	photoID := strings.TrimSpace(req.PhotoID)
	filled := 0
	if movieID != "" {
		filled++
	}
	if comicID != "" {
		filled++
	}
	if photoID != "" {
		filled++
	}
	if filled != 1 {
		return contracts.AIActionPreviewDTO{}, fmt.Errorf("movieId, comicId, or photoId is required")
	}
	if movieID != "" {
		return a.runMovieDisplayAction(ctx, completer, name, req)
	}
	return a.runBookTitleAction(ctx, completer, req, comicID, photoID)
}

// runMovieDisplayAction 为影片标题或简介生成翻译预览，不改刮削列。
func (a *App) runMovieDisplayAction(ctx context.Context, completer commentCompleter, name string, req contracts.AIActionRequest) (contracts.AIActionPreviewDTO, error) {
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

// runBookTitleAction 为漫画或写真展示标题生成翻译预览，写入独立 user_title。
func (a *App) runBookTitleAction(ctx context.Context, completer commentCompleter, req contracts.AIActionRequest, comicID, photoID string) (contracts.AIActionPreviewDTO, error) {
	locale := strings.TrimSpace(req.TargetLocale)
	if locale == "" {
		locale = strings.TrimSpace(req.Locale)
	}
	draft := strings.TrimSpace(req.Body)
	original := draft
	toolName := core.UpdateComicTitleName
	idField := "comicId"
	entityID := comicID
	if photoID != "" {
		toolName = core.UpdatePhotoTitleName
		idField = "photoId"
		entityID = photoID
		detail, err := a.GetPhotoBookDetail(ctx, photoID)
		if err != nil {
			return contracts.AIActionPreviewDTO{}, err
		}
		if original == "" {
			original = strings.TrimSpace(detail.Title)
		}
	} else {
		detail, err := a.GetComicBookDetail(ctx, comicID)
		if err != nil {
			return contracts.AIActionPreviewDTO{}, err
		}
		if original == "" {
			original = strings.TrimSpace(detail.Title)
		}
	}
	if original == "" {
		return contracts.AIActionPreviewDTO{}, fmt.Errorf("title is empty")
	}
	proposed, err := completer.Complete(ctx, []llm.ChatMessage{
		{Role: "system", Content: prompts.TranslateTitlePrompt(original, locale)},
		{Role: "user", Content: prompts.FormatPlainUser("Title", original)},
	}, 0)
	if err != nil {
		return contracts.AIActionPreviewDTO{}, err
	}
	proposed = strings.TrimSpace(sanitizeActionText(proposed))
	if proposed == "" {
		return contracts.AIActionPreviewDTO{}, fmt.Errorf("provider returned empty text")
	}
	argsMap := map[string]string{idField: entityID, "title": proposed}
	args, err := json.Marshal(argsMap)
	if err != nil {
		return contracts.AIActionPreviewDTO{}, err
	}
	sessionID := newAgentID("act_")
	result := a.ensureAgentGateway().Invoke(ctx, core.Call{
		Name:      toolName,
		Args:      args,
		SessionID: sessionID,
		Channel:   core.ChannelAction,
		Sanitize:  core.SanitizeFull,
	})
	if result.Error != nil {
		return contracts.AIActionPreviewDTO{}, fmt.Errorf("%s", result.Error.Message)
	}
	return contracts.AIActionPreviewDTO{
		Action:       prompts.ActionTranslateTitle,
		Name:         toolName,
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
	projection := a.aiProjection(a.currentAIProviderConfig().BaseURL)
	overview := gw.Invoke(ctx, core.Call{
		Name:      "get_insights_overview",
		Args:      mustJSON(map[string]string{"range": rangeValue, "timezone": timezone}),
		SessionID: sessionID,
		Channel:   core.ChannelAction,
		Sanitize:  projection,
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
			Sanitize:  projection,
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

package app

import (
	"context"
	"curated-backend/internal/agent/core"
	"curated-backend/internal/agent/prompts"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// Each field has an independent durable checkpoint; retries retain successful fields.
func (a *App) runOrganizationTranslations(ctx context.Context, id string) bool {
	if err := a.store.UpdateTagOrganization(ctx, id, "running", "translating_metadata", ""); err != nil {
		return false
	}
	for ctx.Err() == nil {
		job, err := a.store.GetTagOrganization(ctx, id)
		if err != nil || job.Status != "running" {
			return false
		}
		input, err := a.store.NextOrganizationTranslation(ctx, id)
		if errors.Is(err, sql.ErrNoRows) {
			return true
		}
		if err != nil {
			return false
		}
		if !input.Available || strings.TrimSpace(input.Source) == "" || len(input.Source) > 24000 {
			reason := "TRANSLATION_SOURCE_EMPTY"
			if !input.Available {
				reason = "MOVIE_UNAVAILABLE"
			}
			if len(input.Source) > 24000 {
				reason = "SOURCE_TOO_LONG"
			}
			if a.store.CompleteOrganizationTranslation(ctx, id, input, "", "failed", reason) != nil {
				return false
			}
			continue
		}
		prompt := prompts.TranslateTitlePrompt(job.Locale)
		if input.Kind == "summary" {
			prompt = prompts.TranslateSummaryPrompt(job.Locale)
		}
		raw, err := a.topicCompleteAttempt(ctx, prompts.Definition{Text: prompt, Version: "organization-translation-v1"}, map[string]string{"text": input.Source})
		if ctx.Err() != nil {
			return false
		}
		if err != nil {
			_ = a.store.UpdateTagOrganization(ctx, id, "blocked", "translating_metadata", topicOrganizationErrorCode(err))
			return false
		}
		proposed := sanitizeActionText(raw)
		if proposed == "" {
			if a.store.CompleteOrganizationTranslation(ctx, id, input, "", "failed", "TRANSLATION_EMPTY") != nil {
				return false
			}
			continue
		}
		if input.Kind == "summary" {
			proposed = clipUTF8Bytes(proposed, core.MaxMovieSummaryBytes)
		}
		reg := core.NewRegistry()
		_ = reg.Register(core.ToolDefinition{Name: core.OrganizationTranslationApplyName, Permission: core.PermissionUserTagTask, Domain: core.DomainUserWrite,
			ParamsSchema: core.Schema{Type: "object", Required: []string{"movieId"}, Properties: map[string]core.Schema{"movieId": {Type: "string"}}},
			Handler: func(ctx context.Context, call core.Call) (core.Result, error) {
				err := a.store.CompleteOrganizationTranslation(ctx, id, input, proposed, "succeeded", "")
				return core.Result{OK: err == nil}, err
			}})
		gw := a.ensureAgentGateway().WithRegistry(reg)
		args, _ := json.Marshal(map[string]string{"movieId": input.MovieID})
		for {
			a.agentRT.applyMu.Lock()
			result := gw.Invoke(core.WithUserTagTaskGrant(ctx, id), core.Call{Name: core.OrganizationTranslationApplyName, SessionID: id, Channel: core.ChannelAction, Args: args})
			a.agentRT.applyMu.Unlock()
			if result.Error == nil {
				break
			}
			if result.Error.Code != "AI_RATE_LIMITED" {
				_ = a.store.UpdateTagOrganization(ctx, id, "blocked", "translating_metadata", result.Error.Code)
				return false
			}
			_ = a.store.UpdateTagOrganization(ctx, id, "running", "waiting_quota", "")
			a.publishTagOrganization(id)
			select {
			case <-ctx.Done():
				return false
			case <-time.After(time.Second):
			}
		}
		if a.store.UpdateTagOrganization(ctx, id, "running", "translating_metadata", "") != nil {
			return false
		}
		a.publishTagOrganization(id)
	}
	return false
}

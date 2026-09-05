package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"curated-backend/internal/agent/core"
	"curated-backend/internal/agent/prompts"
	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/llm"
)

func (a *App) AIGovernanceSettings() contracts.AIGovernanceDTO {
	a.aiProviderMu.RLock()
	defer a.aiProviderMu.RUnlock()
	value := config.DefaultAIGovernance()
	if a.cfg.AIGovernance != nil {
		value = *a.cfg.AIGovernance
	}
	return contracts.AIGovernanceDTO(value)
}
func (a *App) SetAIGovernanceSettings(value contracts.AIGovernanceDTO) error {
	cfg := config.AIGovernanceConfig(value)
	if err := cfg.Validate(); err != nil {
		return err
	}
	if a.librarySettingsPath == "" {
		return fmt.Errorf("library settings path not configured")
	}
	a.agentRT.applyMu.Lock()
	defer a.agentRT.applyMu.Unlock()
	a.aiProviderMu.Lock()
	defer a.aiProviderMu.Unlock()
	if err := config.WriteLibrarySettingsMerge(a.librarySettingsPath, func(m map[string]any) error { m["aiGovernance"] = cfg; return nil }); err != nil {
		return err
	}
	a.cfg.AIGovernance = &cfg
	// Stop active generations before another model/tool request can use old policy.
	a.agentRT.runsMu.Lock()
	for _, cancel := range a.agentRT.activeRuns {
		cancel()
	}
	a.agentRT.runsMu.Unlock()
	return nil
}

func (a *App) aiPermission(write bool) error {
	cfg := a.AIGovernanceSettings()
	if !cfg.Enabled {
		return &core.ToolError{Code: "AI_DISABLED", Message: "AI is disabled in settings"}
	}
	if write && cfg.ReadOnly {
		return &core.ToolError{Code: "AI_READ_ONLY", Message: "AI is read-only in settings"}
	}
	return nil
}
func (a *App) aiProjection(baseURL string) string {
	if a.AIGovernanceSettings().Privacy == "minimal" {
		return core.SanitizeMinimal
	}
	return sanitizeModeForProvider(baseURL)
}

func (a *App) CleanupAIRecords(ctx context.Context) (contracts.AICleanupDTO, error) {
	if a.store == nil {
		return contracts.AICleanupDTO{}, nil
	}
	return a.store.CleanupAIRecords(ctx, time.Now().UTC().AddDate(0, 0, -a.AIGovernanceSettings().RetentionDays))
}
func (a *App) GetAIReport(ctx context.Context, q contracts.AIReportQuery) (contracts.AIReportDTO, error) {
	if a.store == nil {
		return contracts.AIReportDTO{Items: []contracts.AIRunDTO{}, Limit: q.Limit, Offset: q.Offset}, nil
	}
	if _, err := a.CleanupAIRecords(ctx); err != nil {
		return contracts.AIReportDTO{}, err
	}
	return a.store.GetAIReport(ctx, q)
}
func (a *App) ListAIAudit(ctx context.Context, q contracts.AIReportQuery) (contracts.AIAuditPageDTO, error) {
	if a.store == nil {
		return contracts.AIAuditPageDTO{Items: []contracts.AIAuditDTO{}, Limit: q.Limit, Offset: q.Offset}, nil
	}
	if _, err := a.CleanupAIRecords(ctx); err != nil {
		return contracts.AIAuditPageDTO{}, err
	}
	return a.store.ListAIAudit(ctx, q)
}

type aiRunContextKey struct{}
type aiRunObservation struct {
	started time.Time
	row     contracts.AIRunDTO
}

func (a *App) beginAIRun(ctx context.Context, channel, action string) (context.Context, *aiRunObservation, func(error)) {
	cfg := a.currentAIProviderConfig()
	r := &aiRunObservation{started: time.Now(), row: contracts.AIRunDTO{ID: newAgentID("run_"), Channel: channel, Action: action, Provider: config.NormalizeAIProviderKind(cfg.Kind), Model: cfg.Model, PromptVersion: prompts.Version, Status: "completed"}}
	if channel == "action" {
		r.row.PromptVersion = "agent-actions-v1"
	}
	if channel == "test" {
		r.row.PromptVersion = "provider-probe-v1"
	}
	r.row.StartedAt = r.started.UTC().Format(time.RFC3339Nano)
	ctx, cancel := context.WithCancel(context.WithValue(ctx, aiRunContextKey{}, r))
	a.agentRT.runsMu.Lock()
	if a.agentRT.activeRuns == nil {
		a.agentRT.activeRuns = map[string]context.CancelFunc{}
	}
	a.agentRT.activeRuns[r.row.ID] = cancel
	a.agentRT.runsMu.Unlock()
	finish := func(err error) {
		a.agentRT.runsMu.Lock()
		delete(a.agentRT.activeRuns, r.row.ID)
		a.agentRT.runsMu.Unlock()
		defer cancel()
		if err != nil {
			r.row.Status = "failed"
			if r.row.ErrorCode == "" {
				r.row.ErrorCode = aiRunError(err)
			}
		}
		if ctx.Err() != nil {
			r.row.Status = "cancelled"
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				r.row.Status = "failed"
			}
			r.row.ErrorCode = llm.ErrorCategory(ctx.Err())
		}
		if r.row.Status == "failed" && r.row.ErrorCode == "" {
			r.row.ErrorCode = "empty_response"
		}
		r.row.DurationMs = time.Since(r.started).Milliseconds()
		if a.store == nil {
			return
		}
		saveCtx, saveCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer saveCancel()
		if err := a.store.InsertAIRun(saveCtx, r.row); err != nil {
			if a.logger != nil {
				a.logger.Warn("AI statistics persistence failed")
			}
			return
		}
		if _, err := a.CleanupAIRecords(saveCtx); err != nil && a.logger != nil {
			a.logger.Warn("AI record retention cleanup failed")
		}
	}
	return ctx, r, finish
}
func aiRunError(err error) string {
	var tool *core.ToolError
	if errors.As(err, &tool) {
		return tool.Code
	}
	category := llm.ErrorCategory(err)
	if category == "invalid_response" {
		return "operation_failed"
	}
	return category
}

func (a *App) auditAIDeniedApply(ctx context.Context, req contracts.AIToolApplyRequest, err error) {
	name := req.Name
	if name != core.SaveMovieCommentName && name != core.UpdateMovieDisplayOverridesName && name != core.CreateSavedViewName {
		name = "unknown"
	}
	_ = (agentAuditSink{store: a.store}).RecordInvocation(ctx, core.AuditRecord{Channel: core.ChannelAction, ToolName: name, Permission: core.PermissionWriteApply, Result: core.ResultRejected, ErrorCode: aiRunError(err)})
}
func (r *aiRunObservation) observe(o llm.Observation) {
	r.row.ModelCalls++
	if o.FirstTextMs != nil && r.row.FirstTextMs == nil {
		v := time.Since(r.started).Milliseconds() - o.DurationMs + *o.FirstTextMs
		if v < 0 {
			v = 0
		}
		r.row.FirstTextMs = &v
	}
	if o.Usage != nil {
		r.row.UsageCalls++
		r.row.PromptTokens += o.Usage.PromptTokens
		r.row.CompletionTokens += o.Usage.CompletionTokens
		r.row.TotalTokens += o.Usage.TotalTokens
	}
	if o.ErrorCode != "" {
		r.row.ErrorCode = o.ErrorCode
	}
}

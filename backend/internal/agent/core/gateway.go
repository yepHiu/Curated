package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrToolNotFound     = errors.New("tool not found")
	ErrInvalidArgs      = errors.New("invalid tool arguments")
	ErrPermissionDenied = errors.New("tool permission denied")
	ErrRateLimited      = errors.New("tool rate limited")
	ErrConfirmRequired  = errors.New("confirm token required")
	ErrConfirmExpired   = errors.New("confirm token expired")
)

// Gateway runs the seven-step tool pipeline. Every channel must enter here.
type Gateway struct {
	registry   *Registry
	confirm    *ConfirmStore
	audit      AuditSink
	settings   func() Settings
	budget     *budgetTracker
	movieRefs  *MovieRefStore
	actorRefs  *ActorRefStore
	sourceURLs *SourceURLStore
	now        func() time.Time
}

func NewGateway(registry *Registry, confirm *ConfirmStore, audit AuditSink, settings func() Settings) *Gateway {
	if settings == nil {
		settings = func() Settings { return Settings{} }
	}
	if confirm == nil {
		confirm = NewConfirmStore()
	}
	return &Gateway{
		registry:   registry,
		confirm:    confirm,
		audit:      audit,
		settings:   settings,
		budget:     newBudgetTracker(),
		movieRefs:  NewMovieRefStore(),
		actorRefs:  NewActorRefStore(),
		sourceURLs: NewSourceURLStore(),
		now:        time.Now,
	}
}

func (g *Gateway) Invoke(ctx context.Context, call Call) Result {
	started := g.now()
	if call.Channel == "" {
		call.Channel = ChannelChat
	}
	if call.Sanitize == "" {
		call.Sanitize = SanitizeFull
	}

	result, perm, resultKind, errCode := g.invoke(ctx, call)
	if g.audit != nil {
		_ = g.audit.RecordInvocation(ctx, AuditRecord{
			Channel:     call.Channel,
			SessionID:   call.SessionID,
			ToolName:    call.Name,
			Permission:  perm,
			ArgsSummary: ArgsSummary(call.Args),
			Result:      resultKind,
			ErrorCode:   errCode,
			DurationMs:  g.now().Sub(started).Milliseconds(),
		})
	}
	return result
}

func (g *Gateway) invoke(ctx context.Context, call Call) (Result, string, string, string) {
	def, ok := g.registry.Get(call.Name)
	if !ok {
		return fail(ErrToolNotFound, "AI_TOOL_NOT_FOUND", "unknown tool"), "", ResultRejected, "AI_TOOL_NOT_FOUND"
	}
	settings := g.settings()

	if def.NormalizeArgs != nil {
		normalized, err := def.NormalizeArgs(call.Args)
		if err != nil {
			return fail(ErrInvalidArgs, "AI_TOOL_INVALID_ARGS", err.Error()), def.Permission, ResultRejected, "AI_TOOL_INVALID_ARGS"
		}
		call.Args = normalized
	}

	if _, err := ValidateArgs(def.ParamsSchema, call.Args); err != nil {
		return fail(ErrInvalidArgs, "AI_TOOL_INVALID_ARGS", err.Error()), def.Permission, ResultRejected, "AI_TOOL_INVALID_ARGS"
	}

	apply := strings.TrimSpace(call.ConfirmTok) != "" && (def.Permission == PermissionWriteApply || def.Apply != nil)

	if err := allowCall(def, call, settings); err != nil {
		code := "AI_TOOL_PERMISSION_DENIED"
		if errors.Is(err, ErrConfirmRequired) || (call.ConfirmTok == "" && def.Permission == PermissionWriteApply) {
			code = "AI_CONFIRM_REQUIRED"
		}
		return fail(ErrPermissionDenied, code, err.Error()), def.Permission, ResultRejected, code
	}

	step := g.budget.addStep(call.SessionID)
	if step > settings.stepLimit() {
		return fail(ErrRateLimited, "AI_RATE_LIMITED", "tool step limit reached"), def.Permission, ResultRejected, "AI_RATE_LIMITED"
	}
	if apply {
		if g.budget.writeCount(call.SessionID, g.now()) >= settings.writePerMinute() {
			return fail(ErrRateLimited, "AI_RATE_LIMITED", "write rate limit reached"), def.Permission, ResultRejected, "AI_RATE_LIMITED"
		}
	}

	if apply {
		if err := g.confirm.Consume(call.ConfirmTok, call.SessionID, call.Name, call.Args); err != nil {
			return fail(ErrConfirmExpired, "AI_CONFIRM_EXPIRED", err.Error()), def.Permission, ResultRejected, "AI_CONFIRM_EXPIRED"
		}
	}

	timeout := ReadTimeout
	if def.Permission != PermissionRead || apply {
		timeout = WriteTimeout
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	handler := def.Handler
	if apply && def.Apply != nil {
		handler = def.Apply
	}
	raw, err := handler(callCtx, call)
	if err != nil {
		return fail(err, "AI_CHAT_FAILED", err.Error()), def.Permission, ResultError, "AI_CHAT_FAILED"
	}
	if raw.Error != nil {
		kind := ResultError
		if raw.Error.Code == "AI_CONFIRM_REQUIRED" {
			kind = ResultPreviewed
		}
		return raw, def.Permission, kind, raw.Error.Code
	}

	if !apply && def.Permission == PermissionWritePreview && raw.ConfirmToken == "" && len(raw.Changes) > 0 {
		rec, issueErr := g.confirm.Issue(call.SessionID, call.Name, call.Args)
		if issueErr == nil {
			raw.ConfirmToken = rec.Token
			raw.ExpiresAt = rec.ExpiresAt.UTC().Format(time.RFC3339)
			raw.ConfirmArgs = append(json.RawMessage(nil), call.Args...)
		}
	}
	if apply {
		g.budget.addWrite(call.SessionID, g.now())
	}

	raw.Data = Project(raw.Data, call.Sanitize)
	if !raw.OK && raw.Error == nil {
		raw.OK = true
	}
	kind := ResultOK
	if apply {
		kind = ResultConfirmed
	} else if def.Permission == PermissionWritePreview {
		kind = ResultPreviewed
	} else if def.Permission == PermissionWriteApply {
		kind = ResultConfirmed
	}
	return raw, def.Permission, kind, ""
}

func fail(err error, code, message string) Result {
	if message == "" && err != nil {
		message = err.Error()
	}
	return Result{
		OK: false,
		Error: &ToolError{
			Code:    code,
			Message: message,
		},
	}
}

func (g *Gateway) Registry() *Registry { return g.registry }

func (g *Gateway) StepLimit() int {
	if g == nil {
		return DefaultStepLimit
	}
	return g.settings().stepLimit()
}

// ResetSessionSteps clears the per-turn tool-step counter. P-10 bounds one
// user turn, not the lifetime of a persisted session.
func (g *Gateway) ResetSessionSteps(sessionID string) {
	if g == nil || g.budget == nil {
		return
	}
	g.budget.resetSteps(sessionID)
}

func (g *Gateway) MovieRefs() *MovieRefStore {
	if g == nil {
		return nil
	}
	return g.movieRefs
}

func (g *Gateway) RememberMovieRefs(sessionID string, refs []MovieRef) {
	if g == nil || g.movieRefs == nil {
		return
	}
	g.movieRefs.Remember(sessionID, refs)
}

func (g *Gateway) ResetMovieRefs(sessionID string) {
	if g == nil || g.movieRefs == nil {
		return
	}
	g.movieRefs.Reset(sessionID)
}

func (g *Gateway) ActorRefs() *ActorRefStore {
	if g == nil {
		return nil
	}
	return g.actorRefs
}

func (g *Gateway) RememberActorNames(sessionID string, names []string) {
	if g == nil || g.actorRefs == nil {
		return
	}
	g.actorRefs.Remember(sessionID, names)
}

func (g *Gateway) ResetActorRefs(sessionID string) {
	if g == nil || g.actorRefs == nil {
		return
	}
	g.actorRefs.Reset(sessionID)
}

func (g *Gateway) SourceURLs() *SourceURLStore {
	if g == nil {
		return nil
	}
	return g.sourceURLs
}

func (g *Gateway) RememberSourceURLs(sessionID string, urls []string) {
	if g == nil || g.sourceURLs == nil {
		return
	}
	g.sourceURLs.Remember(sessionID, urls)
}

func (g *Gateway) ResetSourceURLs(sessionID string) {
	if g == nil || g.sourceURLs == nil {
		return
	}
	g.sourceURLs.Reset(sessionID)
}

func ResultJSON(result Result) json.RawMessage {
	encoded, err := json.Marshal(result)
	if err != nil {
		return json.RawMessage(`{"ok":false,"error":{"code":"AI_CHAT_FAILED","message":"encode failed"}}`)
	}
	return encoded
}

func (g *Gateway) IssuePreview(sessionID, toolName string, args json.RawMessage) (ConfirmRecord, error) {
	if g == nil || g.confirm == nil {
		return ConfirmRecord{}, fmt.Errorf("confirm store unavailable")
	}
	return g.confirm.Issue(sessionID, toolName, args)
}

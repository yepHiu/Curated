package app

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"curated-backend/internal/agent/core"
	"curated-backend/internal/agent/run"
	"curated-backend/internal/agent/tools"
	"curated-backend/internal/contracts"
	"curated-backend/internal/llm"
	"curated-backend/internal/storage"
)

type agentRuntime struct {
	runsMu     sync.Mutex
	activeRuns map[string]context.CancelFunc
	applyMu    sync.Mutex
	once       sync.Once
	gateway    *core.Gateway
}

type agentAuditSink struct {
	store *storage.SQLiteStore
}

func (s agentAuditSink) RecordInvocation(ctx context.Context, rec core.AuditRecord) error {
	if run, ok := ctx.Value(aiRunContextKey{}).(*aiRunObservation); ok {
		run.row.ToolCalls++
		run.row.SessionID = rec.SessionID
		if rec.ErrorCode != "" && run.row.ErrorCode == "" {
			run.row.ErrorCode = rec.ErrorCode
		}
	}
	if s.store == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	return s.store.InsertAIToolInvocation(
		ctx,
		rec.Channel,
		rec.SessionID,
		rec.ToolName,
		rec.Permission,
		"", // New audit records contain metadata only, never user text or arguments.
		rec.Result,
		rec.ErrorCode,
		rec.DurationMs,
	)
}

func (a *App) ensureAgentGateway() *core.Gateway {
	a.agentRT.once.Do(func() {
		reg := core.NewRegistry()
		if err := tools.RegisterQueryTools(reg, a); err != nil && a.logger != nil {
			a.logger.Warn("register agent query tools failed")
		}
		gw := core.NewGateway(reg, core.NewConfirmStore(), agentAuditSink{store: a.store}, func() core.Settings {
			cfg := a.AIGovernanceSettings()
			return core.Settings{Disabled: !cfg.Enabled, ReadOnly: cfg.ReadOnly, GlobalWriteLimit: true, StepLimit: cfg.StepLimit, WritePerMinute: cfg.WritePerMinute}
		})
		if err := tools.RegisterPresentTools(reg, gw.MovieRefs()); err != nil && a.logger != nil {
			a.logger.Warn("register agent present tools failed")
		}
		if err := tools.RegisterProviderTools(reg, a, a, a, gw.MovieRefs(), gw.ActorRefs(), gw.SourceURLs()); err != nil && a.logger != nil {
			a.logger.Warn("register agent provider tools failed")
		}
		if err := tools.RegisterWriteTools(reg, a); err != nil && a.logger != nil {
			a.logger.Warn("register agent write tools failed")
		}
		a.agentRT.gateway = gw
	})
	return a.agentRT.gateway
}

// StreamAIChat runs one experimental agent turn (E2: read tools + session persistence).
func (a *App) StreamAIChat(ctx context.Context, req contracts.AIChatRequest, emit func(contracts.AIChatSSEEvent)) (retErr error) {
	ctx, observation, finish := a.beginAIRun(ctx, "chat", "")
	var firstPublishedMs *int64
	// 聊天首字延迟按用户实际收到的已校验正文计算，而非上游草稿。
	defer func() { observation.row.FirstTextMs = firstPublishedMs; finish(retErr) }()
	cfg, err := normalizeAIProviderConfig(a.currentAIProviderConfig())
	if err != nil {
		return fmt.Errorf("%w: %v", llm.ErrInvalidConfig, err)
	}
	if strings.TrimSpace(cfg.BaseURL) == "" || strings.TrimSpace(cfg.Model) == "" {
		return ErrAIProviderNotConfigured
	}
	observation.row.Model = cfg.Model
	if err := a.aiPermission(false); err != nil {
		return err
	}
	client, err := newAIHTTPClient(a.currentProxyConfig(), 0)
	if err != nil {
		return err
	}
	streamer := llm.NewClient(llm.ClientConfig{
		BaseURL: cfg.BaseURL,
		APIKey:  cfg.APIKey,
		Model:   cfg.Model,
	}, client)
	streamer.Observe = observation.observe

	session, err := a.resolveAIChatSession(ctx, req)
	if err != nil {
		return err
	}
	observation.row.SessionID = session.ID
	lastUser := lastAIChatUser(req.Messages)
	if lastUser == nil {
		return fmt.Errorf("messages must include at least one user message")
	}
	if a.store != nil {
		if strings.TrimSpace(session.Title) == "" {
			_ = a.store.UpdateAIChatSessionTitle(ctx, session.ID, sessionTitleFrom(lastUser.Content))
		}
		if _, err := a.store.AppendAIChatMessage(ctx, session.ID, "user", lastUser.Content, "", ""); err != nil {
			return err
		}
	}

	history, err := a.llmHistoryForSession(ctx, session.ID)
	if err != nil || len(history) == 0 {
		history = requestHistoryWithoutSystem(req.Messages)
	}

	messageID := newAgentID("msg_")
	loop := run.NewLoop(a.ensureAgentGateway(), streamer, a.aiProjection(cfg.BaseURL), strings.TrimSpace(req.Locale))
	var assistant strings.Builder
	var events []contracts.AIChatSSEEvent
	wrapped := func(ev contracts.AIChatSSEEvent) {
		if ev.Outcome != nil {
			observation.row.Status = ev.Outcome.Status
			if ev.Outcome.ReasonCode == "answer_rejected" {
				observation.row.ErrorCode = "answer_rejected"
			}
		}
		if ev.SessionID == "" {
			ev.SessionID = session.ID
		}
		if ev.MessageID == "" {
			ev.MessageID = messageID
		}
		switch ev.Type {
		case "text_delta":
			if firstPublishedMs == nil && ev.Delta != "" {
				elapsed := time.Since(observation.started).Milliseconds()
				firstPublishedMs = &elapsed
			}
			assistant.WriteString(ev.Delta)
			// Only published text reaches storage; upstream drafts never enter this emitter.
		case "tool_call_result", "movie_cards", "message_done", "confirm_required":
			stored := ev
			if stored.ConfirmToken != "" {
				stored.ReceiptID = storage.NewAIApplyReceiptKey(stored.ConfirmToken, session.ID, stored.Name, "").TokenHash
			}
			// History is evidence, never a source of write authority.
			stored.ConfirmToken = ""
			stored.Arguments = nil
			events = append(events, stored)
		}
		if emit != nil {
			emit(ev)
		}
	}
	page := a.projectAIChatContext(ctx, req.Context)
	runErr := loop.Run(ctx, session.ID, messageID, history, page, wrapped)
	if err := a.persistAIChatTurn(ctx, session.ID, assistant.String(), events); err != nil {
		return errors.Join(runErr, err)
	}
	return runErr
}

// Finish persistence even after browser cancellation, but never wait indefinitely.
func (a *App) persistAIChatTurn(ctx context.Context, sessionID, text string, events []contracts.AIChatSSEEvent) error {
	if a.store == nil {
		return nil
	}
	if len(events) == 0 || events[len(events)-1].Type != "message_done" {
		status := "failed"
		if ctx.Err() != nil {
			status = "cancelled"
		}
		events = append(events, contracts.AIChatSSEEvent{Type: "message_done", Outcome: &contracts.AIChatOutcomeDTO{Status: status}})
	}
	finishCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	_, err := a.store.AppendAIChatMessage(finishCtx, sessionID, "assistant", text, "", "", events...)
	if err != nil {
		return fmt.Errorf("save AI turn: %w", err)
	}
	return nil
}

// projectAIChatContext resolves the v1 explicit selections against the local
// library before the loop can seed its per-turn reference stores. Browser
// values are suggestions only: unknown movie ids and actor names are removed
// rather than becoming trusted present/provider anchors.
func (a *App) projectAIChatContext(ctx context.Context, input *contracts.AIChatContext) *contracts.AIChatContext {
	if input == nil {
		return nil
	}
	page := *input
	page.Mentions = append([]contracts.AIChatMention(nil), input.Mentions...)
	page.SelectedMovieIDs = nil
	page.SelectedActors = nil
	if input.ActiveFilters != nil {
		filters := *input.ActiveFilters
		page.ActiveFilters = &filters
	}
	if input.ContextVersion != 1 || a == nil || a.store == nil {
		return &page
	}
	for _, id := range input.SelectedMovieIDs {
		exists, err := a.store.MovieExists(ctx, id)
		if err == nil && exists {
			page.SelectedMovieIDs = append(page.SelectedMovieIDs, id)
		}
	}
	for _, name := range input.SelectedActors {
		profile, err := a.store.GetActorProfile(ctx, name)
		if err != nil || strings.TrimSpace(profile.Name) == "" {
			continue
		}
		page.SelectedActors = append(page.SelectedActors, profile.Name)
	}
	return &page
}

func (a *App) ListAIChatSessions(ctx context.Context) (contracts.AIChatSessionListDTO, error) {
	if a.store == nil {
		return contracts.AIChatSessionListDTO{Items: []contracts.AIChatSessionDTO{}}, nil
	}
	items, err := a.store.ListAIChatSessions(ctx, 20)
	if err != nil {
		return contracts.AIChatSessionListDTO{}, err
	}
	if items == nil {
		items = []contracts.AIChatSessionDTO{}
	}
	return contracts.AIChatSessionListDTO{Items: items}, nil
}

func (a *App) CreateAIChatSession(ctx context.Context, title string) (contracts.AIChatSessionDTO, error) {
	if a.store == nil {
		return contracts.AIChatSessionDTO{}, fmt.Errorf("store unavailable")
	}
	return a.store.CreateAIChatSession(ctx, title)
}

func (a *App) GetAIChatSession(ctx context.Context, id string, cursor ...string) (contracts.AIChatSessionDetailDTO, error) {
	if a.store == nil {
		return contracts.AIChatSessionDetailDTO{}, sql.ErrNoRows
	}
	session, err := a.store.GetAIChatSession(ctx, id)
	if err != nil {
		return contracts.AIChatSessionDetailDTO{}, err
	}
	before := ""
	if len(cursor) > 0 {
		before = cursor[0]
	}
	messages, nextCursor, err := a.store.ListAIChatMessagePage(ctx, id, before)
	if err != nil {
		return contracts.AIChatSessionDetailDTO{}, err
	}
	if messages == nil {
		messages = []contracts.AIChatStoredMessageDTO{}
	}
	if err := a.store.RestoreAIReceiptStates(ctx, id, messages); err != nil {
		return contracts.AIChatSessionDetailDTO{}, err
	}
	projectAIConfirmationOutcomes(messages)
	return contracts.AIChatSessionDetailDTO{AIChatSessionDTO: session, Messages: messages, NextCursor: nextCursor}, nil
}

func (a *App) DeleteAIChatSession(ctx context.Context, id string) error {
	if a.store == nil {
		return sql.ErrNoRows
	}
	return a.store.DeleteAIChatSession(ctx, id)
}

func (a *App) resolveAIChatSession(ctx context.Context, req contracts.AIChatRequest) (contracts.AIChatSessionDTO, error) {
	if a.store == nil {
		return contracts.AIChatSessionDTO{ID: newAgentID("ses_")}, nil
	}
	id := strings.TrimSpace(req.SessionID)
	if id == "" {
		return a.store.CreateAIChatSession(ctx, sessionTitleFrom(lastAIChatUserContent(req.Messages)))
	}
	session, err := a.store.GetAIChatSession(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return contracts.AIChatSessionDTO{}, err
		}
		return contracts.AIChatSessionDTO{}, err
	}
	return session, nil
}

func (a *App) llmHistoryForSession(ctx context.Context, sessionID string) ([]llm.ChatMessage, error) {
	if a.store == nil {
		return nil, nil
	}
	rows, err := a.store.ListAIChatContext(ctx, sessionID, 80)
	if err != nil {
		return nil, err
	}
	out := make([]llm.ChatMessage, 0, len(rows))
	for _, row := range rows {
		switch row.Role {
		case "user", "assistant":
			if strings.TrimSpace(row.Content) == "" {
				continue
			}
			out = append(out, llm.ChatMessage{Role: row.Role, Content: row.Content})
		}
	}
	return out, nil
}

func lastAIChatUser(messages []contracts.AIChatMessage) *contracts.AIChatMessage {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" && strings.TrimSpace(messages[i].Content) != "" {
			msg := messages[i]
			return &msg
		}
	}
	return nil
}

func lastAIChatUserContent(messages []contracts.AIChatMessage) string {
	if msg := lastAIChatUser(messages); msg != nil {
		return msg.Content
	}
	return ""
}

func requestHistoryWithoutSystem(messages []contracts.AIChatMessage) []llm.ChatMessage {
	out := make([]llm.ChatMessage, 0, len(messages))
	for _, m := range messages {
		if m.Role == "system" {
			continue
		}
		out = append(out, llm.ChatMessage{Role: m.Role, Content: m.Content})
	}
	return out
}

func sessionTitleFrom(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	if utf8.RuneCountInString(content) <= 40 {
		return content
	}
	return string([]rune(content)[:40])
}

func sanitizeModeForProvider(baseURL string) string {
	u, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || u.Hostname() == "" {
		return core.SanitizeSanitized
	}
	host := u.Hostname()
	if host == "localhost" || host == "::1" {
		return core.SanitizeFull
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return core.SanitizeFull
	}
	return core.SanitizeSanitized
}

func newAgentID(prefix string) string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return prefix + "fallback"
	}
	return prefix + hex.EncodeToString(buf[:])
}

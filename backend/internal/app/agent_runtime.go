package app

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
	"unicode/utf8"

	"curated-backend/internal/agent/core"
	"curated-backend/internal/agent/run"
	"curated-backend/internal/agent/tools"
	"curated-backend/internal/contracts"
	"curated-backend/internal/llm"
	"curated-backend/internal/storage"
)

type agentRuntime struct {
	once    sync.Once
	gateway *core.Gateway
}

type agentAuditSink struct {
	store *storage.SQLiteStore
}

func (s agentAuditSink) RecordInvocation(ctx context.Context, rec core.AuditRecord) error {
	if s.store == nil {
		return nil
	}
	return s.store.InsertAIToolInvocation(
		ctx,
		rec.Channel,
		rec.SessionID,
		rec.ToolName,
		rec.Permission,
		rec.ArgsSummary,
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
			return core.Settings{}
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
func (a *App) StreamAIChat(ctx context.Context, req contracts.AIChatRequest, emit func(contracts.AIChatSSEEvent)) error {
	cfg, err := normalizeAIProviderConfig(a.currentAIProviderConfig())
	if err != nil {
		return fmt.Errorf("%w: %v", llm.ErrInvalidConfig, err)
	}
	if strings.TrimSpace(cfg.BaseURL) == "" || strings.TrimSpace(cfg.Model) == "" {
		return ErrAIProviderNotConfigured
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

	session, err := a.resolveAIChatSession(ctx, req)
	if err != nil {
		return err
	}
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
	loop := run.NewLoop(a.ensureAgentGateway(), streamer, sanitizeModeForProvider(cfg.BaseURL), strings.TrimSpace(req.Locale))
	var assistant strings.Builder
	wrapped := func(ev contracts.AIChatSSEEvent) {
		if ev.SessionID == "" {
			ev.SessionID = session.ID
		}
		if ev.MessageID == "" {
			ev.MessageID = messageID
		}
		switch ev.Type {
		case "text_delta":
			assistant.WriteString(ev.Delta)
		case "tool_call_result":
			if a.store != nil {
				content := ev.Summary
				if ev.Name == core.PresentMoviesName && len(ev.Movies) > 0 {
					if encoded, err := json.Marshal(map[string]any{"movies": ev.Movies}); err == nil {
						content = string(encoded)
					}
				}
				_, _ = a.store.AppendAIChatMessage(ctx, session.ID, "tool", content, ev.Name, ev.ToolCallID)
			}
		}
		if emit != nil {
			emit(ev)
		}
	}
	page := a.projectAIChatContext(ctx, req.Context)
	runErr := loop.Run(ctx, session.ID, messageID, history, page, wrapped)
	if a.store != nil && assistant.Len() > 0 {
		_, _ = a.store.AppendAIChatMessage(ctx, session.ID, "assistant", assistant.String(), "", "")
	}
	return runErr
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

func (a *App) GetAIChatSession(ctx context.Context, id string) (contracts.AIChatSessionDetailDTO, error) {
	if a.store == nil {
		return contracts.AIChatSessionDetailDTO{}, sql.ErrNoRows
	}
	session, err := a.store.GetAIChatSession(ctx, id)
	if err != nil {
		return contracts.AIChatSessionDetailDTO{}, err
	}
	messages, err := a.store.ListAIChatMessages(ctx, id, 80)
	if err != nil {
		return contracts.AIChatSessionDetailDTO{}, err
	}
	if messages == nil {
		messages = []contracts.AIChatStoredMessageDTO{}
	}
	return contracts.AIChatSessionDetailDTO{AIChatSessionDTO: session, Messages: messages}, nil
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

package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"curated-backend/internal/contracts"
)

func newAIID(prefix string) (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(buf[:]), nil
}

func (s *SQLiteStore) CreateAIChatSession(ctx context.Context, title string) (contracts.AIChatSessionDTO, error) {
	id, err := newAIID("ses_")
	if err != nil {
		return contracts.AIChatSessionDTO{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	title = strings.TrimSpace(title)
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO ai_chat_sessions (id, title, created_at, updated_at)
		VALUES (?, ?, ?, ?)`, id, title, now, now); err != nil {
		return contracts.AIChatSessionDTO{}, err
	}
	return contracts.AIChatSessionDTO{ID: id, Title: title, CreatedAt: now, UpdatedAt: now}, nil
}

func (s *SQLiteStore) GetAIChatSession(ctx context.Context, id string) (contracts.AIChatSessionDTO, error) {
	var dto contracts.AIChatSessionDTO
	err := s.db.QueryRowContext(ctx, `
		SELECT id, title, created_at, updated_at FROM ai_chat_sessions WHERE id = ?`, id,
	).Scan(&dto.ID, &dto.Title, &dto.CreatedAt, &dto.UpdatedAt)
	if err != nil {
		return contracts.AIChatSessionDTO{}, err
	}
	return dto, nil
}

func (s *SQLiteStore) ListAIChatSessions(ctx context.Context, limit int) ([]contracts.AIChatSessionDTO, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, title, created_at, updated_at
		FROM ai_chat_sessions
		ORDER BY updated_at DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]contracts.AIChatSessionDTO, 0, limit)
	for rows.Next() {
		var dto contracts.AIChatSessionDTO
		if err := rows.Scan(&dto.ID, &dto.Title, &dto.CreatedAt, &dto.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, dto)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) DeleteAIChatSession(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM ai_chat_sessions WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("ai chat session not found")
	}
	return nil
}

func (s *SQLiteStore) AppendAIChatMessage(ctx context.Context, sessionID, role, content, toolName, toolCallID string, events ...contracts.AIChatSSEEvent) (contracts.AIChatStoredMessageDTO, error) {
	encoded, err := json.Marshal(events)
	if err != nil {
		return contracts.AIChatStoredMessageDTO{}, err
	}
	id, err := newAIID("msg_")
	if err != nil {
		return contracts.AIChatStoredMessageDTO{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	var seq int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(seq), 0) + 1 FROM ai_chat_messages WHERE session_id = ?`, sessionID,
	).Scan(&seq); err != nil {
		return contracts.AIChatStoredMessageDTO{}, err
	}
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO ai_chat_messages (id, session_id, role, content, tool_name, tool_call_id, seq, created_at, events_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, sessionID, role, content, toolName, toolCallID, seq, now, string(encoded)); err != nil {
		return contracts.AIChatStoredMessageDTO{}, err
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE ai_chat_sessions SET updated_at = ? WHERE id = ?`, now, sessionID); err != nil {
		return contracts.AIChatStoredMessageDTO{}, err
	}
	return contracts.AIChatStoredMessageDTO{
		ID:         id,
		SessionID:  sessionID,
		Role:       role,
		Content:    content,
		ToolName:   toolName,
		ToolCallID: toolCallID,
		Seq:        seq,
		CreatedAt:  now,
		Events:     events,
	}, nil
}

func (s *SQLiteStore) ListAIChatMessages(ctx context.Context, sessionID string, limit int) ([]contracts.AIChatStoredMessageDTO, error) {
	return s.listRecentAIChatMessages(ctx, sessionID, limit, false)
}

// ListAIChatContext excludes tool/event rows before limiting the model window.
func (s *SQLiteStore) ListAIChatContext(ctx context.Context, sessionID string, limit int) ([]contracts.AIChatStoredMessageDTO, error) {
	return s.listRecentAIChatMessages(ctx, sessionID, limit, true)
}

func (s *SQLiteStore) listRecentAIChatMessages(ctx context.Context, sessionID string, limit int, dialogueOnly bool) ([]contracts.AIChatStoredMessageDTO, error) {
	if limit <= 0 || limit > 200 {
		limit = 80
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, session_id, role, content, tool_name, tool_call_id, seq, created_at, events_json FROM (
			SELECT * FROM ai_chat_messages
			WHERE session_id = ? AND (? = 0 OR role IN ('user', 'assistant'))
			ORDER BY seq DESC LIMIT ?
		) ORDER BY seq ASC`, sessionID, dialogueOnly, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]contracts.AIChatStoredMessageDTO, 0, limit)
	for rows.Next() {
		var dto contracts.AIChatStoredMessageDTO
		var encoded string
		if err := rows.Scan(&dto.ID, &dto.SessionID, &dto.Role, &dto.Content, &dto.ToolName, &dto.ToolCallID, &dto.Seq, &dto.CreatedAt, &encoded); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(encoded), &dto.Events); err != nil {
			return nil, fmt.Errorf("decode chat events: %w", err)
		}
		out = append(out, dto)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) UpdateAIChatSessionTitle(ctx context.Context, id, title string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `UPDATE ai_chat_sessions SET title = ?, updated_at = ? WHERE id = ?`,
		title, time.Now().UTC().Format(time.RFC3339Nano), id)
	return err
}

func (s *SQLiteStore) InsertAIToolInvocation(
	ctx context.Context,
	channel, sessionID, toolName, permission, argsSummary, result, errorCode string,
	durationMs int64,
) error {
	id, err := newAIID("inv_")
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO ai_tool_invocations (
			id, channel, session_id, tool_name, permission, args_summary, result, error_code, duration_ms, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, channel, sessionID, toolName, permission, argsSummary, result, errorCode, durationMs, now)
	return err
}

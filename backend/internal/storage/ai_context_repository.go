package storage

import (
	"context"
	"database/sql"
	"errors"

	"curated-backend/internal/contracts"
)

func (s *SQLiteStore) AIContextCheckpoint(ctx context.Context, sessionID string) (int, string, error) {
	var seq int
	var summary string
	err := s.db.QueryRowContext(ctx, `SELECT through_seq, summary FROM ai_context_checkpoints WHERE session_id=?`, sessionID).Scan(&seq, &summary)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	return seq, summary, err
}

// A stale request can never replace a newer checkpoint.
func (s *SQLiteStore) SaveAIContextCheckpoint(ctx context.Context, sessionID string, seq int, summary string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO ai_context_checkpoints(session_id,through_seq,summary) VALUES(?,?,?)
		ON CONFLICT(session_id) DO UPDATE SET through_seq=excluded.through_seq, summary=excluded.summary
		WHERE excluded.through_seq > ai_context_checkpoints.through_seq`, sessionID, seq, summary)
	return err
}

// Read forward so legacy conversations longer than the recent-history window
// can be checkpointed without silently skipping their original goals.
func (s *SQLiteStore) AIContextRange(ctx context.Context, sessionID string, after, before int) ([]contracts.AIChatStoredMessageDTO, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT seq,role,content FROM ai_chat_messages
		WHERE session_id=? AND seq>? AND seq<? AND role IN ('user','assistant') ORDER BY seq,id LIMIT 80`, sessionID, after, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []contracts.AIChatStoredMessageDTO
	for rows.Next() {
		var row contracts.AIChatStoredMessageDTO
		if err := rows.Scan(&row.Seq, &row.Role, &row.Content); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

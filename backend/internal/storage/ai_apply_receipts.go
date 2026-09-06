package storage

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"curated-backend/internal/contracts"
)

type AIApplyReceiptKey struct{ TokenHash, SessionID, ToolName, ArgsHash string }
type aiReceiptContextKey struct{}

func NewAIApplyReceiptKey(token, sessionID, name, argsHash string) AIApplyReceiptKey {
	sum := sha256.Sum256([]byte(token))
	return AIApplyReceiptKey{hex.EncodeToString(sum[:]), sessionID, name, argsHash}
}

// Receipt metadata can only originate in the application confirmation path.
func WithAIApplyReceipt(ctx context.Context, key AIApplyReceiptKey) context.Context {
	return context.WithValue(ctx, aiReceiptContextKey{}, key)
}

var ErrAIReceiptMismatch = errors.New("confirm token receipt does not match this call")

func (s *SQLiteStore) GetAIApplyReceipt(ctx context.Context, key AIApplyReceiptKey) (contracts.AIToolApplyDTO, bool, error) {
	var sessionID, name, argsHash, encoded string
	err := s.db.QueryRowContext(ctx, `SELECT session_id, tool_name, args_hash, result_json FROM ai_apply_receipts WHERE token_hash = ?`, key.TokenHash).Scan(&sessionID, &name, &argsHash, &encoded)
	if errors.Is(err, sql.ErrNoRows) {
		return contracts.AIToolApplyDTO{}, false, nil
	}
	if err != nil {
		return contracts.AIToolApplyDTO{}, false, err
	}
	if sessionID != key.SessionID || name != key.ToolName || argsHash != key.ArgsHash {
		return contracts.AIToolApplyDTO{}, false, ErrAIReceiptMismatch
	}
	var result contracts.AIToolApplyDTO
	if err := json.Unmarshal([]byte(encoded), &result); err != nil {
		return result, false, err
	}
	return result, true, nil
}

func (s *SQLiteStore) RestoreAIReceiptStates(ctx context.Context, sessionID string, messages []contracts.AIChatStoredMessageDTO) error {
	rows, err := s.db.QueryContext(ctx, `SELECT token_hash FROM ai_apply_receipts WHERE session_id=?`, sessionID)
	if err != nil {
		return err
	}
	defer rows.Close()
	committed := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		committed[id] = true
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for i := range messages {
		for j := range messages[i].Events {
			event := &messages[i].Events[j]
			event.Applied = event.Type == "confirm_required" && committed[event.ReceiptID]
		}
	}
	return nil
}

func saveAIApplyReceiptTx(ctx context.Context, tx *sql.Tx, data any) error {
	key, ok := ctx.Value(aiReceiptContextKey{}).(AIApplyReceiptKey)
	if !ok {
		return nil
	}
	encoded, err := json.Marshal(contracts.AIToolApplyDTO{OK: true, Name: key.ToolName, Data: data})
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO ai_apply_receipts(token_hash,session_id,tool_name,args_hash,result_json,created_at) VALUES (?,?,?,?,?,?)`, key.TokenHash, key.SessionID, key.ToolName, key.ArgsHash, string(encoded), nowUTC())
	if err != nil {
		return fmt.Errorf("save AI apply receipt: %w", err)
	}
	return nil
}

// Display tools return only the fields written by this confirmation in receipts.
// This is a snapshot of the committed values, never a later read of movie detail.
func saveAIDisplayReceiptTx(ctx context.Context, tx *sql.Tx, movieID string, patch contracts.PatchMovieInput) error {
	if _, ok := ctx.Value(aiReceiptContextKey{}).(AIApplyReceiptKey); !ok {
		return nil
	}
	var title, summary string
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(NULLIF(TRIM(user_title),''),title),COALESCE(NULLIF(TRIM(user_summary),''),summary) FROM movies WHERE id=?`, movieID).Scan(&title, &summary); err != nil {
		return err
	}
	data := map[string]any{"id": movieID}
	if patch.UserTitleSet {
		data["title"] = title
	}
	if patch.UserSummarySet {
		data["summary"] = summary
	}
	return saveAIApplyReceiptTx(ctx, tx, data)
}

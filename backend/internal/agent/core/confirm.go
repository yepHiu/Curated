package core

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// ConfirmRecord is a short-lived write ticket bound to session+tool+args.
type ConfirmRecord struct {
	Token     string
	SessionID string
	ToolName  string
	ArgsHash  string
	ExpiresAt time.Time
}

// ConfirmStore issues and consumes preview tokens. Process restart clears all tokens.
type ConfirmStore struct {
	mu     sync.Mutex
	tokens map[string]ConfirmRecord
	now    func() time.Time
	ttl    time.Duration
}

func NewConfirmStore() *ConfirmStore {
	return &ConfirmStore{
		tokens: map[string]ConfirmRecord{},
		now:    time.Now,
		ttl:    ConfirmTTL,
	}
}

func HashArgs(raw json.RawMessage) string {
	sum := sha256.Sum256(CanonicalJSON(raw))
	return hex.EncodeToString(sum[:])
}

// CanonicalJSON unmarshals then remarsals so key order and number formatting
// do not drift between the model, SSE, and the browser confirm POST.
func CanonicalJSON(raw json.RawMessage) []byte {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return []byte("{}")
	}
	var value any
	if err := json.Unmarshal(trimmed, &value); err != nil {
		return trimmed
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return trimmed
	}
	return encoded
}

func (s *ConfirmStore) Issue(sessionID, toolName string, args json.RawMessage) (ConfirmRecord, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return ConfirmRecord{}, err
	}
	rec := ConfirmRecord{
		Token:     "cfm_" + hex.EncodeToString(buf[:]),
		SessionID: sessionID,
		ToolName:  toolName,
		ArgsHash:  HashArgs(args),
		ExpiresAt: s.now().Add(s.ttl),
	}
	s.mu.Lock()
	s.tokens[rec.Token] = rec
	s.mu.Unlock()
	return rec, nil
}

func (s *ConfirmStore) Consume(token, sessionID, toolName string, args json.RawMessage) error {
	if token == "" {
		return fmt.Errorf("confirm token is required")
	}
	s.mu.Lock()
	rec, ok := s.tokens[token]
	if ok {
		delete(s.tokens, token)
	}
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("%w: confirm token is invalid", ErrConfirmExpired)
	}
	if s.now().After(rec.ExpiresAt) {
		return fmt.Errorf("%w: confirm token expired", ErrConfirmExpired)
	}
	if rec.SessionID != sessionID || rec.ToolName != toolName {
		return fmt.Errorf("confirm token does not match this call")
	}
	if rec.ArgsHash != HashArgs(args) {
		return fmt.Errorf("confirm token arguments drifted")
	}
	return nil
}

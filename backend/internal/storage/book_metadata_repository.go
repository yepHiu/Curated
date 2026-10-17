package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"curated-backend/internal/contracts"
)

type bookMetadataScanner struct{ target *contracts.BookMetadata }

func (s *bookMetadataScanner) Scan(value any) error {
	var raw []byte
	switch v := value.(type) {
	case string:
		raw = []byte(v)
	case []byte:
		raw = v
	default:
		return fmt.Errorf("invalid book metadata storage")
	}
	return json.Unmarshal(raw, s.target)
}

// Only fixed table names can enter the queries below.
func bookMetadataTable(kind string) (string, error) {
	switch kind {
	case "comic":
		return "comic_books", nil
	case "photo":
		return "photo_books", nil
	}
	return "", fmt.Errorf("invalid book kind")
}

func (s *SQLiteStore) ApplyBookMetadata(ctx context.Context, kind, id string, metadata contracts.BookMetadata, expected string) (contracts.BookMetadataState, error) {
	table, err := bookMetadataTable(kind)
	if err != nil {
		return contracts.BookMetadataState{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return contracts.BookMetadataState{}, err
	}
	defer tx.Rollback()
	var state contracts.BookMetadataState
	err = tx.QueryRowContext(ctx, `SELECT COALESCE(NULLIF(TRIM(user_title), ''), title), title, source_file_name, metadata_json FROM `+table+` WHERE id = ?`, id).Scan(&state.Title, &state.OriginalTitle, &state.SourceFileName, &bookMetadataScanner{target: &state.Metadata})
	if errors.Is(err, sql.ErrNoRows) {
		if kind == "comic" {
			err = ErrComicBookNotFound
		} else {
			err = ErrPhotoBookNotFound
		}
	}
	if err != nil {
		return state, err
	}
	current, err := json.Marshal(state)
	if err != nil {
		return state, err
	}
	if string(current) != expected {
		return state, ErrAIWriteConflict
	}
	if err := contracts.ValidateBookMetadata(kind, metadata, state); err != nil {
		return state, err
	}
	raw, err := json.Marshal(metadata)
	if err != nil {
		return state, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE `+table+` SET user_title = ?, metadata_json = ?, updated_at = ? WHERE id = ?`, metadata.TranslatedTitle, string(raw), nowUTC(), id); err != nil {
		return state, err
	}
	state.Title, state.Metadata = metadata.TranslatedTitle, metadata
	if err = saveAIApplyReceiptTx(ctx, tx, state); err != nil {
		return state, err
	}
	return state, tx.Commit()
}

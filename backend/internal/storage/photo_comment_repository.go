package storage

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"curated-backend/internal/contracts"
)

// GetPhotoComment returns the saved note for a photo book, or an empty DTO when none exists.
func (s *SQLiteStore) GetPhotoComment(ctx context.Context, photoID string) (contracts.PhotoCommentDTO, error) {
	photoID = strings.TrimSpace(photoID)
	if photoID == "" {
		return contracts.PhotoCommentDTO{}, nil
	}
	var body, updatedAt string
	err := s.db.QueryRowContext(ctx,
		`SELECT body, updated_at FROM photo_book_comments WHERE photo_id = ?`, photoID,
	).Scan(&body, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return contracts.PhotoCommentDTO{Body: "", UpdatedAt: ""}, nil
	}
	if err != nil {
		return contracts.PhotoCommentDTO{}, err
	}
	return contracts.PhotoCommentDTO{Body: body, UpdatedAt: updatedAt}, nil
}

// UpsertPhotoComment replaces the note for an existing photo book after trimming and rune-length checks.
func (s *SQLiteStore) UpsertPhotoComment(ctx context.Context, photoID string, body string) (contracts.PhotoCommentDTO, error) {
	photoID = strings.TrimSpace(photoID)
	if photoID == "" {
		return contracts.PhotoCommentDTO{}, ErrPhotoBookNotFound
	}
	body = strings.TrimSpace(body)
	if utf8.RuneCountInString(body) > contracts.MaxBookCommentRunes {
		return contracts.PhotoCommentDTO{}, ErrBookCommentTooLong
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return contracts.PhotoCommentDTO{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var one int
	switch err := tx.QueryRowContext(ctx, `SELECT 1 FROM photo_books WHERE id = ? LIMIT 1`, photoID).Scan(&one); {
	case errors.Is(err, sql.ErrNoRows):
		return contracts.PhotoCommentDTO{}, ErrPhotoBookNotFound
	case err != nil:
		return contracts.PhotoCommentDTO{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO photo_book_comments (photo_id, body, updated_at) VALUES (?, ?, ?)
		ON CONFLICT(photo_id) DO UPDATE SET body = excluded.body, updated_at = excluded.updated_at`,
		photoID, body, now,
	)
	if err != nil {
		return contracts.PhotoCommentDTO{}, err
	}
	if err := tx.Commit(); err != nil {
		return contracts.PhotoCommentDTO{}, err
	}
	return contracts.PhotoCommentDTO{Body: body, UpdatedAt: now}, nil
}

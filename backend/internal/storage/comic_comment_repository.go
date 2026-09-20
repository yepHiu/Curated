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

// ErrBookCommentTooLong is returned when a comic or photo note exceeds MaxBookCommentRunes.
var ErrBookCommentTooLong = errors.New("comment body too long")

// GetComicComment returns the saved note for a comic, or an empty DTO when none exists.
func (s *SQLiteStore) GetComicComment(ctx context.Context, comicID string) (contracts.ComicCommentDTO, error) {
	comicID = strings.TrimSpace(comicID)
	if comicID == "" {
		return contracts.ComicCommentDTO{}, nil
	}
	var body, updatedAt string
	err := s.db.QueryRowContext(ctx,
		`SELECT body, updated_at FROM comic_book_comments WHERE comic_id = ?`, comicID,
	).Scan(&body, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return contracts.ComicCommentDTO{Body: "", UpdatedAt: ""}, nil
	}
	if err != nil {
		return contracts.ComicCommentDTO{}, err
	}
	return contracts.ComicCommentDTO{Body: body, UpdatedAt: updatedAt}, nil
}

// UpsertComicComment replaces the note for an existing comic book after trimming and rune-length checks.
func (s *SQLiteStore) UpsertComicComment(ctx context.Context, comicID string, body string, expected ...string) (contracts.ComicCommentDTO, error) {
	comicID = strings.TrimSpace(comicID)
	if comicID == "" {
		return contracts.ComicCommentDTO{}, ErrComicBookNotFound
	}
	body = strings.TrimSpace(body)
	if utf8.RuneCountInString(body) > contracts.MaxBookCommentRunes {
		return contracts.ComicCommentDTO{}, ErrBookCommentTooLong
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return contracts.ComicCommentDTO{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var one int
	switch err := tx.QueryRowContext(ctx, `SELECT 1 FROM comic_books WHERE id = ? LIMIT 1`, comicID).Scan(&one); {
	case errors.Is(err, sql.ErrNoRows):
		return contracts.ComicCommentDTO{}, ErrComicBookNotFound
	case err != nil:
		return contracts.ComicCommentDTO{}, err
	}

	if len(expected) > 0 {
		var before string
		err := tx.QueryRowContext(ctx, `SELECT body FROM comic_book_comments WHERE comic_id = ?`, comicID).Scan(&before)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return contracts.ComicCommentDTO{}, err
		}
		if before != expected[0] {
			return contracts.ComicCommentDTO{}, ErrAIWriteConflict
		}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO comic_book_comments (comic_id, body, updated_at) VALUES (?, ?, ?)
		ON CONFLICT(comic_id) DO UPDATE SET body = excluded.body, updated_at = excluded.updated_at`,
		comicID, body, now,
	)
	if err != nil {
		return contracts.ComicCommentDTO{}, err
	}
	if err := saveAIApplyReceiptTx(ctx, tx, contracts.ComicCommentDTO{Body: body, UpdatedAt: now}); err != nil {
		return contracts.ComicCommentDTO{}, err
	}
	if err := tx.Commit(); err != nil {
		return contracts.ComicCommentDTO{}, err
	}
	return contracts.ComicCommentDTO{Body: body, UpdatedAt: now}, nil
}

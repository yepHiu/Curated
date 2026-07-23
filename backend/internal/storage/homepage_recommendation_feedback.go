package storage

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"curated-backend/internal/contracts"
)

const MaxHomepageRecommendationFeedback = 500

type HomepageRecommendationFeedbackRecord struct {
	ID               string
	Action           string
	TargetType       string
	TargetValue      string
	NormalizedTarget string
	SourceMovieID    string
	ExpiresAt        string
	CreatedAt        string
	UpdatedAt        string
}

func (s *SQLiteStore) ListActiveHomepageRecommendationFeedback(ctx context.Context, now time.Time) ([]HomepageRecommendationFeedbackRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, action, target_type, target_value, normalized_target, source_movie_id,
		       expires_at, created_at, updated_at
		FROM homepage_recommendation_feedback
		WHERE expires_at = '' OR julianday(expires_at) > julianday(?)
		ORDER BY created_at DESC, id ASC`, now.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]HomepageRecommendationFeedbackRecord, 0)
	for rows.Next() {
		var item HomepageRecommendationFeedbackRecord
		if err := rows.Scan(
			&item.ID,
			&item.Action,
			&item.TargetType,
			&item.TargetValue,
			&item.NormalizedTarget,
			&item.SourceMovieID,
			&item.ExpiresAt,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *SQLiteStore) CreateHomepageRecommendationFeedback(
	ctx context.Context,
	item HomepageRecommendationFeedbackRecord,
	now time.Time,
) (HomepageRecommendationFeedbackRecord, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return HomepageRecommendationFeedbackRecord{}, err
	}
	defer func() { _ = tx.Rollback() }()

	nowText := now.UTC().Format(time.RFC3339Nano)
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM homepage_recommendation_feedback
		WHERE expires_at <> '' AND julianday(expires_at) <= julianday(?)`, nowText); err != nil {
		return HomepageRecommendationFeedbackRecord{}, err
	}

	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM homepage_recommendation_feedback`).Scan(&count); err != nil {
		return HomepageRecommendationFeedbackRecord{}, err
	}
	if count >= MaxHomepageRecommendationFeedback {
		var existingID string
		err := tx.QueryRowContext(ctx, `
			SELECT id FROM homepage_recommendation_feedback
			WHERE action = ? AND target_type = ? AND normalized_target = ?`,
			item.Action, item.TargetType, item.NormalizedTarget,
		).Scan(&existingID)
		if errors.Is(err, sql.ErrNoRows) {
			return HomepageRecommendationFeedbackRecord{}, contracts.ErrRecommendationFeedbackLimitReached
		}
		if err != nil {
			return HomepageRecommendationFeedbackRecord{}, err
		}
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO homepage_recommendation_feedback (
			id, action, target_type, target_value, normalized_target, source_movie_id,
			expires_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(action, target_type, normalized_target) DO NOTHING`,
		item.ID,
		item.Action,
		item.TargetType,
		item.TargetValue,
		item.NormalizedTarget,
		item.SourceMovieID,
		item.ExpiresAt,
		item.CreatedAt,
		item.UpdatedAt,
	)
	if err != nil {
		return HomepageRecommendationFeedbackRecord{}, err
	}

	var out HomepageRecommendationFeedbackRecord
	err = tx.QueryRowContext(ctx, `
		SELECT id, action, target_type, target_value, normalized_target, source_movie_id,
		       expires_at, created_at, updated_at
		FROM homepage_recommendation_feedback
		WHERE action = ? AND target_type = ? AND normalized_target = ?`,
		item.Action, item.TargetType, item.NormalizedTarget,
	).Scan(
		&out.ID,
		&out.Action,
		&out.TargetType,
		&out.TargetValue,
		&out.NormalizedTarget,
		&out.SourceMovieID,
		&out.ExpiresAt,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		return HomepageRecommendationFeedbackRecord{}, err
	}
	if err := tx.Commit(); err != nil {
		return HomepageRecommendationFeedbackRecord{}, err
	}
	return out, nil
}

func (s *SQLiteStore) DeleteHomepageRecommendationFeedback(ctx context.Context, id string) (bool, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM homepage_recommendation_feedback WHERE id = ?`, strings.TrimSpace(id))
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	return affected > 0, err
}

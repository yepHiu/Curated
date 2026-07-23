package app

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

const maxRecommendationFeedbackTargetRunes = 200

func (a *App) ListHomepageRecommendationFeedback(ctx context.Context) (contracts.RecommendationFeedbackListDTO, error) {
	rows, err := a.store.ListActiveHomepageRecommendationFeedback(ctx, time.Now())
	if err != nil {
		return contracts.RecommendationFeedbackListDTO{}, err
	}
	items := make([]contracts.RecommendationFeedbackDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, recommendationFeedbackDTO(row))
	}
	return contracts.RecommendationFeedbackListDTO{Items: items}, nil
}

func (a *App) CreateHomepageRecommendationFeedback(
	ctx context.Context,
	body contracts.CreateRecommendationFeedbackBody,
) (contracts.RecommendationFeedbackDTO, error) {
	action := strings.ToLower(strings.TrimSpace(body.Action))
	targetType := strings.ToLower(strings.TrimSpace(body.TargetType))
	sourceMovieID := strings.TrimSpace(body.SourceMovieID)
	targetValue := strings.TrimSpace(body.TargetValue)
	if sourceMovieID == "" || targetValue == "" || utf8.RuneCountInString(targetValue) > maxRecommendationFeedbackTargetRunes {
		return contracts.RecommendationFeedbackDTO{}, contracts.ErrRecommendationFeedbackInvalid
	}
	if !validRecommendationFeedbackShape(action, targetType, body.DurationDays) {
		return contracts.RecommendationFeedbackDTO{}, contracts.ErrRecommendationFeedbackInvalid
	}

	movie, err := a.store.GetMovieDetail(ctx, sourceMovieID)
	if errors.Is(err, sql.ErrNoRows) {
		return contracts.RecommendationFeedbackDTO{}, contracts.ErrRecommendationFeedbackTargetNotFound
	}
	if err != nil {
		return contracts.RecommendationFeedbackDTO{}, err
	}
	if strings.TrimSpace(movie.TrashedAt) != "" {
		return contracts.RecommendationFeedbackDTO{}, contracts.ErrRecommendationFeedbackTargetNotFound
	}
	sourceMovieID = movie.ID
	canonicalTarget, ok := canonicalRecommendationFeedbackTarget(movie, targetType, targetValue)
	if !ok {
		return contracts.RecommendationFeedbackDTO{}, contracts.ErrRecommendationFeedbackTargetNotFound
	}

	id, err := newRecommendationFeedbackID()
	if err != nil {
		return contracts.RecommendationFeedbackDTO{}, err
	}
	now := time.Now().UTC()
	expiresAt := ""
	if action == "snooze" {
		expiresAt = now.AddDate(0, 0, body.DurationDays).Format(time.RFC3339Nano)
	}
	createdAt := now.Format(time.RFC3339Nano)
	normalizedTarget := strings.ToLower(strings.TrimSpace(canonicalTarget))
	if targetType == "actor" {
		normalizedTarget = storage.NormalizeActorIdentity(canonicalTarget)
	}
	row, err := a.store.CreateHomepageRecommendationFeedback(ctx, storage.HomepageRecommendationFeedbackRecord{
		ID:               id,
		Action:           action,
		TargetType:       targetType,
		TargetValue:      canonicalTarget,
		NormalizedTarget: normalizedTarget,
		SourceMovieID:    sourceMovieID,
		ExpiresAt:        expiresAt,
		CreatedAt:        createdAt,
		UpdatedAt:        createdAt,
	}, now)
	if err != nil {
		return contracts.RecommendationFeedbackDTO{}, err
	}
	return recommendationFeedbackDTO(row), nil
}

func (a *App) DeleteHomepageRecommendationFeedback(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return contracts.ErrRecommendationFeedbackInvalid
	}
	deleted, err := a.store.DeleteHomepageRecommendationFeedback(ctx, id)
	if err != nil {
		return err
	}
	if !deleted {
		return contracts.ErrRecommendationFeedbackTargetNotFound
	}
	return nil
}

func validRecommendationFeedbackShape(action, targetType string, durationDays int) bool {
	switch action {
	case "not_interested":
		return targetType == "movie" && durationDays == 0
	case "snooze":
		return targetType == "movie" && durationDays >= 1 && durationDays <= 365
	case "less":
		return (targetType == "actor" || targetType == "studio" || targetType == "tag") && durationDays == 0
	default:
		return false
	}
}

func canonicalRecommendationFeedbackTarget(movie contracts.MovieDetailDTO, targetType, targetValue string) (string, bool) {
	switch targetType {
	case "movie":
		return movie.ID, strings.EqualFold(movie.ID, targetValue)
	case "actor":
		return equalFoldValue(movie.Actors, targetValue)
	case "studio":
		return strings.TrimSpace(movie.Studio), strings.EqualFold(strings.TrimSpace(movie.Studio), targetValue)
	case "tag":
		return equalFoldValue(append(append([]string{}, movie.Tags...), movie.UserTags...), targetValue)
	default:
		return "", false
	}
}

func equalFoldValue(values []string, wanted string) (string, bool) {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && strings.EqualFold(value, wanted) {
			return value, true
		}
	}
	return "", false
}

func newRecommendationFeedbackID() (string, error) {
	var random [12]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	return "feedback_" + hex.EncodeToString(random[:]), nil
}

func recommendationFeedbackDTO(row storage.HomepageRecommendationFeedbackRecord) contracts.RecommendationFeedbackDTO {
	return contracts.RecommendationFeedbackDTO{
		ID:            row.ID,
		Action:        row.Action,
		TargetType:    row.TargetType,
		TargetValue:   row.TargetValue,
		SourceMovieID: row.SourceMovieID,
		ExpiresAt:     row.ExpiresAt,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
}

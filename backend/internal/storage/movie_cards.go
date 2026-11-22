package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"curated-backend/internal/contracts"
)

// MovieCards preserves requested order without loading details, previews or file lists.
func (s *SQLiteStore) MovieCards(ctx context.Context, requested []string) (contracts.MovieCardsDTO, error) {
	result := contracts.MovieCardsDTO{Items: []contracts.MovieCardDTO{}, MissingIDs: []string{}}
	if len(requested) > contracts.MaxMovieCardIDs {
		return result, fmt.Errorf("too many movie IDs")
	}
	ids := make([]string, 0, len(requested))
	seen := make(map[string]bool, len(requested))
	for _, raw := range requested {
		id := strings.TrimSpace(raw)
		if id != "" && !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	if len(ids) == 0 {
		return result, nil
	}
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`SELECT m.id,
	COALESCE(NULLIF(TRIM(m.user_title), ''), m.title), m.code,
	COALESCE(NULLIF(TRIM(m.user_studio), ''), m.studio),
	COALESCE(m.user_runtime_minutes, m.runtime_minutes), m.rating, m.user_rating,
	m.is_favorite, `+sqlMovieEffectiveYear+`, m.cover_url, m.thumb_url, m.resolution
	FROM movies m WHERE `+sqlMovieActiveClause+` AND m.id IN (%s)`, inClausePlaceholders(len(ids))), inClauseArgs(ids)...)
	if err != nil {
		return result, err
	}
	byID := make(map[string]contracts.MovieCardDTO, len(ids))
	for rows.Next() {
		var card contracts.MovieCardDTO
		var metadataRating float64
		var userRating sql.NullFloat64
		if err := rows.Scan(&card.ID, &card.Title, &card.Code, &card.Studio, &card.RuntimeMinutes, &metadataRating, &userRating, &card.IsFavorite, &card.Year, &card.CoverURL, &card.ThumbURL, &card.Resolution); err != nil {
			_ = rows.Close()
			return result, err
		}
		card.Rating = effectiveRating(metadataRating, userRating)
		byID[card.ID] = card
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return result, err
	}
	if err := rows.Close(); err != nil {
		return result, err
	}
	actors, err := s.lookupActors(ctx, ids)
	if err != nil {
		return result, err
	}
	for _, id := range ids {
		card, ok := byID[id]
		if !ok {
			result.MissingIDs = append(result.MissingIDs, id)
			continue
		}
		card.Actors = actors[id]
		if card.Actors == nil {
			card.Actors = []string{}
		}
		result.Items = append(result.Items, card)
	}
	return result, nil
}

package storage

import (
	"context"
	"database/sql"
	"encoding/json"
)

type HomepageDailyRecommendationSnapshot struct {
	DateUTC                string
	HeroMovieIDs           []string
	RecommendationMovieIDs []string
	GeneratedAt            string
	GenerationVersion      string
}

type HomepageRecommendationState struct {
	MovieID           string
	LastRecommendedAt string
	RecommendCount    int
	SkipUntil         string
	UpdatedAt         string
}

func (s *SQLiteStore) GetHomepageDailyRecommendationSnapshot(ctx context.Context, dateUTC string) (HomepageDailyRecommendationSnapshot, bool, error) {
	var heroJSON string
	var recommendationJSON string
	var snapshot HomepageDailyRecommendationSnapshot

	err := s.db.QueryRowContext(ctx, `
		SELECT date_utc, hero_movie_ids_json, recommendation_movie_ids_json, generated_at, generation_version
		FROM homepage_daily_recommendations
		WHERE date_utc = ?
	`, dateUTC).Scan(
		&snapshot.DateUTC,
		&heroJSON,
		&recommendationJSON,
		&snapshot.GeneratedAt,
		&snapshot.GenerationVersion,
	)
	if err == sql.ErrNoRows {
		return HomepageDailyRecommendationSnapshot{}, false, nil
	}
	if err != nil {
		return HomepageDailyRecommendationSnapshot{}, false, err
	}
	if err := json.Unmarshal([]byte(heroJSON), &snapshot.HeroMovieIDs); err != nil {
		return HomepageDailyRecommendationSnapshot{}, false, err
	}
	if err := json.Unmarshal([]byte(recommendationJSON), &snapshot.RecommendationMovieIDs); err != nil {
		return HomepageDailyRecommendationSnapshot{}, false, err
	}

	return snapshot, true, nil
}

func (s *SQLiteStore) UpsertHomepageDailyRecommendationSnapshot(ctx context.Context, snapshot HomepageDailyRecommendationSnapshot) error {
	heroJSON, err := json.Marshal(snapshot.HeroMovieIDs)
	if err != nil {
		return err
	}
	recommendationJSON, err := json.Marshal(snapshot.RecommendationMovieIDs)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO homepage_daily_recommendations (
			date_utc,
			hero_movie_ids_json,
			recommendation_movie_ids_json,
			generated_at,
			generation_version
		) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(date_utc) DO UPDATE SET
			hero_movie_ids_json = excluded.hero_movie_ids_json,
			recommendation_movie_ids_json = excluded.recommendation_movie_ids_json,
			generated_at = excluded.generated_at,
			generation_version = excluded.generation_version
	`,
		snapshot.DateUTC,
		string(heroJSON),
		string(recommendationJSON),
		snapshot.GeneratedAt,
		snapshot.GenerationVersion,
	)
	return err
}

func (s *SQLiteStore) ListHomepageDailyRecommendationSnapshotsInRange(
	ctx context.Context,
	startDateUTC string,
	endDateUTC string,
) ([]HomepageDailyRecommendationSnapshot, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT date_utc, hero_movie_ids_json, recommendation_movie_ids_json, generated_at, generation_version
		FROM homepage_daily_recommendations
		WHERE date_utc >= ? AND date_utc <= ?
		ORDER BY date_utc DESC
	`, startDateUTC, endDateUTC)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	snapshots := make([]HomepageDailyRecommendationSnapshot, 0)
	for rows.Next() {
		var snapshot HomepageDailyRecommendationSnapshot
		var heroJSON string
		var recommendationJSON string
		if err := rows.Scan(
			&snapshot.DateUTC,
			&heroJSON,
			&recommendationJSON,
			&snapshot.GeneratedAt,
			&snapshot.GenerationVersion,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(heroJSON), &snapshot.HeroMovieIDs); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(recommendationJSON), &snapshot.RecommendationMovieIDs); err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return snapshots, nil
}

func (s *SQLiteStore) ListHomepageRecommendationStates(ctx context.Context) ([]HomepageRecommendationState, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT movie_id, last_recommended_at, recommend_count, skip_until, updated_at
		FROM homepage_recommendation_states
		ORDER BY movie_id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	states := make([]HomepageRecommendationState, 0)
	for rows.Next() {
		var state HomepageRecommendationState
		if err := rows.Scan(
			&state.MovieID,
			&state.LastRecommendedAt,
			&state.RecommendCount,
			&state.SkipUntil,
			&state.UpdatedAt,
		); err != nil {
			return nil, err
		}
		states = append(states, state)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return states, nil
}

func (s *SQLiteStore) UpsertHomepageRecommendationStates(ctx context.Context, states []HomepageRecommendationState) error {
	if len(states) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO homepage_recommendation_states (
			movie_id,
			last_recommended_at,
			recommend_count,
			skip_until,
			updated_at
		) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(movie_id) DO UPDATE SET
			last_recommended_at = excluded.last_recommended_at,
			recommend_count = excluded.recommend_count,
			skip_until = excluded.skip_until,
			updated_at = excluded.updated_at
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, state := range states {
		if _, err := stmt.ExecContext(
			ctx,
			state.MovieID,
			state.LastRecommendedAt,
			state.RecommendCount,
			state.SkipUntil,
			state.UpdatedAt,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

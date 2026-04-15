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

package storage

import (
	"context"
	"database/sql"
	"fmt"
)

type PersonalInsightsOverviewAggregate struct {
	WatchedSeconds  float64
	StartedMovies   int
	CompletedMovies int
	RatedMovies     int
	AverageRating   *float64
}

type PersonalInsightsBreakdownAggregate struct {
	Name           string
	WatchedSeconds float64
	MovieCount     int
}

func (s *SQLiteStore) PersonalInsightsDataSince(ctx context.Context, toDay string) (*string, error) {
	var value sql.NullString
	if err := s.db.QueryRowContext(ctx, `
		SELECT MIN(day_key)
		FROM playback_daily_watch_time
		WHERE day_key <= ? AND watched_sec > 0`, toDay,
	).Scan(&value); err != nil {
		return nil, err
	}
	if !value.Valid || value.String == "" {
		return nil, nil
	}
	day := value.String
	return &day, nil
}

func (s *SQLiteStore) PersonalInsightsOverview(
	ctx context.Context,
	fromDay string,
	toDay string,
	completionThreshold float64,
) (PersonalInsightsOverviewAggregate, error) {
	var out PersonalInsightsOverviewAggregate
	var average sql.NullFloat64
	err := s.db.QueryRowContext(ctx, `
		WITH ranged_movies AS (
			SELECT movie_id, SUM(watched_sec) AS watched_seconds
			FROM playback_daily_watch_time
			WHERE day_key >= ? AND day_key <= ? AND watched_sec > 0
			GROUP BY movie_id
		)
		SELECT
			COALESCE(SUM(r.watched_seconds), 0),
			COUNT(*),
			COALESCE(SUM(CASE
				WHEN p.duration_sec > 0 AND p.position_sec >= p.duration_sec * ? THEN 1
				ELSE 0
			END), 0),
			COALESCE(SUM(CASE WHEN m.user_rating IS NOT NULL THEN 1 ELSE 0 END), 0),
			AVG(m.user_rating)
		FROM ranged_movies r
		JOIN movies m ON m.id = r.movie_id
		LEFT JOIN playback_progress p ON p.movie_id = r.movie_id`,
		fromDay,
		toDay,
		completionThreshold,
	).Scan(
		&out.WatchedSeconds,
		&out.StartedMovies,
		&out.CompletedMovies,
		&out.RatedMovies,
		&average,
	)
	if err != nil {
		return PersonalInsightsOverviewAggregate{}, err
	}
	if average.Valid {
		value := average.Float64
		out.AverageRating = &value
	}
	return out, nil
}

func (s *SQLiteStore) PersonalInsightsBreakdown(
	ctx context.Context,
	fromDay string,
	toDay string,
	dimension string,
	limit int,
) ([]PersonalInsightsBreakdownAggregate, error) {
	var entityCTE string
	switch dimension {
	case "actor":
		entityCTE = `
			SELECT r.movie_id, r.watched_seconds, a.name AS name, printf('actor:%d', a.id) AS entity_key
			FROM ranged_movies r
			JOIN movie_actors ma ON ma.movie_id = r.movie_id
			JOIN actors a ON a.id = ma.actor_id
			WHERE TRIM(a.name) <> ''`
	case "studio":
		entityCTE = `
			SELECT r.movie_id, r.watched_seconds, TRIM(m.studio) AS name,
			       'studio:' || LOWER(TRIM(m.studio)) AS entity_key
			FROM ranged_movies r
			JOIN movies m ON m.id = r.movie_id
			WHERE TRIM(m.studio) <> ''`
	case "tag":
		entityCTE = `
			SELECT r.movie_id, r.watched_seconds, MIN(TRIM(t.name)) AS name,
			       'tag:' || LOWER(TRIM(t.name)) AS entity_key
			FROM ranged_movies r
			JOIN movie_tags mt ON mt.movie_id = r.movie_id
			JOIN tags t ON t.id = mt.tag_id
			WHERE TRIM(t.name) <> ''
			GROUP BY r.movie_id, r.watched_seconds, LOWER(TRIM(t.name))`
	default:
		return nil, fmt.Errorf("unsupported personal insights dimension %q", dimension)
	}

	query := fmt.Sprintf(`
		WITH ranged_movies AS (
			SELECT movie_id, SUM(watched_sec) AS watched_seconds
			FROM playback_daily_watch_time
			WHERE day_key >= ? AND day_key <= ? AND watched_sec > 0
			GROUP BY movie_id
		), entity_movies AS (%s)
		SELECT MIN(name) AS name, SUM(watched_seconds) AS watched_seconds,
		       COUNT(DISTINCT movie_id) AS movie_count
		FROM entity_movies
		GROUP BY entity_key
		ORDER BY watched_seconds DESC, movie_count DESC, name COLLATE NOCASE ASC, name ASC
		LIMIT ?`, entityCTE)
	rows, err := s.db.QueryContext(ctx, query, fromDay, toDay, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]PersonalInsightsBreakdownAggregate, 0)
	for rows.Next() {
		var item PersonalInsightsBreakdownAggregate
		if err := rows.Scan(&item.Name, &item.WatchedSeconds, &item.MovieCount); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

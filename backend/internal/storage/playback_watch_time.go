package storage

import (
	"context"
	"errors"
	"regexp"
	"time"
)

var dayKeyPattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

var (
	ErrPlaybackWatchTimeInvalidDayKey  = errors.New("invalid playback watch time day key")
	ErrPlaybackWatchTimeInvalidSeconds = errors.New("invalid playback watch time seconds")
	ErrPlaybackWatchTimeMovieNotFound  = errors.New("playback watch time movie not found")
)

type PlaybackWatchTimeDailyRow struct {
	DayKey     string
	WatchedSec float64
}

func isValidPlaybackWatchTimeDayKey(dayKey string) bool {
	if !dayKeyPattern.MatchString(dayKey) {
		return false
	}
	_, err := time.Parse("2006-01-02", dayKey)
	return err == nil
}

func (s *SQLiteStore) AddPlaybackWatchTime(ctx context.Context, movieID string, dayKey string, watchedSec float64) error {
	if !isValidPlaybackWatchTimeDayKey(dayKey) {
		return ErrPlaybackWatchTimeInvalidDayKey
	}
	if watchedSec <= 0 || watchedSec > 300 {
		return ErrPlaybackWatchTimeInvalidSeconds
	}
	ok, err := s.MovieExists(ctx, movieID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrPlaybackWatchTimeMovieNotFound
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO playback_daily_watch_time (day_key, movie_id, watched_sec, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(day_key, movie_id) DO UPDATE SET
			watched_sec = watched_sec + excluded.watched_sec,
			updated_at = excluded.updated_at
	`, dayKey, movieID, watchedSec, now)
	return err
}

func (s *SQLiteStore) ListPlaybackWatchTimeDaily(ctx context.Context, days int) ([]PlaybackWatchTimeDailyRow, error) {
	if days <= 0 {
		days = 91
	}
	if days > 91 {
		days = 91
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT day_key, SUM(watched_sec) AS watched_sec
		FROM playback_daily_watch_time
		WHERE day_key >= date('now', 'localtime', '-' || (? - 1) || ' days')
		GROUP BY day_key
		ORDER BY day_key DESC
	`, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []PlaybackWatchTimeDailyRow{}
	for rows.Next() {
		var r PlaybackWatchTimeDailyRow
		if err := rows.Scan(&r.DayKey, &r.WatchedSec); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

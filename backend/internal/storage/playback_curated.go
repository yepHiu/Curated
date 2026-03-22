package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

// PlaybackProgressRow mirrors playback_progress table.
type PlaybackProgressRow struct {
	MovieID     string
	PositionSec float64
	DurationSec float64
	UpdatedAt   string
}

func (s *SQLiteStore) MovieExists(ctx context.Context, movieID string) (bool, error) {
	var one int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM movies WHERE id = ? LIMIT 1`, movieID).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *SQLiteStore) UpsertPlaybackProgress(ctx context.Context, movieID string, positionSec, durationSec float64) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO playback_progress (movie_id, position_sec, duration_sec, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(movie_id) DO UPDATE SET
			position_sec = excluded.position_sec,
			duration_sec = excluded.duration_sec,
			updated_at = excluded.updated_at
	`, movieID, positionSec, durationSec, now)
	return err
}

func (s *SQLiteStore) DeletePlaybackProgress(ctx context.Context, movieID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM playback_progress WHERE movie_id = ?`, movieID)
	return err
}

func (s *SQLiteStore) GetPlaybackProgress(ctx context.Context, movieID string) (*PlaybackProgressRow, error) {
	var r PlaybackProgressRow
	err := s.db.QueryRowContext(ctx, `
		SELECT movie_id, position_sec, duration_sec, updated_at
		FROM playback_progress WHERE movie_id = ?
	`, movieID).Scan(&r.MovieID, &r.PositionSec, &r.DurationSec, &r.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *SQLiteStore) ListPlaybackProgressByUpdatedDesc(ctx context.Context) ([]PlaybackProgressRow, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT movie_id, position_sec, duration_sec, updated_at
		FROM playback_progress
		ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []PlaybackProgressRow
	for rows.Next() {
		var r PlaybackProgressRow
		if err := rows.Scan(&r.MovieID, &r.PositionSec, &r.DurationSec, &r.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// CuratedFrameMeta is list/detail metadata without image bytes.
type CuratedFrameMeta struct {
	ID          string
	MovieID     string
	Title       string
	Code        string
	Actors      []string
	PositionSec float64
	CapturedAt  string
	Tags        []string
}

func (s *SQLiteStore) InsertCuratedFrame(ctx context.Context, meta CuratedFrameMeta, imageBlob []byte) error {
	actorsJSON, err := json.Marshal(meta.Actors)
	if err != nil {
		return err
	}
	tagsJSON, err := json.Marshal(meta.Tags)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO curated_frames (id, movie_id, title, code, actors_json, position_sec, captured_at, tags_json, image_blob)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, meta.ID, meta.MovieID, meta.Title, meta.Code, string(actorsJSON), meta.PositionSec, meta.CapturedAt, string(tagsJSON), imageBlob)
	return err
}

func scanCuratedMeta(actorsJSON, tagsJSON string, dest *CuratedFrameMeta) error {
	if err := json.Unmarshal([]byte(actorsJSON), &dest.Actors); err != nil {
		dest.Actors = nil
	}
	if err := json.Unmarshal([]byte(tagsJSON), &dest.Tags); err != nil {
		dest.Tags = nil
	}
	return nil
}

func (s *SQLiteStore) ListCuratedFramesByCapturedAtDesc(ctx context.Context) ([]CuratedFrameMeta, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, movie_id, title, code, actors_json, position_sec, captured_at, tags_json
		FROM curated_frames
		ORDER BY captured_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CuratedFrameMeta
	for rows.Next() {
		var m CuratedFrameMeta
		var actorsJSON, tagsJSON string
		if err := rows.Scan(&m.ID, &m.MovieID, &m.Title, &m.Code, &actorsJSON, &m.PositionSec, &m.CapturedAt, &tagsJSON); err != nil {
			return nil, err
		}
		_ = scanCuratedMeta(actorsJSON, tagsJSON, &m)
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) GetCuratedFrameImage(ctx context.Context, id string) ([]byte, error) {
	var blob []byte
	err := s.db.QueryRowContext(ctx, `SELECT image_blob FROM curated_frames WHERE id = ?`, id).Scan(&blob)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return blob, nil
}

func (s *SQLiteStore) UpdateCuratedFrameTags(ctx context.Context, id string, tags []string) error {
	tagsJSON, err := json.Marshal(tags)
	if err != nil {
		return err
	}
	res, err := s.db.ExecContext(ctx, `UPDATE curated_frames SET tags_json = ? WHERE id = ?`, string(tagsJSON), id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *SQLiteStore) DeleteCuratedFrame(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM curated_frames WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *SQLiteStore) CountCuratedFrames(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM curated_frames`).Scan(&n)
	return n, err
}

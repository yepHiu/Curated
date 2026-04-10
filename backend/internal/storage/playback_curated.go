package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
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
	return s.InsertCuratedFrameWithThumbnail(ctx, meta, imageBlob, nil)
}

func (s *SQLiteStore) InsertCuratedFrameWithThumbnail(ctx context.Context, meta CuratedFrameMeta, imageBlob []byte, thumbBlob []byte) error {
	actorsJSON, err := json.Marshal(meta.Actors)
	if err != nil {
		return err
	}
	tagsJSON, err := json.Marshal(meta.Tags)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO curated_frames (id, movie_id, title, code, actors_json, position_sec, captured_at, tags_json, image_blob, thumb_blob)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, meta.ID, meta.MovieID, meta.Title, meta.Code, string(actorsJSON), meta.PositionSec, meta.CapturedAt, string(tagsJSON), imageBlob, thumbBlob)
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

type CuratedFrameQuery struct {
	Query   string
	Actor   string
	MovieID string
	Tag     string
	Limit   int
	Offset  int
}

type CuratedFramePage struct {
	Items  []CuratedFrameMeta
	Total  int
	Limit  int
	Offset int
}

type CuratedFrameFacet struct {
	Name  string
	Count int
}

func normalizedCuratedFrameLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	if limit > 200 {
		return 200
	}
	return limit
}

func curatedJSONContainsArg(value string) string {
	b, err := json.Marshal(strings.TrimSpace(value))
	if err != nil {
		return "%"
	}
	return "%" + string(b) + "%"
}

func buildCuratedFrameWhere(q CuratedFrameQuery) (string, []any) {
	clauses := make([]string, 0, 4)
	args := make([]any, 0, 4)
	if s := strings.TrimSpace(q.Query); s != "" {
		like := "%" + strings.ToLower(s) + "%"
		clauses = append(clauses, `LOWER(title || ' ' || code || ' ' || movie_id || ' ' || actors_json || ' ' || tags_json || ' ' || captured_at || ' ' || CAST(position_sec AS TEXT)) LIKE ?`)
		args = append(args, like)
	}
	if s := strings.TrimSpace(q.Actor); s != "" {
		clauses = append(clauses, `actors_json LIKE ?`)
		args = append(args, curatedJSONContainsArg(s))
	}
	if s := strings.TrimSpace(q.MovieID); s != "" {
		clauses = append(clauses, `movie_id = ?`)
		args = append(args, s)
	}
	if s := strings.TrimSpace(q.Tag); s != "" {
		clauses = append(clauses, `tags_json LIKE ?`)
		args = append(args, curatedJSONContainsArg(s))
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func (s *SQLiteStore) QueryCuratedFrames(ctx context.Context, q CuratedFrameQuery) (CuratedFramePage, error) {
	limit := normalizedCuratedFrameLimit(q.Limit)
	offset := q.Offset
	if offset < 0 {
		offset = 0
	}
	where, args := buildCuratedFrameWhere(q)

	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM curated_frames`+where, args...).Scan(&total); err != nil {
		return CuratedFramePage{}, err
	}

	pageArgs := append(append([]any{}, args...), limit, offset)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, movie_id, title, code, actors_json, position_sec, captured_at, tags_json
		FROM curated_frames`+where+`
		ORDER BY captured_at DESC
		LIMIT ? OFFSET ?
	`, pageArgs...)
	if err != nil {
		return CuratedFramePage{}, err
	}
	defer rows.Close()

	out := make([]CuratedFrameMeta, 0, limit)
	for rows.Next() {
		var m CuratedFrameMeta
		var actorsJSON, tagsJSON string
		if err := rows.Scan(&m.ID, &m.MovieID, &m.Title, &m.Code, &actorsJSON, &m.PositionSec, &m.CapturedAt, &tagsJSON); err != nil {
			return CuratedFramePage{}, err
		}
		_ = scanCuratedMeta(actorsJSON, tagsJSON, &m)
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return CuratedFramePage{}, err
	}
	return CuratedFramePage{Items: out, Total: total, Limit: limit, Offset: offset}, nil
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

func listCuratedFrameJSONFacet(ctx context.Context, db *sql.DB, column string) ([]CuratedFrameFacet, error) {
	rows, err := db.QueryContext(ctx, `SELECT `+column+` FROM curated_frames`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := map[string]int{}
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var values []string
		if err := json.Unmarshal([]byte(raw), &values); err != nil {
			continue
		}
		seen := map[string]struct{}{}
		for _, value := range values {
			name := strings.TrimSpace(value)
			if name == "" {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			counts[name]++
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]CuratedFrameFacet, 0, len(counts))
	for name, count := range counts {
		out = append(out, CuratedFrameFacet{Name: name, Count: count})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

func (s *SQLiteStore) ListCuratedFrameActors(ctx context.Context) ([]CuratedFrameFacet, error) {
	return listCuratedFrameJSONFacet(ctx, s.db, "actors_json")
}

func (s *SQLiteStore) ListCuratedFrameTags(ctx context.Context) ([]CuratedFrameFacet, error) {
	return listCuratedFrameJSONFacet(ctx, s.db, "tags_json")
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

func (s *SQLiteStore) GetCuratedFrameThumbnail(ctx context.Context, id string) ([]byte, error) {
	var thumb, image []byte
	err := s.db.QueryRowContext(ctx, `SELECT thumb_blob, image_blob FROM curated_frames WHERE id = ?`, id).Scan(&thumb, &image)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(thumb) > 0 {
		return thumb, nil
	}
	return image, nil
}

func (s *SQLiteStore) FindNearbyCuratedFrame(ctx context.Context, movieID string, positionSec, thresholdSec float64) (*CuratedFrameMeta, error) {
	if strings.TrimSpace(movieID) == "" || thresholdSec <= 0 {
		return nil, nil
	}

	var meta CuratedFrameMeta
	var actorsJSON, tagsJSON string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, movie_id, title, code, actors_json, position_sec, captured_at, tags_json
		FROM curated_frames
		WHERE movie_id = ? AND ABS(position_sec - ?) <= ?
		ORDER BY ABS(position_sec - ?) ASC, captured_at DESC
		LIMIT 1
	`, movieID, positionSec, thresholdSec, positionSec).Scan(
		&meta.ID,
		&meta.MovieID,
		&meta.Title,
		&meta.Code,
		&actorsJSON,
		&meta.PositionSec,
		&meta.CapturedAt,
		&tagsJSON,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	_ = scanCuratedMeta(actorsJSON, tagsJSON, &meta)
	return &meta, nil
}

// ErrCuratedFrameNotFound is returned when a curated frame id does not exist.
var ErrCuratedFrameNotFound = errors.New("curated frame not found")

// CuratedFrameExportRow is metadata plus PNG/JPEG image bytes for WebP export.
type CuratedFrameExportRow struct {
	CuratedFrameMeta
	ImageBlob []byte
}

// ListCuratedFramesForExport returns rows in the same order as ids (after caller dedupes).
func (s *SQLiteStore) ListCuratedFramesForExport(ctx context.Context, ids []string) ([]CuratedFrameExportRow, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	ph := strings.Repeat("?,", len(ids))
	ph = ph[:len(ph)-1]
	q := fmt.Sprintf(`
		SELECT id, movie_id, title, code, actors_json, position_sec, captured_at, tags_json, image_blob
		FROM curated_frames WHERE id IN (%s)
	`, ph)
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byID := make(map[string]CuratedFrameExportRow, len(ids))
	for rows.Next() {
		var r CuratedFrameExportRow
		var actorsJSON, tagsJSON string
		if err := rows.Scan(&r.ID, &r.MovieID, &r.Title, &r.Code, &actorsJSON, &r.PositionSec, &r.CapturedAt, &tagsJSON, &r.ImageBlob); err != nil {
			return nil, err
		}
		_ = scanCuratedMeta(actorsJSON, tagsJSON, &r.CuratedFrameMeta)
		byID[r.ID] = r
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]CuratedFrameExportRow, 0, len(ids))
	for _, id := range ids {
		r, ok := byID[id]
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrCuratedFrameNotFound, id)
		}
		out = append(out, r)
	}
	return out, nil
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

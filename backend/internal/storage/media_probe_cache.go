package storage

import (
	"context"
	"database/sql"
)

// MediaProbeCacheSchema is the probe-field generation stored in media_probe_cache.
// Rows below this value must be re-probed so new fields are populated.
const MediaProbeCacheSchema = 2

// MediaProbeCacheRow persists one ffprobe media snapshot keyed by source path.
// Callers validate size+modtime before trusting a cached row.
type MediaProbeCacheRow struct {
	Path                string
	SizeBytes           int64
	MtimeUnixNs         int64
	Container           string
	VideoCodec          string
	AudioCodec          string
	DurationSec         float64
	RFrameRate          string
	AvgFrameRate        string
	HasNegativeVideoPTS int
	ProbeSchema         int
}

// GetMediaProbeCache returns the cached probe row for path, or nil when absent.
func (s *SQLiteStore) GetMediaProbeCache(ctx context.Context, path string) (*MediaProbeCacheRow, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT path, size_bytes, mtime_unix_ns, container, video_codec, audio_codec, duration_sec,
			r_frame_rate, avg_frame_rate, has_negative_video_pts, probe_schema
		FROM media_probe_cache
		WHERE path = ?
	`, path)
	var cached MediaProbeCacheRow
	err := row.Scan(
		&cached.Path,
		&cached.SizeBytes,
		&cached.MtimeUnixNs,
		&cached.Container,
		&cached.VideoCodec,
		&cached.AudioCodec,
		&cached.DurationSec,
		&cached.RFrameRate,
		&cached.AvgFrameRate,
		&cached.HasNegativeVideoPTS,
		&cached.ProbeSchema,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &cached, nil
}

// UpsertMediaProbeCache stores a probe snapshot for path.
func (s *SQLiteStore) UpsertMediaProbeCache(ctx context.Context, cached MediaProbeCacheRow) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO media_probe_cache (
			path, size_bytes, mtime_unix_ns, container, video_codec, audio_codec, duration_sec,
			r_frame_rate, avg_frame_rate, has_negative_video_pts, probe_schema, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(path) DO UPDATE SET
			size_bytes = excluded.size_bytes,
			mtime_unix_ns = excluded.mtime_unix_ns,
			container = excluded.container,
			video_codec = excluded.video_codec,
			audio_codec = excluded.audio_codec,
			duration_sec = excluded.duration_sec,
			r_frame_rate = excluded.r_frame_rate,
			avg_frame_rate = excluded.avg_frame_rate,
			has_negative_video_pts = excluded.has_negative_video_pts,
			probe_schema = excluded.probe_schema,
			updated_at = CURRENT_TIMESTAMP
	`, cached.Path, cached.SizeBytes, cached.MtimeUnixNs, cached.Container, cached.VideoCodec, cached.AudioCodec, cached.DurationSec, cached.RFrameRate, cached.AvgFrameRate, cached.HasNegativeVideoPTS, cached.ProbeSchema)
	return err
}

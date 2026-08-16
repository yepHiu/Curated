package storage

import (
	"context"
	"database/sql"
)

// MediaProbeCacheRow persists one ffprobe media snapshot keyed by source path.
// Callers validate size+modtime before trusting a cached row.
type MediaProbeCacheRow struct {
	Path         string
	SizeBytes    int64
	MtimeUnixNs  int64
	Container    string
	VideoCodec   string
	AudioCodec   string
	DurationSec  float64
}

// GetMediaProbeCache returns the cached probe row for path, or nil when absent.
func (s *SQLiteStore) GetMediaProbeCache(ctx context.Context, path string) (*MediaProbeCacheRow, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT path, size_bytes, mtime_unix_ns, container, video_codec, audio_codec, duration_sec
		FROM media_probe_cache
		WHERE path = ?
	`, path)
	var cached MediaProbeCacheRow
	err := row.Scan(&cached.Path, &cached.SizeBytes, &cached.MtimeUnixNs, &cached.Container, &cached.VideoCodec, &cached.AudioCodec, &cached.DurationSec)
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
		INSERT INTO media_probe_cache (path, size_bytes, mtime_unix_ns, container, video_codec, audio_codec, duration_sec, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(path) DO UPDATE SET
			size_bytes = excluded.size_bytes,
			mtime_unix_ns = excluded.mtime_unix_ns,
			container = excluded.container,
			video_codec = excluded.video_codec,
			audio_codec = excluded.audio_codec,
			duration_sec = excluded.duration_sec,
			updated_at = CURRENT_TIMESTAMP
	`, cached.Path, cached.SizeBytes, cached.MtimeUnixNs, cached.Container, cached.VideoCodec, cached.AudioCodec, cached.DurationSec)
	return err
}

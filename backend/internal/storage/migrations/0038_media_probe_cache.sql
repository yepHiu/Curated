CREATE TABLE IF NOT EXISTS media_probe_cache (
  path TEXT PRIMARY KEY,
  size_bytes INTEGER NOT NULL,
  mtime_unix_ns INTEGER NOT NULL,
  container TEXT NOT NULL DEFAULT '',
  video_codec TEXT NOT NULL DEFAULT '',
  audio_codec TEXT NOT NULL DEFAULT '',
  duration_sec REAL NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

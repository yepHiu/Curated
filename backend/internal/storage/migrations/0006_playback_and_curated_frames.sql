CREATE TABLE IF NOT EXISTS playback_progress (
  movie_id TEXT PRIMARY KEY,
  position_sec REAL NOT NULL DEFAULT 0,
  duration_sec REAL NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (movie_id) REFERENCES movies(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_playback_progress_updated_at ON playback_progress(updated_at DESC);

CREATE TABLE IF NOT EXISTS curated_frames (
  id TEXT PRIMARY KEY,
  movie_id TEXT NOT NULL,
  title TEXT NOT NULL,
  code TEXT NOT NULL,
  actors_json TEXT NOT NULL DEFAULT '[]',
  position_sec REAL NOT NULL,
  captured_at TEXT NOT NULL,
  tags_json TEXT NOT NULL DEFAULT '[]',
  image_blob BLOB NOT NULL,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (movie_id) REFERENCES movies(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_curated_frames_movie_id ON curated_frames(movie_id);
CREATE INDEX IF NOT EXISTS idx_curated_frames_captured_at ON curated_frames(captured_at DESC);

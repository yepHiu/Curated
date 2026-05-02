CREATE TABLE IF NOT EXISTS playback_daily_watch_time (
  day_key TEXT NOT NULL,
  movie_id TEXT NOT NULL,
  watched_sec REAL NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL,
  PRIMARY KEY (day_key, movie_id),
  FOREIGN KEY (movie_id) REFERENCES movies(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_playback_daily_watch_time_day
  ON playback_daily_watch_time(day_key DESC);

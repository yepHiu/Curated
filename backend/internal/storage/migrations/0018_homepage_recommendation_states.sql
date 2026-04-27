CREATE TABLE IF NOT EXISTS homepage_recommendation_states (
  movie_id TEXT PRIMARY KEY,
  last_recommended_at TEXT NOT NULL DEFAULT '',
  recommend_count INTEGER NOT NULL DEFAULT 0,
  skip_until TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS homepage_recommendation_feedback (
  id TEXT PRIMARY KEY,
  action TEXT NOT NULL CHECK (action IN ('not_interested', 'snooze', 'less')),
  target_type TEXT NOT NULL CHECK (target_type IN ('movie', 'actor', 'studio', 'tag')),
  target_value TEXT NOT NULL,
  normalized_target TEXT NOT NULL,
  source_movie_id TEXT NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
  expires_at TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  CHECK (
    (action IN ('not_interested', 'snooze') AND target_type = 'movie') OR
    (action = 'less' AND target_type IN ('actor', 'studio', 'tag'))
  ),
  UNIQUE (action, target_type, normalized_target)
);

CREATE INDEX IF NOT EXISTS idx_homepage_recommendation_feedback_active
  ON homepage_recommendation_feedback(expires_at, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_homepage_recommendation_feedback_source_movie
  ON homepage_recommendation_feedback(source_movie_id);

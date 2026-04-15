CREATE TABLE IF NOT EXISTS homepage_daily_recommendations (
  date_utc TEXT PRIMARY KEY,
  hero_movie_ids_json TEXT NOT NULL,
  recommendation_movie_ids_json TEXT NOT NULL,
  generated_at TEXT NOT NULL,
  generation_version TEXT NOT NULL
);

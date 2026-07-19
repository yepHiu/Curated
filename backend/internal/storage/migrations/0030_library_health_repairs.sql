CREATE TABLE IF NOT EXISTS movie_metadata_scrape_attempts (
  movie_id TEXT PRIMARY KEY,
  task_id TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('running', 'completed', 'failed')),
  error_code TEXT NOT NULL DEFAULT '',
  error_category TEXT NOT NULL DEFAULT '',
  error_message TEXT NOT NULL DEFAULT '',
  provider TEXT NOT NULL DEFAULT '',
  started_at TEXT NOT NULL,
  finished_at TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL,
  FOREIGN KEY (movie_id) REFERENCES movies(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_movie_metadata_scrape_attempts_status
  ON movie_metadata_scrape_attempts(status, updated_at DESC);

CREATE TABLE IF NOT EXISTS library_health_repair_runs (
  repair_id TEXT PRIMARY KEY,
  task_id TEXT NOT NULL UNIQUE,
  action TEXT NOT NULL CHECK (action IN ('rescrape_metadata')),
  categories_json TEXT NOT NULL CHECK (json_valid(categories_json)),
  status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'completed', 'partial_failed', 'failed', 'cancelled')),
  total_items INTEGER NOT NULL CHECK (total_items >= 0),
  completed_items INTEGER NOT NULL DEFAULT 0 CHECK (completed_items >= 0),
  succeeded_items INTEGER NOT NULL DEFAULT 0 CHECK (succeeded_items >= 0),
  failed_items INTEGER NOT NULL DEFAULT 0 CHECK (failed_items >= 0),
  created_at TEXT NOT NULL,
  started_at TEXT NOT NULL DEFAULT '',
  finished_at TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_library_health_repair_runs_created
  ON library_health_repair_runs(created_at DESC);

CREATE TABLE IF NOT EXISTS library_health_repair_items (
  repair_id TEXT NOT NULL,
  ordinal INTEGER NOT NULL CHECK (ordinal >= 0),
  finding_id TEXT NOT NULL,
  category TEXT NOT NULL,
  movie_id TEXT NOT NULL,
  label TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL CHECK (status IN ('pending', 'queued', 'succeeded', 'failed', 'cancelled')),
  child_task_id TEXT NOT NULL DEFAULT '',
  error_code TEXT NOT NULL DEFAULT '',
  error_message TEXT NOT NULL DEFAULT '',
  started_at TEXT NOT NULL DEFAULT '',
  finished_at TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (repair_id, ordinal),
  UNIQUE (repair_id, finding_id),
  FOREIGN KEY (repair_id) REFERENCES library_health_repair_runs(repair_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_library_health_repair_items_status
  ON library_health_repair_items(repair_id, status, ordinal);

CREATE TABLE IF NOT EXISTS app_update_status (
  status_key TEXT PRIMARY KEY,
  installed_version TEXT NOT NULL,
  latest_version TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  checked_at TEXT NOT NULL,
  published_at TEXT NOT NULL DEFAULT '',
  release_name TEXT NOT NULL DEFAULT '',
  release_url TEXT NOT NULL DEFAULT '',
  release_notes_snippet TEXT NOT NULL DEFAULT '',
  source TEXT NOT NULL DEFAULT '',
  error_message TEXT NOT NULL DEFAULT ''
);

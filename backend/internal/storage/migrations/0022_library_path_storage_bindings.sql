CREATE TABLE IF NOT EXISTS library_path_storage_bindings (
  library_path_id TEXT PRIMARY KEY,
  root_path TEXT NOT NULL,
  volume_id TEXT,
  volume_label TEXT,
  file_system TEXT,
  drive_type TEXT,
  identity_confidence TEXT NOT NULL DEFAULT 'unknown',
  bound_at TEXT,
  last_seen_at TEXT,
  last_checked_at TEXT,
  last_status TEXT,
  last_error TEXT,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (library_path_id) REFERENCES library_paths(id) ON DELETE CASCADE
);


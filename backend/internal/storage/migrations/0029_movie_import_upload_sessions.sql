CREATE TABLE IF NOT EXISTS movie_import_upload_sessions (
  upload_id TEXT PRIMARY KEY,
  task_id TEXT NOT NULL UNIQUE,
  target_library_path_id TEXT NOT NULL,
  target_root TEXT NOT NULL,
  staging_dir TEXT NOT NULL,
  state TEXT NOT NULL CHECK (state IN ('uploading', 'committing', 'committed', 'aborted', 'expired', 'unrecoverable')),
  chunk_size INTEGER NOT NULL CHECK (chunk_size > 0),
  total_bytes INTEGER NOT NULL CHECK (total_bytes > 0),
  bytes_received INTEGER NOT NULL DEFAULT 0 CHECK (bytes_received >= 0 AND bytes_received <= total_bytes),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  expires_at TEXT NOT NULL,
  cleanup_after TEXT NOT NULL DEFAULT '',
  diagnostic_code TEXT NOT NULL DEFAULT '',
  diagnostic_message TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_movie_import_upload_sessions_state_expires
  ON movie_import_upload_sessions(state, expires_at);

CREATE INDEX IF NOT EXISTS idx_movie_import_upload_sessions_cleanup_after
  ON movie_import_upload_sessions(cleanup_after)
  WHERE cleanup_after != '';

CREATE TABLE IF NOT EXISTS movie_import_upload_files (
  upload_id TEXT NOT NULL,
  file_id TEXT NOT NULL,
  ordinal INTEGER NOT NULL CHECK (ordinal >= 0),
  relative_path TEXT NOT NULL,
  size_bytes INTEGER NOT NULL CHECK (size_bytes > 0),
  staging_path TEXT NOT NULL,
  final_path TEXT NOT NULL,
  bytes_received INTEGER NOT NULL DEFAULT 0 CHECK (bytes_received >= 0 AND bytes_received <= size_bytes),
  state TEXT NOT NULL DEFAULT 'pending' CHECK (state IN ('pending', 'committed')),
  committed_at TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (upload_id, file_id),
  UNIQUE (upload_id, ordinal),
  UNIQUE (upload_id, relative_path),
  FOREIGN KEY (upload_id) REFERENCES movie_import_upload_sessions(upload_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS movie_import_upload_chunks (
  upload_id TEXT NOT NULL,
  file_id TEXT NOT NULL,
  chunk_index INTEGER NOT NULL CHECK (chunk_index >= 0),
  offset_bytes INTEGER NOT NULL CHECK (offset_bytes >= 0),
  size_bytes INTEGER NOT NULL CHECK (size_bytes > 0),
  created_at TEXT NOT NULL,
  PRIMARY KEY (upload_id, file_id, chunk_index),
  FOREIGN KEY (upload_id, file_id) REFERENCES movie_import_upload_files(upload_id, file_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_movie_import_upload_chunks_range
  ON movie_import_upload_chunks(upload_id, file_id, offset_bytes);

CREATE TABLE IF NOT EXISTS movie_import_upload_cleanup_audits (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  upload_id TEXT NOT NULL,
  reason TEXT NOT NULL,
  prior_state TEXT NOT NULL,
  staging_dir TEXT NOT NULL,
  outcome TEXT NOT NULL CHECK (outcome IN ('pending', 'removed', 'skipped', 'failed')),
  diagnostic TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  completed_at TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_movie_import_upload_cleanup_audits_created_at
  ON movie_import_upload_cleanup_audits(created_at DESC);

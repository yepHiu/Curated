CREATE TABLE IF NOT EXISTS photo_library_paths (
  id TEXT PRIMARY KEY,
  path TEXT NOT NULL UNIQUE,
  title TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  first_library_scan_pending INTEGER NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_photo_library_paths_path ON photo_library_paths(path);

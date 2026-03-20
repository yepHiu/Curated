CREATE TABLE IF NOT EXISTS scan_items (
  task_id TEXT NOT NULL,
  path TEXT NOT NULL,
  file_name TEXT NOT NULL,
  number TEXT NOT NULL DEFAULT '',
  movie_id TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (task_id, path)
);

CREATE INDEX IF NOT EXISTS idx_scan_items_task_id ON scan_items(task_id);
CREATE INDEX IF NOT EXISTS idx_scan_items_number ON scan_items(number);

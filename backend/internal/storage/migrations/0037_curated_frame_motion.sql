CREATE TABLE IF NOT EXISTS curated_frame_motions (
  frame_id TEXT PRIMARY KEY,
  status TEXT NOT NULL DEFAULT 'processing',
  artifact_name TEXT NOT NULL DEFAULT '',
  content_type TEXT NOT NULL DEFAULT 'image/gif',
  duration_sec REAL NOT NULL DEFAULT 0,
  width INTEGER NOT NULL DEFAULT 0,
  height INTEGER NOT NULL DEFAULT 0,
  fps INTEGER NOT NULL DEFAULT 0,
  file_size INTEGER NOT NULL DEFAULT 0,
  error_message TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (frame_id) REFERENCES curated_frames(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_curated_frame_motions_status
  ON curated_frame_motions(status);

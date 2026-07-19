CREATE TABLE IF NOT EXISTS path_migration_audits (
  id TEXT PRIMARY KEY,
  created_at TEXT NOT NULL,
  from_root TEXT NOT NULL,
  to_root TEXT NOT NULL,
  source_style TEXT NOT NULL,
  target_style TEXT NOT NULL,
  affected_rows INTEGER NOT NULL,
  missing_targets INTEGER NOT NULL,
  unchecked_targets INTEGER NOT NULL,
  allow_missing INTEGER NOT NULL DEFAULT 0,
  backup_path TEXT NOT NULL,
  summary_json TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_path_migration_audits_created_at
  ON path_migration_audits(created_at DESC);

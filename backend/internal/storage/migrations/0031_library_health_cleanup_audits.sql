CREATE TABLE IF NOT EXISTS library_health_cleanup_audits (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  task_id TEXT NOT NULL,
  finding_id TEXT NOT NULL,
  action TEXT NOT NULL CHECK (action IN ('cleanup_orphan_state')),
  entity_type TEXT NOT NULL,
  entity_id TEXT NOT NULL,
  outcome TEXT NOT NULL CHECK (outcome IN ('removed', 'skipped', 'failed')),
  diagnostic TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  completed_at TEXT NOT NULL,
  UNIQUE(task_id, finding_id)
);

CREATE INDEX IF NOT EXISTS idx_library_health_cleanup_audits_created
  ON library_health_cleanup_audits(created_at DESC);

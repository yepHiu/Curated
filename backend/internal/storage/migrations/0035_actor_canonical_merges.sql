ALTER TABLE actors ADD COLUMN normalized_name TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_actors_normalized_name
  ON actors(normalized_name, id);

CREATE TABLE IF NOT EXISTS actor_aliases (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  canonical_actor_id INTEGER NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
  alias TEXT NOT NULL,
  normalized_alias TEXT NOT NULL UNIQUE,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_actor_aliases_canonical_actor
  ON actor_aliases(canonical_actor_id, id);

CREATE TABLE IF NOT EXISTS actor_merge_audits (
  id TEXT PRIMARY KEY,
  source_actor_id INTEGER NOT NULL,
  target_actor_id INTEGER NOT NULL REFERENCES actors(id) ON DELETE RESTRICT,
  source_name TEXT NOT NULL,
  target_name TEXT NOT NULL,
  preview_token TEXT NOT NULL,
  summary_json TEXT NOT NULL,
  applied_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_actor_merge_audits_applied
  ON actor_merge_audits(applied_at DESC, id);

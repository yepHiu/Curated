CREATE TABLE actor_merge_audits_rebuilt (
  id TEXT PRIMARY KEY,
  source_actor_id INTEGER NOT NULL,
  target_actor_id INTEGER NOT NULL,
  source_name TEXT NOT NULL,
  target_name TEXT NOT NULL,
  preview_token TEXT NOT NULL,
  summary_json TEXT NOT NULL,
  applied_at TEXT NOT NULL
);

INSERT INTO actor_merge_audits_rebuilt (
  id,
  source_actor_id,
  target_actor_id,
  source_name,
  target_name,
  preview_token,
  summary_json,
  applied_at
)
SELECT
  id,
  source_actor_id,
  target_actor_id,
  source_name,
  target_name,
  preview_token,
  summary_json,
  applied_at
FROM actor_merge_audits;

DROP TABLE actor_merge_audits;

ALTER TABLE actor_merge_audits_rebuilt RENAME TO actor_merge_audits;

CREATE INDEX idx_actor_merge_audits_applied
  ON actor_merge_audits(applied_at DESC, id);

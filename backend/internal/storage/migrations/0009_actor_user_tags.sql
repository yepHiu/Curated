CREATE TABLE IF NOT EXISTS actor_user_tags (
  actor_id INTEGER NOT NULL,
  tag TEXT NOT NULL,
  PRIMARY KEY (actor_id, tag),
  FOREIGN KEY (actor_id) REFERENCES actors(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_actor_user_tags_tag ON actor_user_tags(tag);

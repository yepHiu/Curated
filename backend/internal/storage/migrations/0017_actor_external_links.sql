CREATE TABLE IF NOT EXISTS actor_external_links (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  actor_id INTEGER NOT NULL,
  url TEXT NOT NULL,
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (actor_id) REFERENCES actors(id) ON DELETE CASCADE,
  UNIQUE(actor_id, url)
);

CREATE INDEX IF NOT EXISTS idx_actor_external_links_actor_sort
  ON actor_external_links(actor_id, sort_order, id);

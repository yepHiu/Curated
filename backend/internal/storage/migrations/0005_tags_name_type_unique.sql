-- Allow same display name for metadata (nfo) vs user tags via composite uniqueness.
PRAGMA foreign_keys = OFF;

CREATE TABLE tags_new (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  type TEXT NOT NULL DEFAULT 'nfo',
  UNIQUE(name, type)
);

INSERT INTO tags_new (id, name, type)
SELECT id, name, CASE WHEN type IS NULL OR trim(type) = '' THEN 'nfo' ELSE type END
FROM tags;

DROP TABLE tags;

ALTER TABLE tags_new RENAME TO tags;

CREATE INDEX IF NOT EXISTS idx_movie_tags_movie_id ON movie_tags(movie_id);

PRAGMA foreign_keys = ON;

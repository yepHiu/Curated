ALTER TABLE movies ADD COLUMN provider TEXT NOT NULL DEFAULT '';
ALTER TABLE movies ADD COLUMN homepage TEXT NOT NULL DEFAULT '';
ALTER TABLE movies ADD COLUMN director TEXT NOT NULL DEFAULT '';
ALTER TABLE movies ADD COLUMN release_date TEXT NOT NULL DEFAULT '';
ALTER TABLE movies ADD COLUMN cover_url TEXT NOT NULL DEFAULT '';
ALTER TABLE movies ADD COLUMN thumb_url TEXT NOT NULL DEFAULT '';
ALTER TABLE movies ADD COLUMN preview_video_url TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS actors (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  avatar TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS movie_actors (
  movie_id TEXT NOT NULL,
  actor_id INTEGER NOT NULL,
  PRIMARY KEY (movie_id, actor_id),
  FOREIGN KEY (movie_id) REFERENCES movies(id),
  FOREIGN KEY (actor_id) REFERENCES actors(id)
);

CREATE TABLE IF NOT EXISTS tags (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  type TEXT NOT NULL DEFAULT 'nfo'
);

CREATE TABLE IF NOT EXISTS movie_tags (
  movie_id TEXT NOT NULL,
  tag_id INTEGER NOT NULL,
  PRIMARY KEY (movie_id, tag_id),
  FOREIGN KEY (movie_id) REFERENCES movies(id),
  FOREIGN KEY (tag_id) REFERENCES tags(id)
);

CREATE INDEX IF NOT EXISTS idx_movie_actors_movie_id ON movie_actors(movie_id);
CREATE INDEX IF NOT EXISTS idx_movie_tags_movie_id ON movie_tags(movie_id);

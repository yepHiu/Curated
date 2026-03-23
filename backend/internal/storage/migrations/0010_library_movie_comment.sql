CREATE TABLE IF NOT EXISTS library_movie_comments (
  movie_id TEXT PRIMARY KEY,
  body TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL,
  FOREIGN KEY (movie_id) REFERENCES movies(id) ON DELETE CASCADE
);

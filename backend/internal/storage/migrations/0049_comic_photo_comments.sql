CREATE TABLE IF NOT EXISTS comic_book_comments (
  comic_id TEXT PRIMARY KEY,
  body TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL,
  FOREIGN KEY (comic_id) REFERENCES comic_books(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS photo_book_comments (
  photo_id TEXT PRIMARY KEY,
  body TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL,
  FOREIGN KEY (photo_id) REFERENCES photo_books(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS photo_books (
  id TEXT PRIMARY KEY,
  library_path_id TEXT,
  title TEXT NOT NULL,
  source_file_name TEXT NOT NULL,
  location TEXT NOT NULL UNIQUE,
  file_size INTEGER NOT NULL DEFAULT 0,
  file_modified_at TEXT NOT NULL DEFAULT '',
  page_count INTEGER NOT NULL DEFAULT 0,
  cover_page_index INTEGER NOT NULL DEFAULT 0,
  cover_cache_key TEXT NOT NULL DEFAULT '',
  is_favorite INTEGER NOT NULL DEFAULT 0,
  user_rating REAL,
  added_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  last_viewed_at TEXT NOT NULL DEFAULT '',
  completed_at TEXT NOT NULL DEFAULT '',
  FOREIGN KEY (library_path_id) REFERENCES photo_library_paths(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS photo_pages (
  photo_id TEXT NOT NULL,
  page_index INTEGER NOT NULL,
  entry_path TEXT NOT NULL,
  file_name TEXT NOT NULL,
  image_ext TEXT NOT NULL DEFAULT '',
  width INTEGER NOT NULL DEFAULT 0,
  height INTEGER NOT NULL DEFAULT 0,
  size_bytes INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  PRIMARY KEY (photo_id, page_index),
  FOREIGN KEY (photo_id) REFERENCES photo_books(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS photo_tags (
  name TEXT PRIMARY KEY,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS photo_book_tags (
  photo_id TEXT NOT NULL,
  tag TEXT NOT NULL,
  created_at TEXT NOT NULL,
  PRIMARY KEY (photo_id, tag),
  FOREIGN KEY (photo_id) REFERENCES photo_books(id) ON DELETE CASCADE,
  FOREIGN KEY (tag) REFERENCES photo_tags(name) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS photo_viewing_progress (
  photo_id TEXT PRIMARY KEY,
  current_page_index INTEGER NOT NULL DEFAULT 0,
  completed INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (photo_id) REFERENCES photo_books(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS photo_viewing_preferences (
  photo_id TEXT PRIMARY KEY,
  mode TEXT NOT NULL DEFAULT '',
  fit TEXT NOT NULL DEFAULT '',
  direction TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL,
  FOREIGN KEY (photo_id) REFERENCES photo_books(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS photo_cache_entries (
  cache_key TEXT PRIMARY KEY,
  photo_id TEXT NOT NULL,
  kind TEXT NOT NULL,
  page_index INTEGER NOT NULL DEFAULT -1,
  path TEXT NOT NULL,
  size_bytes INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  last_accessed_at TEXT NOT NULL,
  FOREIGN KEY (photo_id) REFERENCES photo_books(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_photo_books_location ON photo_books(location);
CREATE INDEX IF NOT EXISTS idx_photo_books_added_at ON photo_books(added_at DESC, id ASC);
CREATE INDEX IF NOT EXISTS idx_photo_books_is_favorite ON photo_books(is_favorite);
CREATE INDEX IF NOT EXISTS idx_photo_pages_book_page ON photo_pages(photo_id, page_index);
CREATE INDEX IF NOT EXISTS idx_photo_book_tags_photo ON photo_book_tags(photo_id);
CREATE INDEX IF NOT EXISTS idx_photo_cache_entries_last_accessed ON photo_cache_entries(last_accessed_at);

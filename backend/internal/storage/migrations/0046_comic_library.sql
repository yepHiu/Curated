CREATE TABLE IF NOT EXISTS comic_library_paths (
  id TEXT PRIMARY KEY,
  path TEXT NOT NULL UNIQUE,
  title TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  first_library_scan_pending INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS comic_books (
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
  read_status TEXT NOT NULL DEFAULT 'unread',
  added_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  last_read_at TEXT NOT NULL DEFAULT '',
  completed_at TEXT NOT NULL DEFAULT '',
  FOREIGN KEY (library_path_id) REFERENCES comic_library_paths(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS comic_pages (
  comic_id TEXT NOT NULL,
  page_index INTEGER NOT NULL,
  entry_path TEXT NOT NULL,
  file_name TEXT NOT NULL,
  image_ext TEXT NOT NULL DEFAULT '',
  width INTEGER NOT NULL DEFAULT 0,
  height INTEGER NOT NULL DEFAULT 0,
  size_bytes INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  PRIMARY KEY (comic_id, page_index),
  FOREIGN KEY (comic_id) REFERENCES comic_books(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS comic_tags (
  name TEXT PRIMARY KEY,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS comic_book_tags (
  comic_id TEXT NOT NULL,
  tag TEXT NOT NULL,
  created_at TEXT NOT NULL,
  PRIMARY KEY (comic_id, tag),
  FOREIGN KEY (comic_id) REFERENCES comic_books(id) ON DELETE CASCADE,
  FOREIGN KEY (tag) REFERENCES comic_tags(name) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS comic_reading_progress (
  comic_id TEXT PRIMARY KEY,
  current_page_index INTEGER NOT NULL DEFAULT 0,
  completed INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (comic_id) REFERENCES comic_books(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS comic_reading_preferences (
  comic_id TEXT PRIMARY KEY,
  mode TEXT NOT NULL DEFAULT '',
  fit TEXT NOT NULL DEFAULT '',
  direction TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL,
  FOREIGN KEY (comic_id) REFERENCES comic_books(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS comic_cache_entries (
  cache_key TEXT PRIMARY KEY,
  comic_id TEXT NOT NULL,
  kind TEXT NOT NULL,
  page_index INTEGER NOT NULL DEFAULT -1,
  path TEXT NOT NULL,
  size_bytes INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  last_accessed_at TEXT NOT NULL,
  FOREIGN KEY (comic_id) REFERENCES comic_books(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_comic_library_paths_path ON comic_library_paths(path);
CREATE INDEX IF NOT EXISTS idx_comic_books_location ON comic_books(location);
CREATE INDEX IF NOT EXISTS idx_comic_books_added_at ON comic_books(added_at DESC, id ASC);
CREATE INDEX IF NOT EXISTS idx_comic_books_is_favorite ON comic_books(is_favorite);
CREATE INDEX IF NOT EXISTS idx_comic_books_read_status ON comic_books(read_status);
CREATE INDEX IF NOT EXISTS idx_comic_pages_book_page ON comic_pages(comic_id, page_index);
CREATE INDEX IF NOT EXISTS idx_comic_book_tags_comic ON comic_book_tags(comic_id);
CREATE INDEX IF NOT EXISTS idx_comic_cache_entries_last_accessed ON comic_cache_entries(last_accessed_at);

CREATE TABLE wishlist_items (
 id TEXT PRIMARY KEY,
 code TEXT NOT NULL,
 identity_key TEXT NOT NULL UNIQUE,
 metadata_json TEXT NOT NULL DEFAULT '{}',
 note TEXT NOT NULL DEFAULT '',
 completed INTEGER NOT NULL DEFAULT 0,
 state TEXT NOT NULL DEFAULT 'queued',
 error TEXT NOT NULL DEFAULT '',
 generation INTEGER NOT NULL DEFAULT 1,
 version INTEGER NOT NULL DEFAULT 1,
 created_at TEXT NOT NULL,
 updated_at TEXT NOT NULL
);
CREATE INDEX wishlist_order ON wishlist_items(created_at DESC,id DESC);
CREATE TABLE wishlist_jobs (
 item_id TEXT PRIMARY KEY REFERENCES wishlist_items(id) ON DELETE CASCADE,
 generation INTEGER NOT NULL,
 attempts INTEGER NOT NULL DEFAULT 0,
 next_at INTEGER NOT NULL DEFAULT 0,
 lease_until INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE wishlist_assets (
 id TEXT PRIMARY KEY,
 item_id TEXT NOT NULL REFERENCES wishlist_items(id) ON DELETE CASCADE,
 generation INTEGER NOT NULL,
 role TEXT NOT NULL,
 position INTEGER NOT NULL,
 source_url TEXT NOT NULL,
 path TEXT NOT NULL,
 thumbnail_path TEXT NOT NULL DEFAULT '',
 sha256 TEXT NOT NULL,
 UNIQUE(item_id,generation,role,position)
);
CREATE TABLE wishlist_movie_links (
 item_id TEXT NOT NULL REFERENCES wishlist_items(id) ON DELETE CASCADE,
 movie_id TEXT NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
 excluded INTEGER NOT NULL DEFAULT 0,
 PRIMARY KEY(item_id,movie_id)
);
CREATE TABLE wishlist_tokens (
 id TEXT PRIMARY KEY,
 name TEXT NOT NULL,
 hash TEXT NOT NULL UNIQUE,
 created_at TEXT NOT NULL,
 origin TEXT NOT NULL DEFAULT ''
);

CREATE TABLE ai_book_organization_jobs (
 id TEXT PRIMARY KEY, request_id TEXT NOT NULL UNIQUE, kind TEXT NOT NULL CHECK(kind IN ('comic','photo')),
 locale TEXT NOT NULL, status TEXT NOT NULL, stage TEXT NOT NULL,
 created_at TEXT NOT NULL, updated_at TEXT NOT NULL
);
CREATE TABLE ai_book_organization_items (
 job_id TEXT NOT NULL REFERENCES ai_book_organization_jobs(id) ON DELETE CASCADE,
 book_id TEXT NOT NULL, position INTEGER NOT NULL, title TEXT NOT NULL,
 status TEXT NOT NULL DEFAULT 'pending', revision INTEGER NOT NULL DEFAULT 0,
 snapshot TEXT NOT NULL DEFAULT '', metadata TEXT NOT NULL DEFAULT '{}', reason TEXT NOT NULL DEFAULT '',
 PRIMARY KEY(job_id,book_id)
);

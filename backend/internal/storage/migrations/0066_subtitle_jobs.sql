-- Dialogue checkpoints live in the temporary work directory, never in this table.
CREATE TABLE subtitle_jobs (
 id TEXT PRIMARY KEY,
 request_id TEXT NOT NULL UNIQUE,
 movie_id TEXT NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
 file_id TEXT NOT NULL REFERENCES movie_files(id) ON DELETE CASCADE,
 state TEXT NOT NULL,
 payload TEXT NOT NULL,
 updated_at TEXT NOT NULL
);
CREATE UNIQUE INDEX subtitle_active_file ON subtitle_jobs(file_id) WHERE state IN ('queued','running');
CREATE INDEX subtitle_movie_jobs ON subtitle_jobs(movie_id, updated_at DESC);

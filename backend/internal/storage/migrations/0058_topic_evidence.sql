-- Validated excerpts are stored separately from source metadata; no NFO writes.
ALTER TABLE ai_tag_organization_items ADD COLUMN evidence_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE ai_tag_organization_items ADD COLUMN input_fingerprint TEXT NOT NULL DEFAULT '';
CREATE TABLE movie_user_tag_decisions (
 movie_id TEXT NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
 name TEXT NOT NULL,
 decision TEXT NOT NULL CHECK(decision IN ('keep','exclude')),
 PRIMARY KEY(movie_id,name)
);

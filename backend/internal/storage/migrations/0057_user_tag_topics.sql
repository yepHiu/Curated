-- AI organization owns user tags only. Triggers enforce the mapping invariant
-- even if a caller accidentally supplies an NFO tag ID.
CREATE TABLE library_topics (
  id TEXT PRIMARY KEY,
  tag_id INTEGER NOT NULL UNIQUE REFERENCES tags(id),
  description TEXT NOT NULL DEFAULT '',
  aliases_json TEXT NOT NULL DEFAULT '[]',
  hidden INTEGER NOT NULL DEFAULT 0 CHECK(hidden IN (0,1))
);
CREATE TRIGGER library_topics_user_insert BEFORE INSERT ON library_topics
WHEN NOT EXISTS(SELECT 1 FROM tags WHERE id=NEW.tag_id AND type='user')
BEGIN SELECT RAISE(ABORT, 'topic requires user tag'); END;
CREATE TRIGGER library_topics_user_update BEFORE UPDATE OF tag_id ON library_topics
WHEN NOT EXISTS(SELECT 1 FROM tags WHERE id=NEW.tag_id AND type='user')
BEGIN SELECT RAISE(ABORT, 'topic requires user tag'); END;
CREATE TRIGGER library_topics_preserve_user BEFORE UPDATE OF type ON tags
WHEN NEW.type!='user' AND EXISTS(SELECT 1 FROM library_topics WHERE tag_id=OLD.id)
BEGIN SELECT RAISE(ABORT, 'topic requires user tag'); END;

CREATE TABLE movie_tag_revisions (
  movie_id TEXT PRIMARY KEY REFERENCES movies(id) ON DELETE CASCADE,
  revision INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE movie_topic_decisions (
  movie_id TEXT NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
  topic_id TEXT NOT NULL REFERENCES library_topics(id),
  decision TEXT NOT NULL CHECK(decision IN ('keep','exclude')),
  PRIMARY KEY(movie_id,topic_id)
);
CREATE TABLE ai_tag_organization_jobs (
  id TEXT PRIMARY KEY,
  request_id TEXT NOT NULL UNIQUE,
  status TEXT NOT NULL,
  stage TEXT NOT NULL DEFAULT 'preparing',
  trigger_reason TEXT NOT NULL,
  vocabulary_json TEXT NOT NULL DEFAULT '[]',
  revision INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  error TEXT NOT NULL DEFAULT ''
);
CREATE UNIQUE INDEX ai_tag_organization_one_active ON ai_tag_organization_jobs((1))
WHERE status IN ('queued','running');
CREATE TABLE ai_tag_organization_items (
  job_id TEXT NOT NULL REFERENCES ai_tag_organization_jobs(id),
  movie_id TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  reason TEXT NOT NULL DEFAULT '',
  PRIMARY KEY(job_id,movie_id)
);
-- IDs in history are snapshots, deliberately independent of movie lifecycle.
CREATE TABLE ai_tag_organization_changes (
  job_id TEXT NOT NULL REFERENCES ai_tag_organization_jobs(id),
  movie_id TEXT NOT NULL,
  before_json TEXT NOT NULL,
  after_json TEXT NOT NULL,
  after_revision INTEGER NOT NULL,
  undone INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY(job_id,movie_id)
);
CREATE INDEX ai_tag_organization_items_state ON ai_tag_organization_items(job_id,status);

ALTER TABLE ai_tag_organization_jobs ADD COLUMN tasks_json TEXT NOT NULL DEFAULT '["tags"]';

CREATE TABLE ai_organization_translations (
  job_id TEXT NOT NULL REFERENCES ai_tag_organization_jobs(id),
  movie_id TEXT NOT NULL,
  kind TEXT NOT NULL CHECK(kind IN ('title','summary')),
  status TEXT NOT NULL DEFAULT 'pending',
  reason TEXT NOT NULL DEFAULT '',
  PRIMARY KEY(job_id,movie_id,kind)
);

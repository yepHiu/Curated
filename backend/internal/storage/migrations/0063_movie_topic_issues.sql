-- Failed analysis awaits a user decision; a valid no-match result is complete.
CREATE TABLE movie_topic_issues (
 movie_id TEXT PRIMARY KEY REFERENCES movies(id) ON DELETE CASCADE,
 job_id TEXT NOT NULL REFERENCES ai_tag_organization_jobs(id),
 reason TEXT NOT NULL
);
INSERT INTO movie_topic_issues(movie_id,job_id,reason)
SELECT movie_id,job_id,reason FROM (
 SELECT i.movie_id,i.job_id,i.reason,i.status,
 ROW_NUMBER() OVER(PARTITION BY i.movie_id ORDER BY j.updated_at DESC,j.created_at DESC,i.rowid DESC) AS position
 FROM ai_tag_organization_items i JOIN ai_tag_organization_jobs j ON j.id=i.job_id
 JOIN movies m ON m.id=i.movie_id
 WHERE i.status IN ('succeeded','unresolved','failed','conflict')
) WHERE position=1 AND (status IN ('failed','conflict') OR reason='SOURCE_TOO_LONG');
UPDATE ai_tag_organization_items SET status='failed' WHERE status='unresolved' AND reason='SOURCE_TOO_LONG';
UPDATE ai_tag_organization_jobs SET status='partial_failed'
WHERE status='completed' AND EXISTS(SELECT 1 FROM ai_tag_organization_items i WHERE i.job_id=ai_tag_organization_jobs.id AND i.status='failed');

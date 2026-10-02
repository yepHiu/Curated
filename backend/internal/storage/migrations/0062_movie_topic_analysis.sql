-- Per-movie completion is independent of whether any tag was added.
CREATE TABLE movie_topic_analysis (
 movie_id TEXT PRIMARY KEY REFERENCES movies(id) ON DELETE CASCADE,
 job_id TEXT NOT NULL REFERENCES ai_tag_organization_jobs(id),
 input_fingerprint TEXT NOT NULL,
 outcome TEXT NOT NULL CHECK(outcome IN ('succeeded','unresolved')),
 analyzed_at TEXT NOT NULL
);

-- Older tasks already saved validated input fingerprints. Rank terminal analysis
-- before excluding undone writes so an undo cannot resurrect an older completion.
INSERT INTO movie_topic_analysis(movie_id,job_id,input_fingerprint,outcome,analyzed_at)
SELECT movie_id,job_id,input_fingerprint,status,updated_at FROM (
 SELECT i.movie_id,i.job_id,i.input_fingerprint,i.status,j.updated_at,
        COALESCE(c.undone,0) AS undone,
        ROW_NUMBER() OVER(PARTITION BY i.movie_id ORDER BY j.updated_at DESC,j.created_at DESC,i.rowid DESC) AS position
 FROM ai_tag_organization_items i
 JOIN ai_tag_organization_jobs j ON j.id=i.job_id
 JOIN movies m ON m.id=i.movie_id
 LEFT JOIN ai_tag_organization_changes c ON c.job_id=i.job_id AND c.movie_id=i.movie_id
 WHERE i.status IN ('succeeded','unresolved') AND i.input_fingerprint!=''
) WHERE position=1 AND undone=0;

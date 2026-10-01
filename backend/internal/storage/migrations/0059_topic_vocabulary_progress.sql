-- Persist vocabulary checkpoints separately from classified movie counts.
ALTER TABLE ai_tag_organization_jobs ADD COLUMN vocabulary_processed INTEGER NOT NULL DEFAULT 0;
ALTER TABLE ai_tag_organization_jobs ADD COLUMN vocabulary_ready INTEGER NOT NULL DEFAULT 0;
-- Previous versions only saved the vocabulary after finishing its entire phase.
UPDATE ai_tag_organization_jobs SET vocabulary_ready=1,
 vocabulary_processed=(SELECT COUNT(*) FROM ai_tag_organization_items i WHERE i.job_id=ai_tag_organization_jobs.id)
WHERE vocabulary_json!='[]';

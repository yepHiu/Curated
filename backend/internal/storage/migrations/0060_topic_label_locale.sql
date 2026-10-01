-- Freeze label language per task, including retries and server restarts.
ALTER TABLE ai_tag_organization_jobs ADD COLUMN locale TEXT NOT NULL DEFAULT 'zh-CN' CHECK(locale IN ('zh-CN','en','ja'));

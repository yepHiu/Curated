-- Preserve movie coverage and issue references when users remove task history.
ALTER TABLE ai_tag_organization_jobs ADD COLUMN deleted_at TEXT NOT NULL DEFAULT '';

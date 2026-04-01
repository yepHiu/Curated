ALTER TABLE actors ADD COLUMN avatar_local_path TEXT NOT NULL DEFAULT '';
ALTER TABLE actors ADD COLUMN avatar_last_http_status INTEGER NOT NULL DEFAULT 0;
ALTER TABLE actors ADD COLUMN avatar_last_error TEXT NOT NULL DEFAULT '';
ALTER TABLE actors ADD COLUMN avatar_last_fetched_at TEXT NOT NULL DEFAULT '';

ALTER TABLE media_assets ADD COLUMN source_provider TEXT NOT NULL DEFAULT '';
ALTER TABLE media_assets ADD COLUMN referer_url TEXT NOT NULL DEFAULT '';
ALTER TABLE media_assets ADD COLUMN last_http_status INTEGER NOT NULL DEFAULT 0;
ALTER TABLE media_assets ADD COLUMN last_error TEXT NOT NULL DEFAULT '';
ALTER TABLE media_assets ADD COLUMN last_fetched_at TEXT NOT NULL DEFAULT '';

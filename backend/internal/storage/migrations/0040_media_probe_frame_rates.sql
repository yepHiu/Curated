ALTER TABLE media_probe_cache ADD COLUMN r_frame_rate TEXT NOT NULL DEFAULT '';
ALTER TABLE media_probe_cache ADD COLUMN avg_frame_rate TEXT NOT NULL DEFAULT '';
ALTER TABLE media_probe_cache ADD COLUMN probe_schema INTEGER NOT NULL DEFAULT 0;

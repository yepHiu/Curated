-- Preserve legacy rows that reference missing parents before enforcing foreign keys for new writes.
-- payload_json contains the full source row (BLOBs are losslessly encoded as hexadecimal text).
CREATE TABLE IF NOT EXISTS data_integrity_quarantine (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  source_table TEXT NOT NULL,
  source_key TEXT NOT NULL,
  payload_json TEXT NOT NULL CHECK (json_valid(payload_json)),
  reason TEXT NOT NULL,
  quarantined_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(source_table, source_key, reason)
);

CREATE INDEX IF NOT EXISTS idx_data_integrity_quarantine_source
  ON data_integrity_quarantine(source_table, quarantined_at DESC);

INSERT OR IGNORE INTO data_integrity_quarantine (source_table, source_key, payload_json, reason)
SELECT
  'media_assets',
  id,
  json_object(
    'id', id,
    'movie_id', movie_id,
    'type', type,
    'source_url', source_url,
    'local_path', local_path,
    'created_at', created_at,
    'updated_at', updated_at
  ),
  'missing referenced movie'
FROM media_assets
WHERE NOT EXISTS (SELECT 1 FROM movies WHERE movies.id = media_assets.movie_id);

DELETE FROM media_assets
WHERE NOT EXISTS (SELECT 1 FROM movies WHERE movies.id = media_assets.movie_id);

INSERT OR IGNORE INTO data_integrity_quarantine (source_table, source_key, payload_json, reason)
SELECT
  'movie_actors',
  json_array(movie_id, actor_id),
  json_object('movie_id', movie_id, 'actor_id', actor_id),
  CASE
    WHEN NOT EXISTS (SELECT 1 FROM movies WHERE movies.id = movie_actors.movie_id)
      AND NOT EXISTS (SELECT 1 FROM actors WHERE actors.id = movie_actors.actor_id)
      THEN 'missing referenced movie and actor'
    WHEN NOT EXISTS (SELECT 1 FROM movies WHERE movies.id = movie_actors.movie_id)
      THEN 'missing referenced movie'
    ELSE 'missing referenced actor'
  END
FROM movie_actors
WHERE NOT EXISTS (SELECT 1 FROM movies WHERE movies.id = movie_actors.movie_id)
   OR NOT EXISTS (SELECT 1 FROM actors WHERE actors.id = movie_actors.actor_id);

DELETE FROM movie_actors
WHERE NOT EXISTS (SELECT 1 FROM movies WHERE movies.id = movie_actors.movie_id)
   OR NOT EXISTS (SELECT 1 FROM actors WHERE actors.id = movie_actors.actor_id);

INSERT OR IGNORE INTO data_integrity_quarantine (source_table, source_key, payload_json, reason)
SELECT
  'movie_tags',
  json_array(movie_id, tag_id),
  json_object('movie_id', movie_id, 'tag_id', tag_id),
  CASE
    WHEN NOT EXISTS (SELECT 1 FROM movies WHERE movies.id = movie_tags.movie_id)
      AND NOT EXISTS (SELECT 1 FROM tags WHERE tags.id = movie_tags.tag_id)
      THEN 'missing referenced movie and tag'
    WHEN NOT EXISTS (SELECT 1 FROM movies WHERE movies.id = movie_tags.movie_id)
      THEN 'missing referenced movie'
    ELSE 'missing referenced tag'
  END
FROM movie_tags
WHERE NOT EXISTS (SELECT 1 FROM movies WHERE movies.id = movie_tags.movie_id)
   OR NOT EXISTS (SELECT 1 FROM tags WHERE tags.id = movie_tags.tag_id);

DELETE FROM movie_tags
WHERE NOT EXISTS (SELECT 1 FROM movies WHERE movies.id = movie_tags.movie_id)
   OR NOT EXISTS (SELECT 1 FROM tags WHERE tags.id = movie_tags.tag_id);

INSERT OR IGNORE INTO data_integrity_quarantine (source_table, source_key, payload_json, reason)
SELECT
  'playback_progress',
  movie_id,
  json_object(
    'movie_id', movie_id,
    'position_sec', position_sec,
    'duration_sec', duration_sec,
    'updated_at', updated_at
  ),
  'missing referenced movie'
FROM playback_progress
WHERE NOT EXISTS (SELECT 1 FROM movies WHERE movies.id = playback_progress.movie_id);

DELETE FROM playback_progress
WHERE NOT EXISTS (SELECT 1 FROM movies WHERE movies.id = playback_progress.movie_id);

INSERT OR IGNORE INTO data_integrity_quarantine (source_table, source_key, payload_json, reason)
SELECT
  'curated_frames',
  id,
  json_object(
    'id', id,
    'movie_id', movie_id,
    'title', title,
    'code', code,
    'actors_json', actors_json,
    'position_sec', position_sec,
    'captured_at', captured_at,
    'tags_json', tags_json,
    'image_blob_hex', hex(image_blob),
    'created_at', created_at,
    'thumb_blob_hex', CASE WHEN thumb_blob IS NULL THEN NULL ELSE hex(thumb_blob) END
  ),
  'missing referenced movie'
FROM curated_frames
WHERE NOT EXISTS (SELECT 1 FROM movies WHERE movies.id = curated_frames.movie_id);

DELETE FROM curated_frames
WHERE NOT EXISTS (SELECT 1 FROM movies WHERE movies.id = curated_frames.movie_id);

INSERT OR IGNORE INTO data_integrity_quarantine (source_table, source_key, payload_json, reason)
SELECT
  'library_played_movies',
  movie_id,
  json_object('movie_id', movie_id, 'first_played_at', first_played_at),
  'missing referenced movie'
FROM library_played_movies
WHERE NOT EXISTS (SELECT 1 FROM movies WHERE movies.id = library_played_movies.movie_id);

DELETE FROM library_played_movies
WHERE NOT EXISTS (SELECT 1 FROM movies WHERE movies.id = library_played_movies.movie_id);

INSERT OR IGNORE INTO data_integrity_quarantine (source_table, source_key, payload_json, reason)
SELECT
  'actor_user_tags',
  json_array(actor_id, tag),
  json_object('actor_id', actor_id, 'tag', tag),
  'missing referenced actor'
FROM actor_user_tags
WHERE NOT EXISTS (SELECT 1 FROM actors WHERE actors.id = actor_user_tags.actor_id);

DELETE FROM actor_user_tags
WHERE NOT EXISTS (SELECT 1 FROM actors WHERE actors.id = actor_user_tags.actor_id);

INSERT OR IGNORE INTO data_integrity_quarantine (source_table, source_key, payload_json, reason)
SELECT
  'library_movie_comments',
  movie_id,
  json_object('movie_id', movie_id, 'body', body, 'updated_at', updated_at),
  'missing referenced movie'
FROM library_movie_comments
WHERE NOT EXISTS (SELECT 1 FROM movies WHERE movies.id = library_movie_comments.movie_id);

DELETE FROM library_movie_comments
WHERE NOT EXISTS (SELECT 1 FROM movies WHERE movies.id = library_movie_comments.movie_id);

INSERT OR IGNORE INTO data_integrity_quarantine (source_table, source_key, payload_json, reason)
SELECT
  'actor_external_links',
  CAST(id AS TEXT),
  json_object(
    'id', id,
    'actor_id', actor_id,
    'url', url,
    'sort_order', sort_order,
    'created_at', created_at,
    'updated_at', updated_at
  ),
  'missing referenced actor'
FROM actor_external_links
WHERE NOT EXISTS (SELECT 1 FROM actors WHERE actors.id = actor_external_links.actor_id);

DELETE FROM actor_external_links
WHERE NOT EXISTS (SELECT 1 FROM actors WHERE actors.id = actor_external_links.actor_id);

INSERT OR IGNORE INTO data_integrity_quarantine (source_table, source_key, payload_json, reason)
SELECT
  'playback_daily_watch_time',
  json_array(day_key, movie_id),
  json_object(
    'day_key', day_key,
    'movie_id', movie_id,
    'watched_sec', watched_sec,
    'updated_at', updated_at
  ),
  'missing referenced movie'
FROM playback_daily_watch_time
WHERE NOT EXISTS (SELECT 1 FROM movies WHERE movies.id = playback_daily_watch_time.movie_id);

DELETE FROM playback_daily_watch_time
WHERE NOT EXISTS (SELECT 1 FROM movies WHERE movies.id = playback_daily_watch_time.movie_id);

INSERT OR IGNORE INTO data_integrity_quarantine (source_table, source_key, payload_json, reason)
SELECT
  'library_path_storage_bindings',
  library_path_id,
  json_object(
    'library_path_id', library_path_id,
    'root_path', root_path,
    'volume_id', volume_id,
    'volume_label', volume_label,
    'file_system', file_system,
    'drive_type', drive_type,
    'identity_confidence', identity_confidence,
    'bound_at', bound_at,
    'last_seen_at', last_seen_at,
    'last_checked_at', last_checked_at,
    'last_status', last_status,
    'last_error', last_error,
    'updated_at', updated_at
  ),
  'missing referenced library path'
FROM library_path_storage_bindings
WHERE NOT EXISTS (
  SELECT 1 FROM library_paths WHERE library_paths.id = library_path_storage_bindings.library_path_id
);

DELETE FROM library_path_storage_bindings
WHERE NOT EXISTS (
  SELECT 1 FROM library_paths WHERE library_paths.id = library_path_storage_bindings.library_path_id
);

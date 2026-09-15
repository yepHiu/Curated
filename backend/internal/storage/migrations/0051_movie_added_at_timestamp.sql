-- Promote date-only movie added_at to the insert timestamp so library
-- "added" sort follows ingestion time instead of catalog-code id.
UPDATE movies
SET added_at = CASE
  WHEN instr(created_at, 'T') > 0 THEN created_at
  WHEN instr(trim(created_at), ' ') > 0 THEN replace(substr(trim(created_at), 1, 19), ' ', 'T') || 'Z'
  ELSE added_at
END
WHERE length(trim(added_at)) <= 10
  AND trim(created_at) != '';

DROP INDEX IF EXISTS idx_movies_trash_sort;
CREATE INDEX IF NOT EXISTS idx_movies_trash_sort
  ON movies(trashed_at, added_at DESC, created_at DESC);

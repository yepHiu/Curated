-- Display-title overlay for comic and photo books.
-- Scan continues to update the source `title` from the filename.
-- List/detail APIs expose COALESCE(NULLIF(TRIM(user_title), ''), title).

ALTER TABLE comic_books ADD COLUMN user_title TEXT NOT NULL DEFAULT '';
ALTER TABLE photo_books ADD COLUMN user_title TEXT NOT NULL DEFAULT '';

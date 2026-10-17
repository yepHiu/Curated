ALTER TABLE comic_books ADD COLUMN metadata_json TEXT NOT NULL DEFAULT '{}';
ALTER TABLE photo_books ADD COLUMN metadata_json TEXT NOT NULL DEFAULT '{}';

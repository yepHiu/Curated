-- User-editable display overrides; scraped columns remain the source when overrides are NULL/empty.

ALTER TABLE movies ADD COLUMN user_title TEXT;
ALTER TABLE movies ADD COLUMN user_studio TEXT;
ALTER TABLE movies ADD COLUMN user_summary TEXT;
ALTER TABLE movies ADD COLUMN user_release_date TEXT;
ALTER TABLE movies ADD COLUMN user_runtime_minutes INTEGER;

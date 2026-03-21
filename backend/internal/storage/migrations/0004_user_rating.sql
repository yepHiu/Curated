-- User-owned score override; scraper continues to write movies.rating (metadata).
ALTER TABLE movies ADD COLUMN user_rating REAL;

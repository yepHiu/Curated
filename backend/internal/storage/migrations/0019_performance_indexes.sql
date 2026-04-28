-- 0019_performance_indexes: add covering indexes for frequent query patterns.

-- Active/trash filter + default sort (added_at DESC, id ASC) used by every movie list query.
CREATE INDEX IF NOT EXISTS idx_movies_trash_sort
    ON movies(trashed_at, added_at DESC, id ASC);

-- Favorites filter (is_favorite = 1) used when browsing favorites.
CREATE INDEX IF NOT EXISTS idx_movies_favorite
    ON movies(is_favorite);

-- Actor filtering: EXISTS subquery in buildMovieFilters joins movie_actors by actor_id.
-- Only movie_actors(movie_id) was indexed in 0003; this covers the reverse direction.
CREATE INDEX IF NOT EXISTS idx_movie_actors_actor_id
    ON movie_actors(actor_id);

-- Asset lookup by movie + type is the dominant query pattern for covers/thumbs/previews.
-- Only media_assets(movie_id) was indexed in 0001; this compound index covers both columns.
CREATE INDEX IF NOT EXISTS idx_media_assets_movie_type
    ON media_assets(movie_id, type);

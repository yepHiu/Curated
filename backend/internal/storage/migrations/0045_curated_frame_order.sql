CREATE INDEX IF NOT EXISTS idx_curated_frames_capture_id ON curated_frames(captured_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_curated_frames_movie_capture_id ON curated_frames(movie_id, captured_at DESC, id DESC);

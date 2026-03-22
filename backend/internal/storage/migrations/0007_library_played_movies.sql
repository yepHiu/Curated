-- 设置页「已播放影片」统计：进入过播放页的去重 movie_id（与前端 played-movies 一致）
CREATE TABLE IF NOT EXISTS library_played_movies (
  movie_id TEXT PRIMARY KEY,
  first_played_at TEXT NOT NULL,
  FOREIGN KEY (movie_id) REFERENCES movies(id) ON DELETE CASCADE
);

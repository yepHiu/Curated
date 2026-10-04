CREATE TABLE movie_files (
 id TEXT PRIMARY KEY,
 movie_id TEXT NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
 location TEXT NOT NULL UNIQUE,
 part_index INTEGER NOT NULL DEFAULT 0 CHECK (part_index >= 0),
 created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_movie_files_movie ON movie_files(movie_id, part_index);
INSERT INTO movie_files(id, movie_id, location)
SELECT id || ':primary', id, location FROM movies WHERE TRIM(location) != '';

CREATE TABLE movie_file_progress (
 file_id TEXT PRIMARY KEY REFERENCES movie_files(id) ON DELETE CASCADE,
 position_sec REAL NOT NULL DEFAULT 0,
 duration_sec REAL NOT NULL DEFAULT 0,
 updated_at TEXT NOT NULL
);
ALTER TABLE playback_progress ADD COLUMN file_id TEXT NOT NULL DEFAULT '';
UPDATE playback_progress SET file_id = movie_id || ':primary'
WHERE EXISTS (SELECT 1 FROM movie_files WHERE id = movie_id || ':primary');
INSERT INTO movie_file_progress(file_id, position_sec, duration_sec, updated_at)
SELECT file_id, position_sec, duration_sec, updated_at FROM playback_progress WHERE file_id != '';

ALTER TABLE curated_frames ADD COLUMN file_id TEXT NOT NULL DEFAULT '';
UPDATE curated_frames SET file_id = movie_id || ':primary'
WHERE EXISTS (SELECT 1 FROM movie_files WHERE id = movie_id || ':primary');

-- 兼容既有导入、种子与手工建片入口，不要求每个调用方重复创建首个文件。
CREATE TRIGGER movie_primary_file_insert AFTER INSERT ON movies
WHEN TRIM(NEW.location) != '' BEGIN
 INSERT INTO movie_files(id, movie_id, location) VALUES(NEW.id || ':primary', NEW.id, NEW.location);
END;
-- 单文件旧入口改路径时保持文件 ID 和进度；切换到已登记分片不改变其余路径。
CREATE TRIGGER movie_primary_file_relocate AFTER UPDATE OF location ON movies
WHEN NEW.location != OLD.location
 AND NOT EXISTS (SELECT 1 FROM movie_files WHERE movie_id = NEW.id AND location = NEW.location)
BEGIN
 UPDATE movie_files SET location = NEW.location WHERE movie_id = NEW.id AND location = OLD.location;
END;

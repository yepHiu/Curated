package storage

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"curated-backend/internal/contracts"
	"curated-backend/internal/library/moviecode"
)

// ListMovieFiles 返回作品的所有文件，数字分片顺序稳定，未标序号者按文件名排列。
func (s *SQLiteStore) ListMovieFiles(ctx context.Context, movieID, code string) ([]contracts.MovieFileDTO, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, location, part_index FROM movie_files WHERE movie_id = ?`, movieID)
	if err != nil {
		return nil, err
	}
	files := make([]contracts.MovieFileDTO, 0)
	for rows.Next() {
		var file contracts.MovieFileDTO
		if err := rows.Scan(&file.ID, &file.Location, &file.PartIndex); err != nil {
			rows.Close()
			return nil, err
		}
		file.FileName = filepath.Base(file.Location)
		if file.PartIndex == 0 {
			file.PartIndex = moviecode.ExtractPartIndex(file.Location, code)
		}
		files = append(files, file)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	sort.SliceStable(files, func(i, j int) bool {
		// 未标分片的原始文件在明确序号前，序号按数字而非字典排序。
		if files[i].PartIndex != files[j].PartIndex {
			return files[i].PartIndex < files[j].PartIndex
		}
		return strings.ToLower(files[i].FileName) < strings.ToLower(files[j].FileName)
	})
	return files, nil
}

// GetMoviePlaybackDetail 校验文件归属并选择视频；未指定时恢复上次分片或从首片开始。
func (s *SQLiteStore) GetMoviePlaybackDetail(ctx context.Context, movieID string) (contracts.MovieDetailDTO, error) {
	detail, err := s.GetMovieDetail(ctx, movieID)
	if err != nil {
		return detail, err
	}
	selected := contracts.MovieFileSelection(ctx)
	if selected == "" {
		_ = s.db.QueryRowContext(ctx, `SELECT file_id FROM playback_progress WHERE movie_id = ?`, movieID).Scan(&selected)
	}
	for _, file := range detail.Files {
		if selected == file.ID || selected == "" {
			detail.Location = file.Location
			detail.SelectedFileID = file.ID
			return detail, nil
		}
	}
	if selected != "" {
		return contracts.MovieDetailDTO{}, sql.ErrNoRows
	}
	return detail, nil
}

// saveScanMovieFileTx 追加或重复识别同一作品文件，绝不覆盖另一分片路径。
func saveScanMovieFileTx(ctx context.Context, tx *sql.Tx, movieID, code, path string) error {
	sum := sha256.Sum256([]byte(movieID + "\x00" + filepath.Clean(path)))
	id := fmt.Sprintf("file-%x", sum[:12])
	_, err := tx.ExecContext(ctx, `INSERT INTO movie_files(id, movie_id, location, part_index)
 VALUES(?, ?, ?, ?) ON CONFLICT(location) DO UPDATE SET part_index=excluded.part_index
 WHERE movie_files.movie_id=excluded.movie_id`, id, movieID, path, moviecode.ExtractPartIndex(path, code))
	if err != nil {
		return err
	}
	// location 仅作为旧客户端的首个文件投影；新播放器明确传递文件 ID。
	_, err = tx.ExecContext(ctx, `UPDATE movies SET location=(SELECT location FROM movie_files
 WHERE movie_id=? ORDER BY part_index, location LIMIT 1), updated_at=? WHERE id=?`, movieID, nowUTC(), movieID)
	return err
}

// RelocateMovieFile 在扫描整理改名后同步已登记的子路径和旧主路径投影，不改变文件 ID。
func (s *SQLiteStore) RelocateMovieFile(ctx context.Context, oldPath, newPath string) error {
	if filepath.Clean(oldPath) == filepath.Clean(newPath) {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `UPDATE movie_files SET location=? WHERE location=?`, newPath, oldPath); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE movies SET location=? WHERE location=?`, newPath, oldPath); err != nil {
		return err
	}
	return tx.Commit()
}

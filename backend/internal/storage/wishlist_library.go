package storage

import (
	"context"
	"path/filepath"
)

// MovieFilesForWishlist 仅从现有允许的库/缓存路径读取图片用于愿望持久副本。
func (s *SQLiteStore) MovieFilesForWishlist(ctx context.Context, movieID, cacheDir string) ([]WishlistAssetFile, error) {
	policy := s.loadPosterPathPolicy(ctx, cacheDir)
	rows, e := s.db.QueryContext(ctx, `SELECT type,source_url,local_path FROM media_assets WHERE movie_id=? ORDER BY type,created_at,id`, movieID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []WishlistAssetFile{}
	positions := map[string]int{}
	for rows.Next() {
		var f WishlistAssetFile
		if e = rows.Scan(&f.Role, &f.SourceURL, &f.Path); e != nil {
			return nil, e
		}
		f.Position = positions[f.Role]
		positions[f.Role]++
		abs, e := filepath.Abs(f.Path)
		if e != nil || !mediaAssetPathAllowedWithPolicy(abs, policy) {
			continue
		}
		f.Path = abs
		out = append(out, f)
	}
	return out, rows.Err()
}

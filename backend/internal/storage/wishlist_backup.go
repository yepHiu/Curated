package storage

import (
	"context"
	"curated-backend/internal/processlock"
	"path/filepath"
)

// LockWishlistAssets 协调跨进程备份和资产回收；不可变新增不需要持锁。
func (s *SQLiteStore) LockWishlistAssets() (*processlock.Lock, error) {
	root, e := s.WishlistAssetRoot()
	if e != nil {
		return nil, e
	}
	return processlock.Acquire(filepath.Join(root, ".maintenance.lock"))
}

// WishlistBackupFiles 从同一数据库快照返回所有被引用的文件，兼容没有愿望表的旧备份。
func (s *SQLiteStore) WishlistBackupFiles(ctx context.Context) (map[string]string, error) {
	var count int
	if e := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='wishlist_assets'`).Scan(&count); e != nil {
		return nil, e
	}
	out := map[string]string{}
	if count == 0 {
		return out, nil
	}
	rows, e := s.db.QueryContext(ctx, `SELECT path,thumbnail_path,sha256 FROM wishlist_assets`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	for rows.Next() {
		var path, thumb, hash string
		if e = rows.Scan(&path, &thumb, &hash); e != nil {
			return nil, e
		}
		out[path] = hash
		if thumb != "" {
			out[thumb] = ""
		}
	}
	return out, rows.Err()
}

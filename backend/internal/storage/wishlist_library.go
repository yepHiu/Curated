package storage

import (
	"context"
	"curated-backend/internal/contracts"
	"fmt"
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

// WishlistForMovie 返回唯一已确认关联的愿望资料，回收站不参与复用。
func (s *SQLiteStore) WishlistForMovie(ctx context.Context, movieID string) (contracts.WishlistItemDTO, error) {
	var id string
	err := s.db.QueryRowContext(ctx, `SELECT l.item_id FROM wishlist_movie_links l JOIN movies m ON m.id=l.movie_id JOIN wishlist_items w ON w.id=l.item_id WHERE l.movie_id=? AND l.excluded=0 AND COALESCE(m.trashed_at,'')='' AND w.state IN ('ready','partial') LIMIT 1`, movieID).Scan(&id)
	if err != nil {
		return contracts.WishlistItemDTO{}, err
	}
	return s.GetWishlist(ctx, id)
}

// FillMovieFromWishlist 在事务中仅补空刮削字段，保留人工覆盖和已有 NFO 数据。
func (s *SQLiteStore) FillMovieFromWishlist(ctx context.Context, movieID string, m contracts.WishlistMetadata) error {
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	_, e = tx.ExecContext(ctx, `UPDATE movies SET
 title=CASE WHEN title='' OR title=code THEN ? ELSE title END,
 summary=CASE WHEN summary='' OR summary='Metadata pending scrape.' THEN ? ELSE summary END,
 studio=CASE WHEN studio='' OR studio='Unknown' THEN ? ELSE studio END,
 runtime_minutes=CASE WHEN runtime_minutes=0 THEN ? ELSE runtime_minutes END,
 release_date=CASE WHEN release_date='' THEN ? ELSE release_date END,
 provider=CASE WHEN provider='' THEN ? ELSE provider END,
 homepage=CASE WHEN homepage='' THEN ? ELSE homepage END,updated_at=? WHERE id=? AND COALESCE(trashed_at,'')=''`, m.Title, m.Summary, m.Studio, m.RuntimeMinutes, m.ReleaseDate, m.Provider, m.Homepage, nowUTC(), movieID)
	if e != nil {
		return e
	}
	var count int
	if e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM movie_actors WHERE movie_id=?`, movieID).Scan(&count); e != nil {
		return e
	}
	if count == 0 {
		if e = replaceMovieActors(ctx, tx, movieID, m.Actors); e != nil {
			return e
		}
	}
	if e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM movie_tags WHERE movie_id=?`, movieID).Scan(&count); e != nil {
		return e
	}
	if count == 0 {
		if e = replaceMovieTags(ctx, tx, movieID, m.Tags); e != nil {
			return e
		}
	}
	return tx.Commit()
}

// RegisterWishlistMovieAsset 只填补缺少本地资产的角色，保留已指定图片。
func (s *SQLiteStore) RegisterWishlistMovieAsset(ctx context.Context, movieID string, a WishlistAssetFile, localPath string) error {
	id := movieID + ":" + a.Role
	if a.Role == "preview_image" {
		id = fmt.Sprintf("%s:preview:%02d", movieID, a.Position+1)
	}
	_, e := s.db.ExecContext(ctx, `INSERT INTO media_assets(id,movie_id,type,source_url,local_path,created_at,updated_at) VALUES(?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET source_url=excluded.source_url,local_path=excluded.local_path,updated_at=excluded.updated_at WHERE media_assets.local_path=''`, id, movieID, a.Role, a.SourceURL, localPath, nowUTC(), nowUTC())
	if e != nil {
		return e
	}
	switch a.Role {
	case "cover":
		_, e = s.db.ExecContext(ctx, `UPDATE movies SET cover_url=? WHERE id=? AND cover_url=''`, a.SourceURL, movieID)
	case "thumb":
		_, e = s.db.ExecContext(ctx, `UPDATE movies SET thumb_url=? WHERE id=? AND thumb_url=''`, a.SourceURL, movieID)
	}
	return e
}

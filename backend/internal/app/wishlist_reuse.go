package app

import (
	"context"
	"crypto/sha256"
	"curated-backend/internal/storage"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

// reuseWishlistForMovie 在自动刮削前补齐愿望资料；完整命中避免重复联网。
func (a *App) reuseWishlistForMovie(ctx context.Context, movieID string) bool {
	if e := a.store.ReconcileWishlist(ctx); e != nil {
		return false
	}
	wish, e := a.store.WishlistForMovie(ctx, movieID)
	if e != nil || wish.Metadata.Title == "" {
		return false
	}
	if e = a.store.FillMovieFromWishlist(ctx, movieID, wish.Metadata); e != nil {
		return false
	}
	movie, e := a.store.GetMovieDetail(ctx, movieID)
	if e != nil {
		return false
	}
	root, e := a.store.WishlistAssetRoot()
	if e != nil {
		return false
	}
	files, e := a.store.WishlistAssetFiles(ctx, wish.ID)
	if e != nil {
		return false
	}
	current, _ := a.store.MovieFilesForWishlist(ctx, movieID, a.cfg.CacheDir)
	dest := filepath.Join(a.cfg.CacheDir, movieID)
	if a.OrganizeLibrary() && movie.Location != "" {
		dest = filepath.Dir(movie.Location)
	}
	for _, file := range files {
		exists := false
		for _, existing := range current {
			if existing.Role == file.Role && existing.Position == file.Position {
				exists = true
				break
			}
		}
		if exists {
			continue
		}
		target, e := copyWishlistAsset(root, dest, file)
		if e != nil {
			continue
		}
		// 登记失败由周期复制再次尝试，不能触发整片重复联网刮削。
		_ = a.store.RegisterWishlistMovieAsset(ctx, movieID, file, target)
	}
	// 资料已复用但图像不全仍跳过整片重复刮削，愿望后台刷新和周期复制补偿图片。
	return true
}

// copyWishlistAsset 校验内容哈希后复制到影片受管理目录，不移动愿望原件。
func copyWishlistAsset(root, dest string, file storage.WishlistAssetFile) (string, error) {
	data, e := os.ReadFile(filepath.Join(root, filepath.FromSlash(file.Path)))
	if e != nil {
		return "", e
	}
	hash := sha256.Sum256(data)
	if hex.EncodeToString(hash[:]) != file.SHA256 {
		return "", fmt.Errorf("wishlist asset hash mismatch")
	}
	if e = os.MkdirAll(dest, 0755); e != nil {
		return "", e
	}
	target := filepath.Join(dest, "wishlist-"+file.SHA256+filepath.Ext(file.Path))
	if existing, e := os.ReadFile(target); e == nil {
		h := sha256.Sum256(existing)
		if h == hash {
			return target, nil
		}
		return "", fmt.Errorf("existing asset differs")
	}
	f, e := os.CreateTemp(dest, ".wishlist-copy-*")
	if e != nil {
		return "", e
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if _, e = f.Write(data); e != nil {
		f.Close()
		return "", e
	}
	if e = f.Close(); e != nil {
		return "", e
	}
	if e = os.Rename(tmp, target); e != nil {
		return "", e
	}
	return target, nil
}

// reconcileWishlistLibrary 周期补偿入库提交后崩溃、恢复影片及图片迟到的场景。
func (a *App) reconcileWishlistLibrary(ctx context.Context) {
	if e := a.store.ReconcileWishlist(ctx); e != nil {
		return
	}
	cursor := ""
	for {
		if ctx.Err() != nil {
			return
		}
		page, e := a.store.ListWishlist(ctx, "in_library", "", cursor, 60)
		if e != nil {
			return
		}
		for _, wish := range page.Items {
			for _, id := range wish.MovieIDs {
				a.reuseWishlistForMovie(ctx, id)
			}
		}
		if page.NextCursor == "" {
			return
		}
		cursor = page.NextCursor
	}
}

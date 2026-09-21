package app

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// collectWishlistAssets 在跨进程备份锁保护下延迟回收无引用文件，不跟随符号链接。
func (a *App) collectWishlistAssets(ctx context.Context) {
	lock, e := a.store.LockWishlistAssets()
	if e != nil {
		return
	}
	defer lock.Release()
	root, e := a.store.WishlistAssetRoot()
	if e != nil {
		return
	}
	refs, e := a.store.WishlistBackupFiles(ctx)
	if e != nil {
		return
	}
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error { // 只删除此根下超过一天且没有数据库引用的普通文件。
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if walkErr != nil {
			return nil
		}
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || entry.Name() == ".maintenance.lock" {
			return nil
		}
		rel, e := filepath.Rel(root, path)
		if e != nil || strings.HasPrefix(rel, "..") {
			return nil
		}
		if _, ok := refs[filepath.ToSlash(rel)]; ok {
			return nil
		}
		info, e := entry.Info()
		if e == nil && info.Mode().IsRegular() && time.Since(info.ModTime()) > 24*time.Hour {
			_ = os.Remove(path)
		}
		return nil
	})
}

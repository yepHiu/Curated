package backup

import (
	"archive/zip"
	"context"
	"curated-backend/internal/storage"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const wishlistArchivePrefix = "assets/wishlist/"

// wishlistBackupSources 校验快照引用与原图哈希，资产不可变且回收锁由调用方持有。
func wishlistBackupSources(ctx context.Context, snapshot *storage.SQLiteStore, root string) ([]FileEntry, map[string]string, error) {
	refs, e := snapshot.WishlistBackupFiles(ctx)
	if e != nil {
		return nil, nil, e
	}
	entries := []FileEntry{}
	sources := map[string]string{}
	for relative, expected := range refs {
		if e = validateArchivePath(relative); e != nil {
			return nil, nil, e
		}
		source := filepath.Join(root, filepath.FromSlash(relative))
		entry, e := describeFile("wishlist-asset", wishlistArchivePrefix+relative, source)
		if e != nil {
			return nil, nil, e
		}
		if expected != "" && entry.SHA256 != expected {
			return nil, nil, fmt.Errorf("wishlist original hash mismatch: %s", relative)
		}
		entries = append(entries, entry)
		sources[entry.Path] = source
	}
	return entries, sources, nil
}

// restoreWishlistAssets 先发布不可变资产再替换数据库，中断仅留下无引用文件，不破坏旧库。
func restoreWishlistAssets(archive map[string]*zip.File, manifest Manifest, databasePath string) error {
	root := filepath.Join(filepath.Dir(databasePath), "assets", "wishlist")
	for _, entry := range manifest.Files {
		if entry.Kind != "wishlist-asset" {
			continue
		}
		relative := strings.TrimPrefix(entry.Path, wishlistArchivePrefix)
		if relative == entry.Path {
			return fmt.Errorf("invalid wishlist archive path")
		}
		target := filepath.Join(root, filepath.FromSlash(relative))
		if _, e := os.Stat(target); e == nil {
			existing, e := describeFile("wishlist-asset", entry.Path, target)
			if e != nil {
				return e
			}
			if existing.SHA256 != entry.SHA256 {
				return fmt.Errorf("asset conflict at %s", relative)
			}
			continue
		} else if !os.IsNotExist(e) {
			return e
		}
		if e := os.MkdirAll(filepath.Dir(target), 0700); e != nil {
			return e
		}
		tmp, e := extractEntryToTargetTemp(archive[entry.Path], entry, filepath.Dir(target))
		if e != nil {
			return e
		}
		if e = os.Rename(tmp, target); e != nil {
			os.Remove(tmp)
			return e
		}
	}
	return nil
}

// verifyWishlistReferences 检查数据库引用与包内文件集合，防止完整哈希包仍缺少所需图片。
func verifyWishlistReferences(ctx context.Context, snapshot *storage.SQLiteStore, manifest Manifest) error {
	refs, e := snapshot.WishlistBackupFiles(ctx)
	if e != nil {
		return e
	}
	if len(refs) == 0 {
		return nil
	}
	entries := map[string]FileEntry{}
	for _, entry := range manifest.Files {
		entries[entry.Path] = entry
	}
	for path, hash := range refs {
		entry, ok := entries[wishlistArchivePrefix+path]
		if !ok || entry.Kind != "wishlist-asset" {
			return fmt.Errorf("missing referenced wishlist asset %s", path)
		}
		if hash != "" && entry.SHA256 != hash {
			return fmt.Errorf("wishlist reference hash mismatch %s", path)
		}
	}
	return nil
}

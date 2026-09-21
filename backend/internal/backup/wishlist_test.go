package backup

import (
	"bytes"
	"context"
	"curated-backend/internal/storage"
	"curated-backend/internal/wishlistassets"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// TestWishlistBackupRestore 验证图片进入快照包并可在全新数据根离线恢复。
func TestWishlistBackupRestore(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, e := storage.NewSQLiteStore(filepath.Join(dir, "source", "curated.db"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if e = s.Migrate(ctx); e != nil {
		t.Fatal(e)
	}
	id, _, e := s.AddWishlist(ctx, "SSIS-001")
	if e != nil {
		t.Fatal(e)
	}
	root, e := s.WishlistAssetRoot()
	if e != nil {
		t.Fatal(e)
	}
	var b bytes.Buffer
	png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 32, 48)))
	file, e := wishlistassets.Save(root, id, b.Bytes())
	if e != nil {
		t.Fatal(e)
	}
	if e = s.SaveWishlistAsset(ctx, storage.WishlistAssetFile{ID: "poster", ItemID: id, Generation: 1, Role: "cover", Path: file.Path, ThumbnailPath: file.ThumbnailPath, SHA256: file.Hash}); e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(dir, "wishlist.curated-backup")
	m, e := Create(ctx, CreateOptions{Store: s, DestinationPath: path})
	if e != nil {
		t.Fatal(e)
	}
	if m.FormatVersion != 2 || !m.Scope.WishlistAssetsIncluded {
		t.Fatal("missing asset scope")
	}
	target := filepath.Join(dir, "restored", "curated.db")
	_, e = Restore(ctx, RestoreOptions{Confirm: true, PreflightOptions: PreflightOptions{BackupPath: path, TargetDatabase: target}})
	if e != nil {
		t.Fatal(e)
	}
	restored, e := os.ReadFile(filepath.Join(filepath.Dir(target), "assets", "wishlist", filepath.FromSlash(file.Path)))
	if e != nil || !bytes.Equal(restored, b.Bytes()) {
		t.Fatalf("asset restore %v", e)
	}
	if e = os.Remove(filepath.Join(root, filepath.FromSlash(file.Path))); e != nil {
		t.Fatal(e)
	}
	if _, e = Create(ctx, CreateOptions{Store: s, DestinationPath: filepath.Join(dir, "missing.curated-backup")}); e == nil {
		t.Fatal("missing original accepted")
	}
}

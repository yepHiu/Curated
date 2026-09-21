package app

import (
	"bytes"
	"context"
	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
	"curated-backend/internal/wishlistassets"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// TestWishlistLibraryReuse 验证已保存图片直接复制且完成资料不再次联网。
func TestWishlistLibraryReuse(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, e := storage.NewSQLiteStore(filepath.Join(dir, "curated.db"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if e = s.Migrate(ctx); e != nil {
		t.Fatal(e)
	}
	id, _, _ := s.AddWishlist(ctx, "SSIS-001")
	if e = s.SaveWishlistMetadata(ctx, id, 1, contracts.WishlistMetadata{Title: "Saved title", Actors: []string{"Actor"}, Provider: "test"}); e != nil {
		t.Fatal(e)
	}
	root, _ := s.WishlistAssetRoot()
	var b bytes.Buffer
	png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 30, 40)))
	f, e := wishlistassets.Save(root, id, b.Bytes())
	if e != nil {
		t.Fatal(e)
	}
	if e = s.SaveWishlistAsset(ctx, storage.WishlistAssetFile{ID: "cover", ItemID: id, Generation: 1, Role: "cover", SourceURL: "https://example.com/cover.png", Path: f.Path, ThumbnailPath: f.ThumbnailPath, SHA256: f.Hash}); e != nil {
		t.Fatal(e)
	}
	s.FinishWishlistJob(ctx, id, 1, 1, "ready", "", false)
	movie, e := s.PersistScanMovie(ctx, contracts.ScanFileResultDTO{Number: "SSIS-001", Path: filepath.Join(dir, "movie.mp4")})
	if e != nil {
		t.Fatal(e)
	}
	a := &App{store: s, cfg: config.Config{CacheDir: filepath.Join(dir, "cache")}}
	if !a.reuseWishlistForMovie(ctx, movie.MovieID) {
		t.Fatal("reuse did not complete")
	}
	detail, e := s.GetMovieDetail(ctx, movie.MovieID)
	if e != nil || detail.Title != "Saved title" {
		t.Fatalf("metadata %+v %v", detail, e)
	}
	files, e := s.MovieFilesForWishlist(ctx, movie.MovieID, a.cfg.CacheDir)
	if e != nil || len(files) != 1 {
		t.Fatalf("files %+v %v", files, e)
	}
	if !a.reuseWishlistForMovie(ctx, movie.MovieID) {
		t.Fatal("replay failed")
	}
	if e = s.DeleteWishlist(ctx, id); e != nil {
		t.Fatal(e)
	}
	if _, e = os.Stat(files[0].Path); e != nil {
		t.Fatal("wish deletion destroyed library asset")
	}
}

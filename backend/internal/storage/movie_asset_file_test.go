package storage

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"curated-backend/internal/contracts"
	"curated-backend/internal/scraper"
)

func TestBatchMoviePosterLocalReady_UnderCacheDir(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(root, "test.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	outcome, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
		TaskID:   "task-1",
		Path:     filepath.Join(root, "ABC-123.mp4"),
		FileName: "ABC-123.mp4",
		Number:   "ABC-123",
	})
	if err != nil {
		t.Fatalf("persist: %v", err)
	}

	srcCover := "https://example.com/poster.jpg"
	if err := store.SaveMovieMetadata(ctx, scraper.Metadata{
		MovieID:  outcome.MovieID,
		Number:   "ABC-123",
		Title:    "T",
		Studio:   "S",
		Summary:  "x",
		CoverURL: srcCover,
		ThumbURL: "https://example.com/thumb.jpg",
	}); err != nil {
		t.Fatalf("metadata: %v", err)
	}

	cacheDir := filepath.Join(root, "cache")
	coverPath := filepath.Join(cacheDir, outcome.MovieID, "cover.jpg")
	if err := os.MkdirAll(filepath.Dir(coverPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(coverPath, []byte("fakejpg"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateMediaAssetLocalPath(ctx, outcome.MovieID, "cover", srcCover, coverPath); err != nil {
		t.Fatalf("update local path: %v", err)
	}

	flags, err := store.BatchMoviePosterLocalReady(ctx, []string{outcome.MovieID}, cacheDir)
	if err != nil {
		t.Fatalf("batch: %v", err)
	}
	f := flags[outcome.MovieID]
	if !f.Cover || f.Thumb {
		t.Fatalf("flags = %+v, want Cover=true Thumb=false", f)
	}

	f2, err := store.OpenMovieAssetFile(ctx, outcome.MovieID, "cover", cacheDir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	_ = f2.Close()
}

func TestOpenMovieAssetFile_ForbiddenOutsideCacheAndLibrary(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(root, "test.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	outcome, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
		TaskID:   "task-1",
		Path:     filepath.Join(root, "ABC-999.mp4"),
		FileName: "ABC-999.mp4",
		Number:   "ABC-999",
	})
	if err != nil {
		t.Fatalf("persist: %v", err)
	}

	srcCover := "https://evil.example/p.jpg"
	if err := store.SaveMovieMetadata(ctx, scraper.Metadata{
		MovieID:  outcome.MovieID,
		Number:   "ABC-999",
		Title:    "T",
		Studio:   "S",
		Summary:  "x",
		CoverURL: srcCover,
	}); err != nil {
		t.Fatalf("metadata: %v", err)
	}

	outsideFile := filepath.Join(root, "outside", "nope.jpg")
	if err := os.MkdirAll(filepath.Dir(outsideFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outsideFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateMediaAssetLocalPath(ctx, outcome.MovieID, "cover", srcCover, outsideFile); err != nil {
		t.Fatalf("update local path: %v", err)
	}

	cacheDir := filepath.Join(root, "cache")
	_, err = store.OpenMovieAssetFile(ctx, outcome.MovieID, "cover", cacheDir)
	if err != ErrMovieAssetForbidden {
		t.Fatalf("open outside roots: err=%v want ErrMovieAssetForbidden", err)
	}
}

func TestRewritePreviewImageURLsPreferLocal(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(root, "test.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	outcome, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
		TaskID:   "task-1",
		Path:     filepath.Join(root, "PRE-001.mp4"),
		FileName: "PRE-001.mp4",
		Number:   "PRE-001",
	})
	if err != nil {
		t.Fatalf("persist: %v", err)
	}

	p1 := "https://example.com/a.jpg"
	p2 := "https://example.com/b.jpg"
	if err := store.SaveMovieMetadata(ctx, scraper.Metadata{
		MovieID:       outcome.MovieID,
		Number:        "PRE-001",
		Title:         "T",
		Studio:        "S",
		Summary:       "x",
		PreviewImages: []string{p1, p2},
	}); err != nil {
		t.Fatalf("metadata: %v", err)
	}

	cacheDir := filepath.Join(root, "cache")
	prev1 := filepath.Join(cacheDir, outcome.MovieID, "preview-01.jpg")
	if err := os.MkdirAll(filepath.Dir(prev1), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(prev1, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateMediaAssetLocalPath(ctx, outcome.MovieID, "preview_image", p1, prev1); err != nil {
		t.Fatalf("update preview 1: %v", err)
	}

	remote := []string{p1, p2}
	out := store.RewritePreviewImageURLsPreferLocal(ctx, outcome.MovieID, cacheDir, remote)
	if len(out) != 2 {
		t.Fatalf("len=%d", len(out))
	}
	if !strings.Contains(out[0], "/asset/preview/1") {
		t.Fatalf("slot 0 want local API path: %q", out[0])
	}
	if out[0] == p1 {
		t.Fatalf("slot 0 should not stay remote: %q", out[0])
	}
	if out[1] != p2 {
		t.Fatalf("slot 1 should stay remote: %q", out[1])
	}

	f, err := store.OpenMoviePreviewImageFile(ctx, outcome.MovieID, 1, cacheDir)
	if err != nil {
		t.Fatalf("open preview: %v", err)
	}
	_ = f.Close()
}

func TestOpenActorAvatarFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(root, "test.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	outcome, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
		TaskID:   "task-1",
		Path:     filepath.Join(root, "ABC-123.mp4"),
		FileName: "ABC-123.mp4",
		Number:   "ABC-123",
	})
	if err != nil {
		t.Fatalf("persist: %v", err)
	}
	if err := store.SaveMovieMetadata(ctx, scraper.Metadata{
		MovieID: outcome.MovieID,
		Number:  "ABC-123",
		Title:   "T",
		Studio:  "S",
		Summary: "x",
		Actors:  []string{"Alice"},
	}); err != nil {
		t.Fatalf("metadata: %v", err)
	}

	cacheDir := filepath.Join(root, "cache")
	avatarPath := filepath.Join(cacheDir, "actors", "Alice.jpg")
	if err := os.MkdirAll(filepath.Dir(avatarPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(avatarPath, []byte("fake-avatar-image-content"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateActorAvatarCache(ctx, "Alice", avatarPath, 200, ""); err != nil {
		t.Fatalf("update actor avatar cache: %v", err)
	}
	ready, err := store.BatchActorAvatarLocalReady(ctx, []string{"Alice"}, cacheDir)
	if err != nil {
		t.Fatalf("batch actor avatar: %v", err)
	}
	if !ready["Alice"] {
		t.Fatal("expected actor avatar to be locally ready")
	}
	f, err := store.OpenActorAvatarFile(ctx, "Alice", cacheDir)
	if err != nil {
		t.Fatalf("open actor avatar: %v", err)
	}
	_ = f.Close()
}

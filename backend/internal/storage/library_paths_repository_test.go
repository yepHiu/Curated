package storage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"curated-backend/internal/contracts"
	"curated-backend/internal/scraper"
)

func TestLibraryPathsSeedListDelete(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(root, "libpaths.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	alphaVideos := filepath.Join(root, "alpha", "videos")
	betaMedia := filepath.Join(root, "beta", "media")
	if err := store.SeedLibraryPathsIfEmpty(ctx, []string{alphaVideos, betaMedia}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	n, _ := store.GetLibraryPathCount(ctx)
	if n != 2 {
		t.Fatalf("expected 2 seeded rows, got %d", n)
	}

	// Seed again should not duplicate
	if err := store.SeedLibraryPathsIfEmpty(ctx, []string{filepath.Join(root, "gamma")}); err != nil {
		t.Fatalf("seed2: %v", err)
	}
	n, _ = store.GetLibraryPathCount(ctx)
	if n != 2 {
		t.Fatalf("re-seed should not add rows, got %d", n)
	}

	list, err := store.ListLibraryPaths(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 || list[0].Path > list[1].Path {
		// ordered by path: /alpha before /beta
		if len(list) != 2 {
			t.Fatalf("list len %d", len(list))
		}
	}

	strs, err := store.ListLibraryPathStrings(ctx)
	if err != nil || len(strs) != 2 {
		t.Fatalf("strings: %v %#v", err, strs)
	}

	gammaNew := filepath.Join(root, "gamma", "new")
	dto, err := store.AddLibraryPath(ctx, gammaNew, "Gamma")
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if dto.Title != "Gamma" {
		t.Fatalf("title %q", dto.Title)
	}

	updated, err := store.UpdateLibraryPathTitle(ctx, dto.ID, "Gamma Archive")
	if err != nil || updated.Title != "Gamma Archive" || updated.Path != gammaNew {
		t.Fatalf("update title: %v %#v", err, updated)
	}
	_, err = store.UpdateLibraryPathTitle(ctx, "missing-id", "x")
	if !errors.Is(err, ErrLibraryPathNotFound) {
		t.Fatalf("expected not found on update, got %v", err)
	}

	_, err = store.AddLibraryPath(ctx, alphaVideos, "dup")
	if !errors.Is(err, ErrLibraryPathDuplicate) {
		t.Fatalf("expected duplicate, got %v", err)
	}

	_, err = store.AddLibraryPath(ctx, "relative/subdir", "rel")
	if !errors.Is(err, ErrLibraryPathNotAbsolute) {
		t.Fatalf("expected not absolute error, got %v", err)
	}

	if err := store.DeleteLibraryPath(ctx, dto.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := store.DeleteLibraryPath(ctx, dto.ID); !errors.Is(err, ErrLibraryPathNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestLibraryPathsEmptySeed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(root, "empty.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := store.SeedLibraryPathsIfEmpty(ctx, nil); err != nil {
		t.Fatal(err)
	}
	n, _ := store.GetLibraryPathCount(ctx)
	if n != 0 {
		t.Fatalf("expected 0 rows, got %d", n)
	}
}

func TestDeleteLibraryPathAndPruneOrphanMovies_RemovesOnlyUncovered(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	alpha := filepath.Join(root, "alpha")
	beta := filepath.Join(root, "beta")
	if err := os.MkdirAll(alpha, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(beta, 0o755); err != nil {
		t.Fatal(err)
	}
	videoAlpha := filepath.Join(alpha, "ABC-100.mp4")
	if err := os.WriteFile(videoAlpha, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	store, err := NewSQLiteStore(filepath.Join(root, "prune.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	alphaDTO, err := store.AddLibraryPath(ctx, alpha, "A")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddLibraryPath(ctx, beta, "B"); err != nil {
		t.Fatal(err)
	}

	outcome, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
		TaskID:   "t1",
		Path:     videoAlpha,
		FileName: "ABC-100.mp4",
		Number:   "ABC-100",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveMovieMetadata(ctx, scraper.Metadata{
		MovieID: outcome.MovieID,
		Number:  "ABC-100",
		Title:   "T",
		Summary: "S",
		Studio:  "St",
	}); err != nil {
		t.Fatal(err)
	}

	page, err := store.ListMovies(ctx, contracts.ListMoviesRequest{Limit: 10})
	if err != nil || page.Total != 1 {
		t.Fatalf("before prune: %v total=%d", err, page.Total)
	}

	pruned, err := store.DeleteLibraryPathAndPruneOrphanMovies(ctx, alphaDTO.ID)
	if err != nil || pruned != 1 {
		t.Fatalf("prune: err=%v pruned=%d", err, pruned)
	}
	page, err = store.ListMovies(ctx, contracts.ListMoviesRequest{Limit: 10})
	if err != nil || page.Total != 0 {
		t.Fatalf("after prune alpha: %v total=%d", err, page.Total)
	}

	n, _ := store.GetLibraryPathCount(ctx)
	if n != 1 {
		t.Fatalf("expected 1 library path (beta), got %d", n)
	}
	if _, err := os.Stat(videoAlpha); err != nil {
		t.Fatalf("removing library path must not delete media files on disk: %v", err)
	}
}

func TestDeleteLibraryPathAndPruneOrphanMovies_NestedRootKeepsMovie(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	parent := filepath.Join(root, "lib")
	child := filepath.Join(root, "lib", "nested")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	video := filepath.Join(child, "XYZ-200.mp4")
	if err := os.WriteFile(video, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	store, err := NewSQLiteStore(filepath.Join(root, "nested.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	if _, err := store.AddLibraryPath(ctx, parent, "P"); err != nil {
		t.Fatal(err)
	}
	childDTO, err := store.AddLibraryPath(ctx, child, "C")
	if err != nil {
		t.Fatal(err)
	}

	outcome, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
		TaskID:   "t1",
		Path:     video,
		FileName: "XYZ-200.mp4",
		Number:   "XYZ-200",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveMovieMetadata(ctx, scraper.Metadata{
		MovieID: outcome.MovieID,
		Number:  "XYZ-200",
		Title:   "T",
		Summary: "S",
		Studio:  "St",
	}); err != nil {
		t.Fatal(err)
	}

	pruned, err := store.DeleteLibraryPathAndPruneOrphanMovies(ctx, childDTO.ID)
	if err != nil || pruned != 0 {
		t.Fatalf("nested child delete should not prune (still under parent): err=%v pruned=%d", err, pruned)
	}
	page, err := store.ListMovies(ctx, contracts.ListMoviesRequest{Limit: 10})
	if err != nil || page.Total != 1 {
		t.Fatalf("movie should remain: %v total=%d", err, page.Total)
	}
}

func TestDeleteLibraryPathAndPruneOrphanMovies_DeleteOtherRootUnchanged(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	alpha := filepath.Join(root, "alpha")
	beta := filepath.Join(root, "beta")
	if err := os.MkdirAll(alpha, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(beta, 0o755); err != nil {
		t.Fatal(err)
	}
	videoAlpha := filepath.Join(alpha, "M-1.mp4")
	if err := os.WriteFile(videoAlpha, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	store, err := NewSQLiteStore(filepath.Join(root, "other.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	if _, err := store.AddLibraryPath(ctx, alpha, "A"); err != nil {
		t.Fatal(err)
	}
	betaDTO, err := store.AddLibraryPath(ctx, beta, "B")
	if err != nil {
		t.Fatal(err)
	}

	outcome, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
		TaskID:   "t1",
		Path:     videoAlpha,
		FileName: "M-1.mp4",
		Number:   "M-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveMovieMetadata(ctx, scraper.Metadata{
		MovieID: outcome.MovieID,
		Number:  "M-1",
		Title:   "T",
		Summary: "S",
		Studio:  "St",
	}); err != nil {
		t.Fatal(err)
	}

	pruned, err := store.DeleteLibraryPathAndPruneOrphanMovies(ctx, betaDTO.ID)
	if err != nil || pruned != 0 {
		t.Fatalf("deleting unrelated root: err=%v pruned=%d", err, pruned)
	}
	page, err := store.ListMovies(ctx, contracts.ListMoviesRequest{Limit: 10})
	if err != nil || page.Total != 1 {
		t.Fatalf("movie under alpha intact: %v total=%d", err, page.Total)
	}
}

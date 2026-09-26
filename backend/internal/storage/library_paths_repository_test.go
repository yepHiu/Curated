package storage

import (
	"context"
	"errors"
	"fmt"
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

func TestLibraryPathsStayEmptyAfterRestart(t *testing.T) {
	for _, initiallyEmpty := range []bool{false, true} {
		t.Run(fmt.Sprintf("initiallyEmpty=%t", initiallyEmpty), func(t *testing.T) {
			ctx := context.Background()
			root := t.TempDir()
			databasePath := filepath.Join(root, "library.db")
			defaults := []string{filepath.Join(root, "media")}
			store, err := NewSQLiteStore(databasePath)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = store.Close() }()
			if err := store.Migrate(ctx); err != nil {
				t.Fatal(err)
			}
			initialPaths := defaults
			if initiallyEmpty {
				initialPaths = nil
			}
			if err := store.SeedLibraryPathsIfEmpty(ctx, initialPaths); err != nil {
				t.Fatal(err)
			}
			paths, err := store.ListLibraryPaths(ctx)
			if err != nil {
				t.Fatal(err)
			}
			for _, path := range paths {
				if _, err := store.DeleteLibraryPathAndPruneOrphanMovies(ctx, path.ID); err != nil {
					t.Fatal(err)
				}
			}
			if err := store.Close(); err != nil {
				t.Fatal(err)
			}
			store, err = NewSQLiteStore(databasePath)
			if err != nil {
				t.Fatal(err)
			}
			if err := store.Migrate(ctx); err != nil {
				t.Fatal(err)
			}
			if err := store.SeedLibraryPathsIfEmpty(ctx, defaults); err != nil {
				t.Fatal(err)
			}
			paths, err = store.ListLibraryPaths(ctx)
			if err != nil || len(paths) != 0 {
				t.Fatalf("restart restored paths: %#v, err=%v", paths, err)
			}
		})
	}
}

func TestLibraryPathsUpgradePreservesExistingList(t *testing.T) {
	for _, hasPath := range []bool{false, true} {
		t.Run(fmt.Sprintf("hasPath=%t", hasPath), func(t *testing.T) {
			ctx := context.Background()
			store := newMigratedTestStore(t)
			var want []contracts.LibraryPathDTO
			if hasPath {
				path, err := store.AddLibraryPath(ctx, filepath.Join(t.TempDir(), "existing"), "Existing")
				if err != nil {
					t.Fatal(err)
				}
				want = append(want, path)
			}
			// Recreate the schema state of an old installation before upgrading.
			if _, err := store.db.ExecContext(ctx, `DROP TABLE library_paths_initialization;
				DELETE FROM schema_migrations WHERE name = '0056_library_paths_initialization.sql'`); err != nil {
				t.Fatal(err)
			}
			if err := store.Migrate(ctx); err != nil {
				t.Fatal(err)
			}
			if err := store.SeedLibraryPathsIfEmpty(ctx, []string{filepath.Join(t.TempDir(), "default")}); err != nil {
				t.Fatal(err)
			}
			paths, err := store.ListLibraryPaths(ctx)
			if err != nil || len(paths) != len(want) {
				t.Fatalf("upgrade changed paths: %#v, want %#v, err=%v", paths, want, err)
			}
			if hasPath && paths[0] != want[0] {
				t.Fatalf("upgrade changed existing path: %#v, want %#v", paths[0], want[0])
			}
		})
	}
}

func TestLibraryPathsSeedFailureCanRetry(t *testing.T) {
	ctx := context.Background()
	store := newMigratedTestStore(t)
	paths := []string{filepath.Join(t.TempDir(), "first"), filepath.Join(t.TempDir(), "second")}
	if _, err := store.db.ExecContext(ctx, `CREATE TRIGGER fail_library_seed BEFORE INSERT ON library_paths
		WHEN NEW.id = 'library-2' BEGIN SELECT RAISE(ABORT, 'seed failed'); END`); err != nil {
		t.Fatal(err)
	}
	if err := store.SeedLibraryPathsIfEmpty(ctx, paths); err == nil {
		t.Fatal("expected seed failure")
	}
	if count, err := store.GetLibraryPathCount(ctx); err != nil || count != 0 {
		t.Fatalf("seed was not rolled back: count=%d, err=%v", count, err)
	}
	if _, err := store.db.ExecContext(ctx, `DROP TRIGGER fail_library_seed`); err != nil {
		t.Fatal(err)
	}
	if err := store.SeedLibraryPathsIfEmpty(ctx, paths); err != nil {
		t.Fatal(err)
	}
	if count, err := store.GetLibraryPathCount(ctx); err != nil || count != 2 {
		t.Fatalf("seed retry: count=%d, err=%v", count, err)
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

func TestDeleteLibraryPathAndPruneOrphanMovies_RollsBackPathDeletionWhenMoviePruneFails(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	alpha := filepath.Join(root, "alpha")
	if err := os.MkdirAll(alpha, 0o755); err != nil {
		t.Fatal(err)
	}
	videoAlpha := filepath.Join(alpha, "ROLLBACK-1.mp4")
	if err := os.WriteFile(videoAlpha, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	store, err := NewSQLiteStore(filepath.Join(root, "rollback.db"))
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

	outcome, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
		TaskID:   "t1",
		Path:     videoAlpha,
		FileName: "ROLLBACK-1.mp4",
		Number:   "ROLLBACK-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveMovieMetadata(ctx, scraper.Metadata{
		MovieID: outcome.MovieID,
		Number:  "ROLLBACK-1",
		Title:   "T",
		Summary: "S",
		Studio:  "St",
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := store.db.ExecContext(ctx, `
		CREATE TRIGGER fail_movie_delete
		BEFORE DELETE ON movies
		BEGIN
			SELECT RAISE(ABORT, 'forced movie delete failure');
		END;
	`); err != nil {
		t.Fatal(err)
	}

	if _, err := store.DeleteLibraryPathAndPruneOrphanMovies(ctx, alphaDTO.ID); err == nil {
		t.Fatal("expected delete library path prune to fail")
	}

	n, err := store.GetLibraryPathCount(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("library path delete should roll back on prune failure, got count=%d", n)
	}

	page, err := store.ListMovies(ctx, contracts.ListMoviesRequest{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 {
		t.Fatalf("movie rows should remain intact after rollback, got total=%d", page.Total)
	}

	if _, err := os.Stat(videoAlpha); err != nil {
		t.Fatalf("rollback path delete must not touch files on disk: %v", err)
	}
}

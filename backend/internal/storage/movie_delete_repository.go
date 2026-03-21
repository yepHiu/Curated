package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrMovieNotFound is returned when deleting a non-existent movie id.
var ErrMovieNotFound = errors.New("movie not found")

// DeleteMovie removes a movie row, related join rows, then best-effort deletes files on disk:
// primary video (movies.location), media_assets.local_path entries, and movie.nfo next to the video.
// When assetCacheRoot is non-empty (typically cfg.CacheDir), the whole directory assetCacheRoot/movieID
// is removed last — this clears downloaded posters/previews stored under the cache when organizeLibrary is off,
// including orphans not recorded in media_assets.
// Database deletion is transactional; file removal runs after commit. File errors are ignored (best-effort).
func (s *SQLiteStore) DeleteMovie(ctx context.Context, movieID string, assetCacheRoot string) error {
	movieID = strings.TrimSpace(movieID)
	if movieID == "" {
		return ErrMovieNotFound
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var location string
	err = tx.QueryRowContext(ctx, `SELECT location FROM movies WHERE id = ?`, movieID).Scan(&location)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrMovieNotFound
	}
	if err != nil {
		return err
	}

	rows, err := tx.QueryContext(ctx,
		`SELECT local_path FROM media_assets WHERE movie_id = ? AND local_path != ''`, movieID)
	if err != nil {
		return err
	}
	assetPaths := make([]string, 0)
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			_ = rows.Close()
			return err
		}
		if strings.TrimSpace(p) != "" {
			assetPaths = append(assetPaths, p)
		}
	}
	if err := rows.Close(); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM movie_actors WHERE movie_id = ?`, movieID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM movie_tags WHERE movie_id = ?`, movieID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM media_assets WHERE movie_id = ?`, movieID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM scan_items WHERE movie_id = ?`, movieID); err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `DELETE FROM movies WHERE id = ?`, movieID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrMovieNotFound
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	paths := collectMovieFilePaths(location, assetPaths)
	for _, p := range paths {
		_ = safeRemoveFileIfFile(p)
	}
	bestEffortRemoveMovieAssetCacheDir(assetCacheRoot, movieID)
	return nil
}

func collectMovieFilePaths(location string, assetLocalPaths []string) []string {
	seen := make(map[string]struct{})
	var out []string
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" {
			return
		}
		c := filepath.Clean(p)
		if _, ok := seen[c]; ok {
			return
		}
		seen[c] = struct{}{}
		out = append(out, c)
	}

	add(location)
	for _, p := range assetLocalPaths {
		add(p)
	}
	if location != "" {
		nfo := filepath.Join(filepath.Dir(filepath.Clean(location)), "movie.nfo")
		add(nfo)
	}
	return out
}

// safeRemoveFileIfFile deletes a path only if it is an absolute path to a regular file (not a directory).
func safeRemoveFileIfFile(path string) error {
	clean := filepath.Clean(path)
	if clean == "" || clean == "." {
		return nil
	}
	if !filepath.IsAbs(clean) {
		return fmt.Errorf("refuse non-absolute path: %q", path)
	}
	fi, err := os.Lstat(clean)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if fi.IsDir() {
		return nil
	}
	return os.Remove(clean)
}

func movieIDSafeForCacheSubdir(movieID string) bool {
	if strings.TrimSpace(movieID) == "" {
		return false
	}
	if movieID == "." || movieID == ".." {
		return false
	}
	if strings.ContainsAny(movieID, `/\`) {
		return false
	}
	return true
}

// bestEffortRemoveMovieAssetCacheDir deletes {cacheRoot}/{movieID} if it is a directory confined under cacheRoot.
func bestEffortRemoveMovieAssetCacheDir(cacheRoot, movieID string) {
	cacheRoot = strings.TrimSpace(cacheRoot)
	if cacheRoot == "" || !movieIDSafeForCacheSubdir(movieID) {
		return
	}

	rootAbs, err := filepath.Abs(filepath.Clean(cacheRoot))
	if err != nil || rootAbs == "" || !filepath.IsAbs(rootAbs) {
		return
	}

	target := filepath.Join(rootAbs, movieID)
	targetAbs, err := filepath.Abs(filepath.Clean(target))
	if err != nil {
		return
	}

	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return
	}

	fi, err := os.Lstat(targetAbs)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		return
	}
	if !fi.IsDir() {
		return
	}
	_ = os.RemoveAll(targetAbs)
}

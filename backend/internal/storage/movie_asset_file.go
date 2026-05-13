package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	// ErrMovieAssetNotFound is returned when there is no usable local file for the requested asset.
	ErrMovieAssetNotFound = errors.New("movie asset not found")
	// ErrMovieAssetForbidden is returned when local_path is outside allowed cache/library directories.
	ErrMovieAssetForbidden = errors.New("movie asset path forbidden")
)

// MoviePosterLocalFlags reports which poster assets exist on disk and pass path policy checks.
type MoviePosterLocalFlags struct {
	Cover        bool
	CoverVersion string
	Thumb        bool
	ThumbVersion string
}

// posterPathPolicy is resolved once per batch or per HTTP request; must not be recomputed per media_assets row.
type posterPathPolicy struct {
	cacheAbs string
	rootAbs  []string
}

func absCleanPath(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", fmt.Errorf("empty path")
	}
	return filepath.Abs(filepath.Clean(p))
}

func (s *SQLiteStore) loadPosterPathPolicy(ctx context.Context, cacheDir string) posterPathPolicy {
	var p posterPathPolicy
	cacheAbs, err := absCleanPath(cacheDir)
	if err == nil && cacheAbs != "" {
		p.cacheAbs = cacheAbs
	}
	roots, err := s.ListLibraryPaths(ctx)
	if err != nil {
		return p
	}
	for _, dto := range roots {
		root := strings.TrimSpace(dto.Path)
		if root == "" {
			continue
		}
		rootAbs, err := absCleanPath(root)
		if err != nil {
			continue
		}
		p.rootAbs = append(p.rootAbs, rootAbs)
	}
	return p
}

func localAssetVersionFromStat(st os.FileInfo) string {
	if st == nil {
		return ""
	}
	return strconv.FormatInt(st.ModTime().UTC().UnixNano(), 10) + "-" + strconv.FormatInt(st.Size(), 10)
}

func mediaAssetFileInfoAllowedWithPolicy(absPath string, policy posterPathPolicy) (os.FileInfo, bool) {
	if policy.cacheAbs == "" && len(policy.rootAbs) == 0 {
		return nil, false
	}
	st, err := os.Stat(absPath)
	if err != nil || !st.Mode().IsRegular() {
		return nil, false
	}
	if policy.cacheAbs != "" && pathUnderLibraryRoot(absPath, policy.cacheAbs) {
		return st, true
	}
	for _, r := range policy.rootAbs {
		if pathUnderLibraryRoot(absPath, r) {
			return st, true
		}
	}
	return nil, false
}

func mediaAssetPathAllowedWithPolicy(absPath string, policy posterPathPolicy) bool {
	_, ok := mediaAssetFileInfoAllowedWithPolicy(absPath, policy)
	return ok
}

// mediaAssetPathAllowed reports whether absPath is a regular file under cacheDir or any library root.
func (s *SQLiteStore) mediaAssetPathAllowed(ctx context.Context, absPath, cacheDir string) bool {
	policy := s.loadPosterPathPolicy(ctx, cacheDir)
	return mediaAssetPathAllowedWithPolicy(absPath, policy)
}

// BatchMoviePosterLocalReady returns, per movie id, whether cover/thumb rows point to existing allowed files.
func (s *SQLiteStore) BatchMoviePosterLocalReady(ctx context.Context, movieIDs []string, cacheDir string) (map[string]MoviePosterLocalFlags, error) {
	out := make(map[string]MoviePosterLocalFlags)
	if len(movieIDs) == 0 {
		return out, nil
	}
	policy := s.loadPosterPathPolicy(ctx, cacheDir)
	if policy.cacheAbs == "" && len(policy.rootAbs) == 0 {
		return out, nil
	}

	if err := forEachInClauseBatch(movieIDs, func(batch []string) error {
		q := fmt.Sprintf(
			`SELECT movie_id, type, local_path FROM media_assets
			WHERE movie_id IN (%s) AND type IN ('cover', 'thumb') AND TRIM(local_path) != ''`,
			inClausePlaceholders(len(batch)),
		)
		rows, err := s.db.QueryContext(ctx, q, inClauseArgs(batch)...)
		if err != nil {
			return err
		}

		for rows.Next() {
			var movieID, assetType, localPath string
			if err := rows.Scan(&movieID, &assetType, &localPath); err != nil {
				_ = rows.Close()
				return err
			}
			absPath, err := absCleanPath(localPath)
			if err != nil {
				continue
			}
			st, ok := mediaAssetFileInfoAllowedWithPolicy(absPath, policy)
			if !ok {
				continue
			}
			version := localAssetVersionFromStat(st)
			flags := out[movieID]
			switch assetType {
			case "cover":
				flags.Cover = true
				flags.CoverVersion = version
			case "thumb":
				flags.Thumb = true
				flags.ThumbVersion = version
			}
			out[movieID] = flags
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return err
		}
		return rows.Close()
	}); err != nil {
		return nil, err
	}
	return out, nil
}

// OpenMovieAssetFile opens a downloaded cover or thumb (kind is "cover" or "thumb") after validating path policy.
func (s *SQLiteStore) OpenMovieAssetFile(ctx context.Context, movieID, kind, cacheDir string) (*os.File, error) {
	kind = strings.TrimSpace(strings.ToLower(kind))
	if kind != "cover" && kind != "thumb" {
		return nil, fmt.Errorf("invalid asset kind")
	}
	movieID = strings.TrimSpace(movieID)
	if movieID == "" {
		return nil, fmt.Errorf("empty movie id")
	}

	var localPath string
	err := s.db.QueryRowContext(
		ctx,
		`SELECT local_path FROM media_assets WHERE movie_id = ? AND type = ? AND TRIM(local_path) != '' LIMIT 1`,
		movieID,
		kind,
	).Scan(&localPath)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMovieAssetNotFound
		}
		return nil, err
	}

	absPath, err := absCleanPath(localPath)
	if err != nil {
		return nil, ErrMovieAssetNotFound
	}
	if !s.mediaAssetPathAllowed(ctx, absPath, cacheDir) {
		return nil, ErrMovieAssetForbidden
	}

	f, err := os.Open(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrMovieAssetNotFound
		}
		return nil, err
	}
	return f, nil
}

type previewImageAssetRow struct {
	SourceURL string
	LocalPath string
}

func (s *SQLiteStore) listPreviewImageAssetRows(ctx context.Context, movieID string) ([]previewImageAssetRow, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT source_url, local_path FROM media_assets
		 WHERE movie_id = ? AND type = 'preview_image' AND TRIM(source_url) != ''
		 ORDER BY id ASC`,
		movieID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []previewImageAssetRow
	for rows.Next() {
		var r previewImageAssetRow
		if err := rows.Scan(&r.SourceURL, &r.LocalPath); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// GetMoviePreviewSourceURL returns the remote source URL and referer URL for a numbered preview image (1-based seq).
func (s *SQLiteStore) GetMoviePreviewSourceURL(ctx context.Context, movieID string, seq int) (string, string, error) {
	if seq < 1 {
		return "", "", fmt.Errorf("invalid preview sequence")
	}
	assetID := fmt.Sprintf("%s:preview:%02d", strings.TrimSpace(movieID), seq)
	var sourceURL string
	var refererURL string
	err := s.db.QueryRowContext(ctx,
		`SELECT source_url, referer_url FROM media_assets
		 WHERE id = ? AND type = 'preview_image' AND TRIM(source_url) != ''`,
		assetID,
	).Scan(&sourceURL, &refererURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", ErrMovieAssetNotFound
		}
		return "", "", err
	}
	return strings.TrimSpace(sourceURL), strings.TrimSpace(refererURL), nil
}

// RewritePreviewImageURLsPreferLocal returns a copy of urls with entries replaced by same-origin preview asset paths when a valid local file exists (order matches media_assets.id).
func (s *SQLiteStore) RewritePreviewImageURLsPreferLocal(ctx context.Context, movieID, cacheDir string, urls []string) []string {
	if len(urls) == 0 {
		return urls
	}
	out := append([]string(nil), urls...)
	policy := s.loadPosterPathPolicy(ctx, cacheDir)
	if policy.cacheAbs == "" && len(policy.rootAbs) == 0 {
		return out
	}
	rows, err := s.listPreviewImageAssetRows(ctx, movieID)
	if err != nil || len(rows) == 0 {
		return out
	}
	n := len(out)
	if len(rows) < n {
		n = len(rows)
	}
	enc := url.PathEscape(movieID)
	for i := 0; i < n; i++ {
		lp := strings.TrimSpace(rows[i].LocalPath)
		if lp == "" {
			continue
		}
		absPath, err := absCleanPath(lp)
		if err != nil {
			continue
		}
		st, ok := mediaAssetFileInfoAllowedWithPolicy(absPath, policy)
		if !ok {
			continue
		}
		localURL := "/api/library/movies/" + enc + "/asset/preview/" + strconv.Itoa(i+1)
		if version := localAssetVersionFromStat(st); version != "" {
			localURL += "?v=" + url.QueryEscape(version)
		}
		out[i] = localURL
	}
	return out
}

// OpenMoviePreviewImageFile opens preview seq (1-based, matching media_assets id …:preview:NN).
func (s *SQLiteStore) OpenMoviePreviewImageFile(ctx context.Context, movieID string, seq int, cacheDir string) (*os.File, error) {
	if seq < 1 {
		return nil, fmt.Errorf("invalid preview sequence")
	}
	movieID = strings.TrimSpace(movieID)
	if movieID == "" {
		return nil, fmt.Errorf("empty movie id")
	}
	assetID := fmt.Sprintf("%s:preview:%02d", movieID, seq)

	var localPath string
	err := s.db.QueryRowContext(
		ctx,
		`SELECT local_path FROM media_assets WHERE id = ? AND type = 'preview_image' AND TRIM(local_path) != ''`,
		assetID,
	).Scan(&localPath)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMovieAssetNotFound
		}
		return nil, err
	}

	absPath, err := absCleanPath(localPath)
	if err != nil {
		return nil, ErrMovieAssetNotFound
	}
	if !s.mediaAssetPathAllowed(ctx, absPath, cacheDir) {
		return nil, ErrMovieAssetForbidden
	}

	f, err := os.Open(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrMovieAssetNotFound
		}
		return nil, err
	}
	return f, nil
}

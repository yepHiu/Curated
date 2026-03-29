package storage

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

var (
	// ErrMovieVideoNotFound is returned when the movie id does not exist in SQLite.
	ErrMovieVideoNotFound = errors.New("movie not found")
	// ErrMovieVideoNoLocation is returned when the movie row has an empty location.
	ErrMovieVideoNoLocation = errors.New("movie has no video file path")
	// ErrMovieVideoForbidden is returned when location is not under any configured library root.
	ErrMovieVideoForbidden = errors.New("video path outside library roots")
	// ErrMovieVideoNotFile is returned when location is not a regular file.
	ErrMovieVideoNotFile = errors.New("video path is not a regular file")
)

// ResolvePrimaryVideoPath returns the absolute path to the movie's primary video after the same
// validation as OpenMovieVideoFile (library roots, regular file, exists).
func (s *SQLiteStore) ResolvePrimaryVideoPath(ctx context.Context, movieID string) (string, error) {
	detail, err := s.GetMovieDetail(ctx, movieID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrMovieVideoNotFound
	}
	if err != nil {
		return "", err
	}
	loc := strings.TrimSpace(detail.Location)
	if loc == "" {
		return "", ErrMovieVideoNoLocation
	}

	absVideo, err := filepath.Abs(filepath.Clean(loc))
	if err != nil {
		return "", err
	}

	roots, err := s.ListLibraryPaths(ctx)
	if err != nil {
		return "", err
	}
	if len(roots) == 0 {
		return "", ErrMovieVideoForbidden
	}

	allowed := false
	for _, rootDTO := range roots {
		root := strings.TrimSpace(rootDTO.Path)
		if root == "" {
			continue
		}
		absRoot, err := filepath.Abs(filepath.Clean(root))
		if err != nil {
			continue
		}
		if pathUnderLibraryRoot(absVideo, absRoot) {
			allowed = true
			break
		}
	}
	if !allowed {
		return "", ErrMovieVideoForbidden
	}

	st, err := os.Stat(absVideo)
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrMovieVideoNotFound
		}
		return "", err
	}
	if !st.Mode().IsRegular() {
		return "", ErrMovieVideoNotFile
	}

	return absVideo, nil
}

// OpenMovieVideoFile opens the on-disk primary video for a movie after validating it lies under
// a configured library path (same roots as GET /api/settings libraryPaths).
func (s *SQLiteStore) OpenMovieVideoFile(ctx context.Context, movieID string) (*os.File, string, error) {
	absVideo, err := s.ResolvePrimaryVideoPath(ctx, movieID)
	if err != nil {
		return nil, "", err
	}

	f, err := os.Open(absVideo)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, "", ErrMovieVideoNotFound
		}
		return nil, "", err
	}
	return f, filepath.Base(absVideo), nil
}

// pathUnderLibraryRoot reports whether target is root or a path inside root (filepath.Rel, rejects "..").
func pathUnderLibraryRoot(target, root string) bool {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, "..")
}

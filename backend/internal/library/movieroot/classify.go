// Package movieroot classifies how a video file is organized on disk (loose, curated-manifest, or external number directory).
package movieroot

import (
	"path/filepath"
	"strings"

	"curated-backend/internal/library/curated"
	"curated-backend/internal/library/moviecode"
)

// LayoutKind classifies how a video file is organized on disk relative to its number directory.
type LayoutKind string

const (
	// LayoutLoose means the video sits directly inside a library root without a number subdirectory.
	LayoutLoose    LayoutKind = "loose"
	// LayoutCurated means the video is organized under a Curated.json manifest.
	LayoutCurated  LayoutKind = "curated"
	// LayoutExternal means the video resides in a directory whose name matches the number (no Curated.json).
	LayoutExternal LayoutKind = "external"
)

// ClassifyVideoRoot determines the layout kind and movie root directory for a scanned video file.
func ClassifyVideoRoot(videoAbsPath, number string, extendedImport, firstScanPendingForRoot bool) (kind LayoutKind, movieRootDir string) {
	movieRootDir = filepath.Dir(filepath.Clean(videoAbsPath))
	if !extendedImport || !firstScanPendingForRoot || strings.TrimSpace(number) == "" {
		return LayoutLoose, movieRootDir
	}
	if _, err := curated.LoadAndValidate(movieRootDir); err == nil {
		return LayoutCurated, movieRootDir
	}
	base := filepath.Base(movieRootDir)
	if moviecode.NormalizeForStorageID(base) == moviecode.NormalizeForStorageID(number) {
		return LayoutExternal, movieRootDir
	}
	return LayoutLoose, movieRootDir
}

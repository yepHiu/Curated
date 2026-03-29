package movieroot

import (
	"path/filepath"
	"strings"

	"curated-backend/internal/library/curated"
	"curated-backend/internal/library/moviecode"
)

// LayoutKind describes how a video file sits on disk relative to a 番号 folder.
type LayoutKind string

const (
	LayoutLoose    LayoutKind = "loose"
	LayoutCurated  LayoutKind = "curated"
	LayoutExternal LayoutKind = "external"
)

// ClassifyVideoRoot inspects the directory containing the video when extended first-scan import is active.
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

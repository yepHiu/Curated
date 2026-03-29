package movieroot

import (
	"path/filepath"
	"strings"

	"curated-backend/internal/contracts"
)

// ResolveConfiguredLibraryPath returns the longest configured library root that contains filePath.
func ResolveConfiguredLibraryPath(filePath string, libraryPaths []contracts.LibraryPathDTO) *contracts.LibraryPathDTO {
	filePath = filepath.Clean(filePath)
	var best *contracts.LibraryPathDTO
	bestLen := -1
	for i := range libraryPaths {
		lp := filepath.Clean(strings.TrimSpace(libraryPaths[i].Path))
		if lp == "" {
			continue
		}
		rel, err := filepath.Rel(lp, filePath)
		if err != nil {
			continue
		}
		if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		if len(lp) > bestLen {
			bestLen = len(lp)
			best = &libraryPaths[i]
		}
	}
	return best
}

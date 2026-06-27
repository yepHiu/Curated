package server

import (
	"net/http"
	"path/filepath"
	"strings"

	"curated-backend/internal/comiccache"
	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
)

func (h *Handler) handleGetComicCacheStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if !h.requireComicLibraryEnabled(w) {
		return
	}
	status, err := h.comicCacheService().Status(r.Context())
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to read comic cache status")
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (h *Handler) handleCleanupComicCache(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if !h.requireComicLibraryEnabled(w) {
		return
	}
	status, err := h.comicCacheService().Cleanup(r.Context())
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeComicCacheCleanupFailed, "failed to clean comic cache")
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (h *Handler) comicCacheService() *comiccache.Service {
	cacheRoot := ""
	if dir := strings.TrimSpace(h.cfg.CacheDir); dir != "" {
		cacheRoot = filepath.Join(dir, "comics")
	}
	return comiccache.NewService(cacheRoot, h.comicCacheSettings().MaxBytes, h.store)
}

func (h *Handler) comicCacheSettings() contracts.ComicCacheSettingsDTO {
	if h.comicSettingsCtl != nil {
		return normalizeComicCacheSettingsDTO(h.comicSettingsCtl.ComicCacheSettings())
	}
	return comicCacheSettingsDTOFromConfig(config.NormalizeComicCacheConfig(h.cfg.ComicCache))
}

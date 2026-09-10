package server

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"curated-backend/internal/comicarchive"
	"curated-backend/internal/contracts"
)

func (h *Handler) handleListComicPages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if !h.requireComicLibraryEnabled(w) {
		return
	}
	detail, ok := h.loadComicDetailForRequest(w, r)
	if !ok {
		return
	}
	for i := range detail.Pages {
		h.enrichComicPageURLs(detail.ID, &detail.Pages[i])
	}
	writeJSON(w, http.StatusOK, detail.Pages)
}

func (h *Handler) handleGetComicPageImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if !h.requireComicLibraryEnabled(w) {
		return
	}
	detail, page, ok := h.loadComicPageForRequest(w, r)
	if !ok {
		return
	}
	body, _, err := comicarchive.OpenPage(r.Context(), detail.Location, page.EntryPath)
	if err != nil {
		if errors.Is(err, comicarchive.ErrPageNotFound) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeComicPageNotFound, "comic page not found")
			return
		}
		if h.logger != nil {
			h.logger.Warn("open comic page failed", zap.Error(err), zap.String("comicId", detail.ID), zap.Int("page", page.Index))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeComicArchiveReadFailed, "failed to read comic page")
		return
	}
	defer body.Close()

	w.Header().Set("Content-Type", contentTypeForComicPage(page))
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, body)
}

func (h *Handler) handleGetComicPageThumbnail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if !h.requireComicLibraryEnabled(w) {
		return
	}
	detail, page, ok := h.loadComicPageForRequest(w, r)
	if !ok {
		return
	}
	file, err := h.comicCacheService().GetOrCreateThumbnail(r.Context(), detail, page)
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("create comic thumbnail failed", zap.Error(err), zap.String("comicId", detail.ID), zap.Int("page", page.Index))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeComicArchiveReadFailed, "failed to create comic thumbnail")
		return
	}
	http.ServeFile(w, r, file.Path)
}

func (h *Handler) loadComicPageForRequest(w http.ResponseWriter, r *http.Request) (contracts.ComicBookDetailDTO, contracts.ComicPageDTO, bool) {
	detail, ok := h.loadComicDetailForRequest(w, r)
	if !ok {
		return contracts.ComicBookDetailDTO{}, contracts.ComicPageDTO{}, false
	}
	raw := strings.TrimSpace(r.PathValue("pageIndex"))
	index, err := strconv.Atoi(raw)
	if err != nil || index < 0 {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "pageIndex must be a non-negative integer")
		return contracts.ComicBookDetailDTO{}, contracts.ComicPageDTO{}, false
	}
	page, found := findComicPage(detail, index)
	if !found {
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodeComicPageNotFound, "comic page not found")
		return contracts.ComicBookDetailDTO{}, contracts.ComicPageDTO{}, false
	}
	return detail, page, true
}

func contentTypeForComicPage(page contracts.ComicPageDTO) string {
	ext := strings.ToLower(strings.TrimSpace(page.ImageExt))
	if ext == "" {
		ext = strings.ToLower(path.Ext(page.FileName))
	}
	if ext == ".jpg" {
		return "image/jpeg"
	}
	if ct := mime.TypeByExtension(ext); ct != "" {
		return ct
	}
	return "application/octet-stream"
}

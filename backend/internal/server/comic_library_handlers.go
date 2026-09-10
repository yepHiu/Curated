package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

func (h *Handler) handleListComics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if !h.requireComicLibraryEnabled(w) {
		return
	}
	req := contracts.ListComicBooksRequest{
		Query:      strings.TrimSpace(r.URL.Query().Get("q")),
		Tag:        strings.TrimSpace(r.URL.Query().Get("tag")),
		ReadStatus: strings.TrimSpace(r.URL.Query().Get("readStatus")),
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("favorite")); raw != "" {
		v := raw == "true" || raw == "1"
		req.Favorite = &v
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			req.Limit = n
		}
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("offset")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			req.Offset = n
		}
	}
	page, err := h.store.ListComicBooks(r.Context(), req)
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("list comics failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to list comics")
		return
	}
	for i := range page.Items {
		h.enrichComicListItemURLs(&page.Items[i])
	}
	writeJSON(w, http.StatusOK, page)
}

func (h *Handler) handleGetComic(w http.ResponseWriter, r *http.Request) {
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
	h.enrichComicDetailURLs(&detail)
	writeJSON(w, http.StatusOK, detail)
}

func (h *Handler) handlePatchComic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if !h.requireComicLibraryEnabled(w) {
		return
	}
	comicID := strings.TrimSpace(r.PathValue("comicId"))
	var body contracts.PatchComicBookRequest
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid json body")
			return
		}
	}
	detail, err := h.store.PatchComicBook(r.Context(), comicID, body)
	if err != nil {
		if errors.Is(err, storage.ErrComicBookNotFound) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeComicBookNotFound, "comic not found")
			return
		}
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, err.Error())
		return
	}
	h.enrichComicDetailURLs(&detail)
	writeJSON(w, http.StatusOK, detail)
}

func (h *Handler) handleDeleteComic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if !h.requireComicLibraryEnabled(w) {
		return
	}
	comicID := strings.TrimSpace(r.PathValue("comicId"))
	if err := h.store.DeleteComicBookIndex(r.Context(), comicID); err != nil {
		if errors.Is(err, storage.ErrComicBookNotFound) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeComicBookNotFound, "comic not found")
			return
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to delete comic index")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleRevealComicSource(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
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
	if _, err := os.Stat(detail.Location); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeComicArchiveReadFailed, "comic source file not found")
			return
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to inspect comic source file")
		return
	}
	if err := revealInFileManagerFn(r.Context(), detail.Location); err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to reveal comic source file")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) requireComicLibraryEnabled(w http.ResponseWriter) bool {
	if h.store == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "comic library runtime not available")
		return false
	}
	if !h.comicLibraryEnabled() {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeComicLibraryDisabled, "comic library is disabled")
		return false
	}
	return true
}

func (h *Handler) loadComicDetailForRequest(w http.ResponseWriter, r *http.Request) (contracts.ComicBookDetailDTO, bool) {
	comicID := strings.TrimSpace(r.PathValue("comicId"))
	if comicID == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "comicId is required")
		return contracts.ComicBookDetailDTO{}, false
	}
	detail, err := h.store.GetComicBookDetail(r.Context(), comicID)
	if err != nil {
		if errors.Is(err, storage.ErrComicBookNotFound) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeComicBookNotFound, "comic not found")
			return contracts.ComicBookDetailDTO{}, false
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load comic")
		return contracts.ComicBookDetailDTO{}, false
	}
	return detail, true
}

func (h *Handler) enrichComicDetailURLs(detail *contracts.ComicBookDetailDTO) {
	h.enrichComicListItemURLs(&detail.ComicBookListItemDTO)
	for i := range detail.Pages {
		h.enrichComicPageURLs(detail.ID, &detail.Pages[i])
	}
}

func (h *Handler) enrichComicListItemURLs(item *contracts.ComicBookListItemDTO) {
	if item.PageCount > 0 {
		item.CoverURL = "/api/library/comics/books/" + item.ID + "/pages/0/thumbnail"
	}
}

func (h *Handler) enrichComicPageURLs(comicID string, page *contracts.ComicPageDTO) {
	base := "/api/library/comics/books/" + comicID + "/pages/" + strconv.Itoa(page.Index)
	page.ImageURL = base + "/image"
	page.ThumbURL = base + "/thumbnail"
}

func findComicPage(detail contracts.ComicBookDetailDTO, index int) (contracts.ComicPageDTO, bool) {
	for _, page := range detail.Pages {
		if page.Index == index {
			return page, true
		}
	}
	return contracts.ComicPageDTO{}, false
}

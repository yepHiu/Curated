package server

import (
	"bytes"
	"errors"
	"io"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"curated-backend/internal/contracts"
	"curated-backend/internal/photoarchive"
	"curated-backend/internal/photothumb"
	"curated-backend/internal/storage"
)

func (h *Handler) handleListPhotos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if !h.requirePhotoLibraryEnabled(w) {
		return
	}
	req := contracts.ListPhotoBooksRequest{
		Query: strings.TrimSpace(r.URL.Query().Get("q")),
		Tag:   strings.TrimSpace(r.URL.Query().Get("tag")),
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
	page, err := h.store.ListPhotoBooks(r.Context(), req)
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("list photos failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to list photos")
		return
	}
	for i := range page.Items {
		h.enrichPhotoListItemURLs(&page.Items[i])
	}
	writeJSON(w, http.StatusOK, page)
}

func (h *Handler) handleGetPhoto(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if !h.requirePhotoLibraryEnabled(w) {
		return
	}
	detail, ok := h.loadPhotoDetailForRequest(w, r)
	if !ok {
		return
	}
	h.enrichPhotoDetailURLs(&detail)
	writeJSON(w, http.StatusOK, detail)
}

func (h *Handler) handleListPhotoPages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if !h.requirePhotoLibraryEnabled(w) {
		return
	}
	detail, ok := h.loadPhotoDetailForRequest(w, r)
	if !ok {
		return
	}
	for i := range detail.Pages {
		h.enrichPhotoPageURLs(detail.ID, &detail.Pages[i])
	}
	writeJSON(w, http.StatusOK, detail.Pages)
}

func (h *Handler) handleGetPhotoPageImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if !h.requirePhotoLibraryEnabled(w) {
		return
	}
	detail, page, ok := h.loadPhotoPageForRequest(w, r)
	if !ok {
		return
	}
	body, _, err := photoarchive.OpenPage(r.Context(), detail.Location, page.EntryPath)
	if err != nil {
		if errors.Is(err, photoarchive.ErrPageNotFound) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodePhotoPageNotFound, "photo page not found")
			return
		}
		if h.logger != nil {
			h.logger.Warn("open photo page failed", zap.Error(err), zap.String("photoId", detail.ID), zap.Int("page", page.Index))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodePhotoArchiveReadFailed, "failed to read photo page")
		return
	}
	defer body.Close()

	w.Header().Set("Content-Type", contentTypeForPhotoPage(page))
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, body)
}

var photoThumbnails = photothumb.New()

func (h *Handler) handleGetPhotoPageThumbnail(w http.ResponseWriter, r *http.Request) {
	if !h.requirePhotoLibraryEnabled(w) {
		return
	}
	detail, page, ok := h.loadPhotoPageForRequest(w, r)
	if !ok {
		return
	}
	body, etag, err := photoThumbnails.Get(r.Context(), detail.Location, page.EntryPath)
	if err != nil {
		writeAppError(w, http.StatusUnprocessableEntity, contracts.ErrorCodePhotoArchiveReadFailed, "failed to create photo thumbnail")
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "private, no-cache")
	w.Header().Set("ETag", etag)
	http.ServeContent(w, r, "preview.jpg", time.Time{}, bytes.NewReader(body))
}

func (h *Handler) requirePhotoLibraryEnabled(w http.ResponseWriter) bool {
	if h.store == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "photo library runtime not available")
		return false
	}
	if !h.photoLibraryEnabled() {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodePhotoLibraryDisabled, "photo library is disabled")
		return false
	}
	return true
}

func (h *Handler) loadPhotoDetailForRequest(w http.ResponseWriter, r *http.Request) (contracts.PhotoBookDetailDTO, bool) {
	photoID := strings.TrimSpace(r.PathValue("photoId"))
	if photoID == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "photoId is required")
		return contracts.PhotoBookDetailDTO{}, false
	}
	detail, err := h.store.GetPhotoBookDetail(r.Context(), photoID)
	if err != nil {
		if errors.Is(err, storage.ErrPhotoBookNotFound) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodePhotoBookNotFound, "photo book not found")
			return contracts.PhotoBookDetailDTO{}, false
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load photo book")
		return contracts.PhotoBookDetailDTO{}, false
	}
	return detail, true
}

func (h *Handler) loadPhotoPageForRequest(w http.ResponseWriter, r *http.Request) (contracts.PhotoBookDetailDTO, contracts.PhotoPageDTO, bool) {
	detail, ok := h.loadPhotoDetailForRequest(w, r)
	if !ok {
		return contracts.PhotoBookDetailDTO{}, contracts.PhotoPageDTO{}, false
	}
	raw := strings.TrimSpace(r.PathValue("pageIndex"))
	index, err := strconv.Atoi(raw)
	if err != nil || index < 0 {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "pageIndex must be a non-negative integer")
		return contracts.PhotoBookDetailDTO{}, contracts.PhotoPageDTO{}, false
	}
	page, found := findPhotoPage(detail, index)
	if !found {
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodePhotoPageNotFound, "photo page not found")
		return contracts.PhotoBookDetailDTO{}, contracts.PhotoPageDTO{}, false
	}
	return detail, page, true
}

func (h *Handler) enrichPhotoDetailURLs(detail *contracts.PhotoBookDetailDTO) {
	h.enrichPhotoListItemURLs(&detail.PhotoBookListItemDTO)
	for i := range detail.Pages {
		h.enrichPhotoPageURLs(detail.ID, &detail.Pages[i])
	}
}

func (h *Handler) enrichPhotoListItemURLs(item *contracts.PhotoBookListItemDTO) {
	if item.PageCount > 0 {
		item.CoverURL = "/api/library/photos/books/" + item.ID + "/pages/0/thumbnail"
	}
}

func (h *Handler) enrichPhotoPageURLs(photoID string, page *contracts.PhotoPageDTO) {
	base := "/api/library/photos/books/" + photoID + "/pages/" + strconv.Itoa(page.Index)
	page.ImageURL = base + "/image"
	page.ThumbURL = base + "/thumbnail"
}

func contentTypeForPhotoPage(page contracts.PhotoPageDTO) string {
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

func findPhotoPage(detail contracts.PhotoBookDetailDTO, index int) (contracts.PhotoPageDTO, bool) {
	for _, page := range detail.Pages {
		if page.Index == index {
			return page, true
		}
	}
	return contracts.PhotoPageDTO{}, false
}

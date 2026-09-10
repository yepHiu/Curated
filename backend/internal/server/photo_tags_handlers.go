package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

func (h *Handler) handleReplacePhotoTags(w http.ResponseWriter, r *http.Request) {
	if !h.requirePhotoLibraryEnabled(w) {
		return
	}
	var body contracts.ReplacePhotoTagsRequest
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil || body.Tags == nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "tags array is required")
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid json body")
		return
	}
	id := strings.TrimSpace(r.PathValue("photoId"))
	if err := h.store.ReplacePhotoTags(r.Context(), id, *body.Tags); err != nil {
		switch {
		case errors.Is(err, storage.ErrPhotoBookNotFound):
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodePhotoBookNotFound, "photo not found")
		case errors.Is(err, storage.ErrInvalidUserTags):
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "at most 64 tags, each at most 64 characters")
		default:
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to save photo tags")
		}
		return
	}
	detail, ok := h.loadPhotoDetailForRequest(w, r)
	if !ok {
		return
	}
	h.enrichPhotoDetailURLs(&detail)
	writeJSON(w, http.StatusOK, detail)
}

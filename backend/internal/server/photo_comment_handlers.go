package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
	"go.uber.org/zap"
)

// handleGetPhotoComment returns the personal note for one photo book.
func (h *Handler) handleGetPhotoComment(w http.ResponseWriter, r *http.Request) {
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
	dto, err := h.store.GetPhotoComment(r.Context(), detail.ID)
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("get photo comment failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load comment")
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

// handlePutPhotoComment upserts the personal note for one photo book.
func (h *Handler) handlePutPhotoComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
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
	body, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
	if err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid body")
		return
	}
	var in contracts.PutPhotoCommentRequest
	if err := json.Unmarshal(body, &in); err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid JSON")
		return
	}
	dto, err := h.store.UpsertPhotoComment(r.Context(), detail.ID, in.Body)
	if err != nil {
		if errors.Is(err, storage.ErrPhotoBookNotFound) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodePhotoBookNotFound, "photo not found")
			return
		}
		if errors.Is(err, storage.ErrBookCommentTooLong) {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "comment body too long")
			return
		}
		if h.logger != nil {
			h.logger.Warn("put photo comment failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to save comment")
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

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

// handleGetComicComment returns the personal note for one comic book.
func (h *Handler) handleGetComicComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	detail, ok := h.loadComicDetailWithGate(w, r)
	if !ok {
		return
	}
	dto, err := h.store.GetComicComment(r.Context(), detail.ID)
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("get comic comment failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load comment")
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

// handlePutComicComment upserts the personal note for one comic book.
func (h *Handler) handlePutComicComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	detail, ok := h.loadComicDetailWithGate(w, r)
	if !ok {
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
	if err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid body")
		return
	}
	var in contracts.PutComicCommentRequest
	if err := json.Unmarshal(body, &in); err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid JSON")
		return
	}
	dto, err := h.store.UpsertComicComment(r.Context(), detail.ID, in.Body)
	if err != nil {
		if errors.Is(err, storage.ErrComicBookNotFound) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeComicBookNotFound, "comic not found")
			return
		}
		if errors.Is(err, storage.ErrBookCommentTooLong) {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "comment body too long")
			return
		}
		if h.logger != nil {
			h.logger.Warn("put comic comment failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to save comment")
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

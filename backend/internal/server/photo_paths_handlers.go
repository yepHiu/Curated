package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

func (h *Handler) handleListPhotoLibraryPaths(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.store == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "photo library path runtime not available")
		return
	}
	paths, err := h.store.ListPhotoLibraryPaths(r.Context())
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("list photo library paths failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to list photo library paths")
		return
	}
	writeJSON(w, http.StatusOK, paths)
}

func (h *Handler) handleAddPhotoLibraryPath(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.store == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "photo library path runtime not available")
		return
	}

	var body contracts.AddPhotoLibraryPathRequest
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid json body")
			return
		}
	}

	path := strings.TrimSpace(body.Path)
	if path == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "path is required")
		return
	}
	dto, err := h.store.AddPhotoLibraryPath(r.Context(), path, strings.TrimSpace(body.Title))
	if err != nil {
		if errors.Is(err, storage.ErrPhotoLibraryPathNotAbsolute) {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "photo library path must be an absolute path")
			return
		}
		if errors.Is(err, storage.ErrPhotoLibraryPathDuplicate) {
			writeAppError(w, http.StatusConflict, contracts.ErrorCodeConflict, "photo library path already exists")
			return
		}
		if h.logger != nil {
			h.logger.Error("add photo library path failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to add photo library path")
		return
	}

	h.reloadPhotoLibraryWatchIfAny(r.Context())
	resp := contracts.AddPhotoLibraryPathResponse{PhotoLibraryPathDTO: dto}
	if h.photoScanStarter != nil && h.photoLibraryEnabled() {
		if task, err := h.photoScanStarter.StartPhotoScan(r.Context(), []contracts.PhotoLibraryPathDTO{dto}); err != nil {
			if errors.Is(err, contracts.ErrScanAlreadyRunning) {
				if h.logger != nil {
					h.logger.Warn("add photo library path: initial scan skipped (scan already in progress)", zap.String("path", dto.Path))
				}
			} else if h.logger != nil {
				h.logger.Warn("add photo library path: failed to start initial scan", zap.Error(err), zap.String("path", dto.Path))
			}
		} else {
			resp.ScanTask = &task
		}
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (h *Handler) handlePatchPhotoLibraryPath(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.store == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "photo library path runtime not available")
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "id is required")
		return
	}

	var body contracts.UpdatePhotoLibraryPathRequest
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid json body")
			return
		}
	}
	dto, err := h.store.UpdatePhotoLibraryPathTitle(r.Context(), id, body.Title)
	if err != nil {
		if errors.Is(err, storage.ErrPhotoLibraryPathNotFound) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodePhotoPathNotFound, "photo library path not found")
			return
		}
		if h.logger != nil {
			h.logger.Error("update photo library path failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to update photo library path")
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) handleDeletePhotoLibraryPath(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.store == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "photo library path runtime not available")
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "id is required")
		return
	}
	if err := h.store.DeletePhotoLibraryPath(r.Context(), id); err != nil {
		if errors.Is(err, storage.ErrPhotoLibraryPathNotFound) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodePhotoPathNotFound, "photo library path not found")
			return
		}
		if h.logger != nil {
			h.logger.Error("delete photo library path failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to delete photo library path")
		return
	}
	h.reloadPhotoLibraryWatchIfAny(r.Context())
	w.WriteHeader(http.StatusNoContent)
}

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

func (h *Handler) handleListComicLibraryPaths(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.store == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "comic library path runtime not available")
		return
	}
	paths, err := h.store.ListComicLibraryPaths(r.Context())
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("list comic library paths failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to list comic library paths")
		return
	}
	writeJSON(w, http.StatusOK, paths)
}

func (h *Handler) handleAddComicLibraryPath(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.store == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "comic library path runtime not available")
		return
	}

	var body contracts.AddComicLibraryPathRequest
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
	dto, err := h.store.AddComicLibraryPath(r.Context(), path, strings.TrimSpace(body.Title))
	if err != nil {
		if errors.Is(err, storage.ErrComicLibraryPathNotAbsolute) {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "comic library path must be an absolute path")
			return
		}
		if errors.Is(err, storage.ErrComicLibraryPathDuplicate) {
			writeAppError(w, http.StatusConflict, contracts.ErrorCodeConflict, "comic library path already exists")
			return
		}
		if h.logger != nil {
			h.logger.Error("add comic library path failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to add comic library path")
		return
	}

	writeJSON(w, http.StatusCreated, contracts.AddComicLibraryPathResponse{ComicLibraryPathDTO: dto})
}

func (h *Handler) handlePatchComicLibraryPath(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.store == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "comic library path runtime not available")
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "id is required")
		return
	}

	var body contracts.UpdateComicLibraryPathRequest
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid json body")
			return
		}
	}
	dto, err := h.store.UpdateComicLibraryPathTitle(r.Context(), id, body.Title)
	if err != nil {
		if errors.Is(err, storage.ErrComicLibraryPathNotFound) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeComicPathNotFound, "comic library path not found")
			return
		}
		if h.logger != nil {
			h.logger.Error("update comic library path failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to update comic library path")
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) handleDeleteComicLibraryPath(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.store == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "comic library path runtime not available")
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "id is required")
		return
	}
	if err := h.store.DeleteComicLibraryPath(r.Context(), id); err != nil {
		if errors.Is(err, storage.ErrComicLibraryPathNotFound) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeComicPathNotFound, "comic library path not found")
			return
		}
		if h.logger != nil {
			h.logger.Error("delete comic library path failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to delete comic library path")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

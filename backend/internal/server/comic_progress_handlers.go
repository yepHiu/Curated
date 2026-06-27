package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

func (h *Handler) handleGetComicProgress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	detail, ok := h.loadComicDetailWithGate(w, r)
	if !ok {
		return
	}
	progress, err := h.store.GetComicProgress(r.Context(), detail.ID)
	if err != nil {
		if errors.Is(err, storage.ErrComicBookNotFound) {
			writeJSON(w, http.StatusOK, contracts.ComicReadingProgressDTO{ComicID: detail.ID})
			return
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load comic progress")
		return
	}
	writeJSON(w, http.StatusOK, progress)
}

func (h *Handler) handlePutComicProgress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	detail, ok := h.loadComicDetailWithGate(w, r)
	if !ok {
		return
	}
	var body contracts.PutComicProgressRequest
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid json body")
			return
		}
	}
	if body.PageIndex < 0 || body.PageIndex >= detail.PageCount {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "pageIndex is out of range")
		return
	}
	if err := h.store.SaveComicProgress(r.Context(), detail.ID, body.PageIndex, body.Completed); err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to save comic progress")
		return
	}
	progress, err := h.store.GetComicProgress(r.Context(), detail.ID)
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load comic progress")
		return
	}
	writeJSON(w, http.StatusOK, progress)
}

func (h *Handler) handleDeleteComicProgress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	detail, ok := h.loadComicDetailWithGate(w, r)
	if !ok {
		return
	}
	if err := h.store.DeleteComicProgress(r.Context(), detail.ID); err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to reset comic progress")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleGetComicPreferences(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	detail, ok := h.loadComicDetailWithGate(w, r)
	if !ok {
		return
	}
	prefs, err := h.store.GetComicReadingPreferences(r.Context(), detail.ID)
	if err != nil {
		if errors.Is(err, storage.ErrComicBookNotFound) {
			writeJSON(w, http.StatusOK, h.defaultComicReadingPreferences(detail.ID))
			return
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load comic preferences")
		return
	}
	writeJSON(w, http.StatusOK, prefs)
}

func (h *Handler) handlePutComicPreferences(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	detail, ok := h.loadComicDetailWithGate(w, r)
	if !ok {
		return
	}
	current := h.defaultComicReadingPreferences(detail.ID)
	if saved, err := h.store.GetComicReadingPreferences(r.Context(), detail.ID); err == nil {
		current = saved
	}
	var body contracts.PutComicReadingPreferencesRequest
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid json body")
			return
		}
	}
	if body.Mode != nil {
		current.Mode = config.NormalizeComicReaderMode(*body.Mode)
	}
	if body.Fit != nil {
		current.Fit = config.NormalizeComicFitMode(*body.Fit)
	}
	if body.Direction != nil {
		current.Direction = config.NormalizeComicReadingDirection(*body.Direction)
	}
	if err := h.store.SaveComicReadingPreferences(r.Context(), detail.ID, current); err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to save comic preferences")
		return
	}
	prefs, err := h.store.GetComicReadingPreferences(r.Context(), detail.ID)
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load comic preferences")
		return
	}
	writeJSON(w, http.StatusOK, prefs)
}

func (h *Handler) loadComicDetailWithGate(w http.ResponseWriter, r *http.Request) (contracts.ComicBookDetailDTO, bool) {
	if !h.requireComicLibraryEnabled(w) {
		return contracts.ComicBookDetailDTO{}, false
	}
	return h.loadComicDetailForRequest(w, r)
}

func (h *Handler) defaultComicReadingPreferences(comicID string) contracts.ComicReadingPreferencesDTO {
	reader := comicReaderSettingsDTOFromConfig(h.cfg.ComicReader)
	if h.comicSettingsCtl != nil {
		reader = normalizeComicReaderSettingsDTO(h.comicSettingsCtl.ComicReaderSettings())
	}
	return contracts.ComicReadingPreferencesDTO{
		ComicID:   comicID,
		Mode:      reader.Mode,
		Fit:       reader.Fit,
		Direction: reader.Direction,
	}
}

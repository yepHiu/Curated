package server

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

func normalizeSavedViewFilters(input contracts.SavedViewFiltersV1) (contracts.SavedViewFiltersV1, error) {
	return storage.NormalizeSavedViewFilters(input)
}

func normalizeSavedViewYear(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return ""
	}
	if normalized == "unknown" {
		return "unknown"
	}
	if len(normalized) != 4 {
		return ""
	}
	year, err := strconv.Atoi(normalized)
	if err != nil || year < 1800 || year > 3000 {
		return ""
	}
	return normalized
}

func normalizeSavedViewResolution(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return ""
	case "4k", "2160p", "uhd", "3840x2160":
		return "4k"
	case "1080p", "full hd", "fhd":
		return "1080p"
	case "720p", "hd":
		return "720p"
	case "480p", "sd":
		return "480p"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func normalizeSavedViewName(value string) (string, error) {
	return storage.NormalizeSavedViewName(value)
}

func newSavedViewID() (string, error) {
	var random [12]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	return "view_" + hex.EncodeToString(random[:]), nil
}

func (h *Handler) handleListSavedViews(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.ListSavedViews(r.Context())
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to list saved views")
		return
	}
	writeJSON(w, http.StatusOK, contracts.SavedViewsDTO{Items: items})
}

func (h *Handler) handleCreateSavedView(w http.ResponseWriter, r *http.Request) {
	var body contracts.CreateSavedViewBody
	if err := decodeStrictJSONBody(w, r, &body); err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeSavedViewInvalid, "invalid saved view request")
		return
	}
	name, err := normalizeSavedViewName(body.Name)
	if err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeSavedViewInvalid, err.Error())
		return
	}
	filters, err := normalizeSavedViewFilters(body.Filters)
	if err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeSavedViewInvalid, err.Error())
		return
	}
	id, err := newSavedViewID()
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to allocate saved view")
		return
	}
	item, err := h.store.CreateSavedView(r.Context(), id, name, filters, time.Now())
	if err != nil {
		writeSavedViewError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (h *Handler) handlePatchSavedView(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("savedViewId"))
	if id == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeSavedViewInvalid, "saved view id is required")
		return
	}
	var body contracts.PatchSavedViewBody
	if err := decodeStrictJSONBody(w, r, &body); err != nil || (body.Name == nil && body.Filters == nil) {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeSavedViewInvalid, "invalid saved view patch")
		return
	}
	current, err := h.store.GetSavedView(r.Context(), id)
	if err != nil {
		writeSavedViewError(w, err)
		return
	}
	name := current.Name
	if body.Name != nil {
		name, err = normalizeSavedViewName(*body.Name)
		if err != nil {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeSavedViewInvalid, err.Error())
			return
		}
	}
	filters := current.Filters
	if body.Filters != nil {
		filters, err = normalizeSavedViewFilters(*body.Filters)
		if err != nil {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeSavedViewInvalid, err.Error())
			return
		}
	}
	item, err := h.store.UpdateSavedView(r.Context(), id, name, filters, time.Now())
	if err != nil {
		writeSavedViewError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) handleDeleteSavedView(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("savedViewId"))
	if id == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeSavedViewInvalid, "saved view id is required")
		return
	}
	if err := h.store.DeleteSavedView(r.Context(), id); err != nil {
		writeSavedViewError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleReorderSavedViews(w http.ResponseWriter, r *http.Request) {
	var body contracts.ReorderSavedViewsBody
	if err := decodeStrictJSONBody(w, r, &body); err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeSavedViewInvalid, "invalid saved view order")
		return
	}
	for index := range body.IDs {
		body.IDs[index] = strings.TrimSpace(body.IDs[index])
		if body.IDs[index] == "" {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeSavedViewInvalid, "saved view order contains an empty id")
			return
		}
	}
	if err := h.store.ReorderSavedViews(r.Context(), body.IDs, time.Now()); err != nil {
		writeSavedViewError(w, err)
		return
	}
	items, err := h.store.ListSavedViews(r.Context())
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to list reordered saved views")
		return
	}
	writeJSON(w, http.StatusOK, contracts.SavedViewsDTO{Items: items})
}

func writeSavedViewError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, storage.ErrSavedViewNotFound):
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "saved view not found")
	case errors.Is(err, storage.ErrSavedViewNameConflict):
		writeAppError(w, http.StatusConflict, contracts.ErrorCodeSavedViewNameConflict, "saved view name already exists")
	case errors.Is(err, storage.ErrSavedViewLimit):
		writeAppError(w, http.StatusConflict, contracts.ErrorCodeSavedViewLimit, "saved view limit reached")
	case errors.Is(err, storage.ErrSavedViewOrderInvalid):
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeSavedViewInvalid, "saved view order must contain every view exactly once")
	default:
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "saved view operation failed")
	}
}

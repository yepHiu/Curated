package server

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

const (
	maxSavedViewNameRunes = 40
	maxSavedViewTextRunes = 200
)

func normalizeSavedViewFilters(input contracts.SavedViewFiltersV1) (contracts.SavedViewFiltersV1, error) {
	if input.SchemaVersion != contracts.SavedViewSchemaVersion {
		return contracts.SavedViewFiltersV1{}, errors.New("unsupported saved view schemaVersion")
	}
	out := contracts.SavedViewFiltersV1{
		SchemaVersion: contracts.SavedViewSchemaVersion,
		Mode:          strings.ToLower(strings.TrimSpace(input.Mode)),
		Query:         strings.TrimSpace(input.Query),
		Tag:           strings.TrimSpace(input.Tag),
		Actor:         strings.TrimSpace(input.Actor),
		Studio:        strings.TrimSpace(input.Studio),
		Tab:           strings.ToLower(strings.TrimSpace(input.Tab)),
		PlayState:     strings.ToLower(strings.TrimSpace(input.PlayState)),
		Resolution:    normalizeSavedViewResolution(input.Resolution),
		Year:          normalizeSavedViewYear(input.Year),
		Runtime:       strings.ToLower(strings.TrimSpace(input.Runtime)),
		Catalog:       strings.ToLower(strings.TrimSpace(input.Catalog)),
	}
	if out.Mode == "" {
		out.Mode = "library"
	}
	if !oneOf(out.Mode, "library", "favorites", "recent", "tags", "trash") {
		return contracts.SavedViewFiltersV1{}, errors.New("invalid saved view mode")
	}
	if out.Tab == "" {
		out.Tab = "all"
	}
	if !oneOf(out.Tab, "all", "new", "top-rated") {
		return contracts.SavedViewFiltersV1{}, errors.New("invalid saved view tab")
	}
	if out.PlayState == "" {
		out.PlayState = "all"
	}
	if !oneOf(out.PlayState, "all", "unwatched", "in-progress", "completed") {
		return contracts.SavedViewFiltersV1{}, errors.New("invalid saved view playState")
	}
	for _, value := range []string{out.Query, out.Tag, out.Actor, out.Studio} {
		if utf8.RuneCountInString(value) > maxSavedViewTextRunes {
			return contracts.SavedViewFiltersV1{}, errors.New("saved view filter text is too long")
		}
	}
	if utf8.RuneCountInString(out.Resolution) > 40 {
		return contracts.SavedViewFiltersV1{}, errors.New("saved view resolution is too long")
	}
	if input.UserRating != nil {
		value := *input.UserRating
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 5 {
			return contracts.SavedViewFiltersV1{}, errors.New("saved view userRating must be between 0 and 5")
		}
		out.UserRating = &value
	}
	if input.Unrated {
		out.Unrated = true
		out.UserRating = nil
	}
	if out.Year == "" && strings.TrimSpace(input.Year) != "" {
		return contracts.SavedViewFiltersV1{}, errors.New("saved view year must be unknown or a year from 1800 to 3000")
	}
	if out.Runtime != "" && !oneOf(out.Runtime, "short", "standard", "long") {
		return contracts.SavedViewFiltersV1{}, errors.New("invalid saved view runtime")
	}
	if out.Catalog != "" && !oneOf(out.Catalog, "unscraped", "no-cover") {
		return contracts.SavedViewFiltersV1{}, errors.New("invalid saved view catalog")
	}
	if input.AddedWithinDays < 0 || input.AddedWithinDays > 3650 {
		return contracts.SavedViewFiltersV1{}, errors.New("saved view addedWithinDays must be between 1 and 3650")
	}
	out.AddedWithinDays = input.AddedWithinDays

	if out.Mode == "trash" {
		return contracts.SavedViewFiltersV1{
			SchemaVersion: contracts.SavedViewSchemaVersion,
			Mode:          "trash",
			Tab:           "all",
			PlayState:     "all",
		}, nil
	}
	return out, nil
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
	name := strings.TrimSpace(value)
	if name == "" {
		return "", errors.New("saved view name is required")
	}
	if utf8.RuneCountInString(name) > maxSavedViewNameRunes {
		return "", errors.New("saved view name is too long")
	}
	return name, nil
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

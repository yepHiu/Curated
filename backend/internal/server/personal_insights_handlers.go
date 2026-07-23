package server

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"curated-backend/internal/contracts"
	"go.uber.org/zap"
)

func (h *Handler) handleGetPersonalInsightsOverview(w http.ResponseWriter, r *http.Request) {
	if h.personalInsightsProvider == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "personal insights not configured")
		return
	}
	result, err := h.personalInsightsProvider.GetPersonalInsightsOverview(
		r.Context(),
		r.URL.Query().Get("range"),
		r.URL.Query().Get("timezone"),
	)
	if err != nil {
		h.writePersonalInsightsError(w, "get personal insights overview", err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) handleGetPersonalInsightsBreakdown(w http.ResponseWriter, r *http.Request) {
	if h.personalInsightsProvider == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "personal insights not configured")
		return
	}
	limit := 0
	rawLimit := strings.TrimSpace(r.URL.Query().Get("limit"))
	if rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodePersonalInsightsInvalidLimit, "invalid personal insights limit")
			return
		}
		limit = parsed
	}
	result, err := h.personalInsightsProvider.GetPersonalInsightsBreakdown(
		r.Context(),
		r.URL.Query().Get("range"),
		r.URL.Query().Get("timezone"),
		r.URL.Query().Get("dimension"),
		limit,
	)
	if err != nil {
		h.writePersonalInsightsError(w, "get personal insights breakdown", err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) writePersonalInsightsError(w http.ResponseWriter, operation string, err error) {
	switch {
	case errors.Is(err, contracts.ErrPersonalInsightsInvalidRange):
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodePersonalInsightsInvalidRange, err.Error())
	case errors.Is(err, contracts.ErrPersonalInsightsInvalidTimezone):
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodePersonalInsightsInvalidTimezone, err.Error())
	case errors.Is(err, contracts.ErrPersonalInsightsInvalidDimension):
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodePersonalInsightsInvalidDimension, err.Error())
	case errors.Is(err, contracts.ErrPersonalInsightsInvalidLimit):
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodePersonalInsightsInvalidLimit, err.Error())
	default:
		if h.logger != nil {
			h.logger.Warn(operation+" failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "personal insights failed")
	}
}

package server

import (
	"errors"
	"net/http"
	"strings"

	"curated-backend/internal/contracts"
)

func (h *Handler) handleListHomepageRecommendationFeedback(w http.ResponseWriter, r *http.Request) {
	if h.homepageRecommendationFeedback == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "recommendation feedback not configured")
		return
	}
	dto, err := h.homepageRecommendationFeedback.ListHomepageRecommendationFeedback(r.Context())
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to list recommendation feedback")
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) handleCreateHomepageRecommendationFeedback(w http.ResponseWriter, r *http.Request) {
	if h.homepageRecommendationFeedback == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "recommendation feedback not configured")
		return
	}
	var body contracts.CreateRecommendationFeedbackBody
	if err := decodeStrictJSONBody(w, r, &body); err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeRecommendationFeedbackInvalid, "invalid recommendation feedback request")
		return
	}
	dto, err := h.homepageRecommendationFeedback.CreateHomepageRecommendationFeedback(r.Context(), body)
	if err != nil {
		writeRecommendationFeedbackError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, dto)
}

func (h *Handler) handleDeleteHomepageRecommendationFeedback(w http.ResponseWriter, r *http.Request) {
	if h.homepageRecommendationFeedback == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "recommendation feedback not configured")
		return
	}
	id := strings.TrimSpace(r.PathValue("feedbackId"))
	if err := h.homepageRecommendationFeedback.DeleteHomepageRecommendationFeedback(r.Context(), id); err != nil {
		writeRecommendationFeedbackError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeRecommendationFeedbackError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, contracts.ErrRecommendationFeedbackInvalid):
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeRecommendationFeedbackInvalid, "invalid recommendation feedback")
	case errors.Is(err, contracts.ErrRecommendationFeedbackTargetNotFound):
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodeRecommendationFeedbackTargetNotFound, "recommendation feedback target not found")
	case errors.Is(err, contracts.ErrRecommendationFeedbackLimitReached):
		writeAppError(w, http.StatusConflict, contracts.ErrorCodeRecommendationFeedbackLimit, "recommendation feedback limit reached")
	default:
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "recommendation feedback operation failed")
	}
}

package server

import (
	"net/http"

	"go.uber.org/zap"

	"curated-backend/internal/contracts"
)

func (h *Handler) handleGetHomepageRecommendations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.homepageRecommendations == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "homepage recommendations not configured")
		return
	}

	dto, err := h.homepageRecommendations.GetOrCreateHomepageDailyRecommendations(r.Context(), "")
	if err != nil {
		h.logger.Error("get homepage recommendations", zap.Error(err))
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load homepage recommendations")
		return
	}

	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) handleRefreshHomepageRecommendations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.homepageRecommendations == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "homepage recommendations not configured")
		return
	}

	dto, err := h.homepageRecommendations.RegenerateHomepageDailyRecommendations(r.Context(), "")
	if err != nil {
		h.logger.Error("refresh homepage recommendations", zap.Error(err))
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to refresh homepage recommendations")
		return
	}

	writeJSON(w, http.StatusOK, dto)
}

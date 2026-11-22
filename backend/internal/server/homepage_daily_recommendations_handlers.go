package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

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

	started := time.Now()
	dto, err := h.homepageRecommendations.GetOrCreateHomepageDailyRecommendations(r.Context(), "")
	if err != nil {
		h.logger.Error("get homepage recommendations", zap.Error(err))
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load homepage recommendations")
		return
	}

	if r.URL.Query().Get("include") == "cards" {
		ids := append(append([]string(nil), dto.HeroMovieIDs...), dto.RecommendationMovieIDs...)
		cards, err := h.loadMovieCards(r.Context(), ids)
		if err != nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load homepage cards")
			return
		}
		dto.Cards = &cards.Items
	}
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Server-Timing", fmt.Sprintf("homepage;dur=%.2f", float64(time.Since(started).Microseconds())/1000))

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

	var options contracts.HomepageDailyRecommendationsRefreshOptions
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&options); err != nil && !errors.Is(err, io.EOF) {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid json body")
			return
		}
	}

	dto, err := h.homepageRecommendations.RegenerateHomepageDailyRecommendations(r.Context(), "", options)
	if err != nil {
		h.logger.Error("refresh homepage recommendations", zap.Error(err))
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to refresh homepage recommendations")
		return
	}

	writeJSON(w, http.StatusOK, dto)
}

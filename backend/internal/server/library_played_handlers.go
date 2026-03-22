package server

import (
	"net/http"
	"strings"

	"go.uber.org/zap"

	"jav-shadcn/backend/internal/contracts"
	"jav-shadcn/backend/internal/storage"
)

func (h *Handler) handleListPlayedMovies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	ids, err := h.store.ListPlayedMovieIDs(r.Context())
	if err != nil {
		h.logger.Error("list played movies", zap.Error(err))
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to list played movies")
		return
	}
	if ids == nil {
		ids = []string{}
	}
	writeJSON(w, http.StatusOK, contracts.PlayedMoviesListDTO{MovieIDs: ids})
}

func (h *Handler) handleRecordPlayedMovie(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	movieID := strings.TrimSpace(r.PathValue("movieId"))
	if movieID == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "movieId is required")
		return
	}
	err := h.store.RecordPlayedMovieIfMovieExists(r.Context(), movieID)
	if err != nil {
		if err == storage.ErrPlayedMovieMovieNotFound {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie not found")
			return
		}
		h.logger.Error("record played movie", zap.Error(err))
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to record played movie")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

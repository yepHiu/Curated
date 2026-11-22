package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"curated-backend/internal/contracts"
)

func (h *Handler) loadMovieCards(ctx context.Context, ids []string) (contracts.MovieCardsDTO, error) {
	if h.store == nil {
		return contracts.MovieCardsDTO{}, fmt.Errorf("movie store unavailable")
	}
	result, err := h.store.MovieCards(ctx, ids)
	if err != nil {
		return result, err
	}
	items := make([]contracts.MovieListItemDTO, len(result.Items))
	for i, card := range result.Items {
		items[i] = contracts.MovieListItemDTO{ID: card.ID, CoverURL: card.CoverURL, ThumbURL: card.ThumbURL}
	}
	h.enrichMovieListItemsLocalPosters(ctx, items)
	for i := range result.Items {
		result.Items[i].CoverURL = items[i].CoverURL
		result.Items[i].ThumbURL = items[i].ThumbURL
	}
	return result, nil
}

func (h *Handler) handleMovieCards(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	var request struct {
		MovieIDs []string `json:"movieIds"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
	if err := decoder.Decode(&request); err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid movie IDs")
		return
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF || len(request.MovieIDs) == 0 || len(request.MovieIDs) > contracts.MaxMovieCardIDs {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "expected 1 to 50 movie IDs")
		return
	}
	result, err := h.loadMovieCards(r.Context(), request.MovieIDs)
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load movie cards")
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Server-Timing", fmt.Sprintf("cards;dur=%.2f", float64(time.Since(started).Microseconds())/1000))
	writeJSON(w, http.StatusOK, result)
}

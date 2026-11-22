package server

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"net/http"
	"regexp"
	"time"

	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

type nativePlaybackResolver interface {
	PlanNativePlayback(context.Context, string) (contracts.NativePlaybackPlanDTO, error)
	ResolveNativePlayback(context.Context, string, contracts.NativePlaybackSelection, *float64) (contracts.PlaybackDescriptorDTO, error)
}

var nativeSourceToken = regexp.MustCompile(`^[0-9a-f]{64}$`)

func (h *Handler) handleNativePlayback(w http.ResponseWriter, r *http.Request, movieID string, body contracts.CreatePlaybackSessionRequest) {
	resolver, ok := h.playbackResolver.(nativePlaybackResolver)
	if !ok {
		writeAppError(w, 400, contracts.ErrorCodeBadRequest, "native playback is unavailable")
		return
	}
	ctx, timing := contracts.WithPlaybackTiming(r.Context())
	started := time.Now()
	var result any
	var err error
	if body.Mode == "plan" {
		if body.NativePlayback != nil {
			writeAppError(w, 400, contracts.ErrorCodeBadRequest, "plan does not accept a decoder selection")
			return
		}
		result, err = resolver.PlanNativePlayback(ctx, movieID)
	} else {
		selection := body.NativePlayback
		if selection == nil || selection.Version != 1 || !nativeSourceToken.MatchString(selection.SourceToken) || (selection.ForceTranscode && selection.DirectPlay) {
			writeAppError(w, 400, contracts.ErrorCodeBadRequest, "invalid native playback selection")
			return
		}
		if p := body.AutoStartPositionSec; p != nil && (*p < 0 || math.IsNaN(*p) || math.IsInf(*p, 0)) {
			writeAppError(w, 400, contracts.ErrorCodeBadRequest, "invalid autoStartPositionSec")
			return
		}
		result, err = resolver.ResolveNativePlayback(ctx, movieID, *selection, body.AutoStartPositionSec)
	}
	contracts.RecordPlaybackTiming(ctx, "playback", started)
	w.Header().Set("Server-Timing", timing.Header())
	w.Header().Set("Cache-Control", "no-store")
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, storage.ErrMovieVideoNotFound) {
			writeAppError(w, 404, contracts.ErrorCodeNotFound, "movie not found")
			return
		}
		writeAppError(w, 400, contracts.ErrorCodeBadRequest, "native playback preparation failed")
		return
	}
	status := http.StatusCreated
	if body.Mode == "plan" {
		status = http.StatusOK
	}
	writeJSON(w, status, result)
}

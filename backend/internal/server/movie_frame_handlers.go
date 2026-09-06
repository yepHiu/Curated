package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"curated-backend/internal/contracts"
	"curated-backend/internal/executil"
	"curated-backend/internal/playback"
)

type boundedFrameBuffer struct {
	bytes.Buffer
	limit int
}

func (b *boundedFrameBuffer) Write(p []byte) (int, error) {
	if len(p) > b.limit-b.Len() {
		return 0, fmt.Errorf("frame output exceeds limit")
	}
	return b.Buffer.Write(p)
}

func (h *Handler) handleExtractMovieFrame(w http.ResponseWriter, r *http.Request) {
	var req contracts.ExtractMovieFrameBody
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil || math.IsNaN(req.PositionSec) || math.IsInf(req.PositionSec, 0) || req.PositionSec < 0 || req.PositionSec > 7*24*3600 {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid frame position")
		return
	}
	if h.store == nil {
		writeAppError(w, http.StatusServiceUnavailable, contracts.ErrorCodeInternal, "frame extraction unavailable")
		return
	}
	path, err := h.store.ResolvePrimaryVideoPath(r.Context(), strings.TrimSpace(r.PathValue("movieId")))
	if err != nil {
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "source video unavailable")
		return
	}
	h.initMovieClipQueue()
	select {
	case h.movieClipWorkers <- struct{}{}:
		defer func() { <-h.movieClipWorkers }()
	default:
		writeAppError(w, http.StatusTooManyRequests, contracts.ErrorCodeConflict, "media encoder busy")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	cmd := executil.CommandContext(ctx, playback.ResolveFFmpegCommand(h.cfg.Player.FFmpegCommand), "-hide_banner", "-loglevel", "error", "-threads", "2", "-ss", strconv.FormatFloat(req.PositionSec, 'f', 6, 64), "-i", path, "-frames:v", "1", "-an", "-f", "image2pipe", "-c:v", "png", "pipe:1")
	output := boundedFrameBuffer{limit: 32 << 20}
	stderr := boundedFrameBuffer{limit: 8192}
	cmd.Stdout = &output
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil || output.Len() == 0 {
		writeAppError(w, http.StatusUnprocessableEntity, contracts.ErrorCodeBadRequest, "source frame extraction failed or exceeded limits")
		return
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(output.Bytes()))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > 3840*2160*4 {
		writeAppError(w, http.StatusUnprocessableEntity, contracts.ErrorCodeBadRequest, "source frame exceeds pixel budget")
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Frame-Width", strconv.Itoa(cfg.Width))
	w.Header().Set("X-Frame-Height", strconv.Itoa(cfg.Height))
	_, _ = w.Write(output.Bytes())
}

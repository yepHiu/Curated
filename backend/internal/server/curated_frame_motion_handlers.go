package server

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"curated-backend/internal/contracts"
	"go.uber.org/zap"
)

func curatedFrameMotionRoot(h *Handler) string {
	base := strings.TrimSpace(h.cfg.CacheDir)
	if base == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "curated-frame-motions")
}

func (h *Handler) handleGetCuratedFrameMotion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" || h.store == nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "id is required")
		return
	}
	motion, err := h.store.GetCuratedFrameMotion(r.Context(), id)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("get curated frame motion", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load motion")
		return
	}
	if motion == nil || motion.Status != "ready" || motion.ArtifactName == "" {
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "curated frame motion is not ready")
		return
	}
	root, err := filepath.Abs(curatedFrameMotionRoot(h))
	if err != nil {
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "curated frame motion is unavailable")
		return
	}
	path := filepath.Join(root, filepath.Base(motion.ArtifactName))
	if !pathUnderRoot(path, root) {
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "curated frame motion is unavailable")
		return
	}
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "curated frame motion is unavailable")
			return
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to open motion")
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to stat motion")
		return
	}
	w.Header().Set("Content-Type", motion.ContentType)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	http.ServeContent(w, r, filepath.Base(path), info.ModTime(), f)
}

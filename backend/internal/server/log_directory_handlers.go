package server

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
)

// No client-supplied path is accepted; opening always happens on the Server host.
func (h *Handler) handleRevealLogDirectory(w http.ResponseWriter, r *http.Request) {
	if !requireLocalDesktopOperation(w, r) {
		return
	}
	dir, err := filepath.Abs(config.DefaultLogDir())
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to resolve log directory")
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to create log directory")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	if err := openDirectoryFn(ctx, dir); err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to open log directory")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

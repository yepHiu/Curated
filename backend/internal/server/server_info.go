package server

import (
	"net/http"
	"os"
	"strings"

	"curated-backend/internal/contracts"
	"curated-backend/internal/version"
)

func (h *Handler) serverInfo() contracts.ServerInfoDTO {
	name := strings.TrimSpace(h.cfg.ServerName)
	if name == "" {
		name, _ = os.Hostname()
	}
	if name == "" {
		name = "Curated Server"
	}
	capabilities := []string{"web-ui", "movie-cards-v1", "movie-image-variants-v1"}
	if _, ok := h.playbackResolver.(nativePlaybackResolver); ok {
		capabilities = append(capabilities, "native-playback-v1")
	}
	return contracts.ServerInfoDTO{
		Product: "curated-server", ServerID: h.cfg.ServerID, Name: name,
		Version: version.ProductVersion(), ProtocolVersion: 1,
		DesktopBridgeVersion: 1, Capabilities: capabilities,
	}
}

func (h *Handler) handleServerInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if h.cfg.ServerID == "" {
		writeAppError(w, http.StatusServiceUnavailable, "SERVER_IDENTITY_UNAVAILABLE", "Server identity is not initialized")
		return
	}
	writeJSON(w, http.StatusOK, h.serverInfo())
}

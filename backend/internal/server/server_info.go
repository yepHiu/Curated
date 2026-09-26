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
	return contracts.ServerInfoDTO{
		Product: "curated-server", ServerID: h.cfg.ServerID, Name: name,
		Version: version.ProductVersion(), ProtocolVersion: 1,
		DesktopBridgeVersion: 1, Capabilities: []string{"web-ui"},
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

package server

import (
	"curated-backend/internal/contracts"
	"net/http"
)

// Host-only desktop actions are unavailable to remote clients. Do not infer
// locality from user-controlled forwarding headers or the Desktop marker.
func requireLocalDesktopOperation(w http.ResponseWriter, r *http.Request) bool {
	if isDirectLocalRequest(r) {
		return true
	}
	writeAppError(w, http.StatusForbidden, contracts.ErrorCodeForbidden, "This operation opens a program on the server and is only available from the server itself")
	return false
}

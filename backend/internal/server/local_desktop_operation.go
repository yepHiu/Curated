package server

import (
	"curated-backend/internal/contracts"
	"net"
	"net/http"
)

// Host-only desktop actions are unavailable to remote clients. Do not infer
// locality from user-controlled forwarding headers or the Desktop marker.
func requireLocalDesktopOperation(w http.ResponseWriter, r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	ip := net.ParseIP(host)
	if err == nil && ip != nil && ip.IsLoopback() && r.Header.Get("Forwarded") == "" && r.Header.Get("X-Forwarded-For") == "" {
		return true
	}
	writeAppError(w, http.StatusForbidden, contracts.ErrorCodeForbidden, "This operation opens a program on the server and is only available from the server itself")
	return false
}

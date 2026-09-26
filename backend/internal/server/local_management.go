package server

import (
	"net"
	"net/http"
	"net/url"
	"strings"

	"curated-backend/internal/contracts"
)

// Server-local management requires a direct loopback request. Forwarded and
// ambiguous LAN/hostname connections do not authorize host configuration changes.
func isDirectLocalRequest(r *http.Request) bool {
	peer, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil || !isLocalRequestLoopback(peer) {
		return false
	}
	target, err := url.Parse("http://" + r.Host)
	if err != nil || target.User != nil || !isLocalRequestLoopback(target.Hostname()) {
		return false
	}
	for name := range r.Header {
		name = strings.ToLower(name)
		if name == "forwarded" || name == "x-real-ip" || strings.HasPrefix(name, "x-forwarded-") {
			return false
		}
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		parsed, err := url.Parse(origin)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || !isLocalRequestLoopback(parsed.Hostname()) {
			return false
		}
	}
	return true
}

func isLocalRequestLoopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func requireLocalLibraryPathManagement(w http.ResponseWriter, r *http.Request) bool {
	if isDirectLocalRequest(r) {
		return true
	}
	writeAppError(w, http.StatusForbidden, contracts.ErrorCodeLibraryPathsReadOnly, "Manage storage directories on the Server computer")
	return false
}

func localLibraryPathManagement(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !requireLocalLibraryPathManagement(w, r) {
			return
		}
		next(w, r)
	}
}

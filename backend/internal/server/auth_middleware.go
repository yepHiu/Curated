package server

import (
	"net/http"
	"strings"

	"curated-backend/internal/contracts"
)

func (h *Handler) withAuthLock(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h == nil || h.store == nil || !isAPIPath(r.URL.Path) || isAuthPublicPath(r.Method, r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		settings, err := h.store.GetAppSecuritySettings(r.Context())
		if err != nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to read auth settings")
			return
		}
		if !settings.PINEnabled {
			next.ServeHTTP(w, r)
			return
		}

		if _, ok, err := h.authSessionFromRequest(r); err != nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to validate auth session")
			return
		} else if ok {
			next.ServeHTTP(w, r)
			return
		}

		writeAppError(w, http.StatusLocked, contracts.ErrorCodeAuthLocked, "Curated is locked")
	})
}

func isAPIPath(path string) bool {
	return path == "/api" || strings.HasPrefix(path, "/api/")
}

func isAuthPublicPath(method string, path string) bool {
	if method == http.MethodOptions {
		return true
	}
	switch path {
	case "/api/health":
		return method == http.MethodGet
	case "/api/auth/status":
		return method == http.MethodGet
	case "/api/auth/setup-pin":
		return method == http.MethodPost
	case "/api/auth/unlock":
		return method == http.MethodPost
	case "/api/auth/lock":
		return method == http.MethodPost
	default:
		return false
	}
}

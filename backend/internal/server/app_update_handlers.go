package server

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"go.uber.org/zap"

	"curated-backend/internal/appupdate"
	"curated-backend/internal/contracts"
)

func (h *Handler) handleGetAppUpdateStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.appUpdateProvider == nil {
		writeAppUpdateStatus(w, r, unsupportedAppUpdateStatus())
		return
	}

	dto, err := h.appUpdateProvider.GetAppUpdateStatus(r.Context())
	if err != nil {
		if h.logger != nil {
			h.logger.Error("get app update status failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load app update status")
		return
	}

	writeAppUpdateStatus(w, r, dto)
}

func (h *Handler) handleCheckAppUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.appUpdateProvider == nil {
		writeAppUpdateStatus(w, r, unsupportedAppUpdateStatus())
		return
	}

	dto, err := h.appUpdateProvider.CheckAppUpdateNow(r.Context())
	if err != nil {
		if h.logger != nil {
			h.logger.Error("check app update failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to check app update")
		return
	}

	writeAppUpdateStatus(w, r, dto)
}

func (h *Handler) handleDownloadAppUpdateInstaller(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if !allowsLocalAppUpdate(r) {
		writeAppError(w, http.StatusForbidden, contracts.ErrorCodeAppUpdateRemoteDisabled, "Update Server on its own machine; remote updates are not supported")
		return
	}
	if h.appUpdateProvider == nil {
		writeAppUpdateStatus(w, r, unsupportedAppUpdateStatus())
		return
	}

	dto, err := h.appUpdateProvider.DownloadAppUpdateInstaller(r.Context())
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("download app update installer failed", zap.Error(err))
		}
		writeAppError(w, http.StatusConflict, contracts.ErrorCodeAppUpdateDownloadFailed, err.Error())
		return
	}
	writeAppUpdateStatus(w, r, dto)
}

func (h *Handler) handleInstallAppUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if !allowsLocalAppUpdate(r) {
		writeAppError(w, http.StatusForbidden, contracts.ErrorCodeAppUpdateRemoteDisabled, "Update Server on its own machine; remote updates are not supported")
		return
	}
	if h.appUpdateProvider == nil {
		writeAppUpdateStatus(w, r, unsupportedAppUpdateStatus())
		return
	}

	var body contracts.AppUpdateInstallRequest
	if r.Body != nil && r.Body != http.NoBody {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid app update install request")
			return
		}
	}

	dto, err := h.appUpdateProvider.InstallAppUpdate(r.Context(), body)
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("install app update failed", zap.Error(err))
		}
		writeAppError(w, http.StatusConflict, contracts.ErrorCodeAppUpdateInstallFailed, err.Error())
		return
	}
	writeAppUpdateStatus(w, r, dto)
}

func (h *Handler) handleClearDownloadedAppUpdateInstaller(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if !allowsLocalAppUpdate(r) {
		writeAppError(w, http.StatusForbidden, contracts.ErrorCodeAppUpdateRemoteDisabled, "Update Server on its own machine; remote updates are not supported")
		return
	}
	if h.appUpdateProvider == nil {
		writeAppUpdateStatus(w, r, unsupportedAppUpdateStatus())
		return
	}

	dto, err := h.appUpdateProvider.ClearDownloadedAppUpdateInstaller(r.Context())
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("clear downloaded app update installer failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to clear downloaded app update installer")
		return
	}
	writeAppUpdateStatus(w, r, dto)
}

func unsupportedAppUpdateStatus() contracts.AppUpdateStatusDTO {
	return contracts.AppUpdateStatusDTO{
		Supported:    false,
		Status:       "unsupported",
		ReleaseURL:   appupdate.DefaultReleasePageURL,
		Source:       "github-releases",
		ErrorMessage: "app update checker is not configured",
	}
}

// A local update requires a direct loopback request. Forwarded requests and
// ambiguous LAN/hostname connections cannot authorize a server-side installer.
func allowsLocalAppUpdate(r *http.Request) bool {
	peer, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil || !isAppUpdateLoopback(peer) {
		return false
	}
	target, err := url.Parse("http://" + r.Host)
	if err != nil || target.User != nil || !isAppUpdateLoopback(target.Hostname()) {
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
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || !isAppUpdateLoopback(parsed.Hostname()) {
			return false
		}
	}
	return true
}

func isAppUpdateLoopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func writeAppUpdateStatus(w http.ResponseWriter, r *http.Request, dto contracts.AppUpdateStatusDTO) {
	dto.LocalUpdateAllowed = dto.Supported && allowsLocalAppUpdate(r)
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, dto)
}

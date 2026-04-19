package server

import (
	"net/http"

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
		writeJSON(w, http.StatusOK, unsupportedAppUpdateStatus())
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

	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) handleCheckAppUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.appUpdateProvider == nil {
		writeJSON(w, http.StatusOK, unsupportedAppUpdateStatus())
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

	writeJSON(w, http.StatusOK, dto)
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

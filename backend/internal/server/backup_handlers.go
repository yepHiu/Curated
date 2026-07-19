package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"go.uber.org/zap"

	"curated-backend/internal/contracts"
)

func (h *Handler) handleCreateBackup(w http.ResponseWriter, r *http.Request) {
	if h.backupProvider == nil {
		writeAppError(w, http.StatusServiceUnavailable, contracts.ErrorCodeBackupCreateFailed, "backup service is unavailable")
		return
	}
	var body contracts.BackupCreateRequest
	if err := decodeStrictJSONBody(w, r, &body); err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBackupInvalidRequest, "destinationPath must be an absolute path")
		return
	}
	body.DestinationPath = strings.TrimSpace(body.DestinationPath)
	if !isAbsoluteBackupPath(body.DestinationPath) {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBackupInvalidRequest, "destinationPath must be an absolute path")
		return
	}
	manifest, err := h.backupProvider.CreateBackup(r.Context(), body.DestinationPath)
	if errors.Is(err, contracts.ErrBackupDestinationExists) {
		writeAppError(w, http.StatusConflict, contracts.ErrorCodeBackupConflict, "backup destination already exists")
		return
	}
	if err != nil {
		if h.logger != nil {
			h.logger.Error("create backup failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeBackupCreateFailed, "failed to create backup")
		return
	}
	writeJSON(w, http.StatusCreated, manifest)
}

func (h *Handler) handleVerifyBackup(w http.ResponseWriter, r *http.Request) {
	if h.backupProvider == nil {
		writeAppError(w, http.StatusServiceUnavailable, contracts.ErrorCodeBackupVerifyFailed, "backup service is unavailable")
		return
	}
	body, ok := decodeBackupPathRequest(w, r)
	if !ok {
		return
	}
	verification, err := h.backupProvider.VerifyBackup(r.Context(), body.BackupPath)
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("verify backup failed", zap.Error(err))
		}
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBackupVerifyFailed, "failed to read or verify backup")
		return
	}
	writeJSON(w, http.StatusOK, verification)
}

func (h *Handler) handlePreflightBackupRestore(w http.ResponseWriter, r *http.Request) {
	if h.backupProvider == nil {
		writeAppError(w, http.StatusServiceUnavailable, contracts.ErrorCodeBackupPreflightFailed, "backup service is unavailable")
		return
	}
	body, ok := decodeBackupPathRequest(w, r)
	if !ok {
		return
	}
	preflight, err := h.backupProvider.PreflightBackupRestore(r.Context(), body.BackupPath)
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("preflight backup restore failed", zap.Error(err))
		}
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBackupPreflightFailed, "failed to preflight backup restore")
		return
	}
	writeJSON(w, http.StatusOK, preflight)
}

func decodeBackupPathRequest(w http.ResponseWriter, r *http.Request) (contracts.BackupPathRequest, bool) {
	var body contracts.BackupPathRequest
	if err := decodeStrictJSONBody(w, r, &body); err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBackupInvalidRequest, "backupPath must be an absolute path")
		return body, false
	}
	body.BackupPath = strings.TrimSpace(body.BackupPath)
	if !isAbsoluteBackupPath(body.BackupPath) {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBackupInvalidRequest, "backupPath must be an absolute path")
		return body, false
	}
	return body, true
}

func decodeStrictJSONBody(w http.ResponseWriter, r *http.Request, destination any) error {
	defer r.Body.Close()
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("request contains trailing JSON")
		}
		return err
	}
	return nil
}

func isAbsoluteBackupPath(value string) bool {
	return filepath.IsAbs(strings.TrimSpace(value))
}

package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"strings"

	"go.uber.org/zap"

	"curated-backend/internal/contracts"
	"curated-backend/internal/photoscanner"
)

func (h *Handler) handleStartPhotoScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.store == nil || h.tasks == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "photo scan runtime not available")
		return
	}
	if !h.photoLibraryEnabled() {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodePhotoLibraryDisabled, "photo library is disabled")
		return
	}

	var request contracts.StartScanRequest
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid json body")
			return
		}
	}

	paths, err := h.store.ListPhotoLibraryPaths(r.Context())
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("list photo library paths for scan failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to list photo library paths")
		return
	}
	if len(paths) == 0 {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodePhotoPathNotConfigured, "photo library path is not configured")
		return
	}

	scanPaths := paths
	if len(request.Paths) > 0 {
		requestedPaths := make(map[string]struct{}, len(request.Paths))
		for _, raw := range request.Paths {
			cleaned := cleanPhotoScanPath(raw)
			if cleaned != "" {
				requestedPaths[cleaned] = struct{}{}
			}
		}
		if len(requestedPaths) == 0 {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "photo scan path is required")
			return
		}

		scanPaths = make([]contracts.PhotoLibraryPathDTO, 0, len(requestedPaths))
		for _, path := range paths {
			cleaned := cleanPhotoScanPath(path.Path)
			if _, ok := requestedPaths[cleaned]; !ok {
				continue
			}
			scanPaths = append(scanPaths, path)
			delete(requestedPaths, cleaned)
		}
		if len(requestedPaths) > 0 {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodePhotoPathNotFound, "unknown photo library path")
			return
		}
	}

	task, err := h.startPhotoScan(r.Context(), scanPaths)
	if err != nil {
		if errors.Is(err, contracts.ErrScanAlreadyRunning) {
			writeAppError(w, http.StatusConflict, contracts.ErrorCodeConflict, "photo scan already in progress")
			return
		}
		if h.logger != nil {
			h.logger.Warn("start photo scan failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to start photo scan")
		return
	}

	writeJSON(w, http.StatusAccepted, task)
}

func cleanPhotoScanPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	cleaned := filepath.Clean(trimmed)
	if cleaned == "." {
		return ""
	}
	return cleaned
}

func (h *Handler) startPhotoScan(ctx context.Context, paths []contracts.PhotoLibraryPathDTO) (contracts.TaskDTO, error) {
	if h.photoScanStarter != nil {
		return h.photoScanStarter.StartPhotoScan(ctx, paths)
	}
	if h.tasks == nil {
		return contracts.TaskDTO{}, errors.New("photo scan task manager is not available")
	}
	task := h.tasks.Create(contracts.TaskTypeScanPhotos, map[string]any{
		"libraryPathCount": len(paths),
	})
	task = h.tasks.Start(task.TaskID, "Scanning photo books")
	h.saveTaskSnapshot(ctx, task)

	scanPaths := append([]contracts.PhotoLibraryPathDTO(nil), paths...)
	go h.runPhotoScan(task.TaskID, scanPaths)

	return task, nil
}

func (h *Handler) photoLibraryEnabled() bool {
	if h.photoSettingsCtl != nil {
		return h.photoSettingsCtl.PhotoLibraryEnabled()
	}
	return h.cfg.PhotoLibraryEnabled
}

func (h *Handler) runPhotoScan(taskID string, paths []contracts.PhotoLibraryPathDTO) {
	ctx := context.Background()
	summary, err := photoscanner.NewService(h.store).Scan(ctx, paths)
	if err != nil {
		task := h.tasks.Fail(taskID, contracts.ErrorCodePhotoArchiveReadFailed, err.Error())
		h.saveTaskSnapshot(ctx, task)
		return
	}
	metadata := photoScanSummaryMetadata(summary)
	if len(summary.Errors) > 0 {
		task := h.tasks.PartialFail(taskID, contracts.ErrorCodePhotoArchiveReadFailed, "photo scan completed with errors", metadata)
		h.saveTaskSnapshot(ctx, task)
		return
	}
	task := h.tasks.ProgressWithMetadata(taskID, 100, "Photo scan complete", metadata)
	task = h.tasks.Complete(taskID, "Photo scan complete")
	if task.Metadata == nil {
		task.Metadata = metadata
	} else {
		for k, v := range metadata {
			task.Metadata[k] = v
		}
	}
	h.saveTaskSnapshot(ctx, task)
}

func photoScanSummaryMetadata(summary photoscanner.Summary) map[string]any {
	errorItems := make([]map[string]any, 0, len(summary.Errors))
	for _, item := range summary.Errors {
		errorItems = append(errorItems, map[string]any{
			"path":      item.Path,
			"errorCode": item.ErrorCode,
			"message":   item.Message,
		})
	}
	return map[string]any{
		"filesDiscovered": summary.FilesDiscovered,
		"imported":        summary.Imported,
		"updated":         summary.Updated,
		"skipped":         summary.Skipped,
		"errorItems":      errorItems,
	}
}

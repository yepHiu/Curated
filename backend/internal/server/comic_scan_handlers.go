package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"strings"

	"go.uber.org/zap"

	"curated-backend/internal/comicscanner"
	"curated-backend/internal/contracts"
)

func (h *Handler) handleStartComicScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.store == nil || h.tasks == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "comic scan runtime not available")
		return
	}
	if !h.comicLibraryEnabled() {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeComicLibraryDisabled, "comic library is disabled")
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

	paths, err := h.store.ListComicLibraryPaths(r.Context())
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("list comic library paths for scan failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to list comic library paths")
		return
	}
	if len(paths) == 0 {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeComicPathNotConfigured, "comic library path is not configured")
		return
	}

	scanPaths := paths
	if len(request.Paths) > 0 {
		requestedPaths := make(map[string]struct{}, len(request.Paths))
		for _, raw := range request.Paths {
			cleaned := cleanComicScanPath(raw)
			if cleaned != "" {
				requestedPaths[cleaned] = struct{}{}
			}
		}
		if len(requestedPaths) == 0 {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "comic scan path is required")
			return
		}

		scanPaths = make([]contracts.ComicLibraryPathDTO, 0, len(requestedPaths))
		for _, path := range paths {
			cleaned := cleanComicScanPath(path.Path)
			if _, ok := requestedPaths[cleaned]; !ok {
				continue
			}
			scanPaths = append(scanPaths, path)
			delete(requestedPaths, cleaned)
		}
		if len(requestedPaths) > 0 {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeComicPathNotFound, "unknown comic library path")
			return
		}
	}

	task, err := h.startComicScan(r.Context(), scanPaths)
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("start comic scan failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to start comic scan")
		return
	}

	writeJSON(w, http.StatusAccepted, task)
}

func cleanComicScanPath(path string) string {
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

func (h *Handler) startComicScan(ctx context.Context, paths []contracts.ComicLibraryPathDTO) (contracts.TaskDTO, error) {
	if h.comicScanStarter != nil {
		return h.comicScanStarter.StartComicScan(ctx, paths)
	}
	if h.tasks == nil {
		return contracts.TaskDTO{}, errors.New("comic scan task manager is not available")
	}
	task := h.tasks.Create(contracts.TaskTypeScanComics, map[string]any{
		"libraryPathCount": len(paths),
	})
	task = h.tasks.Start(task.TaskID, "Scanning comics")
	h.saveTaskSnapshot(ctx, task)

	scanPaths := append([]contracts.ComicLibraryPathDTO(nil), paths...)
	go h.runComicScan(task.TaskID, scanPaths)

	return task, nil
}

func (h *Handler) comicLibraryEnabled() bool {
	if h.comicSettingsCtl != nil {
		return h.comicSettingsCtl.ComicLibraryEnabled()
	}
	return h.cfg.ComicLibraryEnabled
}

func (h *Handler) runComicScan(taskID string, paths []contracts.ComicLibraryPathDTO) {
	ctx := context.Background()
	summary, err := comicscanner.NewService(h.store).Scan(ctx, paths)
	if err != nil {
		task := h.tasks.Fail(taskID, contracts.ErrorCodeComicArchiveReadFailed, err.Error())
		h.saveTaskSnapshot(ctx, task)
		return
	}
	metadata := comicScanSummaryMetadata(summary)
	if len(summary.Errors) > 0 {
		task := h.tasks.PartialFail(taskID, contracts.ErrorCodeComicArchiveReadFailed, "comic scan completed with errors", metadata)
		h.saveTaskSnapshot(ctx, task)
		return
	}
	task := h.tasks.ProgressWithMetadata(taskID, 100, "Comic scan complete", metadata)
	task = h.tasks.Complete(taskID, "Comic scan complete")
	if task.Metadata == nil {
		task.Metadata = metadata
	} else {
		for k, v := range metadata {
			task.Metadata[k] = v
		}
	}
	h.saveTaskSnapshot(ctx, task)
}

func comicScanSummaryMetadata(summary comicscanner.Summary) map[string]any {
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

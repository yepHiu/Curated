package server

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

func (h *Handler) handleImportComics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.store == nil || h.tasks == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "comic import runtime not available")
		return
	}
	if !h.comicLibraryEnabled() {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeComicLibraryDisabled, "comic library is disabled")
		return
	}

	targetID := h.defaultComicImportLibraryPathID()
	if targetID == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeComicImportTargetMissing, "default comic import library path is not configured")
		return
	}
	target, err := h.store.GetComicLibraryPath(r.Context(), targetID)
	if err != nil {
		if errors.Is(err, storage.ErrComicLibraryPathNotFound) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeComicPathNotFound, "default comic import library path was not found")
			return
		}
		if h.logger != nil {
			h.logger.Warn("load default comic import library path failed", zap.Error(err), zap.String("libraryPathId", targetID))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load default comic import library path")
		return
	}
	targetRoot := filepath.Clean(strings.TrimSpace(target.Path))
	if targetRoot == "" || targetRoot == "." {
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodeComicImportTargetMissing, "default comic import library path is invalid")
		return
	}
	if stat, err := os.Stat(targetRoot); err != nil || !stat.IsDir() {
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodeComicImportTargetMissing, "default comic import library path is unavailable")
		return
	}
	target.Path = targetRoot

	reader, err := r.MultipartReader()
	if err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "multipart form data is required")
		return
	}

	task := h.tasks.Create(contracts.TaskTypeImportComics, map[string]any{
		"targetComicLibraryPathId": target.ID,
		"targetPath":               targetRoot,
		"stage":                    "copying",
		"completedFiles":           0,
		"failedFiles":              0,
	})
	task = h.tasks.Start(task.TaskID, "Importing comics")
	h.saveTaskSnapshot(r.Context(), task)
	snapshotThrottle := newImportTaskSnapshotThrottle(time.Now(), importTaskSnapshotMinInterval, importTaskSnapshotMinBytes)

	var (
		totalFiles      int
		completedFiles  int
		failedFiles     int
		copiedBytes     int64
		declaredBytes   int64
		pendingRelPath  string
		errorItems      []map[string]any
		copiedAny       bool
		lastCurrentName string
		conflictAny     bool
		unsupportedAny  bool
		copyFailureAny  bool
	)

	progressPatch := func(extra map[string]any) map[string]any {
		patch := map[string]any{
			"targetComicLibraryPathId": target.ID,
			"targetPath":               targetRoot,
			"stage":                    "copying",
			"totalFiles":               totalFiles,
			"completedFiles":           completedFiles,
			"failedFiles":              failedFiles,
			"copiedBytes":              copiedBytes,
			"totalBytes":               declaredBytes,
			"currentFileName":          lastCurrentName,
			"errorItems":               errorItems,
		}
		for k, v := range extra {
			patch[k] = v
		}
		return patch
	}

	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			task = h.tasks.ProgressWithMetadata(task.TaskID, task.Progress, "Failed to read comic import payload", progressPatch(nil))
			h.saveTaskSnapshot(r.Context(), task)
			task = h.tasks.Fail(task.TaskID, contracts.ErrorCodeImportSourceUnavailable, "failed to read comic import payload")
			h.saveTaskSnapshot(r.Context(), task)
			writeJSON(w, http.StatusAccepted, task)
			return
		}

		formName := part.FormName()
		fileName := strings.TrimSpace(part.FileName())
		if fileName == "" {
			value, _ := io.ReadAll(io.LimitReader(part, 64*1024))
			switch formName {
			case "relativePath":
				pendingRelPath = strings.TrimSpace(string(value))
			case "totalBytes":
				if n, err := strconv.ParseInt(strings.TrimSpace(string(value)), 10, 64); err == nil && n > 0 {
					declaredBytes = n
				}
			}
			_ = part.Close()
			continue
		}

		totalFiles++
		lastCurrentName = fileName
		relPath := sanitizeImportRelativePath(pendingRelPath, fileName)
		pendingRelPath = ""
		if !isSupportedImportComicArchivePath(relPath) {
			failedFiles++
			unsupportedAny = true
			errorItems = append(errorItems, importErrorItem(fileName, contracts.ErrorCodeComicArchiveUnsupported, "unsupported comic archive type"))
			_, _ = io.Copy(io.Discard, part)
			_ = part.Close()
			task = h.tasks.ProgressWithMetadata(task.TaskID, importProgressPercent(copiedBytes, declaredBytes), "Skipped unsupported comic archive "+fileName, progressPatch(nil))
			h.saveTaskSnapshot(r.Context(), task)
			continue
		}

		destPath, err := importDestinationPath(targetRoot, relPath)
		if err != nil {
			failedFiles++
			copyFailureAny = true
			errorItems = append(errorItems, importErrorItem(fileName, contracts.ErrorCodeImportSourceUnavailable, "invalid destination path"))
			_, _ = io.Copy(io.Discard, part)
			_ = part.Close()
			task = h.tasks.ProgressWithMetadata(task.TaskID, importProgressPercent(copiedBytes, declaredBytes), "Skipped invalid comic archive path "+fileName, progressPatch(nil))
			h.saveTaskSnapshot(r.Context(), task)
			continue
		}

		if _, err := os.Stat(destPath); err == nil {
			failedFiles++
			conflictAny = true
			errorItems = append(errorItems, importErrorItem(fileName, contracts.ErrorCodeComicImportConflict, "target comic archive already exists"))
			_, _ = io.Copy(io.Discard, part)
			_ = part.Close()
			task = h.tasks.ProgressWithMetadata(task.TaskID, importProgressPercent(copiedBytes, declaredBytes), "Skipped existing comic archive "+fileName, progressPatch(nil))
			h.saveTaskSnapshot(r.Context(), task)
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			failedFiles++
			copyFailureAny = true
			code := classifyImportCopyError(err)
			errorItems = append(errorItems, importErrorItem(fileName, code, err.Error()))
			_, _ = io.Copy(io.Discard, part)
			_ = part.Close()
			task = h.tasks.ProgressWithMetadata(task.TaskID, importProgressPercent(copiedBytes, declaredBytes), "Failed to prepare comic archive "+fileName, progressPatch(nil))
			h.saveTaskSnapshot(r.Context(), task)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			failedFiles++
			copyFailureAny = true
			code := classifyImportCopyError(err)
			errorItems = append(errorItems, importErrorItem(fileName, code, err.Error()))
			_, _ = io.Copy(io.Discard, part)
			_ = part.Close()
			task = h.tasks.ProgressWithMetadata(task.TaskID, importProgressPercent(copiedBytes, declaredBytes), "Failed to prepare comic archive "+fileName, progressPatch(nil))
			h.saveTaskSnapshot(r.Context(), task)
			continue
		}

		tempPath := destPath + "." + task.TaskID + ".tmp"
		n, err := copyImportPartToFile(tempPath, part, func(delta int64) {
			copiedBytes += delta
			task = h.tasks.ProgressWithMetadata(task.TaskID, importProgressPercent(copiedBytes, declaredBytes), "Copying comic archive "+fileName, progressPatch(nil))
			if snapshotThrottle.ShouldSave(time.Now(), copiedBytes, false) {
				h.saveTaskSnapshot(r.Context(), task)
			}
		})
		_ = part.Close()
		if err != nil {
			_ = os.Remove(tempPath)
			failedFiles++
			copyFailureAny = true
			code := classifyImportCopyError(err)
			errorItems = append(errorItems, importErrorItem(fileName, code, err.Error()))
			task = h.tasks.ProgressWithMetadata(task.TaskID, importProgressPercent(copiedBytes, declaredBytes), "Failed to copy comic archive "+fileName, progressPatch(nil))
			h.saveTaskSnapshot(r.Context(), task)
			continue
		}
		if err := os.Rename(tempPath, destPath); err != nil {
			_ = os.Remove(tempPath)
			copiedBytes -= n
			if copiedBytes < 0 {
				copiedBytes = 0
			}
			failedFiles++
			copyFailureAny = true
			code := classifyImportCopyError(err)
			errorItems = append(errorItems, importErrorItem(fileName, code, err.Error()))
			task = h.tasks.ProgressWithMetadata(task.TaskID, importProgressPercent(copiedBytes, declaredBytes), "Failed to finish comic archive "+fileName, progressPatch(nil))
			h.saveTaskSnapshot(r.Context(), task)
			continue
		}
		completedFiles++
		copiedAny = true
		task = h.tasks.ProgressWithMetadata(task.TaskID, importProgressPercent(copiedBytes, declaredBytes), "Copied comic archive "+fileName, progressPatch(nil))
		h.saveTaskSnapshot(r.Context(), task)
	}

	finalPatch := progressPatch(map[string]any{"stage": "completed"})
	if totalFiles == 0 {
		finalPatch["stage"] = "failed"
		task = h.tasks.ProgressWithMetadata(task.TaskID, 100, "No comic archives were provided", finalPatch)
		h.saveTaskSnapshot(r.Context(), task)
		task = h.tasks.Fail(task.TaskID, contracts.ErrorCodeBadRequest, "no comic archives were provided")
		h.saveTaskSnapshot(r.Context(), task)
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "no comic archives were provided")
		return
	}

	var scanErr error
	if copiedAny {
		var scanTask contracts.TaskDTO
		scanTask, scanErr = h.startComicScan(r.Context(), []contracts.ComicLibraryPathDTO{target})
		if scanErr == nil && scanTask.TaskID != "" {
			finalPatch["scanTaskId"] = scanTask.TaskID
		}
	}
	if scanErr != nil {
		finalPatch["scanError"] = scanErr.Error()
		if failedFiles == 0 {
			finalPatch["stage"] = "partial_failed"
			task = h.tasks.ProgressWithMetadata(task.TaskID, 100, "Comics copied, but scan could not start", finalPatch)
			h.saveTaskSnapshot(r.Context(), task)
			task = h.tasks.PartialFail(task.TaskID, contracts.ErrorCodeImportScanFailed, "comics copied, but scan could not start", finalPatch)
			h.saveTaskSnapshot(r.Context(), task)
			writeJSON(w, http.StatusAccepted, task)
			return
		}
	}

	if failedFiles > 0 {
		finalPatch["stage"] = "partial_failed"
		code := comicImportFailureCode(conflictAny, unsupportedAny, copyFailureAny)
		if completedFiles == 0 {
			finalPatch["stage"] = "failed"
			task = h.tasks.ProgressWithMetadata(task.TaskID, 100, "Comic import failed", finalPatch)
			h.saveTaskSnapshot(r.Context(), task)
			task = h.tasks.Fail(task.TaskID, code, "comic import failed")
			h.saveTaskSnapshot(r.Context(), task)
			writeJSON(w, http.StatusAccepted, task)
			return
		}
		task = h.tasks.PartialFail(task.TaskID, code, "comic import partially completed", finalPatch)
		h.saveTaskSnapshot(r.Context(), task)
		writeJSON(w, http.StatusAccepted, task)
		return
	}

	task = h.tasks.ProgressWithMetadata(task.TaskID, 100, "Comic import completed", finalPatch)
	h.saveTaskSnapshot(r.Context(), task)
	task = h.tasks.Complete(task.TaskID, "Comic import completed")
	h.saveTaskSnapshot(r.Context(), task)
	writeJSON(w, http.StatusAccepted, task)
}

func (h *Handler) defaultComicImportLibraryPathID() string {
	if h.comicSettingsCtl != nil {
		return strings.TrimSpace(h.comicSettingsCtl.DefaultComicImportLibraryPathID())
	}
	return strings.TrimSpace(h.cfg.DefaultComicImportLibraryPathID)
}

func isSupportedImportComicArchivePath(path string) bool {
	switch strings.ToLower(filepath.Ext(strings.TrimSpace(path))) {
	case ".zip", ".cbz":
		return true
	default:
		return false
	}
}

func comicImportFailureCode(conflictAny bool, unsupportedAny bool, copyFailureAny bool) string {
	if conflictAny {
		return contracts.ErrorCodeComicImportConflict
	}
	if unsupportedAny && !copyFailureAny {
		return contracts.ErrorCodeComicArchiveUnsupported
	}
	return contracts.ErrorCodeImportCopyFailed
}

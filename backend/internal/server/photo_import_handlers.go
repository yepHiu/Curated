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

// handleImportPhotos copies ZIP/CBZ uploads into the independent photo root, then queues its scan.
func (h *Handler) handleImportPhotos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.store == nil || h.tasks == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "photo import runtime not available")
		return
	}
	if !h.photoLibraryEnabled() {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodePhotoLibraryDisabled, "photo library is disabled")
		return
	}

	targetID := h.defaultPhotoImportLibraryPathID()
	if targetID == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodePhotoImportTargetMissing, "default photo import library path is not configured")
		return
	}
	target, err := h.store.GetPhotoLibraryPath(r.Context(), targetID)
	if err != nil {
		if errors.Is(err, storage.ErrPhotoLibraryPathNotFound) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodePhotoPathNotFound, "default photo import library path was not found")
			return
		}
		if h.logger != nil {
			h.logger.Warn("load default photo import library path failed", zap.Error(err), zap.String("libraryPathId", targetID))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load default photo import library path")
		return
	}
	targetRoot := filepath.Clean(strings.TrimSpace(target.Path))
	if targetRoot == "" || targetRoot == "." {
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodePhotoImportTargetMissing, "default photo import library path is invalid")
		return
	}
	if stat, err := os.Stat(targetRoot); err != nil || !stat.IsDir() {
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodePhotoImportTargetMissing, "default photo import library path is unavailable")
		return
	}
	target.Path = targetRoot
	root, err := os.OpenRoot(targetRoot)
	if err != nil {
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodePhotoImportTargetMissing, "photo import target is unavailable")
		return
	}
	defer root.Close()

	reader, err := r.MultipartReader()
	if err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "multipart form data is required")
		return
	}

	task := h.tasks.Create(contracts.TaskTypeImportPhotos, map[string]any{
		"targetPhotoLibraryPathId": target.ID,
		"targetPath":               targetRoot,
		"stage":                    "copying",
		"completedFiles":           0,
		"failedFiles":              0,
	})
	task = h.tasks.Start(task.TaskID, "Importing photos")
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
			"targetPhotoLibraryPathId": target.ID,
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
			task = h.tasks.ProgressWithMetadata(task.TaskID, task.Progress, "Failed to read photo import payload", progressPatch(nil))
			h.saveTaskSnapshot(r.Context(), task)
			task = h.tasks.Fail(task.TaskID, contracts.ErrorCodeImportSourceUnavailable, "failed to read photo import payload")
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
		if !isSupportedImportPhotoArchivePath(relPath) {
			failedFiles++
			unsupportedAny = true
			errorItems = append(errorItems, importErrorItem(fileName, contracts.ErrorCodePhotoArchiveUnsupported, "unsupported photo archive type"))
			_, _ = io.Copy(io.Discard, part)
			_ = part.Close()
			task = h.tasks.ProgressWithMetadata(task.TaskID, importProgressPercent(copiedBytes, declaredBytes), "Skipped unsupported photo archive "+fileName, progressPatch(nil))
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
			task = h.tasks.ProgressWithMetadata(task.TaskID, importProgressPercent(copiedBytes, declaredBytes), "Skipped invalid photo archive path "+fileName, progressPatch(nil))
			h.saveTaskSnapshot(r.Context(), task)
			continue
		}

		relativeDestination, err := filepath.Rel(targetRoot, destPath)
		if err != nil {
			relativeDestination = "../invalid"
		}
		if _, err := root.Stat(relativeDestination); err == nil {
			failedFiles++
			conflictAny = true
			errorItems = append(errorItems, importErrorItem(fileName, contracts.ErrorCodePhotoImportConflict, "target photo archive already exists"))
			_, _ = io.Copy(io.Discard, part)
			_ = part.Close()
			task = h.tasks.ProgressWithMetadata(task.TaskID, importProgressPercent(copiedBytes, declaredBytes), "Skipped existing photo archive "+fileName, progressPatch(nil))
			h.saveTaskSnapshot(r.Context(), task)
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			failedFiles++
			copyFailureAny = true
			code := classifyImportCopyError(err)
			errorItems = append(errorItems, importErrorItem(fileName, code, err.Error()))
			_, _ = io.Copy(io.Discard, part)
			_ = part.Close()
			task = h.tasks.ProgressWithMetadata(task.TaskID, importProgressPercent(copiedBytes, declaredBytes), "Failed to prepare photo archive "+fileName, progressPatch(nil))
			h.saveTaskSnapshot(r.Context(), task)
			continue
		}

		if err := root.MkdirAll(filepath.Dir(relativeDestination), 0o755); err != nil {
			failedFiles++
			copyFailureAny = true
			code := classifyImportCopyError(err)
			errorItems = append(errorItems, importErrorItem(fileName, code, err.Error()))
			_, _ = io.Copy(io.Discard, part)
			_ = part.Close()
			task = h.tasks.ProgressWithMetadata(task.TaskID, importProgressPercent(copiedBytes, declaredBytes), "Failed to prepare photo archive "+fileName, progressPatch(nil))
			h.saveTaskSnapshot(r.Context(), task)
			continue
		}

		tempPath := relativeDestination + "." + task.TaskID + ".tmp"
		n, err := copyPhotoImportPart(root, tempPath, part, func(delta int64) {
			copiedBytes += delta
			task = h.tasks.ProgressWithMetadata(task.TaskID, importProgressPercent(copiedBytes, declaredBytes), "Copying photo archive "+fileName, progressPatch(nil))
			if snapshotThrottle.ShouldSave(time.Now(), copiedBytes, false) {
				h.saveTaskSnapshot(r.Context(), task)
			}
		})
		_ = part.Close()
		if err != nil {
			_ = root.Remove(tempPath)
			failedFiles++
			copyFailureAny = true
			code := classifyImportCopyError(err)
			errorItems = append(errorItems, importErrorItem(fileName, code, err.Error()))
			task = h.tasks.ProgressWithMetadata(task.TaskID, importProgressPercent(copiedBytes, declaredBytes), "Failed to copy photo archive "+fileName, progressPatch(nil))
			h.saveTaskSnapshot(r.Context(), task)
			continue
		}
		if err := publishPhotoImport(root, tempPath, relativeDestination); err != nil {
			_ = root.Remove(tempPath)
			copiedBytes -= n
			if copiedBytes < 0 {
				copiedBytes = 0
			}
			failedFiles++
			copyFailureAny = true
			code := classifyImportCopyError(err)
			errorItems = append(errorItems, importErrorItem(fileName, code, err.Error()))
			task = h.tasks.ProgressWithMetadata(task.TaskID, importProgressPercent(copiedBytes, declaredBytes), "Failed to finish photo archive "+fileName, progressPatch(nil))
			h.saveTaskSnapshot(r.Context(), task)
			continue
		}
		completedFiles++
		copiedAny = true
		task = h.tasks.ProgressWithMetadata(task.TaskID, importProgressPercent(copiedBytes, declaredBytes), "Copied photo archive "+fileName, progressPatch(nil))
		h.saveTaskSnapshot(r.Context(), task)
	}

	finalPatch := progressPatch(map[string]any{"stage": "completed"})
	if totalFiles == 0 {
		finalPatch["stage"] = "failed"
		task = h.tasks.ProgressWithMetadata(task.TaskID, 100, "No photo archives were provided", finalPatch)
		h.saveTaskSnapshot(r.Context(), task)
		task = h.tasks.Fail(task.TaskID, contracts.ErrorCodeBadRequest, "no photo archives were provided")
		h.saveTaskSnapshot(r.Context(), task)
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "no photo archives were provided")
		return
	}

	var scanErr error
	if copiedAny {
		var scanTask contracts.TaskDTO
		scanTask, scanErr = h.startPhotoScan(r.Context(), []contracts.PhotoLibraryPathDTO{target})
		if scanErr == nil && scanTask.TaskID != "" {
			finalPatch["scanTaskId"] = scanTask.TaskID
		}
	}
	if scanErr != nil {
		finalPatch["scanError"] = scanErr.Error()
		if failedFiles == 0 {
			finalPatch["stage"] = "partial_failed"
			task = h.tasks.ProgressWithMetadata(task.TaskID, 100, "Photos copied, but scan could not start", finalPatch)
			h.saveTaskSnapshot(r.Context(), task)
			task = h.tasks.PartialFail(task.TaskID, contracts.ErrorCodeImportScanFailed, "photos copied, but scan could not start", finalPatch)
			h.saveTaskSnapshot(r.Context(), task)
			writeJSON(w, http.StatusAccepted, task)
			return
		}
	}

	if failedFiles > 0 {
		finalPatch["stage"] = "partial_failed"
		code := photoImportFailureCode(conflictAny, unsupportedAny, copyFailureAny)
		if completedFiles == 0 {
			finalPatch["stage"] = "failed"
			task = h.tasks.ProgressWithMetadata(task.TaskID, 100, "Photo import failed", finalPatch)
			h.saveTaskSnapshot(r.Context(), task)
			task = h.tasks.Fail(task.TaskID, code, "photo import failed")
			h.saveTaskSnapshot(r.Context(), task)
			writeJSON(w, http.StatusAccepted, task)
			return
		}
		task = h.tasks.PartialFail(task.TaskID, code, "photo import partially completed", finalPatch)
		h.saveTaskSnapshot(r.Context(), task)
		writeJSON(w, http.StatusAccepted, task)
		return
	}

	task = h.tasks.ProgressWithMetadata(task.TaskID, 100, "Photo import completed", finalPatch)
	h.saveTaskSnapshot(r.Context(), task)
	task = h.tasks.Complete(task.TaskID, "Photo import completed")
	h.saveTaskSnapshot(r.Context(), task)
	writeJSON(w, http.StatusAccepted, task)
}

// defaultPhotoImportLibraryPathID reads the persisted photo target only.
func (h *Handler) defaultPhotoImportLibraryPathID() string {
	if h.photoSettingsCtl != nil {
		return strings.TrimSpace(h.photoSettingsCtl.DefaultPhotoImportLibraryPathID())
	}
	return strings.TrimSpace(h.cfg.DefaultPhotoImportLibraryPathID)
}

// isSupportedImportPhotoArchivePath mirrors photo scanner archive support.
func isSupportedImportPhotoArchivePath(path string) bool {
	switch strings.ToLower(filepath.Ext(strings.TrimSpace(path))) {
	case ".zip", ".cbz":
		return true
	default:
		return false
	}
}

// photoImportFailureCode preserves the actionable reason for partial imports.
func photoImportFailureCode(conflictAny bool, unsupportedAny bool, copyFailureAny bool) string {
	if conflictAny {
		return contracts.ErrorCodePhotoImportConflict
	}
	if unsupportedAny && !copyFailureAny {
		return contracts.ErrorCodePhotoArchiveUnsupported
	}
	return contracts.ErrorCodeImportCopyFailed
}

// copyPhotoImportPart confines temporary writes to the opened target root.
func copyPhotoImportPart(root *os.Root, name string, source io.Reader, progress func(int64)) (int64, error) {
	file, err := root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, err
	}
	var total int64
	buffer := make([]byte, 128*1024)
	for {
		n, readErr := source.Read(buffer)
		if n > 0 {
			written, writeErr := file.Write(buffer[:n])
			total += int64(written)
			progress(int64(written))
			if writeErr != nil {
				_ = file.Close()
				return total, writeErr
			}
			if written != n {
				_ = file.Close()
				return total, io.ErrShortWrite
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			_ = file.Close()
			return total, readErr
		}
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return total, err
	}
	return total, file.Close()
}

// publishPhotoImport never replaces an existing archive, including a concurrent import.
// O_EXCL fallback supports filesystems without hard links while keeping root confinement.
func publishPhotoImport(root *os.Root, temporary, destination string) error {
	if err := root.Link(temporary, destination); err == nil {
		_ = root.Remove(temporary)
		return nil
	} else if errors.Is(err, os.ErrExist) {
		return err
	}
	src, err := root.Open(temporary)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := root.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(dst, src)
	if copyErr == nil {
		copyErr = dst.Sync()
	}
	closeErr := dst.Close()
	if copyErr == nil {
		copyErr = closeErr
	}
	if copyErr != nil {
		_ = root.Remove(destination)
		return copyErr
	}
	_ = root.Remove(temporary)
	return nil
}

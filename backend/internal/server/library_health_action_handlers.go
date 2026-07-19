package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"go.uber.org/zap"

	"curated-backend/internal/contracts"
)

const libraryHealthActionMaxFindings = 100

var libraryHealthCleanupActions = map[string]string{
	"cleanup_orphan_state":   "orphan_user_state",
	"cleanup_import_staging": "import_staging_residue",
}

func (h *Handler) handleStartLibraryHealthAction(w http.ResponseWriter, r *http.Request) {
	if h.store == nil || h.tasks == nil || h.importUploads == nil || h.libraryPathStorageStatus == nil {
		writeAppError(w, http.StatusServiceUnavailable, contracts.ErrorCodeInternal, "library health action service is unavailable")
		return
	}
	var body contracts.StartLibraryHealthActionRequest
	if err := decodeStrictJSONBodyLimit(w, r, &body, libraryHealthRepairBodyLimit); err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid library health action request")
		return
	}
	body.Action = strings.TrimSpace(body.Action)
	expectedCategory, supported := libraryHealthCleanupActions[body.Action]
	if !supported {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "unsupported library health action")
		return
	}
	if !body.Confirm {
		writeAppError(w, http.StatusConflict, contracts.ErrorCodeHealthRepairConfirmationRequired, "cleanup requires explicit confirmation")
		return
	}
	findingIDs, err := normalizeExactFindingIDs(body.FindingIDs)
	if err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, err.Error())
		return
	}

	report, err := h.scanLibraryHealth(r.Context(), libraryHealthMaxFindingLimit)
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to refresh library health before cleanup")
		return
	}
	findingsByID := make(map[string]contracts.LibraryHealthFindingDTO, len(report.Findings))
	for _, finding := range report.Findings {
		findingsByID[finding.ID] = finding
	}
	selected := make([]contracts.LibraryHealthFindingDTO, 0, len(findingIDs))
	for _, id := range findingIDs {
		finding, ok := findingsByID[id]
		if !ok || finding.Category != expectedCategory || !slices.Contains(finding.RepairActions, body.Action) {
			writeAppError(w, http.StatusConflict, contracts.ErrorCodeHealthRepairNoFindings, "one or more cleanup findings are stale or no longer actionable")
			return
		}
		selected = append(selected, finding)
	}

	task := h.tasks.Create(contracts.TaskTypeLibraryHealthCleanup, map[string]any{
		"action":         body.Action,
		"totalItems":     len(selected),
		"completedItems": 0,
		"removedItems":   0,
		"skippedItems":   0,
		"failedItems":    0,
		"results":        []map[string]any{},
	})
	task = h.tasks.Start(task.TaskID, "Running confirmed library health cleanup")
	if err := h.store.SaveTask(r.Context(), task); err != nil {
		h.tasks.Fail(task.TaskID, contracts.ErrorCodeHealthRepairPersistFailed, "failed to persist cleanup task")
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeHealthRepairPersistFailed, "failed to persist cleanup task")
		return
	}
	go h.runLibraryHealthCleanup(task.TaskID, body.Action, selected)
	writeJSON(w, http.StatusAccepted, task)
}

func normalizeExactFindingIDs(input []string) ([]string, error) {
	if len(input) == 0 || len(input) > libraryHealthActionMaxFindings {
		return nil, fmt.Errorf("findingIds must contain between 1 and 100 entries")
	}
	seen := make(map[string]struct{}, len(input))
	out := make([]string, 0, len(input))
	for _, raw := range input {
		id := strings.TrimSpace(raw)
		if id == "" {
			return nil, fmt.Errorf("findingIds cannot contain empty values")
		}
		if _, duplicate := seen[id]; duplicate {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out, nil
}

func (h *Handler) runLibraryHealthCleanup(taskID, action string, findings []contracts.LibraryHealthFindingDTO) {
	ctx := h.runtimeContext
	if ctx == nil {
		ctx = context.Background()
	}
	removedCount := 0
	skippedCount := 0
	failedCount := 0
	results := make([]map[string]any, 0, len(findings))
	for index, finding := range findings {
		if err := ctx.Err(); err != nil {
			failedCount += len(findings) - index
			results = append(results, map[string]any{
				"findingId": finding.ID, "status": "failed", "error": "backend stopped before cleanup completed",
			})
			break
		}
		removed, err := h.executeLibraryHealthCleanup(ctx, taskID, action, finding)
		status := "removed"
		message := "cleanup completed"
		switch {
		case err != nil:
			status = "failed"
			message = err.Error()
			failedCount++
		case !removed:
			status = "skipped"
			message = "finding no longer required cleanup"
			skippedCount++
		default:
			removedCount++
		}
		results = append(results, map[string]any{
			"findingId":  finding.ID,
			"category":   finding.Category,
			"entityType": finding.EntityType,
			"entityId":   finding.EntityID,
			"label":      finding.Label,
			"path":       finding.Path,
			"status":     status,
			"message":    message,
		})
		progress := (index + 1) * 100 / len(findings)
		task := h.tasks.ProgressWithMetadata(taskID, progress, "Cleaning confirmed library health findings", map[string]any{
			"completedItems": index + 1,
			"removedItems":   removedCount,
			"skippedItems":   skippedCount,
			"failedItems":    failedCount,
			"results":        results,
		})
		_ = h.store.SaveTask(ctx, task)
	}

	var task contracts.TaskDTO
	finalMetadata := map[string]any{
		"completedItems": len(results),
		"removedItems":   removedCount,
		"skippedItems":   skippedCount,
		"failedItems":    failedCount,
		"results":        results,
	}
	switch {
	case failedCount == 0 && skippedCount == 0:
		task = h.tasks.ProgressWithMetadata(taskID, 100, "Library health cleanup completed", finalMetadata)
		task = h.tasks.Complete(taskID, "Library health cleanup completed")
	case removedCount > 0:
		task = h.tasks.PartialFail(taskID, contracts.ErrorCodeHealthCleanupFailed, "Library health cleanup completed with skipped or failed items", finalMetadata)
	default:
		task = h.tasks.ProgressWithMetadata(taskID, 100, "Library health cleanup failed", finalMetadata)
		task = h.tasks.Fail(taskID, contracts.ErrorCodeHealthCleanupFailed, "Library health cleanup failed")
	}
	_ = h.store.SaveTask(context.WithoutCancel(ctx), task)
}

func (h *Handler) executeLibraryHealthCleanup(
	ctx context.Context, taskID, action string, finding contracts.LibraryHealthFindingDTO,
) (bool, error) {
	switch action {
	case "cleanup_orphan_state":
		return h.store.CleanupOrphanUserState(ctx, taskID, finding.ID, finding.EntityType, finding.EntityID, time.Now())
	case "cleanup_import_staging":
		return h.cleanupConfirmedImportStaging(ctx, taskID, finding)
	default:
		return false, fmt.Errorf("unsupported cleanup action %q", action)
	}
}

func (h *Handler) cleanupConfirmedImportStaging(
	ctx context.Context, taskID string, finding contracts.LibraryHealthFindingDTO,
) (bool, error) {
	candidate := filepath.Clean(strings.TrimSpace(finding.Path))
	uploadID := strings.TrimSpace(finding.EntityID)
	if candidate == "." || !filepath.IsAbs(candidate) || !isStrictMovieImportUploadID(uploadID) || filepath.Base(candidate) != uploadID {
		return false, errors.New("staging finding failed strict identity validation")
	}
	sessionIDs, err := h.store.ListMovieImportUploadSessionIDs(ctx)
	if err != nil {
		return false, err
	}
	if _, registered := sessionIDs[uploadID]; registered {
		return false, errors.New("staging directory now belongs to a registered upload session")
	}
	libraryPaths, err := h.store.ListLibraryPaths(ctx)
	if err != nil {
		return false, err
	}
	root := ""
	for _, libraryPath := range libraryPaths {
		expected := filepath.Join(filepath.Clean(libraryPath.Path), movieImportUploadStagingDirName, uploadID)
		if sameUploadRuntimePath(candidate, expected) {
			root = filepath.Clean(libraryPath.Path)
			break
		}
	}
	if root == "" {
		return false, errors.New("staging directory is outside configured library roots")
	}
	rootInfo, err := os.Stat(root)
	if err != nil || !rootInfo.IsDir() {
		return false, errors.New("target storage is unavailable; cleanup deferred")
	}
	stagingRoot := filepath.Join(root, movieImportUploadStagingDirName)
	stagingRootInfo, err := os.Lstat(stagingRoot)
	if err != nil || stagingRootInfo.Mode()&os.ModeSymlink != 0 || !stagingRootInfo.IsDir() {
		return false, errors.New("staging root is missing, invalid, or a symlink")
	}
	if !uploadRuntimePathDescendsFrom(candidate, stagingRoot) {
		return false, errors.New("staging directory failed descendant validation")
	}
	info, err := os.Lstat(candidate)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return false, errors.New("staging candidate is invalid or a symlink")
	}
	auditID, err := h.store.CreateMovieImportUploadCleanupAudit(
		ctx, uploadID, "health_confirmed_orphan_staging", "orphan", candidate,
		fmt.Sprintf("confirmed from finding %s by task %s", finding.ID, taskID), time.Now(),
	)
	if err != nil {
		return false, err
	}
	if err := removeMovieImportUploadStagingDirectory(candidate); err != nil {
		_ = h.store.CompleteMovieImportUploadCleanupAudit(ctx, auditID, "failed", err.Error(), time.Now())
		return false, err
	}
	if err := h.store.CompleteMovieImportUploadCleanupAudit(ctx, auditID, "removed", "confirmed orphan staging directory removed", time.Now()); err != nil {
		return false, err
	}
	if h.logger != nil {
		h.logger.Info("confirmed library health staging cleanup completed", zap.String("taskId", taskID), zap.String("uploadId", uploadID))
	}
	return true, nil
}

package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"go.uber.org/zap"

	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

const (
	libraryHealthRepairDefaultLimit = 25
	libraryHealthRepairMaxLimit     = 100
	libraryHealthRepairBodyLimit    = 64 << 10
)

var libraryHealthMetadataRepairCategories = map[string]struct{}{
	"metadata_missing": {},
	"metadata_failed":  {},
}

func (h *Handler) handleStartLibraryHealthRepair(w http.ResponseWriter, r *http.Request) {
	if h.store == nil || h.tasks == nil || h.movieMetadataRefresher == nil || h.libraryPathStorageStatus == nil {
		writeAppError(w, http.StatusServiceUnavailable, contracts.ErrorCodeInternal, "library health repair service is unavailable")
		return
	}
	var body contracts.StartLibraryHealthRepairRequest
	if err := decodeStrictJSONBodyLimit(w, r, &body, libraryHealthRepairBodyLimit); err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid library health repair request")
		return
	}
	body.Action = strings.TrimSpace(body.Action)
	if body.Action != "rescrape_metadata" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "action must be rescrape_metadata")
		return
	}
	if !body.Confirm {
		writeAppError(w, http.StatusConflict, contracts.ErrorCodeHealthRepairConfirmationRequired, "repair requires explicit confirmation")
		return
	}
	limit := body.Limit
	if limit == 0 {
		limit = libraryHealthRepairDefaultLimit
	}
	if limit < 1 || limit > libraryHealthRepairMaxLimit {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "limit must be between 1 and 100")
		return
	}
	categories, err := normalizeLibraryHealthRepairCategories(body.Categories)
	if err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, err.Error())
		return
	}
	if len(body.FindingIDs) > libraryHealthRepairMaxLimit {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "findingIds cannot contain more than 100 entries")
		return
	}

	report, err := h.scanLibraryHealth(r.Context(), libraryHealthMaxFindingLimit)
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to refresh library health before repair")
		return
	}
	items := selectLibraryHealthMetadataRepairItems(report, categories, body.FindingIDs, limit)
	if len(items) == 0 {
		writeAppError(w, http.StatusConflict, contracts.ErrorCodeHealthRepairNoFindings, "no matching metadata repair findings remain")
		return
	}

	repairID, err := newLibraryHealthRepairID()
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to allocate repair id")
		return
	}
	createdAt := time.Now().UTC()
	task := h.tasks.Create(contracts.TaskTypeLibraryHealthRepair, map[string]any{
		"repairId":       repairID,
		"action":         body.Action,
		"categories":     categories,
		"totalItems":     len(items),
		"completedItems": 0,
		"succeededItems": 0,
		"failedItems":    0,
	})
	categoriesJSON, _ := json.Marshal(categories)
	run := storage.LibraryHealthRepairRun{
		RepairID: repairID, TaskID: task.TaskID, Action: body.Action,
		CategoriesJSON: string(categoriesJSON), Status: contracts.TaskPending,
		CreatedAt: createdAt.Format(time.RFC3339), Items: items,
	}
	if err := h.store.CreateLibraryHealthRepairRun(r.Context(), run); err != nil {
		h.tasks.Fail(task.TaskID, contracts.ErrorCodeHealthRepairPersistFailed, "failed to persist repair queue")
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeHealthRepairPersistFailed, "failed to persist repair queue")
		return
	}
	if err := h.store.SaveTask(r.Context(), task); err != nil {
		_ = h.store.FinishLibraryHealthRepairRun(r.Context(), repairID, contracts.TaskFailed, time.Now())
		h.tasks.Fail(task.TaskID, contracts.ErrorCodeHealthRepairPersistFailed, "failed to persist repair task")
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeHealthRepairPersistFailed, "failed to persist repair task")
		return
	}

	go h.runLibraryHealthMetadataRepair(repairID, task.TaskID, items)
	persisted, err := h.store.GetLibraryHealthRepairRun(r.Context(), repairID)
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeHealthRepairPersistFailed, "failed to read persisted repair queue")
		return
	}
	writeJSON(w, http.StatusAccepted, libraryHealthRepairDTO(persisted))
}

func (h *Handler) handleGetLibraryHealthRepair(w http.ResponseWriter, r *http.Request) {
	if h.store == nil {
		writeAppError(w, http.StatusServiceUnavailable, contracts.ErrorCodeInternal, "library health repair service is unavailable")
		return
	}
	repairID := strings.TrimSpace(r.PathValue("repairId"))
	if repairID == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "repairId is required")
		return
	}
	run, err := h.store.GetLibraryHealthRepairRun(r.Context(), repairID)
	if errors.Is(err, storage.ErrLibraryHealthRepairNotFound) {
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "library health repair not found")
		return
	}
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to read library health repair")
		return
	}
	writeJSON(w, http.StatusOK, libraryHealthRepairDTO(run))
}

func normalizeLibraryHealthRepairCategories(input []string) ([]string, error) {
	if len(input) == 0 {
		return []string{"metadata_missing", "metadata_failed"}, nil
	}
	seen := make(map[string]struct{})
	out := make([]string, 0, len(input))
	for _, raw := range input {
		category := strings.TrimSpace(raw)
		if _, ok := libraryHealthMetadataRepairCategories[category]; !ok {
			return nil, fmt.Errorf("unsupported metadata repair category %q", category)
		}
		if _, exists := seen[category]; exists {
			continue
		}
		seen[category] = struct{}{}
		out = append(out, category)
	}
	slices.Sort(out)
	return out, nil
}

func selectLibraryHealthMetadataRepairItems(
	report contracts.LibraryHealthReportDTO, categories, findingIDs []string, limit int,
) []storage.LibraryHealthRepairItem {
	categorySet := make(map[string]struct{}, len(categories))
	for _, category := range categories {
		categorySet[category] = struct{}{}
	}
	findingSet := make(map[string]struct{}, len(findingIDs))
	for _, id := range findingIDs {
		if id = strings.TrimSpace(id); id != "" {
			findingSet[id] = struct{}{}
		}
	}
	seenMovies := make(map[string]struct{})
	items := make([]storage.LibraryHealthRepairItem, 0, limit)
	for _, finding := range report.Findings {
		if _, ok := categorySet[finding.Category]; !ok || finding.EntityType != "movie" || strings.TrimSpace(finding.EntityID) == "" {
			continue
		}
		if len(findingSet) > 0 {
			if _, ok := findingSet[finding.ID]; !ok {
				continue
			}
		}
		if !slices.Contains(finding.RepairActions, "rescrape_metadata") {
			continue
		}
		if _, duplicate := seenMovies[finding.EntityID]; duplicate {
			continue
		}
		seenMovies[finding.EntityID] = struct{}{}
		items = append(items, storage.LibraryHealthRepairItem{
			Ordinal: len(items), FindingID: finding.ID, Category: finding.Category,
			MovieID: finding.EntityID, Label: finding.Label, Status: contracts.TaskPending,
		})
		if len(items) == limit {
			break
		}
	}
	return items
}

func newLibraryHealthRepairID() (string, error) {
	var value [8]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return "repair_" + hex.EncodeToString(value[:]), nil
}

func (h *Handler) runLibraryHealthMetadataRepair(repairID, taskID string, items []storage.LibraryHealthRepairItem) {
	ctx := h.runtimeContext
	if ctx == nil {
		ctx = context.Background()
	}
	startedAt := time.Now().UTC()
	if err := h.store.StartLibraryHealthRepairRun(ctx, repairID, startedAt); err != nil {
		h.failLibraryHealthRepairTask(ctx, repairID, taskID, contracts.ErrorCodeHealthRepairPersistFailed, err.Error())
		return
	}
	task := h.tasks.Start(taskID, "Queueing metadata repairs")
	_ = h.store.SaveTask(ctx, task)

	type queuedItem struct {
		ordinal int
		taskID  string
	}
	queued := make(map[string]queuedItem)
	for _, item := range items {
		if err := ctx.Err(); err != nil {
			_ = h.store.CompleteLibraryHealthRepairItem(context.WithoutCancel(ctx), repairID, item.Ordinal, "cancelled", contracts.ErrorCodeHealthRepairInterrupted, err.Error(), time.Now())
			continue
		}
		child, err := h.movieMetadataRefresher.StartMovieMetadataRefresh(ctx, item.MovieID)
		if err != nil {
			_ = h.store.CompleteLibraryHealthRepairItem(ctx, repairID, item.Ordinal, "failed", contracts.ErrorCodeScraperRun, err.Error(), time.Now())
			continue
		}
		if err := h.store.QueueLibraryHealthRepairItem(ctx, repairID, item.Ordinal, child.TaskID, time.Now()); err != nil {
			_ = h.store.CompleteLibraryHealthRepairItem(ctx, repairID, item.Ordinal, "failed", contracts.ErrorCodeHealthRepairPersistFailed, err.Error(), time.Now())
			continue
		}
		queued[child.TaskID] = queuedItem{ordinal: item.Ordinal, taskID: child.TaskID}
	}
	h.updateLibraryHealthRepairTask(ctx, repairID, taskID, "Waiting for metadata repairs")

	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for len(queued) > 0 {
		select {
		case <-ctx.Done():
			for childID, item := range queued {
				_ = h.store.CompleteLibraryHealthRepairItem(context.WithoutCancel(ctx), repairID, item.ordinal, "cancelled", contracts.ErrorCodeHealthRepairInterrupted, "backend stopped before child task completed", time.Now())
				delete(queued, childID)
			}
			h.failLibraryHealthRepairTask(context.WithoutCancel(ctx), repairID, taskID, contracts.ErrorCodeHealthRepairInterrupted, "metadata repair interrupted")
			return
		case <-ticker.C:
			for childID, item := range queued {
				child, ok := h.tasks.Get(item.taskID)
				if !ok || !isTerminalTaskStatus(child.Status) {
					continue
				}
				itemStatus := "succeeded"
				if child.Status != contracts.TaskCompleted {
					itemStatus = "failed"
				}
				if err := h.store.CompleteLibraryHealthRepairItem(ctx, repairID, item.ordinal, itemStatus, child.ErrorCode, child.ErrorMessage, time.Now()); err != nil {
					if h.logger != nil {
						h.logger.Error("persist library health repair item failed", zap.String("repairId", repairID), zap.String("childTaskId", childID), zap.Error(err))
					}
					continue
				}
				delete(queued, childID)
			}
			h.updateLibraryHealthRepairTask(ctx, repairID, taskID, "Repairing metadata")
		}
	}
	h.finishLibraryHealthRepairTask(ctx, repairID, taskID)
}

func isTerminalTaskStatus(status string) bool {
	switch status {
	case contracts.TaskCompleted, contracts.TaskPartialFailed, contracts.TaskFailed, contracts.TaskCancelled:
		return true
	default:
		return false
	}
}

func (h *Handler) updateLibraryHealthRepairTask(ctx context.Context, repairID, taskID, message string) {
	run, err := h.store.GetLibraryHealthRepairRun(ctx, repairID)
	if err != nil {
		return
	}
	progress := 0
	if run.TotalItems > 0 {
		progress = run.CompletedItems * 100 / run.TotalItems
	}
	task := h.tasks.ProgressWithMetadata(taskID, progress, message, map[string]any{
		"completedItems": run.CompletedItems,
		"succeededItems": run.SucceededItems,
		"failedItems":    run.FailedItems,
	})
	_ = h.store.SaveTask(ctx, task)
}

func (h *Handler) finishLibraryHealthRepairTask(ctx context.Context, repairID, taskID string) {
	run, err := h.store.GetLibraryHealthRepairRun(ctx, repairID)
	if err != nil {
		h.failLibraryHealthRepairTask(ctx, repairID, taskID, contracts.ErrorCodeHealthRepairPersistFailed, err.Error())
		return
	}
	status := contracts.TaskCompleted
	var task contracts.TaskDTO
	metadata := map[string]any{
		"completedItems": run.CompletedItems,
		"succeededItems": run.SucceededItems,
		"failedItems":    run.FailedItems,
	}
	switch {
	case run.FailedItems == 0:
		task = h.tasks.ProgressWithMetadata(taskID, 100, "Metadata repair completed", metadata)
		task = h.tasks.Complete(taskID, "Metadata repair completed")
	case run.SucceededItems > 0:
		status = contracts.TaskPartialFailed
		task = h.tasks.PartialFail(taskID, contracts.ErrorCodeScraperRun, "Metadata repair partially failed", metadata)
	default:
		status = contracts.TaskFailed
		task = h.tasks.Fail(taskID, contracts.ErrorCodeScraperRun, "Metadata repair failed")
	}
	if err := h.store.FinishLibraryHealthRepairRun(ctx, repairID, status, time.Now()); err != nil && h.logger != nil {
		h.logger.Error("finish library health repair run failed", zap.String("repairId", repairID), zap.Error(err))
	}
	_ = h.store.SaveTask(ctx, task)
}

func (h *Handler) failLibraryHealthRepairTask(ctx context.Context, repairID, taskID, code, message string) {
	task := h.tasks.Fail(taskID, code, message)
	_ = h.store.SaveTask(ctx, task)
	_ = h.store.FinishLibraryHealthRepairRun(ctx, repairID, contracts.TaskFailed, time.Now())
}

func libraryHealthRepairDTO(run storage.LibraryHealthRepairRun) contracts.LibraryHealthRepairDTO {
	var categories []string
	_ = json.Unmarshal([]byte(run.CategoriesJSON), &categories)
	dto := contracts.LibraryHealthRepairDTO{
		RepairID: run.RepairID, TaskID: run.TaskID, Action: run.Action, Categories: categories,
		Status: run.Status, TotalItems: run.TotalItems, CompletedItems: run.CompletedItems,
		SucceededItems: run.SucceededItems, FailedItems: run.FailedItems,
		CreatedAt: run.CreatedAt, StartedAt: run.StartedAt, FinishedAt: run.FinishedAt,
		Items: make([]contracts.LibraryHealthRepairItemDTO, 0, len(run.Items)),
	}
	for _, item := range run.Items {
		dto.Items = append(dto.Items, contracts.LibraryHealthRepairItemDTO{
			Ordinal: item.Ordinal, FindingID: item.FindingID, Category: item.Category,
			MovieID: item.MovieID, Label: item.Label, Status: item.Status,
			ChildTaskID: item.ChildTaskID, ErrorCode: item.ErrorCode, ErrorMessage: item.ErrorMessage,
			StartedAt: item.StartedAt, FinishedAt: item.FinishedAt,
		})
	}
	return dto
}

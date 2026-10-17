package server

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

func (h *Handler) registerTaskCenterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/task-center/summary", h.handleTaskCenterSummary)
	mux.HandleFunc("GET /api/task-center/tasks", h.handleTaskCenterTasks)
	mux.HandleFunc("GET /api/task-center/tasks/{taskKey}", h.handleTaskCenterDetail)
}

func taskCenterError(w http.ResponseWriter, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		writeAppError(w, 404, contracts.ErrorCodeNotFound, "task record unavailable")
		return
	}
	if errors.Is(err, storage.ErrTaskCenterFilter) {
		writeAppError(w, 400, contracts.ErrorCodeBadRequest, "invalid task query")
		return
	}
	writeAppError(w, 500, contracts.ErrorCodeInternal, "cannot read task records")
}

func (h *Handler) handleTaskCenterSummary(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	v, err := h.store.TaskCenterSummary(r.Context())
	if err != nil {
		taskCenterError(w, err)
		return
	}
	writeJSON(w, 200, v)
}

func (h *Handler) handleTaskCenterTasks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	q := r.URL.Query()
	limit := 20
	if q.Has("limit") {
		n, err := strconv.Atoi(q.Get("limit"))
		if err != nil || n < 1 || n > 100 {
			taskCenterError(w, storage.ErrTaskCenterFilter)
			return
		}
		limit = n
	}
	keys, next, err := h.store.TaskCenterKeys(r.Context(), storage.TaskCenterFilter{Kind: q.Get("kind"), State: q.Get("state"), Cursor: q.Get("cursor"), Limit: limit})
	if err != nil {
		taskCenterError(w, err)
		return
	}
	summary, err := h.store.TaskCenterSummary(r.Context())
	if err != nil {
		taskCenterError(w, err)
		return
	}
	out := contracts.TaskCenterPage{Items: []contracts.TaskCenterTask{}, NextCursor: next, Summary: summary}
	for _, key := range keys {
		v, err := h.store.TaskCenterTask(r.Context(), key)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			taskCenterError(w, err)
			return
		}
		v.CanManage = isDirectLocalRequest(r)
		out.Items = append(out.Items, v)
	}
	writeJSON(w, 200, out)
}

func (h *Handler) handleTaskCenterDetail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	v, err := h.store.TaskCenterTask(r.Context(), r.PathValue("taskKey"))
	if err != nil {
		taskCenterError(w, err)
		return
	}
	v.CanManage = isDirectLocalRequest(r)
	if v.Subtitle != nil && h.subtitles != nil {
		for i := range v.Subtitle.Assets {
			_, err := h.subtitles.AssetPath(r.Context(), v.Subtitle.Assets[i].ID)
			v.Subtitle.Assets[i].Missing = err != nil
		}
	}
	writeJSON(w, 200, v)
}

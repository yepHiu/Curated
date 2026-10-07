package server

import (
	"context"
	"curated-backend/internal/agent/core"
	"curated-backend/internal/contracts"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
)

// TopicOrganizationProvider 保持 HTTP 与业务实现解耦。
type TopicOrganizationProvider interface {
	TagOrganizationIssues(context.Context, int, int) ([]contracts.TagOrganizationItemDTO, error)
	HomepageTopics(context.Context) ([]contracts.HomepageTopicGroupDTO, error)
	StartTagOrganization(context.Context, contracts.TagOrganizationRequest) (contracts.TagOrganizationJobDTO, error)
	ListTagOrganizations(context.Context) ([]contracts.TagOrganizationJobDTO, error)
	TagOrganizationStats(context.Context) (contracts.TagOrganizationStatsDTO, error)
	GetTagOrganization(context.Context, string) (contracts.TagOrganizationJobDTO, error)
	TagOrganizationItems(context.Context, string, int, int) ([]contracts.TagOrganizationItemDTO, error)
	CancelTagOrganization(context.Context, string) error
	DeleteTagOrganization(context.Context, string) error
	RetryTagOrganization(context.Context, string) error
	UndoTagOrganization(context.Context, string) (contracts.TagOrganizationUndoDTO, error)
}

// decodeTopicRequest 拒绝未知写字段、超长请求与尾随 JSON。
func decodeTopicRequest(w http.ResponseWriter, r *http.Request, out any) bool {
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64*1024))
	d.DisallowUnknownFields()
	err := d.Decode(out)
	if err == nil {
		var extra any
		if d.Decode(&extra) != io.EOF {
			err = errors.New("trailing JSON")
		}
	}
	if err != nil {
		writeAppError(w, 400, contracts.ErrorCodeBadRequest, "invalid topic request")
		return false
	}
	return true
}

// topicHTTPError 统一映射可重试业务失败，不回显模型原文或内部 SQL。
func topicHTTPError(w http.ResponseWriter, err error) {
	var toolErr *core.ToolError
	if errors.As(err, &toolErr) {
		status := http.StatusBadRequest
		if toolErr.Code == "AI_DISABLED" || toolErr.Code == "AI_READ_ONLY" {
			status = http.StatusForbidden
		}
		writeAppError(w, status, toolErr.Code, toolErr.Message)
		return
	}
	if errors.Is(err, sql.ErrNoRows) {
		writeAppError(w, 404, contracts.ErrorCodeNotFound, "topic or task not found")
		return
	}
	writeAppError(w, 409, "TAG_ORGANIZATION_UNAVAILABLE", "Unable to perform this operation. Check AI settings, task state and selection.")
}

// handleListTopics 返回分页主题目录。
func (h *Handler) handleListTopics(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	v, err := h.store.ListLibraryTopics(r.Context(), limit, offset)
	if err != nil {
		topicHTTPError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"items": v})
}

// handleGetTopic 返回单题材或显式调整隐藏状态/用户标签名称。
func (h *Handler) handleGetTopic(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("topicId")
	if r.Method == http.MethodPatch {
		var body struct {
			Hidden       *bool   `json:"hidden"`
			Name         *string `json:"name"`
			ExpectedName *string `json:"expectedName"`
		}
		if !decodeTopicRequest(w, r, &body) {
			return
		}
		if (body.Hidden == nil) == (body.Name == nil) || (body.Name != nil && body.ExpectedName == nil) || (body.Hidden != nil && body.ExpectedName != nil) {
			writeAppError(w, 400, contracts.ErrorCodeBadRequest, "provide hidden or name with expectedName")
			return
		}
		var err error
		if body.Name != nil {
			err = h.store.RenameLibraryTopic(r.Context(), id, *body.ExpectedName, *body.Name)
		} else {
			err = h.store.SetLibraryTopicHidden(r.Context(), id, *body.Hidden)
		}
		if err != nil {
			topicHTTPError(w, err)
			return
		}
	}
	v, err := h.store.GetLibraryTopic(r.Context(), id)
	if err != nil {
		topicHTTPError(w, err)
		return
	}
	writeJSON(w, 200, v)
}

// handleHomepageTopics 复用图片资源 URL 丰富逻辑。
func (h *Handler) handleHomepageTopics(w http.ResponseWriter, r *http.Request) {
	if h.topicOrganization == nil {
		writeJSON(w, 200, map[string]any{"items": []contracts.HomepageTopicGroupDTO{}})
		return
	}
	v, err := h.topicOrganization.HomepageTopics(r.Context())
	if err != nil {
		topicHTTPError(w, err)
		return
	}
	for i := range v {
		h.enrichMovieListItemsLocalPosters(r.Context(), v[i].Movies)
	}
	writeJSON(w, 200, map[string]any{"items": v})
}

// handleTagOrganizations 处理显式发起与持久任务列表。
func (h *Handler) handleTagOrganizations(w http.ResponseWriter, r *http.Request) {
	if h.topicOrganization == nil {
		writeAppError(w, 503, "TAG_ORGANIZATION_UNAVAILABLE", "organization unavailable")
		return
	}
	if r.Method == http.MethodPost {
		var req contracts.TagOrganizationRequest
		if !decodeTopicRequest(w, r, &req) {
			return
		}
		v, err := h.topicOrganization.StartTagOrganization(r.Context(), req)
		if err != nil {
			topicHTTPError(w, err)
			return
		}
		writeJSON(w, 202, v)
		return
	}
	v, err := h.topicOrganization.ListTagOrganizations(r.Context())
	if err != nil {
		topicHTTPError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"items": v})
}

// handleTagOrganizationIssues is a bounded read; opening it never retries work.
func (h *Handler) handleTagOrganizationIssues(w http.ResponseWriter, r *http.Request) {
	if h.topicOrganization == nil {
		writeAppError(w, 503, "TAG_ORGANIZATION_UNAVAILABLE", "organization unavailable")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	v, err := h.topicOrganization.TagOrganizationIssues(r.Context(), limit, offset)
	if err != nil {
		topicHTTPError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, map[string]any{"items": v})
}

// handleTagOrganizationStats exposes source-aware coverage of the active library.
func (h *Handler) handleTagOrganizationStats(w http.ResponseWriter, r *http.Request) {
	if h.topicOrganization == nil {
		writeAppError(w, 503, "TAG_ORGANIZATION_UNAVAILABLE", "organization unavailable")
		return
	}
	v, err := h.topicOrganization.TagOrganizationStats(r.Context())
	if err != nil {
		topicHTTPError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, v)
}

// handleTagOrganization 提供详情、分页结果和显式取消／重试／撤销。
func (h *Handler) handleTagOrganization(w http.ResponseWriter, r *http.Request) {
	if h.topicOrganization == nil {
		writeAppError(w, 503, "TAG_ORGANIZATION_UNAVAILABLE", "organization unavailable")
		return
	}
	id := r.PathValue("jobId")
	op := r.PathValue("operation")
	if op == "" && r.Method != http.MethodGet && r.Method != http.MethodDelete || op == "items" && r.Method != http.MethodGet || (op == "cancel" || op == "retry" || op == "undo") && r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if _, err := h.topicOrganization.GetTagOrganization(r.Context(), id); err != nil {
		topicHTTPError(w, err)
		return
	}
	if op == "" && r.Method == http.MethodDelete {
		if err := h.topicOrganization.DeleteTagOrganization(r.Context(), id); err != nil {
			topicHTTPError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	switch r.PathValue("operation") {
	case "items":
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		v, err := h.topicOrganization.TagOrganizationItems(r.Context(), id, limit, offset)
		if err != nil {
			topicHTTPError(w, err)
			return
		}
		writeJSON(w, 200, map[string]any{"items": v})
		return
	case "cancel":
		if err := h.topicOrganization.CancelTagOrganization(r.Context(), id); err != nil {
			topicHTTPError(w, err)
			return
		}
	case "retry":
		if err := h.topicOrganization.RetryTagOrganization(r.Context(), id); err != nil {
			topicHTTPError(w, err)
			return
		}
	case "undo":
		v, err := h.topicOrganization.UndoTagOrganization(r.Context(), id)
		if err != nil {
			topicHTTPError(w, err)
			return
		}
		writeJSON(w, 200, v)
		return
	case "":
	default:
		writeAppError(w, 404, contracts.ErrorCodeNotFound, "unknown operation")
		return
	}
	v, err := h.topicOrganization.GetTagOrganization(r.Context(), id)
	if err != nil {
		topicHTTPError(w, err)
		return
	}
	writeJSON(w, 200, v)
}

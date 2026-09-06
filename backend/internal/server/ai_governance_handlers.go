package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"curated-backend/internal/contracts"
)

type AIGovernanceProvider interface {
	AIGovernanceSettings() contracts.AIGovernanceDTO
	SetAIGovernanceSettings(contracts.AIGovernanceDTO) error
	GetAIReport(context.Context, contracts.AIReportQuery) (contracts.AIReportDTO, error)
	ListAIAudit(context.Context, contracts.AIReportQuery) (contracts.AIAuditPageDTO, error)
	CleanupAIRecords(context.Context) (contracts.AICleanupDTO, error)
}

func (h *Handler) handleAIGovernance(w http.ResponseWriter, r *http.Request) {
	p, ok := h.aiChatProvider.(AIGovernanceProvider)
	if !ok {
		writeAppError(w, 503, contracts.ErrorCodeInternal, "AI governance unavailable")
		return
	}
	if r.Method == http.MethodGet {
		writeJSON(w, 200, p.AIGovernanceSettings())
		return
	}
	value := p.AIGovernanceSettings()
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		writeAppError(w, 400, contracts.ErrorCodeBadRequest, "invalid AI settings")
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		writeAppError(w, 400, contracts.ErrorCodeBadRequest, "invalid AI settings")
		return
	}
	if err := p.SetAIGovernanceSettings(value); err != nil {
		writeAppError(w, 400, contracts.ErrorCodeBadRequest, err.Error())
		return
	}
	writeJSON(w, 200, p.AIGovernanceSettings())
}

func parseAIReportQuery(r *http.Request, audit bool) (contracts.AIReportQuery, error) {
	q := contracts.AIReportQuery{Days: 30, Limit: 25, Channel: r.URL.Query().Get("channel"), Status: r.URL.Query().Get("status")}
	for _, field := range []struct {
		name     string
		ptr      *int
		min, max int
	}{{"days", &q.Days, 1, 365}, {"limit", &q.Limit, 1, 100}, {"offset", &q.Offset, 0, 100000}} {
		if raw := r.URL.Query().Get(field.name); raw != "" {
			n, err := strconv.Atoi(raw)
			if err != nil || n < field.min || n > field.max {
				return q, fmt.Errorf("invalid %s", field.name)
			}
			*field.ptr = n
		}
	}
	switch q.Channel {
	case "", "chat", "action", "test":
	default:
		return q, fmt.Errorf("invalid channel")
	}
	allowed := map[string]bool{"": true, "completed": true, "failed": true, "partial": true, "cancelled": true, "needs_input": true}
	if audit {
		allowed = map[string]bool{"": true, "failed": true, "ok": true, "error": true, "previewed": true, "confirmed": true, "rejected": true}
	}
	if !allowed[q.Status] {
		return q, fmt.Errorf("invalid status")
	}
	return q, nil
}

func (h *Handler) handleAIReport(w http.ResponseWriter, r *http.Request) {
	p, ok := h.aiChatProvider.(AIGovernanceProvider)
	if !ok {
		writeAppError(w, 503, contracts.ErrorCodeInternal, "AI governance unavailable")
		return
	}
	q, err := parseAIReportQuery(r, false)
	if err != nil {
		writeAppError(w, 400, contracts.ErrorCodeBadRequest, err.Error())
		return
	}
	result, err := p.GetAIReport(r.Context(), q)
	if err != nil {
		writeAppError(w, 500, contracts.ErrorCodeInternal, "AI statistics unavailable")
		return
	}
	writeJSON(w, 200, result)
}
func (h *Handler) handleAIAudit(w http.ResponseWriter, r *http.Request) {
	p, ok := h.aiChatProvider.(AIGovernanceProvider)
	if !ok {
		writeAppError(w, 503, contracts.ErrorCodeInternal, "AI governance unavailable")
		return
	}
	q, err := parseAIReportQuery(r, true)
	if err != nil {
		writeAppError(w, 400, contracts.ErrorCodeBadRequest, err.Error())
		return
	}
	result, err := p.ListAIAudit(r.Context(), q)
	if err != nil {
		writeAppError(w, 500, contracts.ErrorCodeInternal, "AI audit unavailable")
		return
	}
	writeJSON(w, 200, result)
}
func (h *Handler) handleAICleanup(w http.ResponseWriter, r *http.Request) {
	p, ok := h.aiChatProvider.(AIGovernanceProvider)
	if !ok {
		writeAppError(w, 503, contracts.ErrorCodeInternal, "AI governance unavailable")
		return
	}
	result, err := p.CleanupAIRecords(r.Context())
	if err != nil {
		writeAppError(w, 500, contracts.ErrorCodeInternal, "AI cleanup failed")
		return
	}
	writeJSON(w, 200, result)
}

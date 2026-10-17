package server

import (
	"context"
	"net/http"
	"strconv"

	"curated-backend/internal/contracts"
)

type BookOrganizationProvider interface {
	StartBookOrganization(context.Context, contracts.BookOrganizationRequest) (contracts.BookOrganizationJob, error)
	ControlBookOrganization(context.Context, string, string) error
	ReviewBookOrganization(context.Context, string, contracts.BookOrganizationReview) error
}

func (h *Handler) handleBookOrganizations(w http.ResponseWriter, r *http.Request) {
	provider, ok := h.aiChatProvider.(BookOrganizationProvider)
	if !ok {
		writeAppError(w, 503, "BOOK_ORGANIZATION_UNAVAILABLE", "organization unavailable")
		return
	}
	id := r.PathValue("jobId")
	if id == "" {
		var req contracts.BookOrganizationRequest
		if !decodeTopicRequest(w, r, &req) {
			return
		}
		job, err := provider.StartBookOrganization(r.Context(), req)
		if err != nil {
			topicHTTPError(w, err)
			return
		}
		writeJSON(w, http.StatusAccepted, job)
		return
	}
	var err error
	switch r.PathValue("operation") {
	case "cancel", "retry":
		err = provider.ControlBookOrganization(r.Context(), id, r.PathValue("operation"))
	case "review":
		var req contracts.BookOrganizationReview
		if !decodeTopicRequest(w, r, &req) {
			return
		}
		err = provider.ReviewBookOrganization(r.Context(), id, req)
	default:
		writeAppError(w, 404, contracts.ErrorCodeNotFound, "operation not found")
		return
	}
	if err != nil {
		topicHTTPError(w, err)
		return
	}
	job, err := h.store.GetBookOrganization(r.Context(), id)
	if err != nil {
		topicHTTPError(w, err)
		return
	}
	writeJSON(w, 200, job)
}

func (h *Handler) handleBookOrganizationItems(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("jobId")
	if _, err := h.store.GetBookOrganization(r.Context(), id); err != nil {
		topicHTTPError(w, err)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	items, err := h.store.BookOrganizationItems(r.Context(), id, limit, offset)
	if err != nil {
		topicHTTPError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, map[string]any{"items": items})
}

package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"curated-backend/internal/contracts"
	"go.uber.org/zap"
)

func (h *Handler) handlePreviewActorMerge(w http.ResponseWriter, r *http.Request) {
	if h.actorMergeProvider == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "actor merge not configured")
		return
	}
	var body contracts.ActorMergePreviewRequest
	if err := decodeActorMergeJSON(w, r, &body); err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeActorMergeInvalid, "invalid actor merge preview body")
		return
	}
	preview, err := h.actorMergeProvider.PreviewActorMerge(r.Context(), body)
	if err != nil {
		h.writeActorMergeError(w, "preview actor merge", err)
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

func (h *Handler) handleApplyActorMerge(w http.ResponseWriter, r *http.Request) {
	if h.actorMergeProvider == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "actor merge not configured")
		return
	}
	var body contracts.ApplyActorMergeRequest
	if err := decodeActorMergeJSON(w, r, &body); err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeActorMergeInvalid, "invalid actor merge body")
		return
	}
	audit, err := h.actorMergeProvider.ApplyActorMerge(r.Context(), body)
	if err != nil {
		h.writeActorMergeError(w, "apply actor merge", err)
		return
	}
	writeJSON(w, http.StatusOK, audit)
}

func (h *Handler) handleListActorMergeAudits(w http.ResponseWriter, r *http.Request) {
	if h.actorMergeProvider == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "actor merge not configured")
		return
	}
	limit, _ := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("limit")))
	offset, _ := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("offset")))
	result, err := h.actorMergeProvider.ListActorMergeAudits(r.Context(), limit, offset)
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("list actor merge audits failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to list actor merge audits")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func decodeActorMergeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	if r.Body == nil {
		return errors.New("request body is required")
	}
	defer r.Body.Close()
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err == nil {
		return errors.New("multiple json values")
	} else if !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

func (h *Handler) writeActorMergeError(w http.ResponseWriter, operation string, err error) {
	switch {
	case errors.Is(err, contracts.ErrActorMergeInvalid):
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeActorMergeInvalid, err.Error())
	case errors.Is(err, contracts.ErrActorMergeNotFound):
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodeActorMergeNotFound, "actor not found")
	case errors.Is(err, contracts.ErrActorMergeSelf):
		writeAppError(w, http.StatusConflict, contracts.ErrorCodeActorMergeSelf, err.Error())
	case errors.Is(err, contracts.ErrActorMergeSourceAlias):
		writeAppError(w, http.StatusConflict, contracts.ErrorCodeActorMergeSourceAlias, err.Error())
	case errors.Is(err, contracts.ErrActorMergeStalePreview):
		writeAppError(w, http.StatusConflict, contracts.ErrorCodeActorMergeStalePreview, err.Error())
	case errors.Is(err, contracts.ErrActorMergeLinkLimit):
		writeAppError(w, http.StatusConflict, contracts.ErrorCodeActorMergeLinkLimit, err.Error())
	case errors.Is(err, contracts.ErrActorMergeConflict):
		writeAppError(w, http.StatusConflict, contracts.ErrorCodeActorMergeConflict, err.Error())
	default:
		if h.logger != nil {
			h.logger.Warn(operation+" failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "actor merge failed")
	}
}

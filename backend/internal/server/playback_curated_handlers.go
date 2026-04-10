package server

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"curated-backend/internal/contracts"
	"curated-backend/internal/curatedthumb"
	"curated-backend/internal/storage"
)

const maxCuratedImageBytes = 12 << 20 // 12 MiB raw PNG/JPEG
func mapCuratedFrameItems(rows []storage.CuratedFrameMeta) []contracts.CuratedFrameItemDTO {
	items := make([]contracts.CuratedFrameItemDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, contracts.CuratedFrameItemDTO{
			ID:          row.ID,
			MovieID:     row.MovieID,
			Title:       row.Title,
			Code:        row.Code,
			Actors:      row.Actors,
			PositionSec: row.PositionSec,
			CapturedAt:  row.CapturedAt,
			Tags:        row.Tags,
		})
	}
	return items
}

func mapCuratedFrameFacets(rows []storage.CuratedFrameFacet) []contracts.CuratedFrameFacetItemDTO {
	items := make([]contracts.CuratedFrameFacetItemDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, contracts.CuratedFrameFacetItemDTO{Name: row.Name, Count: row.Count})
	}
	return items
}

func (h *Handler) handleListPlaybackProgress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	ctx := r.Context()
	rows, err := h.store.ListPlaybackProgressByUpdatedDesc(ctx)
	if err != nil {
		h.logger.Error("list playback progress", zap.Error(err))
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to list playback progress")
		return
	}
	items := make([]contracts.PlaybackProgressItemDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, contracts.PlaybackProgressItemDTO{
			MovieID:     row.MovieID,
			PositionSec: row.PositionSec,
			DurationSec: row.DurationSec,
			UpdatedAt:   row.UpdatedAt,
		})
	}
	writeJSON(w, http.StatusOK, contracts.PlaybackProgressListDTO{Items: items})
}

func (h *Handler) handlePutPlaybackProgress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	movieID := strings.TrimSpace(r.PathValue("movieId"))
	if movieID == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "movieId is required")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "failed to read body")
		return
	}
	var req contracts.PutPlaybackProgressBody
	if err := json.Unmarshal(body, &req); err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid json body")
		return
	}
	ctx := r.Context()
	ok, err := h.store.MovieExists(ctx, movieID)
	if err != nil {
		h.logger.Error("movie exists check", zap.Error(err))
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to verify movie")
		return
	}
	if !ok {
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie not found")
		return
	}
	if err := h.store.UpsertPlaybackProgress(ctx, movieID, req.PositionSec, req.DurationSec); err != nil {
		h.logger.Error("upsert playback progress", zap.Error(err))
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to save playback progress")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleDeletePlaybackProgress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	movieID := strings.TrimSpace(r.PathValue("movieId"))
	if movieID == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "movieId is required")
		return
	}
	if err := h.store.DeletePlaybackProgress(r.Context(), movieID); err != nil {
		h.logger.Error("delete playback progress", zap.Error(err))
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to delete playback progress")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleListCuratedFrames(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	ctx := r.Context()
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	page, err := h.store.QueryCuratedFrames(ctx, storage.CuratedFrameQuery{
		Query:   strings.TrimSpace(q.Get("q")),
		Actor:   strings.TrimSpace(q.Get("actor")),
		MovieID: strings.TrimSpace(q.Get("movieId")),
		Tag:     strings.TrimSpace(q.Get("tag")),
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		h.logger.Error("list curated frames", zap.Error(err))
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to list curated frames")
		return
	}
	writeJSON(w, http.StatusOK, contracts.CuratedFramesListDTO{
		Items:  mapCuratedFrameItems(page.Items),
		Total:  page.Total,
		Limit:  page.Limit,
		Offset: page.Offset,
	})
}

func (h *Handler) handleGetCuratedFrameStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	total, err := h.store.CountCuratedFrames(r.Context())
	if err != nil {
		h.logger.Error("count curated frames", zap.Error(err))
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to count curated frames")
		return
	}
	writeJSON(w, http.StatusOK, contracts.CuratedFrameStatsDTO{Total: total})
}

func (h *Handler) handleListCuratedFrameTags(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	items, err := h.store.ListCuratedFrameTags(r.Context())
	if err != nil {
		h.logger.Error("list curated frame tags", zap.Error(err))
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to list curated frame tags")
		return
	}
	writeJSON(w, http.StatusOK, contracts.CuratedFrameFacetListDTO{Items: mapCuratedFrameFacets(items)})
}

func (h *Handler) handleListCuratedFrameActors(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	items, err := h.store.ListCuratedFrameActors(r.Context())
	if err != nil {
		h.logger.Error("list curated frame actors", zap.Error(err))
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to list curated frame actors")
		return
	}
	writeJSON(w, http.StatusOK, contracts.CuratedFrameFacetListDTO{Items: mapCuratedFrameFacets(items)})
}

func (h *Handler) handleGetCuratedFrameImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "id is required")
		return
	}
	ctx := r.Context()
	blob, err := h.store.GetCuratedFrameImage(ctx, id)
	if err != nil {
		h.logger.Error("get curated frame image", zap.Error(err))
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load image")
		return
	}
	if len(blob) == 0 {
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "curated frame not found")
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(blob)
}

func (h *Handler) handleGetCuratedFrameThumbnail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "id is required")
		return
	}
	blob, err := h.store.GetCuratedFrameThumbnail(r.Context(), id)
	if err != nil {
		h.logger.Error("get curated frame thumbnail", zap.Error(err))
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load thumbnail")
		return
	}
	if len(blob) == 0 {
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "curated frame not found")
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(blob)
}

func readCuratedFrameCreateJSON(r *http.Request) (contracts.CreateCuratedFrameBody, []byte, error) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxCuratedImageBytes+2<<20))
	if err != nil {
		return contracts.CreateCuratedFrameBody{}, nil, fmt.Errorf("failed to read body")
	}
	var req contracts.CreateCuratedFrameBody
	if err := json.Unmarshal(body, &req); err != nil {
		return contracts.CreateCuratedFrameBody{}, nil, fmt.Errorf("invalid json body")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(req.ImageBase64))
	if err != nil {
		return contracts.CreateCuratedFrameBody{}, nil, fmt.Errorf("invalid imageBase64")
	}
	return req, raw, nil
}

func readCuratedFrameCreateMultipart(r *http.Request) (contracts.CreateCuratedFrameBody, []byte, error) {
	if err := r.ParseMultipartForm(maxCuratedImageBytes + 2<<20); err != nil {
		return contracts.CreateCuratedFrameBody{}, nil, fmt.Errorf("invalid multipart body")
	}
	metaText := strings.TrimSpace(r.FormValue("metadata"))
	if metaText == "" {
		return contracts.CreateCuratedFrameBody{}, nil, fmt.Errorf("metadata is required")
	}
	var req contracts.CreateCuratedFrameBody
	if err := json.Unmarshal([]byte(metaText), &req); err != nil {
		return contracts.CreateCuratedFrameBody{}, nil, fmt.Errorf("invalid metadata json")
	}
	file, _, err := r.FormFile("image")
	if err != nil {
		return contracts.CreateCuratedFrameBody{}, nil, fmt.Errorf("image is required")
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, maxCuratedImageBytes+1))
	if err != nil {
		return contracts.CreateCuratedFrameBody{}, nil, fmt.Errorf("failed to read image")
	}
	return req, raw, nil
}

func (h *Handler) handlePostCuratedFrame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	var (
		req contracts.CreateCuratedFrameBody
		raw []byte
		err error
	)
	contentType := r.Header.Get("Content-Type")
	mediaType, _, parseErr := mime.ParseMediaType(contentType)
	if parseErr == nil && strings.HasPrefix(mediaType, "multipart/") {
		req, raw, err = readCuratedFrameCreateMultipart(r)
	} else {
		req, raw, err = readCuratedFrameCreateJSON(r)
	}
	req.ID = strings.TrimSpace(req.ID)
	req.MovieID = strings.TrimSpace(req.MovieID)
	if err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, err.Error())
		return
	}
	if req.ID == "" || req.MovieID == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "id and movieId are required")
		return
	}
	if len(raw) == 0 || len(raw) > maxCuratedImageBytes {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "image too large or empty")
		return
	}
	ctx := r.Context()
	ok, err := h.store.MovieExists(ctx, req.MovieID)
	if err != nil {
		h.logger.Error("movie exists check", zap.Error(err))
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to verify movie")
		return
	}
	if !ok {
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie not found")
		return
	}
	meta := storage.CuratedFrameMeta{
		ID:          req.ID,
		MovieID:     req.MovieID,
		Title:       req.Title,
		Code:        req.Code,
		Actors:      req.Actors,
		PositionSec: req.PositionSec,
		CapturedAt:  req.CapturedAt,
		Tags:        req.Tags,
	}
	if meta.Actors == nil {
		meta.Actors = []string{}
	}
	if meta.Tags == nil {
		meta.Tags = []string{}
	}
	thumbBlob, thumbErr := curatedthumb.PNG(raw)
	if thumbErr != nil {
		h.logger.Warn("build curated frame thumbnail", zap.String("id", meta.ID), zap.Error(thumbErr))
		thumbBlob = bytes.Clone(raw)
	}
	if err := h.store.InsertCuratedFrameWithThumbnail(ctx, meta, raw, thumbBlob); err != nil {
		if strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "unique") {
			writeAppError(w, http.StatusConflict, contracts.ErrorCodeConflict, "curated frame id already exists")
			return
		}
		h.logger.Error("insert curated frame", zap.Error(err))
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to save curated frame")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handlePatchCuratedFrameTags(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "id is required")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 256<<10))
	if err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "failed to read body")
		return
	}
	var req contracts.PatchCuratedFrameTagsBody
	if err := json.Unmarshal(body, &req); err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid json body")
		return
	}
	if req.Tags == nil {
		req.Tags = []string{}
	}
	ctx := r.Context()
	if err := h.store.UpdateCuratedFrameTags(ctx, id, req.Tags); err != nil {
		if err == sql.ErrNoRows {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "curated frame not found")
			return
		}
		h.logger.Error("update curated frame tags", zap.Error(err))
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to update tags")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleDeleteCuratedFrame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "id is required")
		return
	}
	ctx := r.Context()
	if err := h.store.DeleteCuratedFrame(ctx, id); err != nil {
		if err == sql.ErrNoRows {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "curated frame not found")
			return
		}
		h.logger.Error("delete curated frame", zap.Error(err))
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to delete curated frame")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

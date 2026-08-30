package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"curated-backend/internal/contracts"
	"curated-backend/internal/library/importcheck"
	"go.uber.org/zap"
)

func (h *Handler) handleCheckImportMovieCodes(w http.ResponseWriter, r *http.Request) {
	if h.store == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "library store is not configured")
		return
	}
	var body contracts.CheckImportMovieCodesRequest
	if err := decodeImportCodeCheckJSON(w, r, &body); err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid import code-check body")
		return
	}
	if msg := importcheck.ValidateNames(body.Names); msg != "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, msg)
		return
	}
	index, err := h.store.ListActiveMovieCodeIndex(r.Context())
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("list movie code index failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to check catalog codes")
		return
	}
	rows := make([]importcheck.IndexItem, 0, len(index))
	for _, item := range index {
		rows = append(rows, importcheck.IndexItem{ID: item.ID, Code: item.Code, Title: item.Title})
	}
	checked := importcheck.Check(body.Names, rows)
	out := contracts.ImportMovieCodeCheckDTO{
		Items: make([]contracts.ImportMovieCodeCheckItemDTO, 0, len(checked)),
	}
	for _, item := range checked {
		dto := contracts.ImportMovieCodeCheckItemDTO{
			Name:          item.Name,
			ExtractedCode: item.ExtractedCode,
			Matches:       make([]contracts.ImportMovieCodeMatchDTO, 0, len(item.Matches)),
		}
		for _, match := range item.Matches {
			dto.Matches = append(dto.Matches, contracts.ImportMovieCodeMatchDTO{
				MovieID:   match.MovieID,
				Code:      match.Code,
				Title:     match.Title,
				MatchKind: match.MatchKind,
			})
		}
		if len(dto.Matches) > 0 {
			out.MatchedCount++
		}
		out.Items = append(out.Items, dto)
	}
	writeJSON(w, http.StatusOK, out)
}

func decodeImportCodeCheckJSON(w http.ResponseWriter, r *http.Request, target any) error {
	if r.Body == nil {
		return errors.New("request body is required")
	}
	defer r.Body.Close()
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
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

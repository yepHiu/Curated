package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
)

type SubtitleSettingsController interface {
	SubtitleSettings() contracts.SubtitleSettings
	SubtitleConfig() config.SubtitleConfig
	SetSubtitleSettings(contracts.SubtitleSettingsPatch) error
	TestSubtitleProvider(context.Context) error
}

func (h *Handler) registerSubtitleRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/library/movies/{movieId}/subtitle-audio-tracks", func(w http.ResponseWriter, r *http.Request) {
		if !h.subtitleAvailable(w) {
			return
		}
		tracks, e := h.subtitles.AudioTracks(r.Context(), r.PathValue("movieId"), r.URL.Query().Get("fileId"))
		if e != nil {
			writeAppError(w, 400, contracts.ErrorCodeBadRequest, "could not read audio tracks")
			return
		}
		writeJSON(w, 200, tracks)
	})
	mux.HandleFunc("POST /api/subtitles/provider/test", localServerManagement(func(w http.ResponseWriter, r *http.Request) {
		if h.subtitleSettingsCtl == nil {
			writeAppError(w, 503, contracts.ErrorCodeInternal, "subtitle settings unavailable")
			return
		}
		e := h.subtitleSettingsCtl.TestSubtitleProvider(r.Context())
		if e != nil {
			writeAppError(w, 400, contracts.ErrorCodeBadRequest, e.Error())
			return
		}
		writeJSON(w, 200, map[string]bool{"ok": true, "uploadPolicyVerified": true})
	}))
	mux.HandleFunc("GET /api/subtitles/settings", func(w http.ResponseWriter, r *http.Request) {
		if h.subtitleSettingsCtl == nil {
			writeAppError(w, 503, contracts.ErrorCodeInternal, "subtitle settings unavailable")
			return
		}
		cfg := h.subtitleSettingsCtl.SubtitleSettings()
		cfg.CanManage = isDirectLocalRequest(r)
		writeJSON(w, 200, cfg)
	})
	mux.HandleFunc("POST /api/library/movies/{movieId}/subtitle-jobs", localServerManagement(h.handleCreateSubtitleJob))
	mux.HandleFunc("GET /api/library/movies/{movieId}/subtitle-jobs", h.handleListSubtitleJobs)
	mux.HandleFunc("GET /api/library/movies/{movieId}/subtitles", h.handleListSubtitleAssets)
	mux.HandleFunc("GET /api/subtitle-jobs/{jobId}", h.handleGetSubtitleJob)
	mux.HandleFunc("POST /api/subtitle-jobs/{jobId}/{action}", localServerManagement(h.handleSubtitleAction))
	mux.HandleFunc("GET /api/subtitles/{assetId}/content", h.handleSubtitleContent)
	mux.HandleFunc("POST /api/subtitles/{assetId}/reveal", localServerManagement(h.handleRevealSubtitle))
}
func (h *Handler) subtitleAvailable(w http.ResponseWriter) bool {
	if h.subtitles == nil {
		writeAppError(w, 503, contracts.ErrorCodeInternal, "subtitle service unavailable")
		return false
	}
	return true
}
func (h *Handler) handleCreateSubtitleJob(w http.ResponseWriter, r *http.Request) {
	if !h.subtitleAvailable(w) {
		return
	}
	var body contracts.SubtitleRequest
	if e := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&body); e != nil {
		writeAppError(w, 400, contracts.ErrorCodeBadRequest, "invalid subtitle request")
		return
	}
	job, e := h.subtitles.Create(r.Context(), r.PathValue("movieId"), body)
	if e != nil {
		writeAppError(w, 400, contracts.ErrorCodeBadRequest, e.Error())
		return
	}
	writeJSON(w, 202, job)
}
func (h *Handler) handleListSubtitleJobs(w http.ResponseWriter, r *http.Request) {
	if !h.subtitleAvailable(w) {
		return
	}
	jobs, e := h.subtitles.List(r.Context(), r.PathValue("movieId"))
	if e != nil {
		writeAppError(w, 500, contracts.ErrorCodeInternal, "cannot read subtitle tasks")
		return
	}
	writeJSON(w, 200, jobs)
}
func (h *Handler) handleGetSubtitleJob(w http.ResponseWriter, r *http.Request) {
	if !h.subtitleAvailable(w) {
		return
	}
	job, e := h.subtitles.Get(r.Context(), r.PathValue("jobId"))
	if e != nil {
		writeAppError(w, 404, contracts.ErrorCodeNotFound, "subtitle task not found")
		return
	}
	writeJSON(w, 200, job)
}
func (h *Handler) handleSubtitleAction(w http.ResponseWriter, r *http.Request) {
	if !h.subtitleAvailable(w) {
		return
	}
	job, e := h.subtitles.Action(r.Context(), r.PathValue("jobId"), r.PathValue("action"))
	if e != nil {
		writeAppError(w, 409, contracts.ErrorCodeBadRequest, e.Error())
		return
	}
	writeJSON(w, 200, job)
}
func (h *Handler) handleListSubtitleAssets(w http.ResponseWriter, r *http.Request) {
	if !h.subtitleAvailable(w) {
		return
	}
	detail, e := h.store.GetMoviePlaybackDetail(contracts.WithMovieFileSelection(r.Context(), r.URL.Query().Get("fileId")), r.PathValue("movieId"))
	if e != nil {
		writeAppError(w, 404, contracts.ErrorCodeNotFound, "movie file not found")
		return
	}
	jobs, e := h.subtitles.List(r.Context(), r.PathValue("movieId"))
	if e != nil {
		writeAppError(w, 500, contracts.ErrorCodeInternal, "cannot read subtitles")
		return
	}
	assets := []contracts.SubtitleAsset{}
	for _, job := range jobs {
		if job.FileID == detail.SelectedFileID && job.State == "completed" {
			valid := []contracts.SubtitleAsset{}
			for _, a := range job.Assets {
				if !a.Missing {
					valid = append(valid, a)
				}
			}
			if len(valid) > 0 {
				assets = valid
				break
			}
		}
	}
	writeJSON(w, 200, assets)
}
func (h *Handler) handleSubtitleContent(w http.ResponseWriter, r *http.Request) {
	if !h.subtitleAvailable(w) {
		return
	}
	path, e := h.subtitles.AssetPath(r.Context(), r.PathValue("assetId"))
	if e != nil {
		writeAppError(w, 404, contracts.ErrorCodeNotFound, "subtitle file unavailable")
		return
	}
	f, e := os.Open(path)
	if e != nil {
		writeAppError(w, 404, contracts.ErrorCodeNotFound, "subtitle file unavailable")
		return
	}
	defer f.Close()
	info, e := f.Stat()
	if e != nil || info.Size() > 16*1024*1024 {
		writeAppError(w, 413, contracts.ErrorCodeBadRequest, "subtitle file too large")
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, no-cache")
	w.Header().Set("ETag", fmt.Sprintf("\"%s-%d\"", r.PathValue("assetId"), info.ModTime().UnixNano()))
	if r.URL.Query().Get("download") == "1" {
		w.Header().Set("Content-Disposition", "attachment; filename="+strconv.Quote(filepath.Base(path)))
	}
	http.ServeContent(w, r, filepath.Base(path), info.ModTime(), f)
}
func (h *Handler) handleRevealSubtitle(w http.ResponseWriter, r *http.Request) {
	if !h.subtitleAvailable(w) {
		return
	}
	path, e := h.subtitles.AssetPath(r.Context(), r.PathValue("assetId"))
	if e == nil {
		e = revealInFileManagerFn(r.Context(), path)
	}
	if e != nil {
		writeAppError(w, 400, contracts.ErrorCodeBadRequest, "cannot reveal subtitle on this Server")
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

package server

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"curated-backend/internal/browserheaders"
	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/proxyenv"
	"curated-backend/internal/shellopen"
	"curated-backend/internal/storage"
	"curated-backend/internal/tasks"
	"curated-backend/internal/version"
)

type ScanStarter interface {
	StartScan(ctx context.Context, paths []string) (contracts.TaskDTO, error)
}

// MovieMetadataRefresher starts an async single-movie metadata rescrape and returns the scrape task.
type MovieMetadataRefresher interface {
	StartMovieMetadataRefresh(ctx context.Context, movieID string) (contracts.TaskDTO, error)
	StartMetadataRefreshForLibraryPaths(ctx context.Context, paths []string) (contracts.MetadataRefreshQueuedDTO, error)
}

// ActorProfileRefresher starts an async actor profile scrape (Metatube) for an existing library actor name.
type ActorProfileRefresher interface {
	StartActorProfileScrape(ctx context.Context, actorName string) (contracts.TaskDTO, error)
}

// OrganizeLibraryController exposes 整理库开关；值来自启动时合并的 library-config.cfg，PATCH 会写回该文件。
type OrganizeLibraryController interface {
	OrganizeLibrary() bool
	SetOrganizeLibrary(v bool) error
}

// ExtendedLibraryImportController toggles first-scan layout detection for newly added library roots (library-config.cfg).
type ExtendedLibraryImportController interface {
	ExtendedLibraryImport() bool
	SetExtendedLibraryImport(v bool) error
}

// AutoLibraryWatchController exposes whether fsnotify-driven library scans are allowed (library-config.cfg).
type AutoLibraryWatchController interface {
	AutoLibraryWatch() bool
	SetAutoLibraryWatch(v bool) error
}

// MetadataScrapeSettings exposes Metatube movie provider preference (empty = auto) and the list of valid provider names.
type MetadataScrapeSettings interface {
	MetadataMovieProvider() string
	SetMetadataMovieProvider(name string) error
	MetadataMovieProviderChain() []string
	SetMetadataMovieProviderChain(chain []string) error
	MetadataMovieScrapeMode() string
	SetMetadataMovieScrapeMode(mode string) error
	ListMetadataMovieProviders() []string
}

// ProviderHealthChecker provides health check capabilities for metadata providers.
type ProviderHealthChecker interface {
	ListProviders() []string
	CheckProviderHealth(ctx context.Context, name string) (status string, latencyMs int64, err error)
}

// ProxyController exposes and updates HTTP proxy configuration (library-config.cfg).
type ProxyController interface {
	Proxy() config.ProxyConfig
	SetProxy(cfg config.ProxyConfig) error
}

// BackendLogSettingsController exposes and updates backend log paths/level (library-config.cfg); restart applies to Zap.
type BackendLogSettingsController interface {
	BackendLogSettings() contracts.BackendLogSettingsDTO
	SetBackendLogPatch(p contracts.PatchBackendLogSettings) error
}

type PlayerSettingsController interface {
	PlayerSettings() contracts.PlayerSettingsDTO
	SetPlayerSettingsPatch(p contracts.PatchPlayerSettingsDTO) error
}

// LibraryWatchReloader rebuilds fsnotify watches after library roots change.
type LibraryWatchReloader interface {
	ReloadLibraryWatches(ctx context.Context) error
}

type PlaybackResolver interface {
	ResolvePlayback(ctx context.Context, movieID string) (contracts.PlaybackDescriptorDTO, error)
	CreatePlaybackSession(ctx context.Context, movieID string, mode contracts.PlaybackMode, startPositionSec float64) (contracts.PlaybackDescriptorDTO, error)
	ResolvePlaybackSessionFile(sessionID string, name string) (string, error)
	DeletePlaybackSession(sessionID string) error
}

type NativePlaybackLauncher interface {
	LaunchNativePlayback(ctx context.Context, movieID string, startPositionSec float64) (contracts.NativePlaybackLaunchDTO, error)
}

type Handler struct {
	cfg                      config.Config
	logger                   *zap.Logger
	store                    *storage.SQLiteStore
	tasks                    *tasks.Manager
	scanStarter              ScanStarter
	organizeLibraryCtl       OrganizeLibraryController
	extendedLibraryImportCtl ExtendedLibraryImportController
	autoLibraryWatchCtl      AutoLibraryWatchController
	metadataScrapeCtl        MetadataScrapeSettings
	providerHealthChecker    ProviderHealthChecker
	proxyCtl                 ProxyController
	backendLogCtl            BackendLogSettingsController
	playerSettingsCtl        PlayerSettingsController
	movieMetadataRefresher   MovieMetadataRefresher
	actorProfileRefresher    ActorProfileRefresher
	libraryWatchReloader     LibraryWatchReloader
	playbackResolver         PlaybackResolver
	nativePlaybackLauncher   NativePlaybackLauncher
}

type Deps struct {
	Cfg                      config.Config
	Logger                   *zap.Logger
	Store                    *storage.SQLiteStore
	Tasks                    *tasks.Manager
	ScanStarter              ScanStarter
	OrganizeLibraryCtl       OrganizeLibraryController
	ExtendedLibraryImportCtl ExtendedLibraryImportController
	AutoLibraryWatchCtl      AutoLibraryWatchController
	MetadataScrapeCtl        MetadataScrapeSettings
	ProviderHealthChecker    ProviderHealthChecker
	ProxyCtl                 ProxyController
	BackendLogCtl            BackendLogSettingsController
	PlayerSettingsCtl        PlayerSettingsController
	MovieMetadataRefresher   MovieMetadataRefresher
	ActorProfileRefresher    ActorProfileRefresher
	LibraryWatchReloader     LibraryWatchReloader
	PlaybackResolver         PlaybackResolver
	NativePlaybackLauncher   NativePlaybackLauncher
}

func NewHandler(deps Deps) *Handler {
	return &Handler{
		cfg:                      deps.Cfg,
		logger:                   deps.Logger,
		store:                    deps.Store,
		tasks:                    deps.Tasks,
		scanStarter:              deps.ScanStarter,
		organizeLibraryCtl:       deps.OrganizeLibraryCtl,
		extendedLibraryImportCtl: deps.ExtendedLibraryImportCtl,
		autoLibraryWatchCtl:      deps.AutoLibraryWatchCtl,
		metadataScrapeCtl:        deps.MetadataScrapeCtl,
		providerHealthChecker:    deps.ProviderHealthChecker,
		proxyCtl:                 deps.ProxyCtl,
		backendLogCtl:            deps.BackendLogCtl,
		playerSettingsCtl:        deps.PlayerSettingsCtl,
		movieMetadataRefresher:   deps.MovieMetadataRefresher,
		actorProfileRefresher:    deps.ActorProfileRefresher,
		libraryWatchReloader:     deps.LibraryWatchReloader,
		playbackResolver:         deps.PlaybackResolver,
		nativePlaybackLauncher:   deps.NativePlaybackLauncher,
	}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", h.handleHealth)
	mux.HandleFunc("GET /api/library/played-movies", h.handleListPlayedMovies)
	mux.HandleFunc("POST /api/library/played-movies/{movieId}", h.handleRecordPlayedMovie)
	mux.HandleFunc("GET /api/library/movies", h.handleListMovies)
	mux.HandleFunc("GET /api/library/actors", h.handleListActors)
	mux.HandleFunc("GET /api/library/actors/profile", h.handleGetActorProfile)
	mux.HandleFunc("POST /api/library/actors/scrape", h.handleScrapeActorProfile)
	mux.HandleFunc("PATCH /api/library/actors/tags", h.handlePatchActorUserTags)
	mux.HandleFunc("GET /api/library/movies/{movieId}/asset/preview/{index}", h.handleGetMoviePreviewAsset)
	mux.HandleFunc("GET /api/library/movies/{movieId}/asset/{kind}", h.handleGetMovieAsset)
	mux.HandleFunc("GET /api/library/movies/{movieId}/playback", h.handleGetMoviePlayback)
	mux.HandleFunc("POST /api/library/movies/{movieId}/playback-session", h.handleCreatePlaybackSession)
	mux.HandleFunc("POST /api/library/movies/{movieId}/native-play", h.handleLaunchNativePlayback)
	mux.HandleFunc("GET /api/library/movies/{movieId}/stream", h.handleStreamMovie)
	mux.HandleFunc("GET /api/playback/sessions/{sessionId}/hls/{file}", h.handleGetPlaybackSessionFile)
	mux.HandleFunc("DELETE /api/playback/sessions/{sessionId}", h.handleDeletePlaybackSession)
	mux.HandleFunc("POST /api/library/movies/{movieId}/reveal", h.handleRevealMovieInFileManager)
	mux.HandleFunc("GET /api/library/movies/{movieId}/comment", h.handleGetMovieComment)
	mux.HandleFunc("PUT /api/library/movies/{movieId}/comment", h.handlePutMovieComment)
	mux.HandleFunc("GET /api/library/movies/{movieId}", h.handleGetMovie)
	mux.HandleFunc("PATCH /api/library/movies/{movieId}", h.handlePatchMovie)
	mux.HandleFunc("POST /api/library/movies/{movieId}/restore", h.handleRestoreMovie)
	mux.HandleFunc("POST /api/library/movies/{movieId}/scrape", h.handleRefreshMovieMetadata)
	mux.HandleFunc("POST /api/library/metadata-scrape", h.handleMetadataScrapeByPaths)
	mux.HandleFunc("DELETE /api/library/movies/{movieId}", h.handleDeleteMovie)
	mux.HandleFunc("GET /api/settings", h.handleGetSettings)
	mux.HandleFunc("PATCH /api/settings", h.handlePatchSettings)
	mux.HandleFunc("POST /api/library/paths", h.handleAddLibraryPath)
	mux.HandleFunc("PATCH /api/library/paths/{id}", h.handlePatchLibraryPath)
	mux.HandleFunc("DELETE /api/library/paths/{id}", h.handleDeleteLibraryPath)
	mux.HandleFunc("POST /api/scans", h.handleStartScan)
	mux.HandleFunc("GET /api/tasks/recent", h.handleGetRecentTasks)
	mux.HandleFunc("GET /api/tasks/{taskId}", h.handleGetTaskStatus)

	mux.HandleFunc("GET /api/playback/progress", h.handleListPlaybackProgress)
	mux.HandleFunc("PUT /api/playback/progress/{movieId}", h.handlePutPlaybackProgress)
	mux.HandleFunc("DELETE /api/playback/progress/{movieId}", h.handleDeletePlaybackProgress)

	mux.HandleFunc("GET /api/curated-frames", h.handleListCuratedFrames)
	mux.HandleFunc("POST /api/curated-frames", h.handlePostCuratedFrame)
	mux.HandleFunc("GET /api/curated-frames/{id}/image", h.handleGetCuratedFrameImage)
	mux.HandleFunc("PATCH /api/curated-frames/{id}/tags", h.handlePatchCuratedFrameTags)
	mux.HandleFunc("DELETE /api/curated-frames/{id}", h.handleDeleteCuratedFrame)
	mux.HandleFunc("POST /api/curated-frames/export", h.handlePostCuratedFramesExport)

	mux.HandleFunc("POST /api/providers/ping", h.handlePingProvider)
	mux.HandleFunc("POST /api/providers/ping-all", h.handlePingAllProviders)

	mux.HandleFunc("POST /api/proxy/ping-javbus", h.handleProxyPingJavbus)
	mux.HandleFunc("POST /api/proxy/ping-google", h.handleProxyPingGoogle)

	return WithAccessLog(h.logger, withCORS(mux))
}

func (h *Handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, contracts.HealthDTO{
		Name:         "curated",
		Version:      version.Stamp(),
		Channel:      version.Channel,
		Transport:    "http",
		DatabasePath: h.cfg.DatabasePath,
	})
}

func (h *Handler) handleListMovies(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	limit, _ := strconv.Atoi(query.Get("limit"))
	offset, _ := strconv.Atoi(query.Get("offset"))
	if limit <= 0 {
		limit = 50
	}

	request := contracts.ListMoviesRequest{
		Mode:   query.Get("mode"),
		Query:  query.Get("q"),
		Actor:  query.Get("actor"),
		Studio: query.Get("studio"),
		Limit:  limit,
		Offset: offset,
	}

	result, err := h.store.ListMovies(r.Context(), request)
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("list movies failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to list movies")
		return
	}
	h.enrichMovieListItemsLocalPosters(r.Context(), result.Items)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) handleGetActorProfile(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "name is required")
		return
	}
	profile, err := h.store.GetActorProfile(r.Context(), name)
	if err != nil {
		if errors.Is(err, contracts.ErrActorNotFound) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "actor not found")
			return
		}
		if h.logger != nil {
			h.logger.Warn("get actor profile failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load actor profile")
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

func (h *Handler) handleScrapeActorProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.actorProfileRefresher == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "actor profile scrape not configured")
		return
	}
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "name is required")
		return
	}
	task, err := h.actorProfileRefresher.StartActorProfileScrape(r.Context(), name)
	if err != nil {
		switch {
		case errors.Is(err, contracts.ErrActorNotFound):
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "actor not found")
		case strings.Contains(err.Error(), "actor name is required"):
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, err.Error())
		default:
			if h.logger != nil {
				h.logger.Warn("start actor profile scrape failed", zap.Error(err))
			}
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to start actor profile scrape")
		}
		return
	}
	writeJSON(w, http.StatusAccepted, task)
}

func (h *Handler) handleListActors(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	if limit <= 0 {
		limit = 50
	}
	req := contracts.ListActorsRequest{
		Q:        strings.TrimSpace(q.Get("q")),
		ActorTag: strings.TrimSpace(q.Get("actorTag")),
		Sort:     strings.TrimSpace(q.Get("sort")),
		Limit:    limit,
		Offset:   offset,
	}
	result, err := h.store.ListActors(r.Context(), req)
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("list actors failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to list actors")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) handlePatchActorUserTags(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "name is required")
		return
	}
	body, err := io.ReadAll(r.Body)
	if r.Body != nil {
		_ = r.Body.Close()
	}
	if err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "failed to read body")
		return
	}
	var in contracts.PatchActorUserTagsBody
	if err := json.Unmarshal(body, &in); err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid json body")
		return
	}
	err = h.store.ReplaceActorUserTagsByName(r.Context(), name, in.UserTags)
	if err != nil {
		if errors.Is(err, contracts.ErrActorNotFound) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "actor not found")
			return
		}
		if errors.Is(err, storage.ErrInvalidUserTags) {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, err.Error())
			return
		}
		if h.logger != nil {
			h.logger.Warn("patch actor user tags failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to update actor tags")
		return
	}
	item, err := h.store.ActorListItemByName(r.Context(), name)
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("load actor after tag patch failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load actor")
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) handleGetMovie(w http.ResponseWriter, r *http.Request) {
	movieID := r.PathValue("movieId")
	if movieID == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "movieId is required")
		return
	}

	movie, err := h.store.GetMovieDetail(r.Context(), movieID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie not found")
			return
		}
		if h.logger != nil {
			h.logger.Warn("get movie detail failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load movie")
		return
	}
	h.enrichMovieDetailLocalPosters(r.Context(), &movie)
	writeJSON(w, http.StatusOK, movie)
}

func (h *Handler) handleStreamMovie(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	movieID := strings.TrimSpace(r.PathValue("movieId"))
	if movieID == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "movieId is required")
		return
	}

	f, dispName, err := h.store.OpenMovieVideoFile(r.Context(), movieID)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrMovieVideoNotFound):
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie or video file not found")
		case errors.Is(err, storage.ErrMovieVideoNoLocation):
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie has no video file")
		case errors.Is(err, storage.ErrMovieVideoForbidden):
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie or video file not found")
		case errors.Is(err, storage.ErrMovieVideoNotFile):
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie or video file not found")
		default:
			if h.logger != nil {
				h.logger.Warn("open movie video stream failed", zap.Error(err))
			}
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to open video file")
		}
		return
	}
	defer func() { _ = f.Close() }()

	st, err := f.Stat()
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("stat movie video failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to read video file")
		return
	}

	http.ServeContent(w, r, dispName, st.ModTime(), f)
}

func (h *Handler) handleGetMoviePlayback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}

	movieID := strings.TrimSpace(r.PathValue("movieId"))
	if movieID == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "movieId is required")
		return
	}

	if h.playbackResolver != nil {
		dto, err := h.playbackResolver.ResolvePlayback(r.Context(), movieID)
		if err == nil {
			writeJSON(w, http.StatusOK, dto)
			return
		}
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, storage.ErrMovieVideoNotFound) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie not found")
			return
		}
		if h.logger != nil {
			h.logger.Warn("resolve playback failed", zap.Error(err), zap.String("movieId", movieID))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to resolve playback")
		return
	}

	detail, err := h.store.GetMovieDetail(r.Context(), movieID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie not found")
			return
		}
		if h.logger != nil {
			h.logger.Warn("get movie playback detail failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load movie")
		return
	}

	progress, err := h.store.GetPlaybackProgress(r.Context(), movieID)
	if err != nil && h.logger != nil {
		h.logger.Warn("get playback progress failed", zap.Error(err), zap.String("movieId", movieID))
	}

	streamURL := "/api/library/movies/" + url.PathEscape(movieID) + "/stream"
	fileName := filepath.Base(strings.TrimSpace(detail.Location))
	mimeType := mime.TypeByExtension(strings.ToLower(filepath.Ext(fileName)))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	dto := contracts.PlaybackDescriptorDTO{
		MovieID:        movieID,
		Mode:           contracts.PlaybackModeDirect,
		URL:            streamURL,
		MimeType:       mimeType,
		FileName:       fileName,
		DurationSec:    float64(detail.RuntimeMinutes) * 60,
		CanDirectPlay:  true,
		AudioTracks:    []contracts.PlaybackAudioTrackDTO{},
		SubtitleTracks: []contracts.PlaybackSubtitleTrackDTO{},
	}
	if progress != nil && progress.PositionSec > 0 {
		dto.ResumePositionSec = progress.PositionSec
		if progress.DurationSec > 0 {
			dto.DurationSec = progress.DurationSec
		}
	}

	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) handleCreatePlaybackSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.playbackResolver == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "playback session manager not configured")
		return
	}
	movieID := strings.TrimSpace(r.PathValue("movieId"))
	if movieID == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "movieId is required")
		return
	}
	var body contracts.CreatePlaybackSessionRequest
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid json body")
			return
		}
	}
	if body.Mode == "" {
		body.Mode = contracts.PlaybackModeDirect
	}
	dto, err := h.playbackResolver.CreatePlaybackSession(r.Context(), movieID, body.Mode, body.StartPositionSec)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, storage.ErrMovieVideoNotFound) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie not found")
			return
		}
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, dto)
}

func (h *Handler) handleLaunchNativePlayback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.nativePlaybackLauncher == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "native playback is not configured")
		return
	}
	movieID := strings.TrimSpace(r.PathValue("movieId"))
	if movieID == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "movieId is required")
		return
	}
	var body contracts.NativePlaybackLaunchRequest
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid json body")
			return
		}
	}
	dto, err := h.nativePlaybackLauncher.LaunchNativePlayback(r.Context(), movieID, body.StartPositionSec)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, storage.ErrMovieVideoNotFound) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie not found")
			return
		}
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) handleGetPlaybackSessionFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.playbackResolver == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "playback session manager not configured")
		return
	}
	sessionID := strings.TrimSpace(r.PathValue("sessionId"))
	name := strings.TrimSpace(r.PathValue("file"))
	if sessionID == "" || name == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "sessionId and file are required")
		return
	}
	absPath, err := h.playbackResolver.ResolvePlaybackSessionFile(sessionID, name)
	if err != nil {
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "session file not found")
		return
	}
	http.ServeFile(w, r, absPath)
}

func (h *Handler) handleDeletePlaybackSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.playbackResolver == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "playback session manager not configured")
		return
	}
	sessionID := strings.TrimSpace(r.PathValue("sessionId"))
	if sessionID == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "sessionId is required")
		return
	}
	if err := h.playbackResolver.DeletePlaybackSession(sessionID); err != nil {
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "playback session not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleRevealMovieInFileManager(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	movieID := strings.TrimSpace(r.PathValue("movieId"))
	if movieID == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "movieId is required")
		return
	}

	absPath, err := h.store.ResolvePrimaryVideoPath(r.Context(), movieID)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrMovieVideoNotFound):
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie or video file not found")
		case errors.Is(err, storage.ErrMovieVideoNoLocation):
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie has no video file")
		case errors.Is(err, storage.ErrMovieVideoForbidden):
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie or video file not found")
		case errors.Is(err, storage.ErrMovieVideoNotFile):
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie or video file not found")
		default:
			if h.logger != nil {
				h.logger.Warn("resolve primary video for reveal failed", zap.Error(err))
			}
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to resolve video path")
		}
		return
	}

	// Use a detached timeout — not r.Context(): when the 204 is sent the request context
	// is often cancelled immediately and would kill the explorer child from exec.CommandContext.
	revealCtx, revealCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer revealCancel()
	if err := shellopen.RevealInFileManager(revealCtx, absPath); err != nil {
		if h.logger != nil {
			h.logger.Warn("reveal in file manager failed", zap.Error(err), zap.String("path", absPath))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to open file manager: "+err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleGetMovieComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	movieID := strings.TrimSpace(r.PathValue("movieId"))
	if movieID == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "movieId is required")
		return
	}
	ok, err := h.store.MovieRowExists(r.Context(), movieID)
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("movie comment get: exists check failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load movie")
		return
	}
	if !ok {
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie not found")
		return
	}
	dto, err := h.store.GetMovieComment(r.Context(), movieID)
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("get movie comment failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load comment")
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) handlePutMovieComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	movieID := strings.TrimSpace(r.PathValue("movieId"))
	if movieID == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "movieId is required")
		return
	}
	trashed, terr := h.store.IsMovieTrashed(r.Context(), movieID)
	if terr != nil {
		if h.logger != nil {
			h.logger.Warn("put movie comment: trashed check failed", zap.Error(terr))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load movie")
		return
	}
	if trashed {
		writeAppError(w, http.StatusConflict, contracts.ErrorCodeConflict, "movie is in trash")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
	if err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid body")
		return
	}
	var in contracts.PutMovieCommentBody
	if err := json.Unmarshal(body, &in); err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid JSON")
		return
	}
	dto, err := h.store.UpsertMovieComment(r.Context(), movieID, in.Body)
	if err != nil {
		if errors.Is(err, storage.ErrMovieNotFound) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie not found")
			return
		}
		if errors.Is(err, storage.ErrMovieCommentTooLong) {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "comment body too long")
			return
		}
		if h.logger != nil {
			h.logger.Warn("put movie comment failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to save comment")
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func parsePatchMovieInput(body []byte) (contracts.PatchMovieInput, error) {
	var in contracts.PatchMovieInput
	if len(bytes.TrimSpace(body)) == 0 {
		return in, errors.New("empty body")
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(body, &m); err != nil {
		return in, err
	}
	if raw, ok := m["isFavorite"]; ok {
		var b bool
		if err := json.Unmarshal(raw, &b); err != nil {
			return in, err
		}
		in.Favorite = &b
	}
	if raw, ok := m["rating"]; ok {
		in.UserRatingSet = true
		if string(bytes.TrimSpace(raw)) == "null" {
			in.UserRatingClear = true
		} else {
			var v float64
			if err := json.Unmarshal(raw, &v); err != nil {
				return in, err
			}
			in.UserRating = v
		}
	}
	if raw, ok := m["userTags"]; ok {
		in.UserTagsSet = true
		var tags []string
		if err := json.Unmarshal(raw, &tags); err != nil {
			return in, err
		}
		in.UserTags = tags
	}
	if raw, ok := m["metadataTags"]; ok {
		in.MetadataTagsSet = true
		var tags []string
		if err := json.Unmarshal(raw, &tags); err != nil {
			return in, err
		}
		in.MetadataTags = tags
	}
	if err := parsePatchOptionalStringField(m, "userTitle", &in.UserTitleSet, &in.UserTitleClear, &in.UserTitle); err != nil {
		return in, err
	}
	if err := parsePatchOptionalStringField(m, "userStudio", &in.UserStudioSet, &in.UserStudioClear, &in.UserStudio); err != nil {
		return in, err
	}
	if err := parsePatchOptionalStringField(m, "userSummary", &in.UserSummarySet, &in.UserSummaryClear, &in.UserSummary); err != nil {
		return in, err
	}
	if err := parsePatchOptionalStringField(m, "userReleaseDate", &in.UserReleaseDateSet, &in.UserReleaseDateClear, &in.UserReleaseDate); err != nil {
		return in, err
	}
	if raw, ok := m["userRuntimeMinutes"]; ok {
		in.UserRuntimeMinutesSet = true
		t := bytes.TrimSpace(raw)
		if len(t) == 0 || string(t) == "null" {
			in.UserRuntimeMinutesClear = true
		} else {
			var f float64
			if err := json.Unmarshal(raw, &f); err != nil {
				return in, err
			}
			if math.IsNaN(f) || math.IsInf(f, 0) || f < 0 || f > 99999 || f != math.Trunc(f) {
				return in, errors.New("invalid userRuntimeMinutes")
			}
			in.UserRuntimeMinutes = int(f)
		}
	}
	return in, nil
}

func parsePatchOptionalStringField(
	m map[string]json.RawMessage,
	key string,
	set *bool,
	clear *bool,
	out *string,
) error {
	raw, ok := m[key]
	if !ok {
		return nil
	}
	*set = true
	t := bytes.TrimSpace(raw)
	if len(t) == 0 || string(t) == "null" {
		*clear = true
		return nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return err
	}
	s = strings.TrimSpace(s)
	if s == "" {
		*clear = true
		return nil
	}
	*out = s
	return nil
}

var patchUserReleaseDateRx = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func validatePatchMovieDisplay(in contracts.PatchMovieInput) error {
	const maxSummary = 120_000
	if in.UserSummarySet && !in.UserSummaryClear && len(in.UserSummary) > maxSummary {
		return errors.New("userSummary too long")
	}
	if in.UserReleaseDateSet && !in.UserReleaseDateClear && in.UserReleaseDate != "" {
		if !patchUserReleaseDateRx.MatchString(in.UserReleaseDate) {
			return errors.New("userReleaseDate must be YYYY-MM-DD")
		}
	}
	if in.UserRuntimeMinutesSet && !in.UserRuntimeMinutesClear {
		if in.UserRuntimeMinutes < 0 || in.UserRuntimeMinutes > 99999 {
			return errors.New("userRuntimeMinutes out of range")
		}
	}
	return nil
}

func patchMovieInputHasFields(in contracts.PatchMovieInput) bool {
	if in.Favorite != nil {
		return true
	}
	if in.UserRatingSet {
		return true
	}
	if in.UserTagsSet {
		return true
	}
	if in.MetadataTagsSet {
		return true
	}
	if in.UserTitleSet || in.UserStudioSet || in.UserSummarySet || in.UserReleaseDateSet || in.UserRuntimeMinutesSet {
		return true
	}
	return false
}

func (h *Handler) handlePatchMovie(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	movieID := strings.TrimSpace(r.PathValue("movieId"))
	if movieID == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "movieId is required")
		return
	}

	trashed, terr := h.store.IsMovieTrashed(r.Context(), movieID)
	if terr != nil {
		if h.logger != nil {
			h.logger.Warn("patch movie: trashed check failed", zap.Error(terr))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load movie")
		return
	}
	if trashed {
		writeAppError(w, http.StatusConflict, contracts.ErrorCodeConflict, "movie is in trash")
		return
	}

	body, err := io.ReadAll(r.Body)
	if r.Body != nil {
		_ = r.Body.Close()
	}
	if err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "failed to read body")
		return
	}

	in, err := parsePatchMovieInput(body)
	if err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid json body")
		return
	}
	if !patchMovieInputHasFields(in) {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "no fields to update")
		return
	}
	if in.UserRatingSet && !in.UserRatingClear && (in.UserRating < 0 || in.UserRating > 5) {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, storage.ErrInvalidUserRating.Error())
		return
	}
	if err := validatePatchMovieDisplay(in); err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, err.Error())
		return
	}

	err = h.store.PatchMovieUserPrefs(r.Context(), movieID, in)
	if err != nil {
		if errors.Is(err, storage.ErrInvalidUserTags) {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, err.Error())
			return
		}
		if errors.Is(err, storage.ErrMovieNotFoundForPatch) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie not found")
			return
		}
		if errors.Is(err, storage.ErrInvalidUserRating) {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, err.Error())
			return
		}
		if h.logger != nil {
			h.logger.Warn("patch movie failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to patch movie")
		return
	}

	movie, err := h.store.GetMovieDetail(r.Context(), movieID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie not found")
			return
		}
		if h.logger != nil {
			h.logger.Warn("get movie after patch failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load movie")
		return
	}
	h.enrichMovieDetailLocalPosters(r.Context(), &movie)
	writeJSON(w, http.StatusOK, movie)
}

func (h *Handler) handleRefreshMovieMetadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.movieMetadataRefresher == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "metadata refresh not configured")
		return
	}
	movieID := strings.TrimSpace(r.PathValue("movieId"))
	if movieID == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "movieId is required")
		return
	}
	trashed, terr := h.store.IsMovieTrashed(r.Context(), movieID)
	if terr != nil {
		if h.logger != nil {
			h.logger.Warn("refresh metadata: trashed check failed", zap.Error(terr))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load movie")
		return
	}
	if trashed {
		writeAppError(w, http.StatusConflict, contracts.ErrorCodeConflict, "movie is in trash")
		return
	}

	task, err := h.movieMetadataRefresher.StartMovieMetadataRefresh(r.Context(), movieID)
	if err != nil {
		switch {
		case errors.Is(err, contracts.ErrScrapeMovieNotFound):
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie not found")
		case errors.Is(err, contracts.ErrScrapeMovieNoCode), errors.Is(err, contracts.ErrScrapeMovieNoLocation):
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, err.Error())
		default:
			if h.logger != nil {
				h.logger.Warn("start movie metadata refresh failed", zap.Error(err))
			}
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to start metadata refresh")
		}
		return
	}

	writeJSON(w, http.StatusAccepted, task)
}

func (h *Handler) handleMetadataScrapeByPaths(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.movieMetadataRefresher == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "metadata refresh not configured")
		return
	}

	var body contracts.StartMetadataRefreshByPathsRequest
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid json body")
			return
		}
	}
	if len(body.Paths) == 0 {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "paths must contain at least one entry")
		return
	}

	dto, err := h.movieMetadataRefresher.StartMetadataRefreshForLibraryPaths(r.Context(), body.Paths)
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("bulk metadata refresh failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to queue metadata refresh")
		return
	}

	writeJSON(w, http.StatusAccepted, dto)
}

// handleRestoreMovie clears trashed_at. 204 on success; 404 if missing; 400 if not in trash.
func (h *Handler) handleRestoreMovie(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	movieID := strings.TrimSpace(r.PathValue("movieId"))
	if movieID == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "movieId is required")
		return
	}
	err := h.store.RestoreMovie(r.Context(), movieID)
	if err != nil {
		if errors.Is(err, storage.ErrMovieNotFound) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie not found")
			return
		}
		if errors.Is(err, storage.ErrMovieNotInTrash) {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "movie is not in trash")
			return
		}
		if h.logger != nil {
			h.logger.Warn("restore movie failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to restore movie")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleDeleteMovie without ?permanent=true moves the movie to trash (204). With permanent=true,
// removes DB rows and on-disk files only if already trashed (see storage.DeleteMoviePermanently).
func (h *Handler) handleDeleteMovie(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	movieID := strings.TrimSpace(r.PathValue("movieId"))
	if movieID == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "movieId is required")
		return
	}
	permanent := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("permanent")), "true")
	if permanent {
		err := h.store.DeleteMoviePermanently(r.Context(), movieID, strings.TrimSpace(h.cfg.CacheDir))
		if err != nil {
			if errors.Is(err, storage.ErrMovieNotFound) {
				writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie not found")
				return
			}
			if errors.Is(err, storage.ErrMovieNotInTrash) {
				writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "movie must be in trash before permanent delete")
				return
			}
			if h.logger != nil {
				h.logger.Warn("permanent delete movie failed", zap.Error(err))
			}
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to delete movie")
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	err := h.store.TrashMovie(r.Context(), movieID)
	if err != nil {
		if errors.Is(err, storage.ErrMovieNotFound) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie not found")
			return
		}
		if h.logger != nil {
			h.logger.Warn("trash movie failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to move movie to trash")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) buildSettingsDTO(ctx context.Context) (contracts.SettingsDTO, error) {
	libraryPaths, err := h.store.ListLibraryPaths(ctx)
	if err != nil {
		return contracts.SettingsDTO{}, err
	}
	org := h.cfg.OrganizeLibrary
	if h.organizeLibraryCtl != nil {
		org = h.organizeLibraryCtl.OrganizeLibrary()
	}
	autoWatch := h.cfg.AutoLibraryWatch
	if h.autoLibraryWatchCtl != nil {
		autoWatch = h.autoLibraryWatchCtl.AutoLibraryWatch()
	}
	extImp := h.cfg.ExtendedLibraryImport
	if h.extendedLibraryImportCtl != nil {
		extImp = h.extendedLibraryImportCtl.ExtendedLibraryImport()
	}
	dto := contracts.SettingsDTO{
		LibraryPaths: libraryPaths,
		Player: contracts.PlayerSettingsDTO{
			HardwareDecode:      h.cfg.Player.HardwareDecode,
			NativePlayerEnabled: h.cfg.Player.NativePlayerEnabled,
			NativePlayerCommand: h.cfg.Player.NativePlayerCommand,
			StreamPushEnabled:   h.cfg.Player.StreamPushEnabled,
			FFmpegCommand:       h.cfg.Player.FFmpegCommand,
			PreferNativePlayer:  h.cfg.Player.PreferNativePlayer,
			SeekForwardStepSec:  h.cfg.Player.SeekForwardStepSec,
			SeekBackwardStepSec: h.cfg.Player.SeekBackwardStepSec,
		},
		OrganizeLibrary:        org,
		ExtendedLibraryImport:  extImp,
		AutoLibraryWatch:       autoWatch,
		MetadataMovieProviders: []string{},
	}
	if strings.TrimSpace(dto.Player.NativePlayerCommand) == "" {
		dto.Player.NativePlayerCommand = "mpv"
	}
	if strings.TrimSpace(dto.Player.FFmpegCommand) == "" {
		dto.Player.FFmpegCommand = "ffmpeg"
	}
	if dto.Player.SeekForwardStepSec <= 0 {
		dto.Player.SeekForwardStepSec = 10
	}
	if dto.Player.SeekBackwardStepSec <= 0 {
		dto.Player.SeekBackwardStepSec = 10
	}
	if h.playerSettingsCtl != nil {
		dto.Player = h.playerSettingsCtl.PlayerSettings()
	}
	if h.metadataScrapeCtl != nil {
		dto.MetadataMovieProvider = h.metadataScrapeCtl.MetadataMovieProvider()
		dto.MetadataMovieProviderChain = h.metadataScrapeCtl.MetadataMovieProviderChain()
		dto.MetadataMovieScrapeMode = h.metadataScrapeCtl.MetadataMovieScrapeMode()
		if list := h.metadataScrapeCtl.ListMetadataMovieProviders(); list != nil {
			dto.MetadataMovieProviders = list
		}
	}
	if h.proxyCtl != nil {
		p := h.proxyCtl.Proxy()
		dto.Proxy = contracts.ProxySettingsDTO{
			Enabled:  p.Enabled,
			URL:      p.URL,
			Username: p.Username,
			Password: p.Password,
		}
	}
	if h.backendLogCtl != nil {
		dto.BackendLog = h.backendLogCtl.BackendLogSettings()
	}
	return dto, nil
}

func patchBackendLogHasChanges(p *contracts.PatchBackendLogSettings) bool {
	if p == nil {
		return false
	}
	return p.LogDir != nil || p.LogFilePrefix != nil || p.LogMaxAgeDays != nil || p.LogLevel != nil
}

func movieProviderNameAllowed(want string, allowed []string) bool {
	if strings.TrimSpace(want) == "" {
		return true
	}
	lw := strings.ToLower(strings.TrimSpace(want))
	for _, a := range allowed {
		if strings.ToLower(strings.TrimSpace(a)) == lw {
			return true
		}
	}
	return false
}

func (h *Handler) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	dto, err := h.buildSettingsDTO(r.Context())
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("build settings dto failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to list library paths")
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) handlePatchSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.organizeLibraryCtl == nil && h.metadataScrapeCtl == nil && h.autoLibraryWatchCtl == nil && h.extendedLibraryImportCtl == nil && h.proxyCtl == nil && h.backendLogCtl == nil && h.playerSettingsCtl == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "settings runtime not available")
		return
	}

	var body contracts.PatchSettingsRequest
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid json body")
			return
		}
	}

	if body.OrganizeLibrary == nil && body.AutoLibraryWatch == nil && body.MetadataMovieProvider == nil && body.ExtendedLibraryImport == nil && body.MetadataMovieProviderChain == nil && body.MetadataMovieScrapeMode == nil && body.Proxy == nil && !patchBackendLogHasChanges(body.BackendLog) && body.Player == nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "no supported fields to update")
		return
	}

	if body.OrganizeLibrary != nil {
		if h.organizeLibraryCtl == nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "organize library settings not available")
			return
		}
		if err := h.organizeLibraryCtl.SetOrganizeLibrary(*body.OrganizeLibrary); err != nil {
			if h.logger != nil {
				h.logger.Warn("failed to persist organizeLibrary", zap.Error(err))
			}
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to save library settings")
			return
		}
	}

	if body.ExtendedLibraryImport != nil {
		if h.extendedLibraryImportCtl == nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "extended library import settings not available")
			return
		}
		if err := h.extendedLibraryImportCtl.SetExtendedLibraryImport(*body.ExtendedLibraryImport); err != nil {
			if h.logger != nil {
				h.logger.Warn("failed to persist extendedLibraryImport", zap.Error(err))
			}
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to save library settings")
			return
		}
	}

	if body.AutoLibraryWatch != nil {
		if h.autoLibraryWatchCtl == nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "auto library watch settings not available")
			return
		}
		if err := h.autoLibraryWatchCtl.SetAutoLibraryWatch(*body.AutoLibraryWatch); err != nil {
			if h.logger != nil {
				h.logger.Warn("failed to persist autoLibraryWatch", zap.Error(err))
			}
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to save library settings")
			return
		}
	}

	if body.MetadataMovieProvider != nil {
		if h.metadataScrapeCtl == nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "metadata scrape settings not available")
			return
		}
		name := strings.TrimSpace(*body.MetadataMovieProvider)
		if name != "" {
			allowed := h.metadataScrapeCtl.ListMetadataMovieProviders()
			if len(allowed) == 0 || !movieProviderNameAllowed(name, allowed) {
				writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "unknown metadataMovieProvider")
				return
			}
		}
		if err := h.metadataScrapeCtl.SetMetadataMovieProvider(name); err != nil {
			if h.logger != nil {
				h.logger.Warn("failed to persist metadataMovieProvider", zap.Error(err))
			}
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to save library settings")
			return
		}
	}

	if body.MetadataMovieProviderChain != nil {
		if h.metadataScrapeCtl == nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "metadata scrape settings not available")
			return
		}
		chain := *body.MetadataMovieProviderChain
		// Validate chain providers
		if len(chain) > 0 {
			allowed := h.metadataScrapeCtl.ListMetadataMovieProviders()
			for _, name := range chain {
				name = strings.TrimSpace(name)
				if name == "" {
					continue
				}
				if len(allowed) == 0 || !movieProviderNameAllowed(name, allowed) {
					writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "unknown provider in chain: "+name)
					return
				}
			}
		}
		if err := h.metadataScrapeCtl.SetMetadataMovieProviderChain(chain); err != nil {
			if h.logger != nil {
				h.logger.Warn("failed to persist metadataMovieProviderChain", zap.Error(err))
			}
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to save library settings")
			return
		}
	}

	if body.MetadataMovieScrapeMode != nil {
		if h.metadataScrapeCtl == nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "metadata scrape settings not available")
			return
		}
		mode := strings.TrimSpace(strings.ToLower(*body.MetadataMovieScrapeMode))
		if mode != "auto" && mode != "specified" && mode != "chain" {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid metadataMovieScrapeMode")
			return
		}
		if err := h.metadataScrapeCtl.SetMetadataMovieScrapeMode(mode); err != nil {
			if h.logger != nil {
				h.logger.Warn("failed to persist metadataMovieScrapeMode", zap.Error(err))
			}
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to save library settings")
			return
		}
	}

	if body.Proxy != nil {
		if h.proxyCtl == nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "proxy settings not available")
			return
		}
		cfg := config.ProxyConfig{
			Enabled:  body.Proxy.Enabled,
			URL:      strings.TrimSpace(body.Proxy.URL),
			Username: strings.TrimSpace(body.Proxy.Username),
			Password: body.Proxy.Password,
		}
		if cfg.Enabled && cfg.URL == "" {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "proxy URL is required when enabled")
			return
		}
		if err := h.proxyCtl.SetProxy(cfg); err != nil {
			if h.logger != nil {
				h.logger.Warn("failed to persist proxy settings", zap.Error(err))
			}
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to save proxy settings")
			return
		}
	}

	if body.BackendLog != nil && patchBackendLogHasChanges(body.BackendLog) {
		if h.backendLogCtl == nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "backend log settings not available")
			return
		}
		if err := h.backendLogCtl.SetBackendLogPatch(*body.BackendLog); err != nil {
			if h.logger != nil {
				h.logger.Warn("failed to persist backend log settings", zap.Error(err))
			}
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, err.Error())
			return
		}
	}

	if body.Player != nil {
		if h.playerSettingsCtl == nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "player settings not available")
			return
		}
		if err := h.playerSettingsCtl.SetPlayerSettingsPatch(*body.Player); err != nil {
			if h.logger != nil {
				h.logger.Warn("failed to persist player settings", zap.Error(err))
			}
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, err.Error())
			return
		}
	}

	dto, err := h.buildSettingsDTO(r.Context())
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("build settings dto failed after patch", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to list library paths")
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) handleAddLibraryPath(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}

	var body contracts.AddLibraryPathRequest
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid json body")
			return
		}
	}

	path := strings.TrimSpace(body.Path)
	if path == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "path is required")
		return
	}

	dto, err := h.store.AddLibraryPath(r.Context(), path, strings.TrimSpace(body.Title))
	if err != nil {
		if errors.Is(err, storage.ErrLibraryPathNotAbsolute) {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "library path must be an absolute path")
			return
		}
		if errors.Is(err, storage.ErrLibraryPathDuplicate) {
			writeAppError(w, http.StatusConflict, contracts.ErrorCodeConflict, "library path already exists")
			return
		}
		h.logger.Error("add library path failed", zap.Error(err))
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to add library path")
		return
	}

	h.reloadLibraryWatchIfAny(r.Context())

	resp := contracts.AddLibraryPathResponse{LibraryPathDTO: dto}
	if h.scanStarter != nil {
		if task, err := h.scanStarter.StartScan(r.Context(), []string{dto.Path}); err != nil {
			if errors.Is(err, contracts.ErrScanAlreadyRunning) {
				if h.logger != nil {
					h.logger.Warn("add library path: initial scan skipped (scan already in progress)", zap.String("path", dto.Path))
				}
			} else if h.logger != nil {
				h.logger.Warn("add library path: failed to start initial scan", zap.Error(err), zap.String("path", dto.Path))
			}
		} else {
			resp.ScanTask = &task
		}
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (h *Handler) handleDeleteLibraryPath(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}

	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "id is required")
		return
	}

	pruned, err := h.store.DeleteLibraryPathAndPruneOrphanMovies(r.Context(), id)
	if err != nil {
		if errors.Is(err, storage.ErrLibraryPathNotFound) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "library path not found")
			return
		}
		h.logger.Error("delete library path failed", zap.Error(err))
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to delete library path")
		return
	}
	if pruned > 0 {
		h.logger.Info("library path removed; pruned orphan movies", zap.String("id", id), zap.Int("prunedMovies", pruned))
	}

	h.reloadLibraryWatchIfAny(r.Context())
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handlePatchLibraryPath(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}

	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "id is required")
		return
	}

	var body contracts.UpdateLibraryPathRequest
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid json body")
			return
		}
	}

	dto, err := h.store.UpdateLibraryPathTitle(r.Context(), id, body.Title)
	if err != nil {
		if errors.Is(err, storage.ErrLibraryPathNotFound) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "library path not found")
			return
		}
		h.logger.Error("update library path title failed", zap.Error(err))
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to update library path")
		return
	}

	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) handleStartScan(w http.ResponseWriter, r *http.Request) {
	var request contracts.StartScanRequest
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid json body")
			return
		}
	}

	task, err := h.scanStarter.StartScan(r.Context(), request.Paths)
	if err != nil {
		if errors.Is(err, contracts.ErrScanAlreadyRunning) {
			writeAppError(w, http.StatusConflict, contracts.ErrorCodeConflict, "scan already in progress")
			return
		}
		if h.logger != nil {
			h.logger.Warn("start scan failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to start scan")
		return
	}

	writeJSON(w, http.StatusAccepted, task)
}

func (h *Handler) handleGetTaskStatus(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskId")
	if taskID == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "taskId is required")
		return
	}

	task, ok := h.tasks.Get(taskID)
	if !ok {
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "task not found")
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (h *Handler) handleGetRecentTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	limit := 30
	if q := strings.TrimSpace(r.URL.Query().Get("limit")); q != "" {
		if n, err := strconv.Atoi(q); err == nil && n > 0 {
			limit = n
		}
	}
	tasks := h.tasks.ListRecentFinished(limit)
	writeJSON(w, http.StatusOK, contracts.RecentTasksDTO{Tasks: tasks})
}

func (h *Handler) reloadLibraryWatchIfAny(ctx context.Context) {
	if h.libraryWatchReloader == nil {
		return
	}
	if err := h.libraryWatchReloader.ReloadLibraryWatches(ctx); err != nil && h.logger != nil {
		h.logger.Warn("library watch reload failed", zap.Error(err))
	}
}

func ListenAndServe(ctx context.Context, addr string, handler http.Handler, logger *zap.Logger) error {
	srv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		<-ctx.Done()
		if logger != nil {
			logger.Info("HTTP server shutdown initiated")
		}
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil && logger != nil {
			logger.Warn("HTTP server shutdown error", zap.Error(err))
		} else if logger != nil {
			logger.Info("HTTP server stopped")
		}
	}()

	if logger != nil {
		logger.Info("HTTP server listening", zap.String("addr", addr))
	}
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeAppError(w http.ResponseWriter, status int, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(contracts.AppError{
		Code:      code,
		Message:   message,
		Retryable: status >= 500,
	})
}

func (h *Handler) handlePingProvider(w http.ResponseWriter, r *http.Request) {
	var req contracts.PingProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("invalid ping provider request body", zap.Error(err))
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "Invalid request body")
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "Provider name is required")
		return
	}

	// Verify provider exists
	found := false
	for _, p := range h.providerHealthChecker.ListProviders() {
		if strings.EqualFold(p, name) {
			found = true
			name = p // Use the canonical name from the list
			break
		}
	}
	if !found {
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodeProviderNotFound, "Provider not found: "+name)
		return
	}

	status, latencyMs, err := h.providerHealthChecker.CheckProviderHealth(r.Context(), name)
	if err != nil {
		h.logger.Warn("provider health check failed", zap.String("provider", name), zap.Error(err))
	}

	dto := contracts.ProviderHealthDTO{
		Name:      name,
		Status:    contracts.ProviderHealthStatus(status),
		LatencyMs: latencyMs,
	}
	if err != nil {
		dto.Message = err.Error()
	}

	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) handlePingAllProviders(w http.ResponseWriter, r *http.Request) {
	providers := h.providerHealthChecker.ListProviders()
	results := make([]contracts.ProviderHealthDTO, 0, len(providers))

	okCount := 0
	failCount := 0

	for _, name := range providers {
		status, latencyMs, err := h.providerHealthChecker.CheckProviderHealth(r.Context(), name)
		dto := contracts.ProviderHealthDTO{
			Name:      name,
			Status:    contracts.ProviderHealthStatus(status),
			LatencyMs: latencyMs,
		}
		if err != nil {
			dto.Message = err.Error()
			failCount++
		} else if status == "ok" {
			okCount++
		}
		results = append(results, dto)
	}

	writeJSON(w, http.StatusOK, contracts.PingAllProvidersResponse{
		Providers: results,
		Total:     len(providers),
		OK:        okCount,
		Fail:      failCount,
	})
}

func (h *Handler) handleProxyPingJavbus(w http.ResponseWriter, r *http.Request) {
	h.handleProxyPingURL(w, r, "https://www.javbus.com/")
}

func (h *Handler) handleProxyPingGoogle(w http.ResponseWriter, r *http.Request) {
	h.handleProxyPingURL(w, r, "https://www.google.com/")
}

// handleProxyPingURL GETs targetURL using optional body.proxy or persisted proxy (same contract as ping-javbus).
func (h *Handler) handleProxyPingURL(w http.ResponseWriter, r *http.Request, targetURL string) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "Method not allowed")
		return
	}

	var req contracts.ProxyJavBusPingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		h.logger.Warn("invalid proxy outbound ping body", zap.Error(err))
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "Invalid request body")
		return
	}

	var cfg config.ProxyConfig
	if req.Proxy != nil {
		cfg = config.ProxyConfig{
			Enabled:  req.Proxy.Enabled,
			URL:      strings.TrimSpace(req.Proxy.URL),
			Username: strings.TrimSpace(req.Proxy.Username),
			Password: req.Proxy.Password,
		}
	} else {
		if h.proxyCtl == nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "proxy settings not available")
			return
		}
		cfg = h.proxyCtl.Proxy()
	}

	if cfg.Enabled && strings.TrimSpace(cfg.URL) == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "Proxy URL is required when proxy is enabled")
		return
	}

	// 与设置页「测试 JavBus / Google 连通」一致：超过 5s 即视为超时失败
	const proxyOutboundPingTimeout = 5 * time.Second
	client, err := proxyenv.NewHTTPClientForProxy(cfg, proxyOutboundPingTimeout)
	if err != nil {
		writeJSON(w, http.StatusOK, contracts.ProxyJavBusPingResponse{
			OK:      false,
			Message: err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), proxyOutboundPingTimeout)
	defer cancel()

	start := time.Now()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		writeJSON(w, http.StatusOK, contracts.ProxyJavBusPingResponse{OK: false, Message: err.Error()})
		return
	}
	httpReq.Header.Set("User-Agent", browserheaders.UserAgentChrome120)
	httpReq.Header.Set("Accept", browserheaders.AcceptLikeChrome)

	resp, err := client.Do(httpReq)
	latencyMs := time.Since(start).Milliseconds()
	if err != nil {
		writeJSON(w, http.StatusOK, contracts.ProxyJavBusPingResponse{
			OK:        false,
			LatencyMs: latencyMs,
			Message:   err.Error(),
		})
		return
	}
	defer resp.Body.Close()
	_, _ = io.CopyN(io.Discard, resp.Body, 64*1024)

	ok := resp.StatusCode < 500
	msg := ""
	if !ok {
		msg = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	writeJSON(w, http.StatusOK, contracts.ProxyJavBusPingResponse{
		OK:         ok,
		LatencyMs:  latencyMs,
		HTTPStatus: resp.StatusCode,
		Message:    msg,
	})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

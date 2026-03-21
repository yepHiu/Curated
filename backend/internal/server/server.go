package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"jav-shadcn/backend/internal/config"
	"jav-shadcn/backend/internal/contracts"
	"jav-shadcn/backend/internal/library"
	"jav-shadcn/backend/internal/storage"
	"jav-shadcn/backend/internal/tasks"
	"jav-shadcn/backend/internal/version"
)

type ScanStarter interface {
	StartScan(ctx context.Context, paths []string) (contracts.TaskDTO, error)
}

// MovieMetadataRefresher starts an async single-movie metadata rescrape and returns the scrape task.
type MovieMetadataRefresher interface {
	StartMovieMetadataRefresh(ctx context.Context, movieID string) (contracts.TaskDTO, error)
}

// OrganizeLibraryController exposes 整理库开关；值来自启动时合并的 library-config.cfg，PATCH 会写回该文件。
type OrganizeLibraryController interface {
	OrganizeLibrary() bool
	SetOrganizeLibrary(v bool) error
}

type Handler struct {
	cfg                    config.Config
	logger                 *zap.Logger
	store                  *storage.SQLiteStore
	library                *library.Service
	tasks                  *tasks.Manager
	scanStarter            ScanStarter
	organizeLibraryCtl     OrganizeLibraryController
	movieMetadataRefresher MovieMetadataRefresher
}

type Deps struct {
	Cfg                    config.Config
	Logger                 *zap.Logger
	Store                  *storage.SQLiteStore
	Library                *library.Service
	Tasks                  *tasks.Manager
	ScanStarter            ScanStarter
	OrganizeLibraryCtl     OrganizeLibraryController
	MovieMetadataRefresher MovieMetadataRefresher
}

func NewHandler(deps Deps) *Handler {
	return &Handler{
		cfg:                    deps.Cfg,
		logger:                 deps.Logger,
		store:                  deps.Store,
		library:                deps.Library,
		tasks:                  deps.Tasks,
		scanStarter:            deps.ScanStarter,
		organizeLibraryCtl:     deps.OrganizeLibraryCtl,
		movieMetadataRefresher: deps.MovieMetadataRefresher,
	}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", h.handleHealth)
	mux.HandleFunc("GET /api/library/movies", h.handleListMovies)
	mux.HandleFunc("GET /api/library/movies/{movieId}/stream", h.handleStreamMovie)
	mux.HandleFunc("GET /api/library/movies/{movieId}", h.handleGetMovie)
	mux.HandleFunc("POST /api/library/movies/{movieId}/scrape", h.handleRefreshMovieMetadata)
	mux.HandleFunc("DELETE /api/library/movies/{movieId}", h.handleDeleteMovie)
	mux.HandleFunc("GET /api/settings", h.handleGetSettings)
	mux.HandleFunc("PATCH /api/settings", h.handlePatchSettings)
	mux.HandleFunc("POST /api/library/paths", h.handleAddLibraryPath)
	mux.HandleFunc("PATCH /api/library/paths/{id}", h.handlePatchLibraryPath)
	mux.HandleFunc("DELETE /api/library/paths/{id}", h.handleDeleteLibraryPath)
	mux.HandleFunc("POST /api/scans", h.handleStartScan)
	mux.HandleFunc("GET /api/tasks/{taskId}", h.handleGetTaskStatus)

	return withCORS(mux)
}

func (h *Handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, contracts.HealthDTO{
		Name:         "javd",
		Version:      version.Version,
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
		Limit:  limit,
		Offset: offset,
	}

	result, err := h.store.ListMovies(r.Context(), request)
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to list movies")
		return
	}
	if result.Total == 0 {
		result = h.library.ListMovies(request)
	}
	writeJSON(w, http.StatusOK, result)
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
			movie, err = h.library.GetMovie(movieID)
		} else {
			if h.logger != nil {
				h.logger.Warn("get movie detail failed", zap.Error(err))
			}
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load movie")
			return
		}
	}
	if err != nil {
		if library.IsNotFound(err) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie not found")
			return
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load movie")
		return
	}
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

// handleDeleteMovie removes the movie row and related DB rows in a transaction, then best-effort deletes
// files on disk (see storage.DeleteMovie). Success returns 204 No Content; missing id returns 404.
func (h *Handler) handleDeleteMovie(w http.ResponseWriter, r *http.Request) {
	movieID := r.PathValue("movieId")
	if movieID == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "movieId is required")
		return
	}

	err := h.store.DeleteMovie(r.Context(), movieID, strings.TrimSpace(h.cfg.CacheDir))
	if err != nil {
		if errors.Is(err, storage.ErrMovieNotFound) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie not found")
			return
		}
		if h.logger != nil {
			h.logger.Warn("delete movie failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to delete movie")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	libraryPaths, err := h.store.ListLibraryPaths(r.Context())
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to list library paths")
		return
	}

	org := h.cfg.OrganizeLibrary
	if h.organizeLibraryCtl != nil {
		org = h.organizeLibraryCtl.OrganizeLibrary()
	}
	writeJSON(w, http.StatusOK, contracts.SettingsDTO{
		LibraryPaths: libraryPaths,
		Player: contracts.PlayerSettingsDTO{
			HardwareDecode: h.cfg.Player.HardwareDecode,
		},
		OrganizeLibrary: org,
	})
}

func (h *Handler) handlePatchSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.organizeLibraryCtl == nil {
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

	if body.OrganizeLibrary == nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "no supported fields to update")
		return
	}

	if err := h.organizeLibraryCtl.SetOrganizeLibrary(*body.OrganizeLibrary); err != nil {
		if h.logger != nil {
			h.logger.Warn("failed to persist organizeLibrary", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to save library settings")
		return
	}

	libraryPaths, err := h.store.ListLibraryPaths(r.Context())
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to list library paths")
		return
	}

	writeJSON(w, http.StatusOK, contracts.SettingsDTO{
		LibraryPaths: libraryPaths,
		Player: contracts.PlayerSettingsDTO{
			HardwareDecode: h.cfg.Player.HardwareDecode,
		},
		OrganizeLibrary: h.organizeLibraryCtl.OrganizeLibrary(),
	})
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

	writeJSON(w, http.StatusCreated, dto)
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

	if err := h.store.DeleteLibraryPath(r.Context(), id); err != nil {
		if errors.Is(err, storage.ErrLibraryPathNotFound) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "library path not found")
			return
		}
		h.logger.Error("delete library path failed", zap.Error(err))
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to delete library path")
		return
	}

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
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	logger.Info("HTTP server listening", zap.String("addr", addr))
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

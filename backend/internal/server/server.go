package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"go.uber.org/zap"

	"jav-shadcn/backend/internal/config"
	"jav-shadcn/backend/internal/contracts"
	"jav-shadcn/backend/internal/library"
	"jav-shadcn/backend/internal/storage"
	"jav-shadcn/backend/internal/tasks"
)

type ScanStarter interface {
	StartScan(ctx context.Context, paths []string) (contracts.TaskDTO, error)
}

type Handler struct {
	cfg          config.Config
	logger       *zap.Logger
	store        *storage.SQLiteStore
	library      *library.Service
	tasks        *tasks.Manager
	scanStarter  ScanStarter
}

type Deps struct {
	Cfg         config.Config
	Logger      *zap.Logger
	Store       *storage.SQLiteStore
	Library     *library.Service
	Tasks       *tasks.Manager
	ScanStarter ScanStarter
}

func NewHandler(deps Deps) *Handler {
	return &Handler{
		cfg:         deps.Cfg,
		logger:      deps.Logger,
		store:       deps.Store,
		library:     deps.Library,
		tasks:       deps.Tasks,
		scanStarter: deps.ScanStarter,
	}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", h.handleHealth)
	mux.HandleFunc("GET /api/library/movies", h.handleListMovies)
	mux.HandleFunc("GET /api/library/movies/{movieId}", h.handleGetMovie)
	mux.HandleFunc("GET /api/settings", h.handleGetSettings)
	mux.HandleFunc("POST /api/scans", h.handleStartScan)
	mux.HandleFunc("GET /api/tasks/{taskId}", h.handleGetTaskStatus)

	return withCORS(mux)
}

func (h *Handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, contracts.HealthDTO{
		Name:         "javd",
		Version:      "0.1.0",
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
		movie, err = h.library.GetMovie(movieID)
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

func (h *Handler) handleGetSettings(w http.ResponseWriter, _ *http.Request) {
	libraryPaths := make([]contracts.LibraryPathDTO, 0, len(h.cfg.LibraryPaths))
	for index, path := range h.cfg.LibraryPaths {
		libraryPaths = append(libraryPaths, contracts.LibraryPathDTO{
			ID:    fmt.Sprintf("library-%d", index+1),
			Path:  path,
			Title: fmt.Sprintf("Library path %d", index+1),
		})
	}

	writeJSON(w, http.StatusOK, contracts.SettingsDTO{
		LibraryPaths:        libraryPaths,
		ScanIntervalSeconds: h.cfg.ScanIntervalSeconds,
		Player: contracts.PlayerSettingsDTO{
			HardwareDecode: h.cfg.Player.HardwareDecode,
		},
	})
}

func (h *Handler) handleStartScan(w http.ResponseWriter, r *http.Request) {
	var request contracts.StartScanRequest
	if r.Body != nil {
		defer r.Body.Close()
		_ = json.NewDecoder(r.Body).Decode(&request)
	}

	task, err := h.scanStarter.StartScan(r.Context(), request.Paths)
	if err != nil {
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
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

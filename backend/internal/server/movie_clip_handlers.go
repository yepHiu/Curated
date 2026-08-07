package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"curated-backend/internal/contracts"
	"curated-backend/internal/playback"
)

const (
	maxMovieClipDurationSec = 6.0
	minMovieClipDurationSec = 0.4
	defaultMovieClipFPS     = 10
	defaultMovieClipWidth   = 480
	movieClipArtifactMaxAge = 24 * time.Hour
)

func movieClipArtifactRoot(h *Handler) string {
	base := strings.TrimSpace(h.cfg.CacheDir)
	if base == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "curated-clips")
}

func (h *Handler) startMovieClipArtifactJanitor(ctx context.Context) {
	go func() {
		cleanup := func() {
			entries, err := os.ReadDir(movieClipArtifactRoot(h))
			if err != nil {
				return
			}
			cutoff := time.Now().Add(-movieClipArtifactMaxAge)
			for _, entry := range entries {
				if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".gif") {
					continue
				}
				path := filepath.Join(movieClipArtifactRoot(h), entry.Name())
				info, err := entry.Info()
				if err == nil && info.ModTime().Before(cutoff) {
					_ = os.Remove(path)
				}
			}
		}
		cleanup()
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cleanup()
			}
		}
	}()
}

func (h *Handler) handleCreateMovieClip(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
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
	var req contracts.CreateMovieClipBody
	if len(strings.TrimSpace(string(body))) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid json body")
			return
		}
	}
	format := strings.ToLower(strings.TrimSpace(req.Format))
	if format == "" {
		format = "gif"
	}
	if format != "gif" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, `format must be "gif"`)
		return
	}
	if req.StartSec < 0 || req.EndSec <= req.StartSec {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "endSec must be greater than startSec")
		return
	}
	duration := req.EndSec - req.StartSec
	if duration < minMovieClipDurationSec || duration > maxMovieClipDurationSec {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, fmt.Sprintf("clip duration must be between %.1f and %.0f seconds", minMovieClipDurationSec, maxMovieClipDurationSec))
		return
	}
	fps := req.FPS
	if fps == 0 {
		fps = defaultMovieClipFPS
	}
	if fps < 1 || fps > 20 {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "fps must be between 1 and 20")
		return
	}
	width := req.Width
	if width == 0 {
		width = defaultMovieClipWidth
	}
	if width < 160 || width > 960 {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "width must be between 160 and 960")
		return
	}
	if h.store == nil || h.tasks == nil {
		writeAppError(w, http.StatusServiceUnavailable, contracts.ErrorCodeInternal, "clip export is not configured")
		return
	}
	sourcePath, err := h.store.ResolvePrimaryVideoPath(r.Context(), movieID)
	if err != nil {
		status := http.StatusInternalServerError
		code := contracts.ErrorCodeInternal
		if errors.Is(err, os.ErrNotExist) {
			status, code = http.StatusNotFound, contracts.ErrorCodeNotFound
		}
		writeAppError(w, status, code, "movie video is not available")
		return
	}

	task := h.tasks.Create("movie_clip_gif", map[string]any{
		"movieId":  movieID,
		"format":   format,
		"startSec": req.StartSec,
		"endSec":   req.EndSec,
		"fps":      fps,
		"width":    width,
	})
	go h.runMovieClipTask(task.TaskID, sourcePath, req.StartSec, duration, fps, width)
	writeJSON(w, http.StatusAccepted, task)
}

func (h *Handler) runMovieClipTask(taskID, sourcePath string, startSec, duration float64, fps, width int) {
	ctx := h.runtimeContext
	if ctx == nil {
		ctx = context.Background()
	}
	root := movieClipArtifactRoot(h)
	if err := os.MkdirAll(root, 0o755); err != nil {
		h.tasks.Fail(taskID, "clip_artifact_directory_failed", "failed to prepare clip output directory")
		return
	}
	outputPath := filepath.Join(root, "curated-clip-"+taskID+".gif")
	ffmpeg := playback.ResolveFFmpegCommand(h.cfg.Player.FFmpegCommand)
	filter := fmt.Sprintf("fps=%d,scale=%d:-1:flags=lanczos:force_original_aspect_ratio=decrease", fps, width)
	h.tasks.Start(taskID, "generating GIF")
	h.tasks.Progress(taskID, 5, "generating GIF")
	cmd := exec.CommandContext(ctx, ffmpeg,
		"-hide_banner", "-loglevel", "error", "-y",
		"-ss", strconv.FormatFloat(startSec, 'f', 3, 64),
		"-i", sourcePath,
		"-t", strconv.FormatFloat(duration, 'f', 3, 64),
		"-vf", filter,
		"-an", "-loop", "0", outputPath,
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		_ = os.Remove(outputPath)
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = "ffmpeg failed to generate GIF"
		}
		h.tasks.Fail(taskID, "clip_ffmpeg_failed", message)
		return
	}
	if info, err := os.Stat(outputPath); err != nil || info.Size() == 0 {
		_ = os.Remove(outputPath)
		h.tasks.Fail(taskID, "clip_artifact_missing", "GIF output was not created")
		return
	}
	h.tasks.ProgressWithMetadata(taskID, 100, "GIF ready", map[string]any{
		"artifactUrl": "/api/tasks/" + taskID + "/artifact",
		"contentType": "image/gif",
		"filename":    "curated-clip-" + taskID + ".gif",
	})
	if h.movieClipArtifacts != nil {
		h.movieClipArtifacts.Store(taskID, outputPath)
	}
	h.tasks.Complete(taskID, "GIF ready")
}

func (h *Handler) handleGetMovieClipArtifact(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	task, ok := h.tasks.Get(strings.TrimSpace(r.PathValue("taskId")))
	if !ok {
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "task not found")
		return
	}
	if task.Status != contracts.TaskCompleted || task.Type != "movie_clip_gif" {
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "clip artifact is not ready")
		return
	}
	var pathValue string
	if h.movieClipArtifacts != nil {
		if value, ok := h.movieClipArtifacts.Load(strings.TrimSpace(r.PathValue("taskId"))); ok {
			pathValue, _ = value.(string)
		}
	}
	root, err := filepath.Abs(movieClipArtifactRoot(h))
	if err != nil || !pathUnderRoot(pathValue, root) {
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "clip artifact is unavailable")
		return
	}
	name, _ := task.Metadata["filename"].(string)
	if name == "" {
		name = filepath.Base(pathValue)
	}
	w.Header().Set("Content-Type", "image/gif")
	w.Header().Set("Content-Disposition", contentDispositionAttachment(name, "curated-clip.gif"))
	http.ServeFile(w, r, pathValue)
}

func pathUnderRoot(pathValue, root string) bool {
	absPath, err := filepath.Abs(filepath.Clean(pathValue))
	if err != nil {
		return false
	}
	cleanRoot := filepath.Clean(root)
	if strings.EqualFold(absPath, cleanRoot) {
		return true
	}
	prefix := cleanRoot + string(os.PathSeparator)
	return strings.HasPrefix(strings.ToLower(absPath), strings.ToLower(prefix))
}

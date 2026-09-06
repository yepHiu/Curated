package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"curated-backend/internal/contracts"
	"curated-backend/internal/executil"
	"curated-backend/internal/playback"
	"curated-backend/internal/storage"
)

func (h *Handler) initMovieClipQueue() {
	h.movieClipInit.Do(func() {
		h.movieClipSlots = make(chan struct{}, 8)
		h.movieClipWorkers = make(chan struct{}, 2)
	})
}

func (h *Handler) handleCancelMovieClip(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("taskId"))
	if cancel, ok := h.movieClipCancels.Load(id); ok {
		cancel.(context.CancelFunc)()
		w.WriteHeader(http.StatusNoContent)
		return
	}
	writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "clip task is not active")
}

const (
	maxMovieClipDurationSec = 6.0
	minMovieClipDurationSec = 0.4
	defaultMovieClipFPS     = 10
	defaultMovieClipWidth   = 640
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
				if entry.IsDir() || !isMovieClipExtension(filepath.Ext(entry.Name())) {
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
	if !isMovieClipExtension("." + format) {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, `format must be gif, mp4 or webm`)
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
	curatedFrameID := strings.TrimSpace(req.CuratedFrameID)
	if curatedFrameID != "" {
		if h.store == nil {
			writeAppError(w, http.StatusServiceUnavailable, contracts.ErrorCodeInternal, "clip export is not configured")
			return
		}
		exists, err := h.store.CuratedFrameExists(r.Context(), curatedFrameID)
		if err != nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to verify curated frame")
			return
		}
		if !exists {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "curated frame not found")
			return
		}
		if err := h.store.UpsertCuratedFrameMotion(r.Context(), storage.CuratedFrameMotionMeta{
			FrameID: curatedFrameID, Status: "processing", ContentType: movieClipContentType(format), DurationSec: duration, Width: width, FPS: fps,
		}); err != nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to prepare curated frame motion")
			return
		}
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
		if curatedFrameID != "" {
			_ = h.store.UpsertCuratedFrameMotion(r.Context(), storage.CuratedFrameMotionMeta{
				FrameID: curatedFrameID, Status: "error", ContentType: movieClipContentType(format), DurationSec: duration, Width: width, FPS: fps,
				ErrorMessage: "movie video is not available",
			})
		}
		writeAppError(w, status, code, "movie video is not available")
		return
	}

	task := h.tasks.Create("movie_clip_"+format, map[string]any{
		"movieId":  movieID,
		"format":   format,
		"startSec": req.StartSec,
		"endSec":   req.EndSec,
		"fps":      fps,
		"width":    width,
	})
	h.initMovieClipQueue()
	select {
	case h.movieClipSlots <- struct{}{}:
	default:
		h.tasks.Fail(task.TaskID, "clip_queue_full", "clip queue is full")
		if curatedFrameID != "" {
			_ = h.store.UpsertCuratedFrameMotion(r.Context(), storage.CuratedFrameMotionMeta{FrameID: curatedFrameID, Status: "error", ContentType: movieClipContentType(format), ErrorMessage: "clip queue is full"})
		}
		writeAppError(w, http.StatusTooManyRequests, contracts.ErrorCodeConflict, "clip queue is full")
		return
	}
	parent := h.runtimeContext
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, 2*time.Minute)
	h.movieClipCancels.Store(task.TaskID, context.CancelFunc(cancel))
	go func() {
		defer func() { <-h.movieClipSlots; h.movieClipCancels.Delete(task.TaskID); cancel() }()
		select {
		case h.movieClipWorkers <- struct{}{}:
			defer func() { <-h.movieClipWorkers }()
			h.runMovieClipTaskContext(ctx, task.TaskID, sourcePath, req.StartSec, duration, fps, width, curatedFrameID)
		case <-ctx.Done():
			h.tasks.Fail(task.TaskID, "clip_cancelled", "clip cancelled or timed out")
			if curatedFrameID != "" {
				_ = h.store.UpsertCuratedFrameMotion(context.Background(), storage.CuratedFrameMotionMeta{FrameID: curatedFrameID, Status: "error", ContentType: movieClipContentType(format), ErrorMessage: "clip cancelled or timed out"})
			}
		}
	}()
	writeJSON(w, http.StatusAccepted, task)
}

func (h *Handler) runMovieClipTask(taskID, sourcePath string, startSec, duration float64, fps, width int, curatedFrameID string) {
	ctx := h.runtimeContext
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	h.runMovieClipTaskContext(ctx, taskID, sourcePath, startSec, duration, fps, width, curatedFrameID)
}

func (h *Handler) runMovieClipTaskContext(ctx context.Context, taskID, sourcePath string, startSec, duration float64, fps, width int, curatedFrameID string) {
	format := "gif"
	if task, ok := h.tasks.Get(taskID); ok {
		if value, ok := task.Metadata["format"].(string); ok && isMovieClipExtension("."+value) {
			format = value
		}
	}

	root := movieClipArtifactRoot(h)
	if err := os.MkdirAll(root, 0o755); err != nil {
		if curatedFrameID != "" && h.store != nil {
			_ = h.store.UpsertCuratedFrameMotion(context.Background(), storage.CuratedFrameMotionMeta{FrameID: curatedFrameID, Status: "error", ContentType: movieClipContentType(format), DurationSec: duration, Width: width, FPS: fps, ErrorMessage: "failed to prepare clip output directory"})
		}
		h.tasks.Fail(taskID, "clip_artifact_directory_failed", "failed to prepare clip output directory")
		return
	}
	outputPath := filepath.Join(root, "curated-clip-"+taskID+"."+format)
	ffmpeg := playback.ResolveFFmpegCommand(h.cfg.Player.FFmpegCommand)
	filter := fmt.Sprintf("fps=%d,scale=%d:-2:flags=lanczos:force_original_aspect_ratio=decrease", fps, width)
	h.tasks.Start(taskID, "generating clip")
	h.tasks.Progress(taskID, 5, "generating clip")

	args := []string{"-hide_banner", "-loglevel", "error", "-y", "-threads", "2", "-filter_threads", "1", "-ss", strconv.FormatFloat(startSec, 'f', 3, 64), "-i", sourcePath, "-t", strconv.FormatFloat(duration, 'f', 3, 64), "-an"}
	switch format {
	case "gif":
		filter += ",split[a][b];[a]palettegen=stats_mode=diff[p];[b][p]paletteuse=dither=bayer:bayer_scale=3"
		args = append(args, "-vf", filter, "-loop", "0")
	case "mp4":
		args = append(args, "-vf", filter, "-c:v", "libx264", "-threads", "2", "-preset", "fast", "-crf", "23", "-pix_fmt", "yuv420p", "-movflags", "+faststart")
	case "webm":
		args = append(args, "-vf", filter, "-c:v", "libvpx-vp9", "-threads", "2", "-deadline", "good", "-cpu-used", "4", "-crf", "32", "-b:v", "0", "-pix_fmt", "yuv420p")
	}
	args = append(args, "-fs", "134217728", outputPath)
	cmd := executil.CommandContext(ctx, ffmpeg, args...)

	if output, err := cmd.CombinedOutput(); err != nil {
		_ = os.Remove(outputPath)
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = "ffmpeg failed to generate GIF"
		}
		if curatedFrameID != "" && h.store != nil {
			_ = h.store.UpsertCuratedFrameMotion(context.Background(), storage.CuratedFrameMotionMeta{FrameID: curatedFrameID, Status: "error", ContentType: movieClipContentType(format), DurationSec: duration, Width: width, FPS: fps, ErrorMessage: message})
		}
		h.tasks.Fail(taskID, "clip_ffmpeg_failed", message)
		return
	}
	info, statErr := os.Stat(outputPath)
	if statErr != nil || info.Size() == 0 {
		_ = os.Remove(outputPath)
		if curatedFrameID != "" && h.store != nil {
			_ = h.store.UpsertCuratedFrameMotion(context.Background(), storage.CuratedFrameMotionMeta{FrameID: curatedFrameID, Status: "error", ContentType: movieClipContentType(format), DurationSec: duration, Width: width, FPS: fps, ErrorMessage: "GIF output was not created"})
		}
		h.tasks.Fail(taskID, "clip_artifact_missing", "GIF output was not created")
		return
	}
	artifactURL := "/api/tasks/" + taskID + "/artifact"
	filename := "curated-clip-" + taskID + "." + format
	if curatedFrameID != "" {
		motionRoot := curatedFrameMotionRoot(h)
		if err := os.MkdirAll(motionRoot, 0o755); err != nil {
			_ = os.Remove(outputPath)
			_ = h.store.UpsertCuratedFrameMotion(context.Background(), storage.CuratedFrameMotionMeta{FrameID: curatedFrameID, Status: "error", ContentType: movieClipContentType(format), DurationSec: duration, Width: width, FPS: fps, ErrorMessage: "failed to prepare curated frame motion directory"})
			h.tasks.Fail(taskID, "clip_artifact_directory_failed", "failed to prepare curated frame motion directory")
			return
		}
		filename = curatedFrameID + "." + format
		finalPath := filepath.Join(motionRoot, filename)
		if err := os.Remove(finalPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			_ = os.Remove(outputPath)
			h.tasks.Fail(taskID, "clip_artifact_move_failed", "failed to replace curated frame motion")
			return
		}
		if err := os.Rename(outputPath, finalPath); err != nil {
			_ = os.Remove(outputPath)
			_ = h.store.UpsertCuratedFrameMotion(context.Background(), storage.CuratedFrameMotionMeta{FrameID: curatedFrameID, Status: "error", ContentType: movieClipContentType(format), DurationSec: duration, Width: width, FPS: fps, ErrorMessage: "failed to persist curated frame motion"})
			h.tasks.Fail(taskID, "clip_artifact_move_failed", "failed to persist curated frame motion")
			return
		}
		if err := h.store.UpsertCuratedFrameMotion(context.Background(), storage.CuratedFrameMotionMeta{FrameID: curatedFrameID, Status: "ready", ArtifactName: filename, ContentType: movieClipContentType(format), DurationSec: duration, Width: width, FPS: fps, FileSize: info.Size()}); err != nil {
			h.tasks.Fail(taskID, "clip_metadata_persist_failed", "failed to persist curated frame motion metadata")
			return
		}
		artifactURL = "/api/curated-frames/" + url.PathEscape(curatedFrameID) + "/motion"
	}
	h.tasks.ProgressWithMetadata(taskID, 100, "clip ready", map[string]any{
		"artifactUrl":    artifactURL,
		"contentType":    movieClipContentType(format),
		"filename":       filename,
		"curatedFrameId": curatedFrameID,
	})
	if h.movieClipArtifacts != nil && curatedFrameID == "" {
		h.movieClipArtifacts.Store(taskID, outputPath)
	}
	h.tasks.Complete(taskID, "clip ready")
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
	if task.Status != contracts.TaskCompleted || !isMovieClipExtension("."+strings.TrimPrefix(task.Type, "movie_clip_")) {
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
	w.Header().Set("Content-Type", movieClipContentType(strings.TrimPrefix(task.Type, "movie_clip_")))
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

func isMovieClipExtension(ext string) bool {
	switch strings.ToLower(ext) {
	case ".gif", ".mp4", ".webm":
		return true
	default:
		return false
	}
}
func movieClipContentType(format string) string {
	switch format {
	case "mp4":
		return "video/mp4"
	case "webm":
		return "video/webm"
	default:
		return "image/gif"
	}
}

package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"curated-backend/internal/config"
	"curated-backend/internal/executil"
	"curated-backend/internal/tasks"
	"go.uber.org/zap"
)

func TestMovieClipRealFormats(t *testing.T) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("FFmpeg not installed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	dir := t.TempDir()
	source := filepath.Join(dir, "source.mp4")
	if output, err := executil.CommandContext(ctx, ffmpeg, "-hide_banner", "-loglevel", "error", "-f", "lavfi", "-i", "testsrc2=size=320x180:rate=10", "-t", "1", "-c:v", "libx264", "-pix_fmt", "yuv420p", source).CombinedOutput(); err != nil {
		t.Fatalf("synthetic source: %v %s", err, output)
	}
	h := &Handler{cfg: config.Config{CacheDir: dir}, tasks: tasks.NewManager(), movieClipArtifacts: &sync.Map{}, runtimeContext: ctx}
	for _, format := range []string{"gif", "mp4", "webm"} {
		task := h.tasks.Create("movie_clip_"+format, map[string]any{"format": format})
		h.runMovieClipTask(task.TaskID, source, 0, 0.6, 10, 320, "")
		result, _ := h.tasks.Get(task.TaskID)
		if string(result.Status) != "completed" {
			t.Fatalf("%s: %+v", format, result)
		}
		if result.Metadata["contentType"] != movieClipContentType(format) {
			t.Fatalf("wrong MIME: %+v", result)
		}
		path, _ := h.movieClipArtifacts.Load(task.TaskID)
		info, err := os.Stat(path.(string))
		if err != nil || info.Size() == 0 {
			t.Fatalf("missing artifact: %v", err)
		}
		t.Logf("%s: %d bytes", format, info.Size())
	}
}

func TestMovieClipQueueBudgetAndCancellation(t *testing.T) {
	h := &Handler{}
	h.initMovieClipQueue()
	if cap(h.movieClipSlots) != 8 || cap(h.movieClipWorkers) != 2 {
		t.Fatal("unexpected clip resource budget")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h.movieClipCancels.Store("clip-test", context.CancelFunc(cancel))
	req := httptest.NewRequest(http.MethodDelete, "/api/tasks/clip-test/clip", nil)
	req.SetPathValue("taskId", "clip-test")
	rec := httptest.NewRecorder()
	h.handleCancelMovieClip(rec, req)
	if rec.Code != http.StatusNoContent || ctx.Err() != context.Canceled {
		t.Fatalf("cancel: %d %v", rec.Code, ctx.Err())
	}
}

func TestMovieClipDefaultWidth(t *testing.T) {
	if defaultMovieClipWidth != 640 {
		t.Fatalf("default movie clip width = %d, want 640", defaultMovieClipWidth)
	}
}

func TestHandleCreateMovieClipRejectsInvalidRangesBeforeStorageLookup(t *testing.T) {
	h := &Handler{tasks: tasks.NewManager(), logger: zap.NewNop()}
	for _, tc := range []struct {
		name string
		body string
	}{
		{name: "negative start", body: `{"startSec":-1,"endSec":1}`},
		{name: "reversed range", body: `{"startSec":2,"endSec":1}`},
		{name: "too short", body: `{"startSec":1,"endSec":1.1}`},
		{name: "too long", body: `{"startSec":1,"endSec":8}`},
		{name: "wrong format", body: `{"startSec":1,"endSec":2,"format":"avi"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/library/movies/movie-1/clips", strings.NewReader(tc.body))
			req.SetPathValue("movieId", "movie-1")
			rec := httptest.NewRecorder()
			h.handleCreateMovieClip(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
			}
		})
	}
}

func TestHandleGetMovieClipArtifactServesOnlyRegisteredTaskOutput(t *testing.T) {
	cacheDir := t.TempDir()
	tm := tasks.NewManager()
	h := &Handler{
		cfg:                config.Config{CacheDir: cacheDir},
		tasks:              tm,
		logger:             zap.NewNop(),
		movieClipArtifacts: &sync.Map{},
	}
	task := tm.Create("movie_clip_gif", map[string]any{"movieId": "movie-1"})
	root := movieClipArtifactRoot(h)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir artifact root: %v", err)
	}
	path := filepath.Join(root, "curated-clip-"+task.TaskID+".gif")
	if err := os.WriteFile(path, []byte("GIF89a"), 0o644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	tm.ProgressWithMetadata(task.TaskID, 100, "GIF ready", map[string]any{
		"artifactUrl": "/api/tasks/" + task.TaskID + "/artifact",
		"filename":    "clip.gif",
	})
	tm.Complete(task.TaskID, "GIF ready")
	h.movieClipArtifacts.Store(task.TaskID, path)

	req := httptest.NewRequest(http.MethodGet, "/api/tasks/"+task.TaskID+"/artifact", nil)
	req.SetPathValue("taskId", task.TaskID)
	rec := httptest.NewRecorder()
	h.handleGetMovieClipArtifact(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "image/gif") {
		t.Fatalf("content-type = %q, want image/gif", got)
	}
	if rec.Body.String() != "GIF89a" {
		t.Fatalf("artifact body = %q", rec.Body.String())
	}
}

func TestPathUnderRootRejectsSiblingAndTraversalPaths(t *testing.T) {
	root := t.TempDir()
	if !pathUnderRoot(root+"/nested/clip.gif", root) {
		t.Fatal("expected nested artifact to be inside root")
	}
	if pathUnderRoot(root+"-sibling/clip.gif", root) {
		t.Fatal("sibling path must not be accepted")
	}
	if pathUnderRoot(root+"/nested/../..//clip.gif", root) {
		t.Fatal("traversal path must not be accepted")
	}
}

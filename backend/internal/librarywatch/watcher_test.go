package librarywatch

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
)

type testPathLister struct {
	roots []string
}

func (l testPathLister) ListLibraryPathStrings(context.Context) ([]string, error) {
	return l.roots, nil
}

type testScanQueue struct {
	roots chan []string
}

func (q testScanQueue) EnqueueLibraryWatchScanRoots(roots []string) {
	q.roots <- roots
}

func TestWatcherCreateDirectorySchedulesScan(t *testing.T) {
	root := t.TempDir()
	movieDir := filepath.Join(root, "ABC-100")
	if err := os.MkdirAll(movieDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(movieDir, "ABC-100.mp4"), []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}

	queue := testScanQueue{roots: make(chan []string, 1)}
	w, err := New(Options{
		Enabled:  true,
		Debounce: time.Millisecond,
		Logger:   zap.NewNop(),
		Lister:   testPathLister{roots: []string{root}},
		Queue:    queue,
	})
	if err != nil {
		t.Fatal(err)
	}

	w.handleEvent(context.Background(), fsnotify.Event{Name: movieDir, Op: fsnotify.Create})

	select {
	case got := <-queue.roots:
		if len(got) != 1 || got[0] != root {
			t.Fatalf("queued roots = %#v, want [%q]", got, root)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for directory create to enqueue a scan")
	}
}

func TestIsVideoPathUsesSharedWhitelist(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{filepath.Join("library", "ABC-123.mp4"), true},
		{filepath.Join("library", "ABC-123.rmvb"), true},
		{filepath.Join("library", "ABC-123.wmv"), true},
		{filepath.Join("library", "ABC-123.iso"), true},
		{filepath.Join("library", "ABC-123.m2ts"), true},
		{filepath.Join("library", "video.part"), false},
		{filepath.Join("library", "still-copying.mp4.tmp"), false},
		{filepath.Join("library", ".ABC-123.mp4"), false},
		{filepath.Join("library", "ABC-123.mp4~"), false},
		{filepath.Join("library", "notes.txt"), false},
	}
	for _, tc := range cases {
		if got := isVideoPath(tc.path); got != tc.want {
			t.Fatalf("isVideoPath(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

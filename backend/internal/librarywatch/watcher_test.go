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

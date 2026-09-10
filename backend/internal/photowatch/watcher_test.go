package photowatch

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
)

type testPhotoPathLister struct {
	roots []string
}

func (l testPhotoPathLister) ListPhotoLibraryPathStrings(context.Context) ([]string, error) {
	return l.roots, nil
}

type testPhotoScanQueue struct {
	roots chan []string
}

func (q testPhotoScanQueue) EnqueuePhotoLibraryWatchScanRoots(roots []string) {
	q.roots <- roots
}

func TestWatcherArchiveWriteSchedulesPhotoScan(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "Portrait Set.cbz")
	if err := os.WriteFile(archivePath, []byte("fake archive"), 0o644); err != nil {
		t.Fatal(err)
	}

	queue := testPhotoScanQueue{roots: make(chan []string, 1)}
	w, err := New(Options{
		Enabled:  true,
		Debounce: time.Millisecond,
		Logger:   zap.NewNop(),
		Lister:   testPhotoPathLister{roots: []string{root}},
		Queue:    queue,
	})
	if err != nil {
		t.Fatal(err)
	}

	w.handleEvent(context.Background(), fsnotify.Event{Name: archivePath, Op: fsnotify.Write})

	assertQueuedPhotoRoot(t, queue, root)
}

func TestWatcherIgnoresNonPhotoAndTemporaryFiles(t *testing.T) {
	root := t.TempDir()
	queue := testPhotoScanQueue{roots: make(chan []string, 1)}
	w, err := New(Options{
		Enabled:  true,
		Debounce: time.Millisecond,
		Logger:   zap.NewNop(),
		Lister:   testPhotoPathLister{roots: []string{root}},
		Queue:    queue,
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"movie.mp4", ".hidden.cbz", "download.cbz.part", "backup.zip~"} {
		w.handleEvent(context.Background(), fsnotify.Event{Name: filepath.Join(root, name), Op: fsnotify.Write})
	}

	select {
	case got := <-queue.roots:
		t.Fatalf("unexpected queued roots = %#v", got)
	case <-time.After(50 * time.Millisecond):
	}
}

func assertQueuedPhotoRoot(t *testing.T, queue testPhotoScanQueue, root string) {
	t.Helper()
	select {
	case got := <-queue.roots:
		if len(got) != 1 || got[0] != root {
			t.Fatalf("queued roots = %#v, want [%q]", got, root)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for photo watcher to enqueue a scan")
	}
}

package comicwatch

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
)

type testComicPathLister struct {
	roots []string
}

func (l testComicPathLister) ListComicLibraryPathStrings(context.Context) ([]string, error) {
	return l.roots, nil
}

type testComicScanQueue struct {
	roots chan []string
}

func (q testComicScanQueue) EnqueueComicLibraryWatchScanRoots(roots []string) {
	q.roots <- roots
}

func TestWatcherArchiveWriteSchedulesComicScan(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "Long Story.cbz")
	if err := os.WriteFile(archivePath, []byte("fake archive"), 0o644); err != nil {
		t.Fatal(err)
	}

	queue := testComicScanQueue{roots: make(chan []string, 1)}
	w, err := New(Options{
		Enabled:  true,
		Debounce: time.Millisecond,
		Logger:   zap.NewNop(),
		Lister:   testComicPathLister{roots: []string{root}},
		Queue:    queue,
	})
	if err != nil {
		t.Fatal(err)
	}

	w.handleEvent(context.Background(), fsnotify.Event{Name: archivePath, Op: fsnotify.Write})

	assertQueuedComicRoot(t, queue, root)
}

func TestWatcherCreateDirectoryWithArchiveSchedulesComicScan(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "Series")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Chapter 01.zip"), []byte("fake archive"), 0o644); err != nil {
		t.Fatal(err)
	}

	queue := testComicScanQueue{roots: make(chan []string, 1)}
	w, err := New(Options{
		Enabled:  true,
		Debounce: time.Millisecond,
		Logger:   zap.NewNop(),
		Lister:   testComicPathLister{roots: []string{root}},
		Queue:    queue,
	})
	if err != nil {
		t.Fatal(err)
	}

	w.handleEvent(context.Background(), fsnotify.Event{Name: dir, Op: fsnotify.Create})

	assertQueuedComicRoot(t, queue, root)
}

func TestWatcherIgnoresNonComicAndTemporaryFiles(t *testing.T) {
	root := t.TempDir()
	queue := testComicScanQueue{roots: make(chan []string, 1)}
	w, err := New(Options{
		Enabled:  true,
		Debounce: time.Millisecond,
		Logger:   zap.NewNop(),
		Lister:   testComicPathLister{roots: []string{root}},
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

func assertQueuedComicRoot(t *testing.T, queue testComicScanQueue, root string) {
	t.Helper()
	select {
	case got := <-queue.roots:
		if len(got) != 1 || got[0] != root {
			t.Fatalf("queued roots = %#v, want [%q]", got, root)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for comic watcher to enqueue a scan")
	}
}

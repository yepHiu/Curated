// Package librarywatch watches configured library roots with fsnotify and debounces
// scan requests when video files appear or change. Network filesystems may be unreliable.
package librarywatch

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
)

// PathLister returns current library root paths (absolute).
type PathLister interface {
	ListLibraryPathStrings(ctx context.Context) ([]string, error)
}

// ScanQueue receives merged library roots to scan after debouncing.
type ScanQueue interface {
	EnqueueLibraryWatchScanRoots(roots []string)
}

// Options configures the watcher.
type Options struct {
	Enabled  bool
	Debounce time.Duration
	Logger   *zap.Logger
	Lister   PathLister
	Queue    ScanQueue
}

// Watcher watches library directories for new/changed video files.
type Watcher struct {
	opts Options

	mu      sync.Mutex
	watcher *fsnotify.Watcher
	watched map[string]struct{}

	debounceMu    sync.Mutex
	debounceTimer *time.Timer
	pendingRoots  map[string]struct{}
}

var videoExtensions = map[string]struct{}{
	"mp4": {}, "mkv": {}, "avi": {}, "mov": {}, "ts": {},
}

// New builds a watcher (does not start I/O until Run).
func New(opts Options) (*Watcher, error) {
	if opts.Logger == nil {
		opts.Logger = zap.NewNop()
	}
	if opts.Debounce <= 0 {
		opts.Debounce = 1500 * time.Millisecond
	}
	if opts.Lister == nil || opts.Queue == nil {
		return nil, os.ErrInvalid
	}
	return &Watcher{
		opts:   opts,
		watched: make(map[string]struct{}),
	}, nil
}

// Run blocks until ctx is cancelled. It recreates the fsnotify watcher whenever
// the Events channel is closed (including after Reload).
func (w *Watcher) Run(ctx context.Context) error {
	if !w.opts.Enabled {
		<-ctx.Done()
		return ctx.Err()
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := w.bootstrap(ctx); err != nil {
			w.opts.Logger.Warn("library watch bootstrap failed, retrying", zap.Error(err))
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(5 * time.Second):
			}
			continue
		}

		w.mu.Lock()
		fsn := w.watcher
		w.mu.Unlock()
		if fsn == nil {
			continue
		}

		closed := w.consumeEvents(ctx, fsn)
		w.shutdownWatcher()

		if ctx.Err() != nil {
			return ctx.Err()
		}
		if closed {
			// Reload or Close; loop to re-bootstrap.
			continue
		}
	}
}

// Reload closes the current watcher so Run re-bootstraps from the database.
func (w *Watcher) Reload(ctx context.Context) error {
	_ = ctx
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.watcher != nil {
		_ = w.watcher.Close()
		w.watcher = nil
	}
	w.watched = make(map[string]struct{})
	return nil
}

func (w *Watcher) bootstrap(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.watcher != nil {
		_ = w.watcher.Close()
		w.watcher = nil
		w.watched = make(map[string]struct{})
	}

	roots, err := w.opts.Lister.ListLibraryPathStrings(ctx)
	if err != nil {
		return err
	}
	roots = cleanRoots(roots)
	if len(roots) == 0 {
		w.opts.Logger.Debug("library watch: no library paths configured")
	}

	fsn, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	w.watcher = fsn
	w.watched = make(map[string]struct{})

	for _, root := range roots {
		if st, err := os.Stat(root); err != nil || !st.IsDir() {
			w.opts.Logger.Debug("library watch: skip missing or non-dir root", zap.String("root", root), zap.Error(err))
			continue
		}
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return nil
			}
			if !d.IsDir() {
				return nil
			}
			if err := w.addWatchLocked(path); err != nil {
				w.opts.Logger.Debug("library watch: add failed", zap.String("path", path), zap.Error(err))
			}
			return nil
		})
	}
	w.opts.Logger.Info("library fsnotify watching", zap.Int("dirCount", len(w.watched)))
	return nil
}

func (w *Watcher) addWatchLocked(dir string) error {
	dir = filepath.Clean(dir)
	if dir == "" || dir == "." {
		return nil
	}
	if _, ok := w.watched[dir]; ok {
		return nil
	}
	if err := w.watcher.Add(dir); err != nil {
		return err
	}
	w.watched[dir] = struct{}{}
	return nil
}

func (w *Watcher) removeWatchLocked(dir string) {
	dir = filepath.Clean(dir)
	if _, ok := w.watched[dir]; !ok {
		return
	}
	_ = w.watcher.Remove(dir)
	delete(w.watched, dir)
}

func (w *Watcher) shutdownWatcher() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.watcher != nil {
		_ = w.watcher.Close()
		w.watcher = nil
	}
	w.watched = make(map[string]struct{})
}

func (w *Watcher) consumeEvents(ctx context.Context, fsn *fsnotify.Watcher) (channelClosed bool) {
	for {
		select {
		case <-ctx.Done():
			return false
		case err, ok := <-fsn.Errors:
			if !ok {
				return true
			}
			if err != nil {
				w.opts.Logger.Debug("library watch fsnotify error", zap.Error(err))
			}
		case ev, ok := <-fsn.Events:
			if !ok {
				return true
			}
			w.handleEvent(ctx, ev)
		}
	}
}

func (w *Watcher) handleEvent(ctx context.Context, ev fsnotify.Event) {
	path := filepath.Clean(ev.Name)
	if path == "" {
		return
	}

	switch {
	case ev.Has(fsnotify.Create):
		if st, err := os.Stat(path); err == nil && st.IsDir() {
			w.mu.Lock()
			if w.watcher != nil {
				_ = filepath.WalkDir(path, func(p string, d os.DirEntry, we error) error {
					if we != nil || !d.IsDir() {
						return nil
					}
					if err := w.addWatchLocked(p); err != nil {
						w.opts.Logger.Debug("library watch: add subtree", zap.String("path", p), zap.Error(err))
					}
					return nil
				})
			}
			w.mu.Unlock()
		} else if isVideoPath(path) {
			w.scheduleScanForPath(ctx, path)
		}
	case ev.Has(fsnotify.Write), ev.Has(fsnotify.Rename):
		if isVideoPath(path) {
			w.scheduleScanForPath(ctx, path)
		}
	case ev.Has(fsnotify.Remove):
		w.mu.Lock()
		if w.watcher != nil {
			w.removeWatchLocked(path)
		}
		w.mu.Unlock()
	}
}

func (w *Watcher) scheduleScanForPath(ctx context.Context, filePath string) {
	roots, err := w.opts.Lister.ListLibraryPathStrings(ctx)
	if err != nil {
		w.opts.Logger.Warn("library watch: list roots failed", zap.Error(err))
		return
	}
	roots = cleanRoots(roots)
	root := longestMatchingRoot(roots, filePath)
	if root == "" {
		return
	}
	w.debounceMu.Lock()
	defer w.debounceMu.Unlock()
	if w.pendingRoots == nil {
		w.pendingRoots = make(map[string]struct{})
	}
	w.pendingRoots[root] = struct{}{}
	if w.debounceTimer != nil {
		w.debounceTimer.Stop()
	}
	d := w.opts.Debounce
	w.debounceTimer = time.AfterFunc(d, w.flushDebounce)
}

func (w *Watcher) flushDebounce() {
	w.debounceMu.Lock()
	roots := make([]string, 0, len(w.pendingRoots))
	for r := range w.pendingRoots {
		roots = append(roots, r)
	}
	w.pendingRoots = make(map[string]struct{})
	w.debounceTimer = nil
	w.debounceMu.Unlock()

	if len(roots) == 0 {
		return
	}
	slices.Sort(roots)
	w.opts.Logger.Info("library watch: enqueue scan", zap.Strings("roots", roots))
	w.opts.Queue.EnqueueLibraryWatchScanRoots(roots)
}

func cleanRoots(in []string) []string {
	out := make([]string, 0, len(in))
	seen := make(map[string]struct{})
	for _, p := range in {
		p = filepath.Clean(strings.TrimSpace(p))
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	slices.SortFunc(out, func(a, b string) int {
		return len(b) - len(a)
	})
	return out
}

func longestMatchingRoot(roots []string, path string) string {
	path = filepath.Clean(path)
	for _, root := range roots {
		if path == root {
			return root
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			continue
		}
		if rel == "." {
			return root
		}
		if !strings.HasPrefix(rel, "..") {
			return root
		}
	}
	return ""
}

func isVideoPath(p string) bool {
	base := filepath.Base(p)
	if base == "" || strings.HasSuffix(strings.ToLower(base), ".part") || strings.HasSuffix(strings.ToLower(base), ".tmp") {
		return false
	}
	if strings.HasPrefix(base, ".") || strings.HasSuffix(base, "~") {
		return false
	}
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(base)), ".")
	if ext == "" {
		return false
	}
	_, ok := videoExtensions[ext]
	return ok
}

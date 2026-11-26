// library_watch.go 保存影片、漫画和写真的目录监听编排。
// 这些方法仍挂在 App 上，只是从 app.go 拆出，避免启动和扫描逻辑继续堆在同一个文件里。
package app

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"slices"
	"strings"

	"go.uber.org/zap"

	"curated-backend/internal/comicwatch"
	"curated-backend/internal/contracts"
	"curated-backend/internal/librarywatch"
	"curated-backend/internal/photowatch"
)

// ReloadLibraryWatches re-reads library roots from the database and rebuilds directory watches.
func (a *App) ReloadLibraryWatches(ctx context.Context) error {
	a.libWatchMu.Lock()
	w := a.libWatch
	a.libWatchMu.Unlock()
	if w == nil {
		return nil
	}
	return w.Reload(ctx)
}

// ReloadComicLibraryWatches re-reads comic roots from the database and rebuilds comic directory watches.
func (a *App) ReloadComicLibraryWatches(ctx context.Context) error {
	a.comicWatchMu.Lock()
	w := a.comicWatch
	a.comicWatchMu.Unlock()
	if w == nil {
		return a.EnsureComicLibraryWatchRunning()
	}
	return w.Reload(ctx)
}

// ReloadPhotoLibraryWatches re-reads photo roots from the database and rebuilds photo directory watches.
func (a *App) ReloadPhotoLibraryWatches(ctx context.Context) error {
	a.photoWatchMu.Lock()
	w := a.photoWatch
	a.photoWatchMu.Unlock()
	if w == nil {
		return a.EnsurePhotoLibraryWatchRunning()
	}
	return w.Reload(ctx)
}

// EnqueueLibraryWatchScanRoots queues library roots for a debounced fsnotify-driven scan.
func (a *App) EnqueueLibraryWatchScanRoots(roots []string) {
	if !a.AutoLibraryWatch() {
		return
	}
	if len(roots) == 0 {
		return
	}
	a.watchScanMu.Lock()
	for _, r := range roots {
		r = filepath.Clean(strings.TrimSpace(r))
		if r != "" {
			a.watchScanPending[r] = struct{}{}
		}
	}
	a.watchScanMu.Unlock()
	go a.tryDrainWatchScanQueue()
}

// tryDrainWatchScanQueue 把待扫描的影片库根交给一次 fsnotify 触发的扫描；扫描已在进行时把根放回队列。
func (a *App) tryDrainWatchScanQueue() {
	a.watchScanDrainMu.Lock()
	defer a.watchScanDrainMu.Unlock()

	a.watchScanMu.Lock()
	if len(a.watchScanPending) == 0 {
		a.watchScanMu.Unlock()
		return
	}
	roots := make([]string, 0, len(a.watchScanPending))
	for r := range a.watchScanPending {
		roots = append(roots, r)
	}
	slices.Sort(roots)
	a.watchScanPending = make(map[string]struct{})
	a.watchScanMu.Unlock()

	_, err := a.startLibraryScan(a.appCtx, io.Discard, roots, map[string]any{"trigger": "fsnotify"})
	if err != nil {
		if errors.Is(err, contracts.ErrScanAlreadyRunning) {
			a.watchScanMu.Lock()
			for _, r := range roots {
				a.watchScanPending[r] = struct{}{}
			}
			a.watchScanMu.Unlock()
			return
		}
		a.logger.Warn("library watch: failed to start scan", zap.Error(err))
		a.watchScanMu.Lock()
		for _, r := range roots {
			a.watchScanPending[r] = struct{}{}
		}
		a.watchScanMu.Unlock()
	}
}

// EnqueueComicLibraryWatchScanRoots queues comic roots for a debounced fsnotify-driven scan.
func (a *App) EnqueueComicLibraryWatchScanRoots(roots []string) {
	if !a.ComicLibraryEnabled() || !a.AutoComicLibraryWatch() {
		return
	}
	if len(roots) == 0 {
		return
	}
	a.comicWatchScanMu.Lock()
	if a.comicWatchScanPending == nil {
		a.comicWatchScanPending = make(map[string]struct{})
	}
	for _, r := range roots {
		r = filepath.Clean(strings.TrimSpace(r))
		if r != "" {
			a.comicWatchScanPending[r] = struct{}{}
		}
	}
	a.comicWatchScanMu.Unlock()
	go a.tryDrainComicWatchScanQueue()
}

// tryDrainComicWatchScanQueue 把待扫描的漫画库根交给一次 fsnotify 触发的扫描。
func (a *App) tryDrainComicWatchScanQueue() {
	a.comicWatchScanDrainMu.Lock()
	defer a.comicWatchScanDrainMu.Unlock()

	a.comicWatchScanMu.Lock()
	if len(a.comicWatchScanPending) == 0 {
		a.comicWatchScanMu.Unlock()
		return
	}
	roots := make([]string, 0, len(a.comicWatchScanPending))
	for r := range a.comicWatchScanPending {
		roots = append(roots, r)
	}
	slices.Sort(roots)
	a.comicWatchScanPending = make(map[string]struct{})
	a.comicWatchScanMu.Unlock()

	if !a.ComicLibraryEnabled() || !a.AutoComicLibraryWatch() {
		return
	}
	if a.store == nil {
		if a.logger != nil {
			a.logger.Warn("comic watch: scan store is not available")
		}
		return
	}

	ctx := a.appCtx
	if ctx == nil {
		ctx = context.Background()
	}
	allPaths, err := a.store.ListComicLibraryPaths(ctx)
	if err != nil {
		if a.logger != nil {
			a.logger.Warn("comic watch: list comic paths failed", zap.Error(err))
		}
		a.requeueComicWatchScanRoots(roots)
		return
	}

	pendingRoots := make(map[string]struct{}, len(roots))
	for _, root := range roots {
		pendingRoots[filepath.Clean(root)] = struct{}{}
	}
	scanPaths := make([]contracts.ComicLibraryPathDTO, 0, len(allPaths))
	for _, p := range allPaths {
		cleanPath := filepath.Clean(strings.TrimSpace(p.Path))
		if _, ok := pendingRoots[cleanPath]; ok {
			scanPaths = append(scanPaths, p)
		}
	}
	if len(scanPaths) == 0 {
		return
	}

	_, err = a.startComicScan(ctx, scanPaths, map[string]any{
		"trigger": "fsnotify",
		"paths":   roots,
	})
	if err != nil {
		if errors.Is(err, contracts.ErrScanAlreadyRunning) {
			a.requeueComicWatchScanRoots(roots)
			return
		}
		if a.logger != nil {
			a.logger.Warn("comic watch: failed to start scan", zap.Error(err))
		}
		a.requeueComicWatchScanRoots(roots)
	}
}

// requeueComicWatchScanRoots 在扫描未能开始时把漫画库根放回待处理集合。
func (a *App) requeueComicWatchScanRoots(roots []string) {
	a.comicWatchScanMu.Lock()
	if a.comicWatchScanPending == nil {
		a.comicWatchScanPending = make(map[string]struct{})
	}
	for _, root := range roots {
		root = filepath.Clean(strings.TrimSpace(root))
		if root != "" {
			a.comicWatchScanPending[root] = struct{}{}
		}
	}
	a.comicWatchScanMu.Unlock()
}

// EnqueuePhotoLibraryWatchScanRoots queues photo roots for a debounced fsnotify-driven scan.
func (a *App) EnqueuePhotoLibraryWatchScanRoots(roots []string) {
	if !a.PhotoLibraryEnabled() || !a.AutoPhotoLibraryWatch() {
		return
	}
	if len(roots) == 0 {
		return
	}
	a.photoWatchScanMu.Lock()
	if a.photoWatchScanPending == nil {
		a.photoWatchScanPending = make(map[string]struct{})
	}
	for _, r := range roots {
		r = filepath.Clean(strings.TrimSpace(r))
		if r != "" {
			a.photoWatchScanPending[r] = struct{}{}
		}
	}
	a.photoWatchScanMu.Unlock()
	go a.tryDrainPhotoWatchScanQueue()
}

// tryDrainPhotoWatchScanQueue 把待扫描的写真库根交给一次 fsnotify 触发的扫描。
func (a *App) tryDrainPhotoWatchScanQueue() {
	a.photoWatchScanDrainMu.Lock()
	defer a.photoWatchScanDrainMu.Unlock()

	a.photoWatchScanMu.Lock()
	if len(a.photoWatchScanPending) == 0 {
		a.photoWatchScanMu.Unlock()
		return
	}
	roots := make([]string, 0, len(a.photoWatchScanPending))
	for r := range a.photoWatchScanPending {
		roots = append(roots, r)
	}
	slices.Sort(roots)
	a.photoWatchScanPending = make(map[string]struct{})
	a.photoWatchScanMu.Unlock()

	if !a.PhotoLibraryEnabled() || !a.AutoPhotoLibraryWatch() {
		return
	}
	if a.store == nil {
		if a.logger != nil {
			a.logger.Warn("photo watch: scan store is not available")
		}
		return
	}

	ctx := a.appCtx
	if ctx == nil {
		ctx = context.Background()
	}
	allPaths, err := a.store.ListPhotoLibraryPaths(ctx)
	if err != nil {
		if a.logger != nil {
			a.logger.Warn("photo watch: list photo paths failed", zap.Error(err))
		}
		a.requeuePhotoWatchScanRoots(roots)
		return
	}

	pendingRoots := make(map[string]struct{}, len(roots))
	for _, root := range roots {
		pendingRoots[filepath.Clean(root)] = struct{}{}
	}
	scanPaths := make([]contracts.PhotoLibraryPathDTO, 0, len(allPaths))
	for _, p := range allPaths {
		cleanPath := filepath.Clean(strings.TrimSpace(p.Path))
		if _, ok := pendingRoots[cleanPath]; ok {
			scanPaths = append(scanPaths, p)
		}
	}
	if len(scanPaths) == 0 {
		return
	}

	_, err = a.startPhotoScan(ctx, scanPaths, map[string]any{
		"trigger": "fsnotify",
		"paths":   roots,
	})
	if err != nil {
		if errors.Is(err, contracts.ErrScanAlreadyRunning) {
			a.requeuePhotoWatchScanRoots(roots)
			return
		}
		if a.logger != nil {
			a.logger.Warn("photo watch: failed to start scan", zap.Error(err))
		}
		a.requeuePhotoWatchScanRoots(roots)
	}
}

// requeuePhotoWatchScanRoots 在扫描未能开始时把写真库根放回待处理集合。
func (a *App) requeuePhotoWatchScanRoots(roots []string) {
	a.photoWatchScanMu.Lock()
	if a.photoWatchScanPending == nil {
		a.photoWatchScanPending = make(map[string]struct{})
	}
	for _, root := range roots {
		root = filepath.Clean(strings.TrimSpace(root))
		if root != "" {
			a.photoWatchScanPending[root] = struct{}{}
		}
	}
	a.photoWatchScanMu.Unlock()
}

// EnsureLibraryWatchRunning starts the fsnotify Run loop when yaml LibraryWatchOn and AutoLibraryWatch are true.
func (a *App) EnsureLibraryWatchRunning() error {
	if !a.cfg.LibraryWatchOn() {
		return nil
	}
	if !a.AutoLibraryWatch() {
		return nil
	}
	a.libWatchMu.Lock()
	if a.watchLoopCancel != nil {
		a.libWatchMu.Unlock()
		return nil
	}
	if a.libWatch == nil {
		lw, err := librarywatch.New(librarywatch.Options{
			Enabled:  true,
			Debounce: a.cfg.LibraryWatchDebounce(),
			Logger:   a.logger,
			Lister:   a.store,
			Queue:    a,
		})
		if err != nil {
			a.libWatchMu.Unlock()
			return err
		}
		a.libWatch = lw
	}
	w := a.libWatch
	a.watchLoopSession++
	sess := a.watchLoopSession
	watchCtx, cancel := context.WithCancel(a.appCtx)
	a.watchLoopCancel = cancel
	a.libWatchMu.Unlock()

	go func(sess uint64) {
		// 在独立 goroutine 里跑影片库监听，只在会话号未变时清掉 cancel。
		err := w.Run(watchCtx)
		a.libWatchMu.Lock()
		if a.watchLoopSession == sess {
			a.watchLoopCancel = nil
		}
		a.libWatchMu.Unlock()
		if err != nil && !errors.Is(err, context.Canceled) {
			a.logger.Warn("library fsnotify watcher exited", zap.Error(err))
		}
	}(sess)

	return nil
}

// StopLibraryWatchLoop cancels the active fsnotify Run context (watcher instance is kept for Reload / restart).
func (a *App) StopLibraryWatchLoop() {
	a.libWatchMu.Lock()
	cancel := a.watchLoopCancel
	a.watchLoopCancel = nil
	a.watchLoopSession++
	a.libWatchMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// EnsureComicLibraryWatchRunning starts the comic fsnotify Run loop when all comic watch gates are open.
func (a *App) EnsureComicLibraryWatchRunning() error {
	if !a.cfg.LibraryWatchOn() {
		return nil
	}
	if !a.ComicLibraryEnabled() || !a.AutoComicLibraryWatch() {
		return nil
	}
	if a.store == nil {
		return nil
	}
	a.comicWatchMu.Lock()
	if a.comicWatchLoopCancel != nil {
		a.comicWatchMu.Unlock()
		return nil
	}
	if a.comicWatch == nil {
		cw, err := comicwatch.New(comicwatch.Options{
			Enabled:  true,
			Debounce: a.cfg.LibraryWatchDebounce(),
			Logger:   a.logger,
			Lister:   a.store,
			Queue:    a,
		})
		if err != nil {
			a.comicWatchMu.Unlock()
			return err
		}
		a.comicWatch = cw
	}
	w := a.comicWatch
	a.comicWatchLoopSession++
	sess := a.comicWatchLoopSession
	baseCtx := a.appCtx
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	watchCtx, cancel := context.WithCancel(baseCtx)
	a.comicWatchLoopCancel = cancel
	a.comicWatchMu.Unlock()

	go func(sess uint64) {
		// 在独立 goroutine 里跑漫画库监听，只在会话号未变时清掉 cancel。
		err := w.Run(watchCtx)
		a.comicWatchMu.Lock()
		if a.comicWatchLoopSession == sess {
			a.comicWatchLoopCancel = nil
		}
		a.comicWatchMu.Unlock()
		if err != nil && !errors.Is(err, context.Canceled) && a.logger != nil {
			a.logger.Warn("comic fsnotify watcher exited", zap.Error(err))
		}
	}(sess)

	return nil
}

// StopComicLibraryWatchLoop cancels the active comic fsnotify Run context.
func (a *App) StopComicLibraryWatchLoop() {
	a.comicWatchMu.Lock()
	cancel := a.comicWatchLoopCancel
	a.comicWatchLoopCancel = nil
	a.comicWatchLoopSession++
	a.comicWatchMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// EnsurePhotoLibraryWatchRunning starts the photo fsnotify Run loop when all photo watch gates are open.
func (a *App) EnsurePhotoLibraryWatchRunning() error {
	if !a.cfg.LibraryWatchOn() {
		return nil
	}
	if !a.PhotoLibraryEnabled() || !a.AutoPhotoLibraryWatch() {
		return nil
	}
	if a.store == nil {
		return nil
	}
	a.photoWatchMu.Lock()
	if a.photoWatchLoopCancel != nil {
		a.photoWatchMu.Unlock()
		return nil
	}
	if a.photoWatch == nil {
		pw, err := photowatch.New(photowatch.Options{
			Enabled:  true,
			Debounce: a.cfg.LibraryWatchDebounce(),
			Logger:   a.logger,
			Lister:   a.store,
			Queue:    a,
		})
		if err != nil {
			a.photoWatchMu.Unlock()
			return err
		}
		a.photoWatch = pw
	}
	w := a.photoWatch
	a.photoWatchLoopSession++
	sess := a.photoWatchLoopSession
	baseCtx := a.appCtx
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	watchCtx, cancel := context.WithCancel(baseCtx)
	a.photoWatchLoopCancel = cancel
	a.photoWatchMu.Unlock()

	go func(sess uint64) {
		// 在独立 goroutine 里跑写真库监听，只在会话号未变时清掉 cancel。
		err := w.Run(watchCtx)
		a.photoWatchMu.Lock()
		if a.photoWatchLoopSession == sess {
			a.photoWatchLoopCancel = nil
		}
		a.photoWatchMu.Unlock()
		if err != nil && !errors.Is(err, context.Canceled) && a.logger != nil {
			a.logger.Warn("photo fsnotify watcher exited", zap.Error(err))
		}
	}(sess)

	return nil
}

// StopPhotoLibraryWatchLoop cancels the active photo fsnotify Run context.
func (a *App) StopPhotoLibraryWatchLoop() {
	a.photoWatchMu.Lock()
	cancel := a.photoWatchLoopCancel
	a.photoWatchLoopCancel = nil
	a.photoWatchLoopSession++
	a.photoWatchMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

package app

import (
	"context"
	"time"

	"go.uber.org/zap"
)

const (
	autoActorProfileScrapeCooldown    = 24 * time.Hour
	autoActorProfileSweepBatchLimit   = 50
	autoActorProfileSweepInterval     = 15 * time.Minute
	autoActorProfileSweepStartupDelay = 20 * time.Second
	autoActorProfileSweepTrigger      = "auto.missing-profile"
)

func (a *App) autoActorSweepContext() context.Context {
	if a != nil && a.appCtx != nil {
		return a.appCtx
	}
	return context.Background()
}

// StartAutoActorProfileScrapeLoop backfills missing actor profiles while the setting stays on.
func (a *App) StartAutoActorProfileScrapeLoop(ctx context.Context) {
	if a == nil {
		return
	}
	go func() {
		timer := time.NewTimer(autoActorProfileSweepStartupDelay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			a.enqueueMissingLibraryActorProfileSweep(ctx)
		}
		ticker := time.NewTicker(autoActorProfileSweepInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				a.enqueueMissingLibraryActorProfileSweep(ctx)
			}
		}
	}()
}

func (a *App) enqueueMissingLibraryActorProfileSweep(ctx context.Context) {
	if a == nil || a.store == nil || !a.AutoActorProfileScrape() {
		return
	}
	if !a.autoActorProfileSweepMu.TryLock() {
		return
	}
	defer a.autoActorProfileSweepMu.Unlock()

	if ctx == nil {
		ctx = a.autoActorSweepContext()
	}
	names, err := a.store.ListActorsNeedingProfileScrape(ctx, autoActorProfileSweepBatchLimit)
	if err != nil {
		if a.logger != nil {
			a.logger.Warn("failed to list actors needing profile scrape", zap.Error(err))
		}
		return
	}
	if len(names) == 0 {
		return
	}
	if a.logger != nil {
		a.logger.Info("auto actor profile sweep", zap.Int("candidates", len(names)))
	}
	a.enqueueAutoActorProfileScrapes(ctx, names, autoActorProfileSweepTrigger, true)
}

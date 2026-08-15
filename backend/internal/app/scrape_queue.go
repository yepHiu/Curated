package app

import (
	"context"
	"strings"
	"time"

	"go.uber.org/zap"

	"curated-backend/internal/contracts"
)

func (a *App) acquireScrapeSlot(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case a.scrapeSem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (a *App) releaseScrapeSlot() {
	<-a.scrapeSem
}

func (a *App) lookupMovieScrape(movieID string) (string, bool) {
	id := strings.TrimSpace(movieID)
	if id == "" {
		return "", false
	}
	a.scrapeMovieMu.Lock()
	defer a.scrapeMovieMu.Unlock()
	taskID, ok := a.scrapeMovieInflight[id]
	return taskID, ok && strings.TrimSpace(taskID) != ""
}

func (a *App) claimMovieScrape(movieID, taskID string) (existingTaskID string, claimed bool) {
	id := strings.TrimSpace(movieID)
	if id == "" {
		return "", true
	}
	a.scrapeMovieMu.Lock()
	defer a.scrapeMovieMu.Unlock()
	if a.scrapeMovieInflight == nil {
		a.scrapeMovieInflight = make(map[string]string)
	}
	if existing, ok := a.scrapeMovieInflight[id]; ok && strings.TrimSpace(existing) != "" {
		return existing, false
	}
	a.scrapeMovieInflight[id] = strings.TrimSpace(taskID)
	return "", true
}

func (a *App) finishMovieScrape(movieID, taskID string) {
	id := strings.TrimSpace(movieID)
	if id == "" {
		return
	}
	a.scrapeMovieMu.Lock()
	defer a.scrapeMovieMu.Unlock()
	if a.scrapeMovieInflight[id] == strings.TrimSpace(taskID) {
		delete(a.scrapeMovieInflight, id)
	}
}

func (a *App) persistFailedScrapeTask(taskID, message string) contracts.TaskDTO {
	task := a.tasks.Fail(taskID, contracts.ErrorCodeScraperRun, message)
	parent := context.Background()
	if a.appCtx != nil {
		parent = a.appCtx
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(parent), 5*time.Second)
	defer cancel()
	if err := a.store.SaveTask(ctx, task); err != nil && a.logger != nil {
		a.logger.Error("failed to persist cancelled scraper task", zap.Error(err), zap.String("taskId", taskID))
	}
	return task
}

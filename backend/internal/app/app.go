package app

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"net/http"

	"jav-shadcn/backend/internal/assets"
	"jav-shadcn/backend/internal/config"
	"jav-shadcn/backend/internal/contracts"
	"jav-shadcn/backend/internal/library"
	"jav-shadcn/backend/internal/scanner"
	"jav-shadcn/backend/internal/scraper"
	"jav-shadcn/backend/internal/scraper/metatube"
	"jav-shadcn/backend/internal/server"
	"jav-shadcn/backend/internal/storage"
	"jav-shadcn/backend/internal/tasks"
)

const appVersion = "0.1.0"

type App struct {
	cfg     config.Config
	logger  *zap.Logger
	store   *storage.SQLiteStore
	library *library.Service
	scanner *scanner.Service
	scraper scraper.Service
	assets  *assets.Service
	tasks   *tasks.Manager

	appCtx  context.Context
	writeMu sync.Mutex
}

func New(ctx context.Context, cfg config.Config, logger *zap.Logger, store *storage.SQLiteStore) (*App, error) {
	scraperService, err := metatube.NewService(logger, time.Duration(cfg.Scraper.RequestTimeoutSeconds)*time.Second)
	if err != nil {
		return nil, err
	}

	return &App{
		cfg:     cfg,
		logger:  logger,
		store:   store,
		library: library.NewService(),
		scanner: scanner.NewService(logger),
		scraper: scraperService,
		assets:  assets.NewService(logger, cfg.CacheDir, time.Duration(cfg.Assets.RequestTimeoutSeconds)*time.Second),
		tasks:   tasks.NewManager(),
		appCtx:  ctx,
	}, nil
}

func (a *App) Run(ctx context.Context, input io.Reader, output io.Writer) error {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return err
			}
			return nil
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var command contracts.Command
		if err := json.Unmarshal([]byte(line), &command); err != nil {
			if writeErr := a.writeResponse(output, contracts.Response{
				Kind:      "response",
				ID:        "",
				OK:        false,
				Error:     badRequest("invalid command json", map[string]any{"cause": err.Error()}),
				Timestamp: nowUTC(),
			}); writeErr != nil {
				return writeErr
			}
			continue
		}

		if err := a.handleCommand(ctx, output, command); err != nil {
			return err
		}
	}
}

func (a *App) handleCommand(ctx context.Context, output io.Writer, command contracts.Command) error {
	a.logger.Info("handling command", zap.String("command", command.Type), zap.String("commandId", command.ID))

	switch command.Type {
	case contracts.CommandSystemHealth:
		return a.respondOK(output, command.ID, contracts.HealthDTO{
			Name:         "javd",
			Version:      appVersion,
			Transport:    "stdio-jsonl",
			DatabasePath: a.cfg.DatabasePath,
		})

	case contracts.CommandLibraryList:
		var request contracts.ListMoviesRequest
		if err := decodePayload(command.Payload, &request); err != nil {
			return a.respondError(output, command.ID, badRequest("invalid library.list payload", map[string]any{"cause": err.Error()}))
		}

		result, err := a.store.ListMovies(ctx, request)
		if err != nil {
			return a.respondError(output, command.ID, internalError("failed to list movies", map[string]any{"cause": err.Error()}))
		}
		if result.Total == 0 {
			result = a.library.ListMovies(request)
		}
		return a.respondOK(output, command.ID, result)

	case contracts.CommandLibraryDetail:
		var request contracts.GetMovieDetailRequest
		if err := decodePayload(command.Payload, &request); err != nil {
			return a.respondError(output, command.ID, badRequest("invalid library.detail payload", map[string]any{"cause": err.Error()}))
		}
		if request.MovieID == "" {
			return a.respondError(output, command.ID, badRequest("movieId is required", nil))
		}

		movie, err := a.store.GetMovieDetail(ctx, request.MovieID)
		if err != nil {
			if err == sql.ErrNoRows {
				movie, err = a.library.GetMovie(request.MovieID)
			}
		}
		if err != nil {
			if err == sql.ErrNoRows || library.IsNotFound(err) {
				return a.respondError(output, command.ID, notFound("movie was not found", map[string]any{"movieId": request.MovieID}))
			}
			return a.respondError(output, command.ID, internalError("failed to load movie", map[string]any{"movieId": request.MovieID}))
		}

		return a.respondOK(output, command.ID, movie)

	case contracts.CommandSettingsGet:
		libraryPaths := make([]contracts.LibraryPathDTO, 0, len(a.cfg.LibraryPaths))
		for index, path := range a.cfg.LibraryPaths {
			libraryPaths = append(libraryPaths, contracts.LibraryPathDTO{
				ID:    fmt.Sprintf("library-%d", index+1),
				Path:  path,
				Title: fmt.Sprintf("Library path %d", index+1),
			})
		}

		settings := contracts.SettingsDTO{
			LibraryPaths:        libraryPaths,
			ScanIntervalSeconds: a.cfg.ScanIntervalSeconds,
			Player: contracts.PlayerSettingsDTO{
				HardwareDecode: a.cfg.Player.HardwareDecode,
			},
		}
		return a.respondOK(output, command.ID, settings)

	case contracts.CommandScanStart:
		var request contracts.StartScanRequest
		if err := decodePayload(command.Payload, &request); err != nil {
			return a.respondError(output, command.ID, badRequest("invalid scan.start payload", map[string]any{"cause": err.Error()}))
		}

		paths := request.Paths
		if len(paths) == 0 {
			paths = append(paths, a.cfg.LibraryPaths...)
		}

		task := a.tasks.Create("scan.library", map[string]any{"paths": paths})
		task = a.tasks.Start(task.TaskID, fmt.Sprintf("Scanning %d library path(s)", len(paths)))
		if err := a.store.SaveTask(ctx, task); err != nil {
			return a.respondError(output, command.ID, internalError("failed to persist scan task", map[string]any{"cause": err.Error()}))
		}

		if err := a.emitEvent(output, contracts.EventTaskStarted, contracts.TaskEventDTO{Task: task}); err != nil {
			return err
		}
		if err := a.emitEvent(output, contracts.EventScanStarted, contracts.TaskEventDTO{Task: task}); err != nil {
			return err
		}

		go a.runScan(ctx, output, task.TaskID, paths)

		return a.respondOK(output, command.ID, task)

	case contracts.CommandScanStatus:
		var request contracts.GetTaskStatusRequest
		if err := decodePayload(command.Payload, &request); err != nil {
			return a.respondError(output, command.ID, badRequest("invalid scan.status payload", map[string]any{"cause": err.Error()}))
		}
		if request.TaskID == "" {
			return a.respondError(output, command.ID, badRequest("taskId is required", nil))
		}

		task, ok := a.tasks.Get(request.TaskID)
		if !ok {
			return a.respondError(output, command.ID, notFound("scan task was not found", map[string]any{"taskId": request.TaskID}))
		}

		return a.respondOK(output, command.ID, task)

	default:
		return a.respondError(output, command.ID, &contracts.AppError{
			Code:      contracts.ErrorCodeUnsupported,
			Message:   "unsupported command",
			Retryable: false,
			Details: map[string]any{
				"command": command.Type,
			},
		})
	}
}

func (a *App) runScan(parentCtx context.Context, output io.Writer, taskID string, paths []string) {
	ctx, cancel := context.WithTimeout(parentCtx, time.Duration(a.cfg.Tasks.ScanTimeoutSeconds)*time.Second)
	defer cancel()

	importedCount := 0
	updatedCount := 0
	skippedCount := 0
	persistedResults := make([]contracts.ScanFileResultDTO, 0)

	summary, err := a.scanner.Scan(ctx, taskID, paths, scanner.Hooks{
		OnProgress: func(processed, total int, message string) {
			progress := 100
			if total > 0 {
				progress = int(float64(processed) / float64(total) * 100)
			}

			task := a.tasks.Progress(taskID, progress, message)
			if saveErr := a.store.SaveTask(ctx, task); saveErr != nil {
				a.logger.Error("failed to persist task progress", zap.Error(saveErr), zap.String("taskId", taskID))
			}
			if emitErr := a.emitEvent(output, contracts.EventTaskProgress, contracts.TaskEventDTO{Task: task}); emitErr != nil {
				a.logger.Error("failed to emit task progress", zap.Error(emitErr), zap.String("taskId", taskID))
				return
			}
			if emitErr := a.emitEvent(output, contracts.EventScanProgress, contracts.TaskEventDTO{Task: task}); emitErr != nil {
				a.logger.Error("failed to emit scan progress", zap.Error(emitErr), zap.String("taskId", taskID))
			}
		},
		OnFileDetected: func(result contracts.ScanFileResultDTO) error {
			if result.Number == "" {
				skippedCount++
				if err := a.store.SaveScanItem(ctx, result); err != nil {
					return err
				}
				persistedResults = append(persistedResults, result)

				if emitErr := a.emitEvent(output, contracts.EventScanFileSkipped, result); emitErr != nil {
					a.logger.Error("failed to emit scan skip event", zap.Error(emitErr), zap.String("taskId", taskID))
				}
				return nil
			}

			outcome, err := a.store.PersistScanMovie(ctx, result)
			if err != nil {
				return err
			}

			result.MovieID = outcome.MovieID
			result.Status = outcome.Status
			result.Reason = outcome.Reason

			if err := a.store.SaveScanItem(ctx, result); err != nil {
				return err
			}

			if result.Status == "imported" || result.Status == "updated" {
				a.library.UpsertScannedMovie(result)
			}

			switch result.Status {
			case "imported":
				importedCount++
				if emitErr := a.emitEvent(output, contracts.EventScanFileImported, result); emitErr != nil {
					a.logger.Error("failed to emit scan import event", zap.Error(emitErr), zap.String("taskId", taskID))
				}
				go a.runScrape(parentCtx, output, result)
			case "updated":
				updatedCount++
				if emitErr := a.emitEvent(output, contracts.EventScanFileUpdated, result); emitErr != nil {
					a.logger.Error("failed to emit scan update event", zap.Error(emitErr), zap.String("taskId", taskID))
				}
				go a.runScrape(parentCtx, output, result)
			default:
				skippedCount++
				if emitErr := a.emitEvent(output, contracts.EventScanFileSkipped, result); emitErr != nil {
					a.logger.Error("failed to emit scan skip event", zap.Error(emitErr), zap.String("taskId", taskID))
				}
			}

			persistedResults = append(persistedResults, result)
			return nil
		},
	})
	if err != nil {
		code := contracts.ErrorCodeScanWalk
		if err == context.Canceled {
			code = contracts.ErrorCodeScanStart
		}
		task := a.tasks.Fail(taskID, code, err.Error())
		if saveErr := a.store.SaveTask(ctx, task); saveErr != nil {
			a.logger.Error("failed to persist failed task", zap.Error(saveErr), zap.String("taskId", taskID))
		}
		_ = a.emitEvent(output, contracts.EventTaskFailed, contracts.TaskEventDTO{Task: task})
		return
	}

	summary.Results = persistedResults
	summary.FilesImported = importedCount
	summary.FilesUpdated = updatedCount
	summary.FilesSkipped = skippedCount

	taskMessage := fmt.Sprintf(
		"Scan finished: %d discovered, %d imported, %d updated, %d skipped",
		summary.FilesDiscovered,
		summary.FilesImported,
		summary.FilesUpdated,
		summary.FilesSkipped,
	)
	task := a.tasks.Complete(taskID, taskMessage)
	if saveErr := a.store.SaveTask(ctx, task); saveErr != nil {
		a.logger.Error("failed to persist completed task", zap.Error(saveErr), zap.String("taskId", taskID))
	}
	if err := a.emitEvent(output, contracts.EventTaskCompleted, contracts.TaskEventDTO{Task: task}); err != nil {
		a.logger.Error("failed to emit task completion", zap.Error(err), zap.String("taskId", taskID))
		return
	}

	if err := a.emitEvent(output, contracts.EventScanCompleted, struct {
		Task    contracts.TaskDTO        `json:"task"`
		Summary contracts.ScanSummaryDTO `json:"summary"`
	}{
		Task:    task,
		Summary: summary,
	}); err != nil {
		a.logger.Error("failed to emit scan completion", zap.Error(err), zap.String("taskId", taskID))
	}
}

func (a *App) runScrape(parentCtx context.Context, output io.Writer, result contracts.ScanFileResultDTO) {
	ctx, cancel := context.WithTimeout(parentCtx, time.Duration(a.cfg.Scraper.TaskTimeoutSeconds)*time.Second)
	defer cancel()

	task := a.tasks.Create("scrape.movie", map[string]any{
		"movieId": result.MovieID,
		"number":  result.Number,
		"path":    result.Path,
	})
	task = a.tasks.Start(task.TaskID, fmt.Sprintf("Scraping metadata for %s", result.Number))
	if err := a.store.SaveTask(ctx, task); err != nil {
		a.logger.Error("failed to persist scraper task", zap.Error(err), zap.String("taskId", task.TaskID))
	}
	if err := a.emitEvent(output, contracts.EventTaskStarted, contracts.TaskEventDTO{Task: task}); err != nil {
		a.logger.Error("failed to emit scraper task start", zap.Error(err), zap.String("taskId", task.TaskID))
	}

	metadata, err := a.scraper.Scrape(ctx, result.MovieID, result.Number)
	if err != nil {
		code := contracts.ErrorCodeScraperRun
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			code = contracts.ErrorCodeScraperRun
		}
		task = a.tasks.Fail(task.TaskID, code, err.Error())
		if saveErr := a.store.SaveTask(ctx, task); saveErr != nil {
			a.logger.Error("failed to persist failed scraper task", zap.Error(saveErr), zap.String("taskId", task.TaskID))
		}
		_ = a.emitEvent(output, contracts.EventTaskFailed, contracts.TaskEventDTO{Task: task})
		_ = a.emitEvent(output, contracts.EventScraperFailed, struct {
			Task    contracts.TaskDTO `json:"task"`
			MovieID string            `json:"movieId"`
			Number  string            `json:"number"`
			Error   string            `json:"error"`
		}{
			Task:    task,
			MovieID: result.MovieID,
			Number:  result.Number,
			Error:   err.Error(),
		})
		return
	}

	task = a.tasks.Progress(task.TaskID, 80, "Persisting scraped metadata")
	if err := a.store.SaveTask(ctx, task); err != nil {
		a.logger.Error("failed to persist scraper progress", zap.Error(err), zap.String("taskId", task.TaskID))
	}
	if err := a.emitEvent(output, contracts.EventTaskProgress, contracts.TaskEventDTO{Task: task}); err != nil {
		a.logger.Error("failed to emit scraper progress", zap.Error(err), zap.String("taskId", task.TaskID))
	}

	if err := a.store.SaveMovieMetadata(ctx, metadata); err != nil {
		task = a.tasks.Fail(task.TaskID, contracts.ErrorCodeScraperRun, err.Error())
		if saveErr := a.store.SaveTask(ctx, task); saveErr != nil {
			a.logger.Error("failed to persist scraper failure", zap.Error(saveErr), zap.String("taskId", task.TaskID))
		}
		_ = a.emitEvent(output, contracts.EventTaskFailed, contracts.TaskEventDTO{Task: task})
		_ = a.emitEvent(output, contracts.EventScraperFailed, struct {
			Task    contracts.TaskDTO `json:"task"`
			MovieID string            `json:"movieId"`
			Number  string            `json:"number"`
			Error   string            `json:"error"`
		}{
			Task:    task,
			MovieID: result.MovieID,
			Number:  result.Number,
			Error:   err.Error(),
		})
		return
	}

	a.library.ApplyScrapedMetadata(metadata)

	task = a.tasks.Complete(task.TaskID, fmt.Sprintf("Metadata saved for %s", result.Number))
	if err := a.store.SaveTask(ctx, task); err != nil {
		a.logger.Error("failed to persist completed scraper task", zap.Error(err), zap.String("taskId", task.TaskID))
	}
	if err := a.emitEvent(output, contracts.EventTaskCompleted, contracts.TaskEventDTO{Task: task}); err != nil {
		a.logger.Error("failed to emit scraper completion", zap.Error(err), zap.String("taskId", task.TaskID))
	}
	if err := a.emitEvent(output, contracts.EventScraperMetadataSaved, struct {
		Task     contracts.TaskDTO `json:"task"`
		MovieID  string            `json:"movieId"`
		Number   string            `json:"number"`
		Title    string            `json:"title"`
		Provider string            `json:"provider"`
	}{
		Task:     task,
		MovieID:  metadata.MovieID,
		Number:   metadata.Number,
		Title:    metadata.Title,
		Provider: metadata.Provider,
	}); err != nil {
		a.logger.Error("failed to emit metadata saved event", zap.Error(err), zap.String("taskId", task.TaskID))
	}

	go a.runAssetDownload(parentCtx, output, metadata)
}

func (a *App) runAssetDownload(parentCtx context.Context, output io.Writer, metadata scraper.Metadata) {
	ctx, cancel := context.WithTimeout(parentCtx, time.Duration(a.cfg.Assets.TaskTimeoutSeconds)*time.Second)
	defer cancel()

	task := a.tasks.Create("asset.download", map[string]any{
		"movieId": metadata.MovieID,
		"number":  metadata.Number,
	})
	task = a.tasks.Start(task.TaskID, fmt.Sprintf("Downloading assets for %s", metadata.Number))
	if err := a.store.SaveTask(ctx, task); err != nil {
		a.logger.Error("failed to persist asset task", zap.Error(err), zap.String("taskId", task.TaskID))
	}
	_ = a.emitEvent(output, contracts.EventTaskStarted, contracts.TaskEventDTO{Task: task})

	downloaded, err := a.assets.DownloadAll(ctx, metadata)
	if err != nil {
		task = a.tasks.Fail(task.TaskID, contracts.ErrorCodeAssetDownload, err.Error())
		if saveErr := a.store.SaveTask(ctx, task); saveErr != nil {
			a.logger.Error("failed to persist failed asset task", zap.Error(saveErr), zap.String("taskId", task.TaskID))
		}
		_ = a.emitEvent(output, contracts.EventTaskFailed, contracts.TaskEventDTO{Task: task})
		_ = a.emitEvent(output, contracts.EventAssetDownloadFailed, struct {
			Task    contracts.TaskDTO `json:"task"`
			MovieID string            `json:"movieId"`
			Number  string            `json:"number"`
			Error   string            `json:"error"`
		}{
			Task:    task,
			MovieID: metadata.MovieID,
			Number:  metadata.Number,
			Error:   err.Error(),
		})
		return
	}

	for index, asset := range downloaded {
		progress := int(float64(index+1) / float64(len(downloaded)) * 100)
		task = a.tasks.Progress(task.TaskID, progress, fmt.Sprintf("Downloaded %s asset", asset.Type))
		if err := a.store.SaveTask(ctx, task); err != nil {
			a.logger.Error("failed to persist asset progress", zap.Error(err), zap.String("taskId", task.TaskID))
		}
		if err := a.store.UpdateMediaAssetLocalPath(ctx, metadata.MovieID, asset.Type, asset.SourceURL, asset.LocalPath); err != nil {
			a.logger.Error("failed to persist asset local path", zap.Error(err), zap.String("taskId", task.TaskID))
		}
		_ = a.emitEvent(output, contracts.EventTaskProgress, contracts.TaskEventDTO{Task: task})
		_ = a.emitEvent(output, contracts.EventAssetDownloaded, struct {
			Task      contracts.TaskDTO `json:"task"`
			MovieID   string            `json:"movieId"`
			Number    string            `json:"number"`
			Type      string            `json:"type"`
			SourceURL string            `json:"sourceUrl"`
			LocalPath string            `json:"localPath"`
		}{
			Task:      task,
			MovieID:   metadata.MovieID,
			Number:    metadata.Number,
			Type:      asset.Type,
			SourceURL: asset.SourceURL,
			LocalPath: asset.LocalPath,
		})
	}

	task = a.tasks.Complete(task.TaskID, fmt.Sprintf("Downloaded %d assets for %s", len(downloaded), metadata.Number))
	if err := a.store.SaveTask(ctx, task); err != nil {
		a.logger.Error("failed to persist completed asset task", zap.Error(err), zap.String("taskId", task.TaskID))
	}
	_ = a.emitEvent(output, contracts.EventTaskCompleted, contracts.TaskEventDTO{Task: task})
}

func (a *App) respondOK(output io.Writer, id string, data any) error {
	return a.writeResponse(output, contracts.Response{
		Kind:      "response",
		ID:        id,
		OK:        true,
		Data:      data,
		Timestamp: nowUTC(),
	})
}

func (a *App) respondError(output io.Writer, id string, appErr *contracts.AppError) error {
	return a.writeResponse(output, contracts.Response{
		Kind:      "response",
		ID:        id,
		OK:        false,
		Error:     appErr,
		Timestamp: nowUTC(),
	})
}

func (a *App) emitEvent(output io.Writer, eventType string, payload any) error {
	return a.writeEvent(output, contracts.Event{
		Kind:      "event",
		Type:      eventType,
		Payload:   payload,
		Timestamp: nowUTC(),
	})
}

func (a *App) writeResponse(output io.Writer, response contracts.Response) error {
	a.writeMu.Lock()
	defer a.writeMu.Unlock()

	return json.NewEncoder(output).Encode(response)
}

func (a *App) writeEvent(output io.Writer, event contracts.Event) error {
	a.writeMu.Lock()
	defer a.writeMu.Unlock()

	return json.NewEncoder(output).Encode(event)
}

func decodePayload(payload json.RawMessage, target any) error {
	if len(payload) == 0 {
		return nil
	}
	return json.Unmarshal(payload, target)
}

func badRequest(message string, details map[string]any) *contracts.AppError {
	return &contracts.AppError{
		Code:      contracts.ErrorCodeBadRequest,
		Message:   message,
		Retryable: false,
		Details:   details,
	}
}

func notFound(message string, details map[string]any) *contracts.AppError {
	return &contracts.AppError{
		Code:      contracts.ErrorCodeNotFound,
		Message:   message,
		Retryable: false,
		Details:   details,
	}
}

func internalError(message string, details map[string]any) *contracts.AppError {
	return &contracts.AppError{
		Code:      contracts.ErrorCodeInternal,
		Message:   message,
		Retryable: true,
		Details:   details,
	}
}

func (a *App) StartScan(_ context.Context, paths []string) (contracts.TaskDTO, error) {
	if len(paths) == 0 {
		paths = append(paths, a.cfg.LibraryPaths...)
	}

	task := a.tasks.Create("scan.library", map[string]any{"paths": paths})
	task = a.tasks.Start(task.TaskID, fmt.Sprintf("Scanning %d library path(s)", len(paths)))
	if err := a.store.SaveTask(a.appCtx, task); err != nil {
		return contracts.TaskDTO{}, err
	}

	go a.runScan(a.appCtx, io.Discard, task.TaskID, paths)

	return task, nil
}

func (a *App) HTTPHandler() http.Handler {
	return server.NewHandler(server.Deps{
		Cfg:         a.cfg,
		Logger:      a.logger,
		Store:       a.store,
		Library:     a.library,
		Tasks:       a.tasks,
		ScanStarter: a,
	}).Routes()
}

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}

package app

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
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
	"jav-shadcn/backend/internal/version"
)

type App struct {
	cfg     config.Config
	logger  *zap.Logger
	store   *storage.SQLiteStore
	library *library.Service
	scanner *scanner.Service
	scraper scraper.Service
	assets  *assets.Service
	tasks   *tasks.Manager

	// organizeLibrary is toggled via Settings UI / PATCH and persisted to library-config.cfg.
	organizeLibrary     bool
	organizeMu          sync.RWMutex
	librarySettingsPath string // JSON file under config/ (organizeLibrary, future keys)

	appCtx   context.Context
	writeMu  sync.Mutex
	scanning atomic.Bool

	// scrapeSem limits concurrent scrape.movie pipelines (network + DB).
	scrapeSem chan struct{}
}

// New 构造并返回可运行的后端 App（依赖注入入口），由 cmd/javd 在加载配置、合并 library-config.cfg、
// 打开数据库并完成迁移后调用。
//
// 参数说明：
//   - ctx：应用生命周期 context，用于取消与超时，存入 App.appCtx。
//   - cfg：已合并主配置与库设置文件的 config.Config（含 OrganizeLibrary、CacheDir、各类超时等）。
//   - logger / store：日志与 SQLite 存储。
//   - librarySettingsPath：持久化库行为开关的 JSON 文件路径（如 config/library-config.cfg），
//     供 SetOrganizeLibrary 原子写回；可为空字符串但此时 PATCH 整理库开关会失败。
//
// 若 Metatube 刮削服务初始化失败，返回 (nil, err)。
func New(ctx context.Context, cfg config.Config, logger *zap.Logger, store *storage.SQLiteStore, librarySettingsPath string) (*App, error) {
	scraperService, err := metatube.NewService(logger, time.Duration(cfg.Scraper.RequestTimeoutSeconds)*time.Second)
	if err != nil {
		return nil, err
	}

	scrapeConc := cfg.Scraper.MaxConcurrent
	if scrapeConc <= 0 {
		scrapeConc = 4
	}

	return &App{
		cfg:     cfg,
		logger:  logger,
		store:   store,
		library: library.NewService(),
		scanner: scanner.NewService(logger),
		scraper: scraperService,
		assets: assets.NewService(
			logger,
			cfg.CacheDir,
			time.Duration(cfg.Assets.RequestTimeoutSeconds)*time.Second,
			cfg.Assets.MaxConcurrentDownloads,
			cfg.Assets.MaxResponseBodyMB,
		),
		tasks:               tasks.NewManager(),
		organizeLibrary:     cfg.OrganizeLibrary,
		librarySettingsPath: strings.TrimSpace(librarySettingsPath),
		appCtx:              ctx,
		scrapeSem:           make(chan struct{}, scrapeConc),
	}, nil
}

// OrganizeLibrary returns whether scan/scrape should move files and write NFO/assets into 番号 folders.
func (a *App) OrganizeLibrary() bool {
	a.organizeMu.RLock()
	defer a.organizeMu.RUnlock()
	return a.organizeLibrary
}

// SetOrganizeLibrary persists organizeLibrary to library-config.cfg, then updates in-memory state.
// Fails without mutating memory if the file cannot be written.
func (a *App) SetOrganizeLibrary(v bool) error {
	path := a.librarySettingsPath
	if path == "" {
		return fmt.Errorf("library settings path not configured")
	}
	if err := config.WriteLibrarySettingsMerge(path, func(m map[string]any) error {
		m["organizeLibrary"] = v
		return nil
	}); err != nil {
		return err
	}

	a.organizeMu.Lock()
	a.organizeLibrary = v
	a.cfg.OrganizeLibrary = v
	a.organizeMu.Unlock()
	return nil
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
			Version:      version.Version,
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
		libraryPaths, err := a.store.ListLibraryPaths(ctx)
		if err != nil {
			return a.respondError(output, command.ID, internalError("failed to list library paths", map[string]any{"cause": err.Error()}))
		}

		settings := contracts.SettingsDTO{
			LibraryPaths: libraryPaths,
			Player: contracts.PlayerSettingsDTO{
				HardwareDecode: a.cfg.Player.HardwareDecode,
			},
			OrganizeLibrary: a.OrganizeLibrary(),
		}
		return a.respondOK(output, command.ID, settings)

	case contracts.CommandScanStart:
		var request contracts.StartScanRequest
		if err := decodePayload(command.Payload, &request); err != nil {
			return a.respondError(output, command.ID, badRequest("invalid scan.start payload", map[string]any{"cause": err.Error()}))
		}

		task, err := a.startLibraryScan(ctx, output, request.Paths)
		if err != nil {
			if errors.Is(err, contracts.ErrScanAlreadyRunning) {
				return a.respondError(output, command.ID, &contracts.AppError{
					Code:      contracts.ErrorCodeConflict,
					Message:   "scan already in progress",
					Retryable: false,
				})
			}
			return a.respondError(output, command.ID, internalError("failed to start scan", map[string]any{"cause": err.Error()}))
		}

		if err := a.emitEvent(output, contracts.EventTaskStarted, contracts.TaskEventDTO{Task: task}); err != nil {
			return err
		}
		if err := a.emitEvent(output, contracts.EventScanStarted, contracts.TaskEventDTO{Task: task}); err != nil {
			return err
		}

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

	const (
		minScanProgressInterval = 250 * time.Millisecond
		scanProgressFileStep    = 50
	)
	var lastProgressEmit time.Time

	summary, err := a.scanner.Scan(ctx, taskID, paths, scanner.Hooks{
		OnProgress: func(processed, total int, message string) {
			now := time.Now()
			shouldEmit := processed == 0 || processed == total ||
				now.Sub(lastProgressEmit) >= minScanProgressInterval ||
				(processed > 0 && processed%scanProgressFileStep == 0)
			if !shouldEmit {
				return
			}
			lastProgressEmit = now

			progress := 100
			if total > 0 {
				progress = int(float64(processed) / float64(total) * 100)
			}

			patch := map[string]any{
				"scanTotal":     total,
				"scanProcessed": processed,
				"scanImported":  importedCount,
				"scanUpdated":   updatedCount,
				"scanSkipped":   skippedCount,
			}
			task := a.tasks.ProgressWithMetadata(taskID, progress, message, patch)
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

			if a.OrganizeLibrary() {
				newPath, orgErr := library.OrganizeVideoFile(result.Path, result.Number)
				if orgErr != nil {
					skippedCount++
					result.Status = "skipped"
					if errors.Is(orgErr, library.ErrOrganizeConflict) {
						result.Reason = "organize_conflict: " + orgErr.Error()
					} else {
						result.Reason = "organize_failed: " + orgErr.Error()
					}
					if err := a.store.SaveScanItem(ctx, result); err != nil {
						return err
					}
					persistedResults = append(persistedResults, result)
					if emitErr := a.emitEvent(output, contracts.EventScanFileSkipped, result); emitErr != nil {
						a.logger.Error("failed to emit scan skip event", zap.Error(emitErr), zap.String("taskId", taskID))
					}
					return nil
				}
				result.Path = newPath
				result.FileName = filepath.Base(newPath)
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
				a.enqueueScrape(parentCtx, output, result)
			case "updated":
				updatedCount++
				if emitErr := a.emitEvent(output, contracts.EventScanFileUpdated, result); emitErr != nil {
					a.logger.Error("failed to emit scan update event", zap.Error(emitErr), zap.String("taskId", taskID))
				}
				a.enqueueScrape(parentCtx, output, result)
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
		if errors.Is(err, context.Canceled) {
			code = contracts.ErrorCodeScanCancelled
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

// enqueueScrape runs runScrape in a goroutine bounded by scrapeSem (config scraper.maxConcurrent).
func (a *App) enqueueScrape(parentCtx context.Context, output io.Writer, result contracts.ScanFileResultDTO) {
	go func(r contracts.ScanFileResultDTO) {
		a.scrapeSem <- struct{}{}
		defer func() { <-a.scrapeSem }()
		a.runScrape(parentCtx, output, r)
	}(result)
}

// beginMovieScrapeTask creates and persists the scrape.movie task and emits task started.
func (a *App) beginMovieScrapeTask(ctx context.Context, output io.Writer, result contracts.ScanFileResultDTO) contracts.TaskDTO {
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
	return task
}

// runMovieScrapeBody runs scraper, persists metadata, NFO/assets hooks; ctx must be the scrape timeout context.
func (a *App) runMovieScrapeBody(ctx context.Context, parentCtx context.Context, output io.Writer, task contracts.TaskDTO, result contracts.ScanFileResultDTO) {
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

	if a.OrganizeLibrary() {
		nfoDir := filepath.Dir(result.Path)
		if err := library.WriteMovieNFO(nfoDir, metadata); err != nil {
			a.logger.Warn("failed to write movie.nfo", zap.Error(err), zap.String("taskId", task.TaskID), zap.String("dir", nfoDir))
		}
	}

	assetDest := filepath.Join(a.cfg.CacheDir, metadata.MovieID)
	if a.OrganizeLibrary() {
		assetDest = filepath.Dir(result.Path)
	}
	go a.runAssetDownload(parentCtx, output, metadata, assetDest)
}

func (a *App) runScrape(parentCtx context.Context, output io.Writer, result contracts.ScanFileResultDTO) {
	ctx, cancel := context.WithTimeout(parentCtx, time.Duration(a.cfg.Scraper.TaskTimeoutSeconds)*time.Second)
	defer cancel()
	task := a.beginMovieScrapeTask(ctx, output, result)
	a.runMovieScrapeBody(ctx, parentCtx, output, task, result)
}

func scanFileResultFromMovieDetail(detail contracts.MovieDetailDTO) (contracts.ScanFileResultDTO, error) {
	if strings.TrimSpace(detail.Code) == "" {
		return contracts.ScanFileResultDTO{}, contracts.ErrScrapeMovieNoCode
	}
	if strings.TrimSpace(detail.Location) == "" {
		return contracts.ScanFileResultDTO{}, contracts.ErrScrapeMovieNoLocation
	}
	return contracts.ScanFileResultDTO{
		Path:     detail.Location,
		FileName: filepath.Base(detail.Location),
		Number:   detail.Code,
		MovieID:  detail.ID,
		Status:   "updated",
	}, nil
}

// startAsyncMovieMetadataScrape enqueues runMovieScrapeBody in a goroutine (same pipeline as scan-triggered scrape).
func (a *App) startAsyncMovieMetadataScrape(ctx context.Context, detail contracts.MovieDetailDTO) (contracts.TaskDTO, error) {
	result, err := scanFileResultFromMovieDetail(detail)
	if err != nil {
		return contracts.TaskDTO{}, err
	}
	task := a.beginMovieScrapeTask(ctx, io.Discard, result)
	go func() {
		scrapeCtx, cancel := context.WithTimeout(a.appCtx, time.Duration(a.cfg.Scraper.TaskTimeoutSeconds)*time.Second)
		defer cancel()
		a.runMovieScrapeBody(scrapeCtx, a.appCtx, io.Discard, task, result)
	}()
	return task, nil
}

// StartMovieMetadataRefresh enqueues a single-movie rescrape (same pipeline as scan-triggered scrape).
// It returns the scrape task immediately; work continues in a background goroutine.
func (a *App) StartMovieMetadataRefresh(ctx context.Context, movieID string) (contracts.TaskDTO, error) {
	detail, err := a.store.GetMovieDetail(ctx, movieID)
	if errors.Is(err, sql.ErrNoRows) {
		return contracts.TaskDTO{}, contracts.ErrScrapeMovieNotFound
	}
	if err != nil {
		return contracts.TaskDTO{}, err
	}
	return a.startAsyncMovieMetadataScrape(ctx, detail)
}

// matchConfiguredLibraryPath returns the canonical configured path if req matches a library path (case-fold on Windows).
func matchConfiguredLibraryPath(req string, configured []contracts.LibraryPathDTO) (canonical string, ok bool) {
	reqClean := filepath.Clean(strings.TrimSpace(req))
	if reqClean == "." || reqClean == "" {
		return "", false
	}
	for _, c := range configured {
		cp := filepath.Clean(strings.TrimSpace(c.Path))
		if cp == "" || cp == "." {
			continue
		}
		if strings.EqualFold(reqClean, cp) {
			return cp, true
		}
	}
	return "", false
}

// StartMetadataRefreshForLibraryPaths queues metadata rescrape for all indexed movies under the given paths.
// Each requested path must match a configured library root; non-matching paths are listed in invalidPaths.
func (a *App) StartMetadataRefreshForLibraryPaths(ctx context.Context, requestedPaths []string) (contracts.MetadataRefreshQueuedDTO, error) {
	out := contracts.MetadataRefreshQueuedDTO{}
	if len(requestedPaths) == 0 {
		return out, nil
	}

	configured, err := a.store.ListLibraryPaths(ctx)
	if err != nil {
		return out, err
	}

	resolvedRoots := make([]string, 0, len(requestedPaths))
	seenRoot := make(map[string]struct{})
	for _, raw := range requestedPaths {
		p := strings.TrimSpace(raw)
		if p == "" {
			continue
		}
		canon, ok := matchConfiguredLibraryPath(p, configured)
		if !ok {
			out.InvalidPaths = append(out.InvalidPaths, raw)
			continue
		}
		key := strings.ToLower(canon)
		if _, dup := seenRoot[key]; dup {
			continue
		}
		seenRoot[key] = struct{}{}
		resolvedRoots = append(resolvedRoots, canon)
	}

	if len(resolvedRoots) == 0 {
		return out, nil
	}

	ids, err := a.store.ListMovieIDsUnderLibraryRoots(ctx, resolvedRoots)
	if err != nil {
		return out, err
	}

	queued, skipped := 0, 0
	for _, id := range ids {
		detail, derr := a.store.GetMovieDetail(ctx, id)
		if derr != nil {
			skipped++
			continue
		}
		_, serr := a.startAsyncMovieMetadataScrape(ctx, detail)
		if serr != nil {
			skipped++
			continue
		}
		queued++
	}
	out.Queued = queued
	out.Skipped = skipped
	return out, nil
}

func (a *App) runAssetDownload(parentCtx context.Context, output io.Writer, metadata scraper.Metadata, destDir string) {
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

	downloaded, err := a.assets.DownloadAllTo(ctx, metadata, destDir)
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

func (a *App) resolveScanPaths(ctx context.Context, paths []string) ([]string, error) {
	if len(paths) > 0 {
		return paths, nil
	}
	dbPaths, err := a.store.ListLibraryPathStrings(ctx)
	if err != nil {
		return nil, err
	}
	if len(dbPaths) > 0 {
		return dbPaths, nil
	}
	return append([]string(nil), a.cfg.LibraryPaths...), nil
}

// StartAutoScanLoop runs periodic library scans until ctx is cancelled. No-op if AutoScanIntervalSeconds <= 0.
func (a *App) StartAutoScanLoop(ctx context.Context) {
	secs := a.cfg.AutoScanIntervalSeconds
	if secs <= 0 {
		return
	}
	interval := time.Duration(secs) * time.Second
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, err := a.StartScan(context.Background(), nil)
				if err != nil {
					if errors.Is(err, contracts.ErrScanAlreadyRunning) {
						a.logger.Debug("auto scan skipped: scan already in progress")
						continue
					}
					a.logger.Warn("auto scan failed", zap.Error(err))
				}
			}
		}
	}()
}

func (a *App) StartScan(ctx context.Context, paths []string) (contracts.TaskDTO, error) {
	return a.startLibraryScan(ctx, io.Discard, paths)
}

func (a *App) startLibraryScan(ctx context.Context, output io.Writer, paths []string) (contracts.TaskDTO, error) {
	if !a.scanning.CompareAndSwap(false, true) {
		return contracts.TaskDTO{}, contracts.ErrScanAlreadyRunning
	}

	resolved, err := a.resolveScanPaths(ctx, paths)
	if err != nil {
		a.scanning.Store(false)
		return contracts.TaskDTO{}, err
	}
	paths = resolved

	task := a.tasks.Create("scan.library", map[string]any{"paths": paths})
	task = a.tasks.Start(task.TaskID, fmt.Sprintf("Scanning %d library path(s)", len(paths)))
	if err := a.store.SaveTask(a.appCtx, task); err != nil {
		a.scanning.Store(false)
		return contracts.TaskDTO{}, err
	}

	go func() {
		defer a.scanning.Store(false)
		a.runScan(a.appCtx, output, task.TaskID, paths)
	}()

	return task, nil
}

func (a *App) HTTPHandler() http.Handler {
	return server.NewHandler(server.Deps{
		Cfg:                    a.cfg,
		Logger:                 a.logger,
		Store:                  a.store,
		Library:                a.library,
		Tasks:                  a.tasks,
		ScanStarter:            a,
		OrganizeLibraryCtl:     a,
		MovieMetadataRefresher: a,
	}).Routes()
}

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}

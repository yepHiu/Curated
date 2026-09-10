// Package server provides HTTP handlers and routing for the Curated REST API.
package server

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"curated-backend/internal/browserheaders"
	"curated-backend/internal/clienttracker"
	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/diskutil"
	"curated-backend/internal/proxyenv"
	"curated-backend/internal/scraper/metatube"
	"curated-backend/internal/shellopen"
	"curated-backend/internal/storage"
	"curated-backend/internal/tasks"
	"curated-backend/internal/version"
)

const (
	importCopyBufferSize          = 4 * 1024 * 1024
	importTaskSnapshotMinBytes    = 64 * 1024 * 1024
	importTaskSnapshotMinInterval = time.Second
)

var (
	openDirectoryFn       = shellopen.OpenDirectory
	revealInFileManagerFn = shellopen.RevealInFileManager
)

// ScanStarter starts an async library scan task and returns its task descriptor.
type ScanStarter interface {
	StartScan(ctx context.Context, paths []string) (contracts.TaskDTO, error)
}

// ComicScanStarter starts an async comic library scan task and returns its task descriptor.
type ComicScanStarter interface {
	StartComicScan(ctx context.Context, paths []contracts.ComicLibraryPathDTO) (contracts.TaskDTO, error)
}

// PhotoScanStarter starts an async photo library scan task and returns its task descriptor.
type PhotoScanStarter interface {
	StartPhotoScan(ctx context.Context, paths []contracts.PhotoLibraryPathDTO) (contracts.TaskDTO, error)
}

// MovieMetadataRefresher starts an async single-movie metadata rescrape and returns the scrape task.
type MovieMetadataRefresher interface {
	StartMovieMetadataRefresh(ctx context.Context, movieID string) (contracts.TaskDTO, error)
	StartMetadataRefreshForLibraryPaths(ctx context.Context, paths []string) (contracts.MetadataRefreshQueuedDTO, error)
}

// ActorProfileRefresher starts an async actor profile scrape (Metatube) for an existing library actor name.
type ActorProfileRefresher interface {
	StartActorProfileScrape(ctx context.Context, actorName string) (contracts.TaskDTO, error)
}

// OrganizeLibraryController exposes whether the library auto-organize feature is enabled (persisted in library-config.cfg).
type OrganizeLibraryController interface {
	OrganizeLibrary() bool
	SetOrganizeLibrary(v bool) error
}

// AutoLibraryWatchController exposes whether fsnotify-driven library scans are allowed (library-config.cfg).
type AutoLibraryWatchController interface {
	AutoLibraryWatch() bool
	SetAutoLibraryWatch(v bool) error
}

// AutoActorProfileScrapeController exposes whether movie metadata scrapes may auto-enqueue missing actor profiles.
type AutoActorProfileScrapeController interface {
	AutoActorProfileScrape() bool
	SetAutoActorProfileScrape(v bool) error
}

// AutoDownloadUpdatesController exposes whether startup update checks may auto-download verified installers.
type AutoDownloadUpdatesController interface {
	AutoDownloadUpdates() bool
	SetAutoDownloadUpdates(v bool) error
}

// MetadataScrapeSettings exposes Metatube movie provider preference (empty = auto) and the list of valid provider names.
type MetadataScrapeSettings interface {
	MetadataMovieProvider() string
	SetMetadataMovieProvider(name string) error
	MetadataMovieProviderChain() []string
	SetMetadataMovieProviderChain(chain []string) error
	MetadataMovieScrapeMode() string
	SetMetadataMovieScrapeMode(mode string) error
	MetadataMovieStrategy() string
	SetMetadataMovieStrategy(strategy string) error
	ListMetadataMovieProviders() []string
}

// ProviderHealthChecker provides health check capabilities for metadata providers.
type ProviderHealthChecker interface {
	ListProviders() []string
	CheckProviderHealth(ctx context.Context, name string) (status string, latencyMs int64, err error)
}

// ProxyController exposes and updates HTTP proxy configuration (library-config.cfg).
type ProxyController interface {
	Proxy() config.ProxyConfig
	SetProxy(cfg config.ProxyConfig) error
}

// BackendLogSettingsController exposes and updates backend log paths/level (library-config.cfg); restart applies to Zap.
type BackendLogSettingsController interface {
	BackendLogSettings() contracts.BackendLogSettingsDTO
	SetBackendLogPatch(p contracts.PatchBackendLogSettings) error
}

// PlayerSettingsController exposes and updates player configuration (persisted in library-config.cfg).
type PlayerSettingsController interface {
	PlayerSettings() contracts.PlayerSettingsDTO
	SetPlayerSettingsPatch(p contracts.PatchPlayerSettingsDTO) error
}

// AISettingsController exposes and updates the experimental agent provider configuration (library-config.cfg).
type AISettingsController interface {
	AIProviderSettings() contracts.AIProviderSettingsDTO
	SetAIProviderSettingsPatch(p contracts.PatchAIProviderSettings) error
}

// AIChatProvider streams experimental agent chat completions and tests provider connectivity.
type AIChatProvider interface {
	StreamAIChat(ctx context.Context, req contracts.AIChatRequest, emit func(contracts.AIChatSSEEvent)) error
	TestAIProvider(ctx context.Context, override *contracts.AIProviderSettingsDTO) contracts.AIProviderTestResponse
	ListAIChatSessions(ctx context.Context) (contracts.AIChatSessionListDTO, error)
	CreateAIChatSession(ctx context.Context, title string) (contracts.AIChatSessionDTO, error)
	GetAIChatSession(ctx context.Context, id string, cursor ...string) (contracts.AIChatSessionDetailDTO, error)
	DeleteAIChatSession(ctx context.Context, id string) error
	RunAIAction(ctx context.Context, name string, req contracts.AIActionRequest) (contracts.AIActionPreviewDTO, error)
	ApplyAITool(ctx context.Context, req contracts.AIToolApplyRequest) (contracts.AIToolApplyDTO, error)
}

// LaunchAtLoginController exposes whether Windows login autostart is enabled and whether the current runtime supports it.
type LaunchAtLoginController interface {
	LaunchAtLogin() bool
	LaunchAtLoginSupported() bool
	SetLaunchAtLogin(v bool) error
}

// CuratedFrameExportFormatController exposes and updates the export format preference for curated frames.
type CuratedFrameExportFormatController interface {
	CuratedFrameExportFormat() string
	SetCuratedFrameExportFormat(v string) error
}

// CuratedFrameExportModeController exposes and updates the default export mode preference for curated frames.
type CuratedFrameExportModeController interface {
	CuratedFrameExportMode() string
	SetCuratedFrameExportMode(v string) error
}

// DefaultImportLibraryPathController exposes the library root used by top-bar movie import.
type DefaultImportLibraryPathController interface {
	DefaultImportLibraryPathID() string
	SetDefaultImportLibraryPathID(id string) error
}

// BackupDirectoryController exposes the remembered backup package destination directory.
type BackupDirectoryController interface {
	BackupDirectory() string
	SetBackupDirectory(directory string) error
}

// ComicSettingsController exposes and updates optional comic library settings.
type ComicSettingsController interface {
	ComicLibraryEnabled() bool
	SetComicLibraryEnabled(v bool) error
	AutoComicLibraryWatch() bool
	SetAutoComicLibraryWatch(v bool) error
	DefaultComicImportLibraryPathID() string
	SetDefaultComicImportLibraryPathID(id string) error
	ComicReaderSettings() contracts.ComicReaderSettingsDTO
	SetComicReaderSettings(v contracts.ComicReaderSettingsDTO) error
	ComicCacheSettings() contracts.ComicCacheSettingsDTO
	SetComicCacheSettings(v contracts.ComicCacheSettingsDTO) error
}

// PhotoSettingsController exposes and updates optional photo book library settings.
type PhotoSettingsController interface {
	PhotoLibraryEnabled() bool
	SetPhotoLibraryEnabled(v bool) error
	AutoPhotoLibraryWatch() bool
	SetAutoPhotoLibraryWatch(v bool) error
	DefaultPhotoImportLibraryPathID() string
	SetDefaultPhotoImportLibraryPathID(id string) error
	PhotoViewerSettings() contracts.PhotoViewerSettingsDTO
	SetPhotoViewerSettings(v contracts.PhotoViewerSettingsDTO) error
	PhotoCacheSettings() contracts.PhotoCacheSettingsDTO
	SetPhotoCacheSettings(v contracts.PhotoCacheSettingsDTO) error
}

// LibraryPathStorageStatusProvider checks whether configured library paths' backing storage is available.
type LibraryPathStorageStatusProvider interface {
	ListLibraryPathStorageStatus(ctx context.Context) (contracts.LibraryPathStorageStatusListDTO, error)
	CheckLibraryPathStorageStatus(ctx context.Context, libraryPathIDs []string) (contracts.LibraryPathStorageStatusListDTO, error)
	RebindLibraryPathStorage(ctx context.Context, libraryPathID string) (contracts.LibraryPathStorageStatusDTO, error)
}

// LibraryWatchReloader rebuilds fsnotify watches after library roots change.
type LibraryWatchReloader interface {
	ReloadLibraryWatches(ctx context.Context) error
}

// ComicLibraryWatchReloader rebuilds comic fsnotify watches after comic roots change.
type ComicLibraryWatchReloader interface {
	ReloadComicLibraryWatches(ctx context.Context) error
}

// PhotoLibraryWatchReloader rebuilds photo fsnotify watches after photo roots change.
type PhotoLibraryWatchReloader interface {
	ReloadPhotoLibraryWatches(ctx context.Context) error
}

// DevPerformanceProvider returns a development-only CPU and runtime performance summary.
type DevPerformanceProvider interface {
	GetDevPerformanceSummary(ctx context.Context) contracts.DevPerformanceSummaryDTO
}

// PlaybackResolver resolves movie playback descriptors and manages HLS playback sessions.
type PlaybackResolver interface {
	// clientVideoCodecs optionally carries browser-reported decodable mp4-family
	// video codecs (the `clientVideoCodecs` query parameter).
	ResolvePlayback(ctx context.Context, movieID string, clientVideoCodecs []string, startPositionSec *float64) (contracts.PlaybackDescriptorDTO, error)
	CreatePlaybackSession(ctx context.Context, movieID string, mode contracts.PlaybackMode, startPositionSec float64) (contracts.PlaybackDescriptorDTO, error)
	GetPlaybackSession(ctx context.Context, sessionID string) (contracts.PlaybackSessionStatusDTO, error)
	ListRecentPlaybackSessions(ctx context.Context, limit int) (contracts.PlaybackSessionListDTO, error)
	ResolvePlaybackSessionFile(sessionID string, name string) (string, error)
	DeletePlaybackSession(sessionID string) error
}

// NativePlaybackLauncher launches an external native player process for a movie.
type NativePlaybackLauncher interface {
	LaunchNativePlayback(ctx context.Context, movieID string, startPositionSec float64) (contracts.NativePlaybackLaunchDTO, error)
}

// HomepageRecommendationsProvider generates and persists daily UTC-day homepage recommendation snapshots.
type HomepageRecommendationsProvider interface {
	GetOrCreateHomepageDailyRecommendations(ctx context.Context, dateUTC string) (contracts.HomepageDailyRecommendationsDTO, error)
	RegenerateHomepageDailyRecommendations(ctx context.Context, dateUTC string, options ...contracts.HomepageDailyRecommendationsRefreshOptions) (contracts.HomepageDailyRecommendationsDTO, error)
}

type HomepageRecommendationFeedbackProvider interface {
	ListHomepageRecommendationFeedback(ctx context.Context) (contracts.RecommendationFeedbackListDTO, error)
	CreateHomepageRecommendationFeedback(ctx context.Context, body contracts.CreateRecommendationFeedbackBody) (contracts.RecommendationFeedbackDTO, error)
	DeleteHomepageRecommendationFeedback(ctx context.Context, id string) error
}

type ActorMergeProvider interface {
	PreviewActorMerge(ctx context.Context, req contracts.ActorMergePreviewRequest) (contracts.ActorMergePreviewDTO, error)
	ApplyActorMerge(ctx context.Context, req contracts.ApplyActorMergeRequest) (contracts.ActorMergeAuditDTO, error)
	ListActorMergeAudits(ctx context.Context, limit, offset int) (contracts.ActorMergeAuditListDTO, error)
}

type PersonalInsightsProvider interface {
	GetPersonalInsightsOverview(ctx context.Context, rangeValue string, timezone string) (contracts.PersonalInsightsOverviewDTO, error)
	GetPersonalInsightsBreakdown(ctx context.Context, rangeValue string, timezone string, dimension string, limit int) (contracts.PersonalInsightsBreakdownDTO, error)
}

// AppUpdateProvider checks and returns packaged-app update status from GitHub Releases.
type AppUpdateProvider interface {
	GetAppUpdateStatus(ctx context.Context) (contracts.AppUpdateStatusDTO, error)
	CheckAppUpdateNow(ctx context.Context) (contracts.AppUpdateStatusDTO, error)
	DownloadAppUpdateInstaller(ctx context.Context) (contracts.AppUpdateStatusDTO, error)
	InstallAppUpdate(ctx context.Context, req contracts.AppUpdateInstallRequest) (contracts.AppUpdateStatusDTO, error)
	ClearDownloadedAppUpdateInstaller(ctx context.Context) (contracts.AppUpdateStatusDTO, error)
}

type BackupProvider interface {
	CreateBackup(ctx context.Context, destinationPath string) (contracts.BackupManifestDTO, error)
	VerifyBackup(ctx context.Context, backupPath string) (contracts.BackupVerificationDTO, error)
	PreflightBackupRestore(ctx context.Context, backupPath string) (contracts.BackupRestorePreflightDTO, error)
}

// Handler holds all HTTP handler dependencies and implements the API route handlers.
type Handler struct {
	runtimeContext                 context.Context
	cfg                            config.Config
	logger                         *zap.Logger
	store                          *storage.SQLiteStore
	tasks                          *tasks.Manager
	scanStarter                    ScanStarter
	organizeLibraryCtl             OrganizeLibraryController
	autoLibraryWatchCtl            AutoLibraryWatchController
	autoActorProfileScrapeCtl      AutoActorProfileScrapeController
	autoDownloadUpdatesCtl         AutoDownloadUpdatesController
	launchAtLoginCtl               LaunchAtLoginController
	curatedFrameExportFormatCtl    CuratedFrameExportFormatController
	curatedFrameExportModeCtl      CuratedFrameExportModeController
	defaultImportLibraryPathCtl    DefaultImportLibraryPathController
	backupDirectoryCtl             BackupDirectoryController
	libraryPathStorageStatus       LibraryPathStorageStatusProvider
	metadataScrapeCtl              MetadataScrapeSettings
	providerHealthChecker          ProviderHealthChecker
	proxyCtl                       ProxyController
	backendLogCtl                  BackendLogSettingsController
	playerSettingsCtl              PlayerSettingsController
	aiSettingsCtl                  AISettingsController
	aiChatProvider                 AIChatProvider
	movieMetadataRefresher         MovieMetadataRefresher
	actorProfileRefresher          ActorProfileRefresher
	libraryWatchReloader           LibraryWatchReloader
	devPerformanceProvider         DevPerformanceProvider
	playbackResolver               PlaybackResolver
	nativePlaybackLauncher         NativePlaybackLauncher
	homepageRecommendations        HomepageRecommendationsProvider
	homepageRecommendationFeedback HomepageRecommendationFeedbackProvider
	actorMergeProvider             ActorMergeProvider
	personalInsightsProvider       PersonalInsightsProvider
	appUpdateProvider              AppUpdateProvider
	backupProvider                 BackupProvider
	importUploads                  *movieImportUploadSessionStore
	clientTracker                  *clienttracker.Tracker
	authAttempts                   *authAttemptLimiter
	movieClipArtifacts             *sync.Map
	movieClipInit                  sync.Once
	movieClipSlots                 chan struct{}
	movieClipWorkers               chan struct{}
	movieClipCancels               sync.Map
	comicScanStarter               ComicScanStarter
	photoScanStarter               PhotoScanStarter
	comicSettingsCtl               ComicSettingsController
	photoSettingsCtl               PhotoSettingsController
	comicLibraryWatchReloader      ComicLibraryWatchReloader
	photoLibraryWatchReloader      PhotoLibraryWatchReloader
}

// Deps bundles all dependencies needed to construct a Handler.
type Deps struct {
	RuntimeContext                   context.Context
	Cfg                              config.Config
	Logger                           *zap.Logger
	Store                            *storage.SQLiteStore
	Tasks                            *tasks.Manager
	ScanStarter                      ScanStarter
	ComicScanStarter                 ComicScanStarter
	PhotoScanStarter                 PhotoScanStarter
	OrganizeLibraryCtl               OrganizeLibraryController
	AutoLibraryWatchCtl              AutoLibraryWatchController
	AutoActorProfileScrapeCtl        AutoActorProfileScrapeController
	AutoDownloadUpdatesCtl           AutoDownloadUpdatesController
	LaunchAtLoginCtl                 LaunchAtLoginController
	CuratedFrameExportFormatCtl      CuratedFrameExportFormatController
	CuratedFrameExportModeCtl        CuratedFrameExportModeController
	DefaultImportLibraryPathCtl      DefaultImportLibraryPathController
	BackupDirectoryCtl               BackupDirectoryController
	ComicSettingsCtl                 ComicSettingsController
	PhotoSettingsCtl                 PhotoSettingsController
	LibraryPathStorageStatusProvider LibraryPathStorageStatusProvider
	MetadataScrapeCtl                MetadataScrapeSettings
	ProviderHealthChecker            ProviderHealthChecker
	ProxyCtl                         ProxyController
	BackendLogCtl                    BackendLogSettingsController
	PlayerSettingsCtl                PlayerSettingsController
	AISettingsCtl                    AISettingsController
	AIChatProvider                   AIChatProvider
	MovieMetadataRefresher           MovieMetadataRefresher
	ActorProfileRefresher            ActorProfileRefresher
	LibraryWatchReloader             LibraryWatchReloader
	ComicLibraryWatchReloader        ComicLibraryWatchReloader
	PhotoLibraryWatchReloader        PhotoLibraryWatchReloader
	DevPerformanceProvider           DevPerformanceProvider
	PlaybackResolver                 PlaybackResolver
	NativePlaybackLauncher           NativePlaybackLauncher
	HomepageRecommendations          HomepageRecommendationsProvider
	HomepageRecommendationFeedback   HomepageRecommendationFeedbackProvider
	ActorMergeProvider               ActorMergeProvider
	PersonalInsightsProvider         PersonalInsightsProvider
	AppUpdateProvider                AppUpdateProvider
	BackupProvider                   BackupProvider
	ClientTracker                    *clienttracker.Tracker
}

// NewHandler creates a Handler wired with all provided dependencies.
func NewHandler(deps Deps) *Handler {
	tracker := deps.ClientTracker
	if tracker == nil {
		tracker = clienttracker.New()
	}
	importUploads := newMovieImportUploadSessionStore(deps.Store, deps.Tasks, deps.Logger)
	if deps.Store != nil && deps.Tasks != nil {
		recoveryContext := deps.RuntimeContext
		if recoveryContext == nil {
			recoveryContext = context.Background()
		}
		if err := importUploads.recover(recoveryContext); err != nil {
			importUploads.initErr = err
			if deps.Logger != nil {
				deps.Logger.Error("recover movie import upload sessions failed", zap.Error(err))
			}
		}
		if deps.RuntimeContext != nil {
			importUploads.startJanitor(deps.RuntimeContext)
		}
	}
	if deps.Store != nil && deps.RuntimeContext != nil {
		if err := deps.Store.InterruptLibraryHealthRepairRuns(deps.RuntimeContext, time.Now()); err != nil && deps.Logger != nil {
			deps.Logger.Error("mark interrupted library health repairs failed", zap.Error(err))
		}
	}
	h := &Handler{
		comicScanStarter:               deps.ComicScanStarter,
		photoScanStarter:               deps.PhotoScanStarter,
		comicSettingsCtl:               deps.ComicSettingsCtl,
		photoSettingsCtl:               deps.PhotoSettingsCtl,
		comicLibraryWatchReloader:      deps.ComicLibraryWatchReloader,
		photoLibraryWatchReloader:      deps.PhotoLibraryWatchReloader,
		runtimeContext:                 deps.RuntimeContext,
		cfg:                            deps.Cfg,
		logger:                         deps.Logger,
		store:                          deps.Store,
		tasks:                          deps.Tasks,
		scanStarter:                    deps.ScanStarter,
		organizeLibraryCtl:             deps.OrganizeLibraryCtl,
		autoLibraryWatchCtl:            deps.AutoLibraryWatchCtl,
		autoActorProfileScrapeCtl:      deps.AutoActorProfileScrapeCtl,
		autoDownloadUpdatesCtl:         deps.AutoDownloadUpdatesCtl,
		launchAtLoginCtl:               deps.LaunchAtLoginCtl,
		curatedFrameExportFormatCtl:    deps.CuratedFrameExportFormatCtl,
		curatedFrameExportModeCtl:      deps.CuratedFrameExportModeCtl,
		defaultImportLibraryPathCtl:    deps.DefaultImportLibraryPathCtl,
		backupDirectoryCtl:             deps.BackupDirectoryCtl,
		libraryPathStorageStatus:       deps.LibraryPathStorageStatusProvider,
		metadataScrapeCtl:              deps.MetadataScrapeCtl,
		providerHealthChecker:          deps.ProviderHealthChecker,
		proxyCtl:                       deps.ProxyCtl,
		backendLogCtl:                  deps.BackendLogCtl,
		playerSettingsCtl:              deps.PlayerSettingsCtl,
		aiSettingsCtl:                  deps.AISettingsCtl,
		aiChatProvider:                 deps.AIChatProvider,
		movieMetadataRefresher:         deps.MovieMetadataRefresher,
		actorProfileRefresher:          deps.ActorProfileRefresher,
		libraryWatchReloader:           deps.LibraryWatchReloader,
		devPerformanceProvider:         deps.DevPerformanceProvider,
		playbackResolver:               deps.PlaybackResolver,
		nativePlaybackLauncher:         deps.NativePlaybackLauncher,
		homepageRecommendations:        deps.HomepageRecommendations,
		homepageRecommendationFeedback: deps.HomepageRecommendationFeedback,
		actorMergeProvider:             deps.ActorMergeProvider,
		personalInsightsProvider:       deps.PersonalInsightsProvider,
		appUpdateProvider:              deps.AppUpdateProvider,
		backupProvider:                 deps.BackupProvider,
		importUploads:                  importUploads,
		clientTracker:                  tracker,
		authAttempts:                   newAuthAttemptLimiter(),
		movieClipArtifacts:             &sync.Map{},
	}
	if deps.RuntimeContext != nil {
		h.startMovieClipArtifactJanitor(deps.RuntimeContext)
	}
	return h
}

// Routes builds the HTTP mux with all registered API routes, access logging, and CORS.
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", h.handleHealth)
	mux.HandleFunc("GET /api/auth/status", h.handleAuthStatus)
	mux.HandleFunc("POST /api/auth/setup-pin", h.handleSetupPIN)
	mux.HandleFunc("POST /api/auth/unlock", h.handleUnlockPIN)
	mux.HandleFunc("POST /api/auth/change-pin", h.handleChangePIN)
	mux.HandleFunc("POST /api/auth/lock", h.handleLockPIN)
	mux.HandleFunc("PATCH /api/auth/settings", h.handlePatchAuthSettings)
	mux.HandleFunc("GET /api/auth/sessions", h.handleListAuthSessions)
	mux.HandleFunc("DELETE /api/auth/sessions/{publicId}", h.handleRevokeAuthSession)
	mux.HandleFunc("POST /api/auth/sessions/revoke-others", h.handleRevokeOtherAuthSessions)
	mux.HandleFunc("GET /api/connected-clients", h.handleConnectedClients)
	mux.HandleFunc("GET /api/events", h.handleEvents)
	mux.HandleFunc("GET /api/dev/performance", h.handleDevPerformance)
	mux.HandleFunc("GET /api/app-update/status", h.handleGetAppUpdateStatus)
	mux.HandleFunc("POST /api/app-update/check", h.handleCheckAppUpdate)
	mux.HandleFunc("POST /api/app-update/download", h.handleDownloadAppUpdateInstaller)
	mux.HandleFunc("POST /api/app-update/install", h.handleInstallAppUpdate)
	mux.HandleFunc("DELETE /api/app-update/downloaded-installer", h.handleClearDownloadedAppUpdateInstaller)
	mux.HandleFunc("POST /api/maintenance/backups", h.handleCreateBackup)
	mux.HandleFunc("POST /api/maintenance/backups/verify", h.handleVerifyBackup)
	mux.HandleFunc("POST /api/maintenance/backups/preflight", h.handlePreflightBackupRestore)
	mux.HandleFunc("POST /api/library/health/scan", h.handleScanLibraryHealth)
	mux.HandleFunc("POST /api/library/health/repairs", h.handleStartLibraryHealthRepair)
	mux.HandleFunc("GET /api/library/health/repairs/{repairId}", h.handleGetLibraryHealthRepair)
	mux.HandleFunc("POST /api/library/health/actions", h.handleStartLibraryHealthAction)
	mux.HandleFunc("GET /api/homepage/recommendations", h.handleGetHomepageRecommendations)
	mux.HandleFunc("POST /api/homepage/recommendations/refresh", h.handleRefreshHomepageRecommendations)
	mux.HandleFunc("GET /api/homepage/recommendations/feedback", h.handleListHomepageRecommendationFeedback)
	mux.HandleFunc("POST /api/homepage/recommendations/feedback", h.handleCreateHomepageRecommendationFeedback)
	mux.HandleFunc("DELETE /api/homepage/recommendations/feedback/{feedbackId}", h.handleDeleteHomepageRecommendationFeedback)
	mux.HandleFunc("GET /api/library/played-movies", h.handleListPlayedMovies)
	mux.HandleFunc("POST /api/library/played-movies/{movieId}", h.handleRecordPlayedMovie)
	mux.HandleFunc("GET /api/library/saved-views", h.handleListSavedViews)
	mux.HandleFunc("POST /api/library/saved-views", h.handleCreateSavedView)
	mux.HandleFunc("PUT /api/library/saved-views/order", h.handleReorderSavedViews)
	mux.HandleFunc("PATCH /api/library/saved-views/{savedViewId}", h.handlePatchSavedView)
	mux.HandleFunc("DELETE /api/library/saved-views/{savedViewId}", h.handleDeleteSavedView)
	mux.HandleFunc("GET /api/library/movies", h.handleListMovies)
	mux.HandleFunc("GET /api/library/actors", h.handleListActors)
	mux.HandleFunc("GET /api/library/actors/profile", h.handleGetActorProfile)
	mux.HandleFunc("GET /api/library/actors/{name}/asset/avatar", h.handleGetActorAvatarAsset)
	mux.HandleFunc("POST /api/library/actors/scrape", h.handleScrapeActorProfile)
	mux.HandleFunc("PATCH /api/library/actors/tags", h.handlePatchActorUserTags)
	mux.HandleFunc("PATCH /api/library/actors/external-links", h.handlePatchActorExternalLinks)
	mux.HandleFunc("POST /api/library/actors/merge-preview", h.handlePreviewActorMerge)
	mux.HandleFunc("POST /api/library/actors/merge", h.handleApplyActorMerge)
	mux.HandleFunc("GET /api/library/actors/merge-audits", h.handleListActorMergeAudits)
	mux.HandleFunc("GET /api/insights/overview", h.handleGetPersonalInsightsOverview)
	mux.HandleFunc("GET /api/insights/breakdown", h.handleGetPersonalInsightsBreakdown)
	mux.HandleFunc("GET /api/library/movies/{movieId}/asset/preview/{index}", h.handleGetMoviePreviewAsset)
	mux.HandleFunc("GET /api/library/movies/{movieId}/asset/{kind}", h.handleGetMovieAsset)
	mux.HandleFunc("GET /api/library/movies/{movieId}/playback", h.handleGetMoviePlayback)
	mux.HandleFunc("POST /api/library/movies/{movieId}/playback-session", h.handleCreatePlaybackSession)
	mux.HandleFunc("POST /api/library/movies/{movieId}/native-play", h.handleLaunchNativePlayback)
	mux.HandleFunc("GET /api/library/movies/{movieId}/stream", h.handleStreamMovie)
	mux.HandleFunc("GET /api/playback/sessions/recent", h.handleGetRecentPlaybackSessions)
	mux.HandleFunc("GET /api/playback/sessions/{sessionId}", h.handleGetPlaybackSession)
	mux.HandleFunc("GET /api/playback/sessions/{sessionId}/hls/{file}", h.handleGetPlaybackSessionFile)
	mux.HandleFunc("DELETE /api/playback/sessions/{sessionId}", h.handleDeletePlaybackSession)
	mux.HandleFunc("POST /api/library/movies/{movieId}/reveal", h.handleRevealMovieInFileManager)
	mux.HandleFunc("GET /api/library/movies/{movieId}/comment", h.handleGetMovieComment)
	mux.HandleFunc("PUT /api/library/movies/{movieId}/comment", h.handlePutMovieComment)
	mux.HandleFunc("GET /api/library/movies/{movieId}", h.handleGetMovie)
	mux.HandleFunc("PATCH /api/library/movies/{movieId}", h.handlePatchMovie)
	mux.HandleFunc("POST /api/library/movies/{movieId}/restore", h.handleRestoreMovie)
	mux.HandleFunc("POST /api/library/movies/{movieId}/scrape", h.handleRefreshMovieMetadata)
	mux.HandleFunc("POST /api/library/movies/{movieId}/clips", h.handleCreateMovieClip)
	mux.HandleFunc("POST /api/library/movies/{movieId}/frame", h.handleExtractMovieFrame)
	mux.HandleFunc("DELETE /api/tasks/{taskId}/clip", h.handleCancelMovieClip)
	mux.HandleFunc("POST /api/library/metadata-scrape", h.handleMetadataScrapeByPaths)
	mux.HandleFunc("DELETE /api/library/movies/{movieId}", h.handleDeleteMovie)
	mux.HandleFunc("GET /api/settings", h.handleGetSettings)
	mux.HandleFunc("PATCH /api/settings", h.handlePatchSettings)
	mux.HandleFunc("POST /api/import/movies", h.handleImportMovies)
	mux.HandleFunc("POST /api/import/movies/code-check", h.handleCheckImportMovieCodes)
	mux.HandleFunc("POST /api/import/movies/uploads", h.handleCreateMovieImportUpload)
	mux.HandleFunc("GET /api/import/movies/uploads/{uploadId}", h.handleGetMovieImportUpload)
	mux.HandleFunc("DELETE /api/import/movies/uploads/{uploadId}", h.handleAbortMovieImportUpload)
	mux.HandleFunc("PUT /api/import/movies/uploads/{uploadId}/files/{fileId}/chunks/{chunkIndex}", h.handlePutMovieImportUploadChunk)
	mux.HandleFunc("POST /api/import/movies/uploads/{uploadId}/commit", h.handleCommitMovieImportUpload)
	mux.HandleFunc("POST /api/import/comics", h.handleImportComics)
	mux.HandleFunc("POST /api/library/paths", h.handleAddLibraryPath)
	mux.HandleFunc("GET /api/library/paths/storage-status", h.handleGetLibraryPathStorageStatus)
	mux.HandleFunc("POST /api/library/paths/storage-status/check", h.handleCheckLibraryPathStorageStatus)
	mux.HandleFunc("POST /api/library/paths/{id}/reveal", h.handleRevealLibraryPathInFileManager)
	mux.HandleFunc("POST /api/library/paths/{id}/storage-binding/rebind", h.handleRebindLibraryPathStorage)
	mux.HandleFunc("PATCH /api/library/paths/{id}", h.handlePatchLibraryPath)
	mux.HandleFunc("DELETE /api/library/paths/{id}", h.handleDeleteLibraryPath)
	mux.HandleFunc("POST /api/scans", h.handleStartScan)
	mux.HandleFunc("GET /api/library/comics/paths", h.handleListComicLibraryPaths)
	mux.HandleFunc("POST /api/library/comics/paths", h.handleAddComicLibraryPath)
	mux.HandleFunc("PATCH /api/library/comics/paths/{id}", h.handlePatchComicLibraryPath)
	mux.HandleFunc("DELETE /api/library/comics/paths/{id}", h.handleDeleteComicLibraryPath)
	mux.HandleFunc("GET /api/library/photos/paths", h.handleListPhotoLibraryPaths)
	mux.HandleFunc("POST /api/library/photos/paths", h.handleAddPhotoLibraryPath)
	mux.HandleFunc("PATCH /api/library/photos/paths/{id}", h.handlePatchPhotoLibraryPath)
	mux.HandleFunc("DELETE /api/library/photos/paths/{id}", h.handleDeletePhotoLibraryPath)
	mux.HandleFunc("POST /api/library/photos/scans", h.handleStartPhotoScan)
	mux.HandleFunc("GET /api/library/photos", h.handleListPhotos)
	mux.HandleFunc("GET /api/library/photos/{photoId}", h.handleGetPhoto)
	mux.HandleFunc("GET /api/library/photos/books/{photoId}/pages", h.handleListPhotoPages)
	mux.HandleFunc("GET /api/library/photos/books/{photoId}/pages/{pageIndex}/image", h.handleGetPhotoPageImage)
	mux.HandleFunc("GET /api/library/photos/books/{photoId}/pages/{pageIndex}/thumbnail", h.handleGetPhotoPageThumbnail)
	mux.HandleFunc("POST /api/library/comics/scans", h.handleStartComicScan)
	mux.HandleFunc("GET /api/library/comics/cache/status", h.handleGetComicCacheStatus)
	mux.HandleFunc("POST /api/library/comics/cache/cleanup", h.handleCleanupComicCache)
	mux.HandleFunc("GET /api/library/comics", h.handleListComics)
	mux.HandleFunc("GET /api/library/comics/{comicId}", h.handleGetComic)
	mux.HandleFunc("PATCH /api/library/comics/{comicId}", h.handlePatchComic)
	mux.HandleFunc("DELETE /api/library/comics/{comicId}", h.handleDeleteComic)
	mux.HandleFunc("POST /api/library/comics/books/{comicId}/reveal", h.handleRevealComicSource)
	mux.HandleFunc("GET /api/library/comics/books/{comicId}/pages", h.handleListComicPages)
	mux.HandleFunc("GET /api/library/comics/books/{comicId}/pages/{pageIndex}/image", h.handleGetComicPageImage)
	mux.HandleFunc("GET /api/library/comics/books/{comicId}/pages/{pageIndex}/thumbnail", h.handleGetComicPageThumbnail)
	mux.HandleFunc("GET /api/library/comics/books/{comicId}/progress", h.handleGetComicProgress)
	mux.HandleFunc("PUT /api/library/comics/books/{comicId}/progress", h.handlePutComicProgress)
	mux.HandleFunc("DELETE /api/library/comics/books/{comicId}/progress", h.handleDeleteComicProgress)
	mux.HandleFunc("GET /api/library/comics/books/{comicId}/preferences", h.handleGetComicPreferences)
	mux.HandleFunc("PUT /api/library/comics/books/{comicId}/preferences", h.handlePutComicPreferences)
	mux.HandleFunc("GET /api/tasks/recent", h.handleGetRecentTasks)
	mux.HandleFunc("GET /api/tasks/{taskId}/artifact", h.handleGetMovieClipArtifact)
	mux.HandleFunc("GET /api/tasks/{taskId}", h.handleGetTaskStatus)

	mux.HandleFunc("GET /api/playback/progress", h.handleListPlaybackProgress)
	mux.HandleFunc("PUT /api/playback/progress/{movieId}", h.handlePutPlaybackProgress)
	mux.HandleFunc("DELETE /api/playback/progress/{movieId}", h.handleDeletePlaybackProgress)
	mux.HandleFunc("GET /api/playback/watch-time/daily", h.handleListPlaybackWatchTimeDaily)
	mux.HandleFunc("POST /api/playback/watch-time/daily", h.handlePostPlaybackWatchTimeDaily)

	mux.HandleFunc("GET /api/curated-frames", h.handleListCuratedFrames)
	mux.HandleFunc("GET /api/curated-frames/stats", h.handleGetCuratedFrameStats)
	mux.HandleFunc("GET /api/curated-frames/tags", h.handleListCuratedFrameTags)
	mux.HandleFunc("GET /api/curated-frames/actors", h.handleListCuratedFrameActors)
	mux.HandleFunc("POST /api/curated-frames", h.handlePostCuratedFrame)
	mux.HandleFunc("GET /api/curated-frames/{id}/thumbnail", h.handleGetCuratedFrameThumbnail)
	mux.HandleFunc("GET /api/curated-frames/{id}/motion", h.handleGetCuratedFrameMotion)
	mux.HandleFunc("GET /api/curated-frames/{id}/image", h.handleGetCuratedFrameImage)
	mux.HandleFunc("PATCH /api/curated-frames/{id}/tags", h.handlePatchCuratedFrameTags)
	mux.HandleFunc("DELETE /api/curated-frames/{id}", h.handleDeleteCuratedFrame)
	mux.HandleFunc("POST /api/curated-frames/export", h.handlePostCuratedFramesExport)

	mux.HandleFunc("POST /api/providers/ping", h.handlePingProvider)
	mux.HandleFunc("POST /api/providers/ping-all", h.handlePingAllProviders)

	mux.HandleFunc("POST /api/proxy/ping-javbus", h.handleProxyPingJavbus)
	mux.HandleFunc("POST /api/proxy/ping-google", h.handleProxyPingGoogle)

	mux.HandleFunc("POST /api/ai/provider/test", h.handleAIProviderTest)
	mux.HandleFunc("POST /api/ai/chat", h.handleAIChat)
	mux.HandleFunc("GET /api/ai/sessions", h.handleListAIChatSessions)
	mux.HandleFunc("POST /api/ai/sessions", h.handleCreateAIChatSession)
	mux.HandleFunc("GET /api/ai/sessions/{sessionId}", h.handleGetAIChatSession)
	mux.HandleFunc("DELETE /api/ai/sessions/{sessionId}", h.handleDeleteAIChatSession)
	mux.HandleFunc("POST /api/ai/actions/{name}", h.handleAIAction)
	mux.HandleFunc("POST /api/ai/confirm", h.handleAIConfirm)
	mux.HandleFunc("GET /api/ai/settings", h.handleAIGovernance)
	mux.HandleFunc("PATCH /api/ai/settings", h.handleAIGovernance)
	mux.HandleFunc("GET /api/ai/usage", h.handleAIReport)
	mux.HandleFunc("GET /api/ai/audit", h.handleAIAudit)
	mux.HandleFunc("POST /api/ai/cleanup", h.handleAICleanup)

	return WithAccessLog(h.logger, withClientTracking(h.withRequestSecurity(h.withAuthLock(mux)), h.clientTracker))
}

func (h *Handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, contracts.HealthDTO{
		Name:             version.BackendName(),
		Version:          version.Stamp(),
		Channel:          version.Channel,
		InstallerVersion: version.PackageVersion(),
		Transport:        "http",
		DatabasePath:     h.cfg.DatabasePath,
	})
}

func (h *Handler) handleDevPerformance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.devPerformanceProvider == nil {
		writeJSON(w, http.StatusOK, contracts.DevPerformanceSummaryDTO{Supported: false})
		return
	}
	writeJSON(w, http.StatusOK, h.devPerformanceProvider.GetDevPerformanceSummary(r.Context()))
}

func (h *Handler) handleListMovies(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	limit, _ := strconv.Atoi(query.Get("limit"))
	offset, _ := strconv.Atoi(query.Get("offset"))
	if limit <= 0 {
		limit = 50
	}

	mode := strings.ToLower(strings.TrimSpace(query.Get("mode")))
	if !oneOf(mode, "", "library", "favorites", "recent", "tags", "trash") {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid mode")
		return
	}

	playState := strings.ToLower(strings.TrimSpace(query.Get("playState")))
	if playState != "" && !oneOf(playState, "all", "unwatched", "in-progress", "completed") {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid playState")
		return
	}
	var userRating *float64
	if raw := strings.TrimSpace(query.Get("userRating")); raw != "" {
		value, err := strconv.ParseFloat(raw, 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 5 {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "userRating must be between 0 and 5")
			return
		}
		userRating = &value
	}
	addedAfter := strings.TrimSpace(query.Get("addedAfter"))
	if addedAfter != "" {
		parsed, err := time.Parse(time.RFC3339Nano, addedAfter)
		if err != nil {
			parsed, err = time.Parse("2006-01-02", addedAfter)
		}
		if err != nil {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "addedAfter must be RFC3339 or YYYY-MM-DD")
			return
		}
		addedAfter = parsed.UTC().Format(time.RFC3339Nano)
	}
	unratedRaw := strings.ToLower(strings.TrimSpace(query.Get("unrated")))
	unrated := unratedRaw == "1" || unratedRaw == "true" || unratedRaw == "yes"
	if unratedRaw != "" && !unrated && unratedRaw != "0" && unratedRaw != "false" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid unrated")
		return
	}
	year := normalizeSavedViewYear(query.Get("year"))
	if raw := strings.TrimSpace(query.Get("year")); raw != "" && year == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "year must be unknown or a year from 1800 to 3000")
		return
	}
	runtimeFilter := strings.ToLower(strings.TrimSpace(query.Get("runtime")))
	if runtimeFilter != "" && !oneOf(runtimeFilter, "short", "standard", "long") {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid runtime")
		return
	}
	catalog := strings.ToLower(strings.TrimSpace(query.Get("catalog")))
	if catalog != "" && !oneOf(catalog, "unscraped", "no-cover") {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid catalog")
		return
	}

	rawTags := make([]string, 0, 8)
	for _, value := range query["tag"] {
		rawTags = append(rawTags, value)
	}
	rawActors := make([]string, 0, 8)
	for _, value := range query["actor"] {
		rawActors = append(rawActors, value)
	}
	rawStudios := make([]string, 0, 8)
	for _, value := range query["studio"] {
		rawStudios = append(rawStudios, value)
	}
	request := contracts.ListMoviesRequest{
		Mode:       mode,
		Query:      query.Get("q"),
		Tags:       storage.ParseMovieTagFilters(rawTags...),
		Actor:      strings.Join(storage.ParseMovieTagFilters(rawActors...), ","),
		Studio:     strings.Join(storage.ParseMovieTagFilters(rawStudios...), ","),
		PlayState:  playState,
		UserRating: userRating,
		Unrated:    unrated,
		Resolution: normalizeSavedViewResolution(query.Get("resolution")),
		AddedAfter: addedAfter,
		Year:       year,
		Runtime:    runtimeFilter,
		Catalog:    catalog,
		Limit:      limit,
		Offset:     offset,
	}

	result, err := h.store.ListMovies(r.Context(), request)
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("list movies failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to list movies")
		return
	}
	h.enrichMovieListItemsLocalPosters(r.Context(), result.Items)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) handleGetActorProfile(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "name is required")
		return
	}
	profile, err := h.store.GetActorProfile(r.Context(), name)
	if err != nil {
		if errors.Is(err, contracts.ErrActorNotFound) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "actor not found")
			return
		}
		if h.logger != nil {
			h.logger.Warn("get actor profile failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load actor profile")
		return
	}
	h.enrichActorProfileLocalAvatar(r.Context(), &profile)
	writeJSON(w, http.StatusOK, profile)
}

func (h *Handler) handleScrapeActorProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.actorProfileRefresher == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "actor profile scrape not configured")
		return
	}
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "name is required")
		return
	}
	task, err := h.actorProfileRefresher.StartActorProfileScrape(r.Context(), name)
	if err != nil {
		switch {
		case errors.Is(err, contracts.ErrActorNotFound):
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "actor not found")
		case strings.Contains(err.Error(), "actor name is required"):
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, err.Error())
		default:
			if h.logger != nil {
				h.logger.Warn("start actor profile scrape failed", zap.Error(err))
			}
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to start actor profile scrape")
		}
		return
	}
	writeJSON(w, http.StatusAccepted, task)
}

func (h *Handler) handleListActors(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	if limit <= 0 {
		limit = 50
	}
	req := contracts.ListActorsRequest{
		Q:        strings.TrimSpace(q.Get("q")),
		ActorTag: strings.TrimSpace(q.Get("actorTag")),
		Sort:     strings.TrimSpace(q.Get("sort")),
		Limit:    limit,
		Offset:   offset,
	}
	result, err := h.store.ListActors(r.Context(), req)
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("list actors failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to list actors")
		return
	}
	h.enrichActorListLocalAvatars(r.Context(), result.Actors)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) handlePatchActorUserTags(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "name is required")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 256<<10))
	if r.Body != nil {
		_ = r.Body.Close()
	}
	if err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "failed to read body")
		return
	}
	var in contracts.PatchActorUserTagsBody
	if err := json.Unmarshal(body, &in); err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid json body")
		return
	}
	err = h.store.ReplaceActorUserTagsByName(r.Context(), name, in.UserTags)
	if err != nil {
		if errors.Is(err, contracts.ErrActorNotFound) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "actor not found")
			return
		}
		if errors.Is(err, storage.ErrInvalidUserTags) {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, err.Error())
			return
		}
		if h.logger != nil {
			h.logger.Warn("patch actor user tags failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to update actor tags")
		return
	}
	item, err := h.store.ActorListItemByName(r.Context(), name)
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("load actor after tag patch failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load actor")
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) handlePatchActorExternalLinks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "name is required")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 256<<10))
	if r.Body != nil {
		_ = r.Body.Close()
	}
	if err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "failed to read body")
		return
	}
	var in contracts.PatchActorExternalLinksBody
	if err := json.Unmarshal(body, &in); err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid json body")
		return
	}
	err = h.store.ReplaceActorExternalLinksByName(r.Context(), name, in.ExternalLinks)
	if err != nil {
		if errors.Is(err, contracts.ErrActorNotFound) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "actor not found")
			return
		}
		if errors.Is(err, storage.ErrInvalidActorExternalLinks) {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, err.Error())
			return
		}
		if h.logger != nil {
			h.logger.Warn("patch actor external links failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to update actor external links")
		return
	}
	profile, err := h.store.GetActorProfile(r.Context(), name)
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("load actor after external links patch failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load actor profile")
		return
	}
	h.enrichActorProfileLocalAvatar(r.Context(), &profile)
	writeJSON(w, http.StatusOK, profile)
}

func (h *Handler) handleGetMovie(w http.ResponseWriter, r *http.Request) {
	movieID := r.PathValue("movieId")
	if movieID == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "movieId is required")
		return
	}

	movie, err := h.store.GetMovieDetail(r.Context(), movieID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie not found")
			return
		}
		if h.logger != nil {
			h.logger.Warn("get movie detail failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load movie")
		return
	}
	h.enrichMovieDetailLocalPosters(r.Context(), &movie)
	writeJSON(w, http.StatusOK, movie)
}

func (h *Handler) handleStreamMovie(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	movieID := strings.TrimSpace(r.PathValue("movieId"))
	if movieID == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "movieId is required")
		return
	}

	f, dispName, err := h.store.OpenMovieVideoFile(r.Context(), movieID)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrMovieVideoNotFound):
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie or video file not found")
		case errors.Is(err, storage.ErrMovieVideoNoLocation):
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie has no video file")
		case errors.Is(err, storage.ErrMovieVideoForbidden):
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie or video file not found")
		case errors.Is(err, storage.ErrMovieVideoNotFile):
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie or video file not found")
		default:
			if h.logger != nil {
				h.logger.Warn("open movie video stream failed", zap.Error(err))
			}
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to open video file")
		}
		return
	}
	defer func() { _ = f.Close() }()

	st, err := f.Stat()
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("stat movie video failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to read video file")
		return
	}

	// A size+modtime ETag lets http.ServeContent answer If-None-Match and
	// If-Range revalidation for byte-range clients without transferring data.
	w.Header().Set("ETag", fmt.Sprintf(`"%x-%x"`, st.Size(), st.ModTime().UnixNano()))
	http.ServeContent(w, r, dispName, st.ModTime(), f)
}

// parseClientVideoCodecs reads the optional comma-separated
// `clientVideoCodecs` query parameter reported by the browser. It returns nil
// when the client reports nothing, keeping the backend's static whitelist.
func parseClientVideoCodecs(r *http.Request) []string {
	raw := strings.TrimSpace(r.URL.Query().Get("clientVideoCodecs"))
	if raw == "" {
		return nil
	}
	codecs := make([]string, 0, 4)
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			codecs = append(codecs, part)
		}
	}
	if len(codecs) == 0 {
		return nil
	}
	return codecs
}

func (h *Handler) handleGetMoviePlayback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}

	movieID := strings.TrimSpace(r.PathValue("movieId"))
	if movieID == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "movieId is required")
		return
	}

	var startPositionSec *float64
	if raw, present := r.URL.Query()["startPositionSec"]; present {
		value, err := strconv.ParseFloat(raw[0], 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid startPositionSec")
			return
		}
		startPositionSec = &value
	}
	if h.playbackResolver != nil {
		dto, err := h.playbackResolver.ResolvePlayback(r.Context(), movieID, parseClientVideoCodecs(r), startPositionSec)
		if err == nil {
			writeJSON(w, http.StatusOK, dto)
			return
		}
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, storage.ErrMovieVideoNotFound) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie not found")
			return
		}
		if h.logger != nil {
			h.logger.Warn("resolve playback failed", zap.Error(err), zap.String("movieId", movieID))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to resolve playback")
		return
	}

	detail, err := h.store.GetMovieDetail(r.Context(), movieID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie not found")
			return
		}
		if h.logger != nil {
			h.logger.Warn("get movie playback detail failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load movie")
		return
	}

	progress, err := h.store.GetPlaybackProgress(r.Context(), movieID)
	if err != nil && h.logger != nil {
		h.logger.Warn("get playback progress failed", zap.Error(err), zap.String("movieId", movieID))
	}

	streamURL := "/api/library/movies/" + url.PathEscape(movieID) + "/stream"
	fileName := filepath.Base(strings.TrimSpace(detail.Location))
	mimeType := mime.TypeByExtension(strings.ToLower(filepath.Ext(fileName)))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	dto := contracts.PlaybackDescriptorDTO{
		MovieID:        movieID,
		Mode:           contracts.PlaybackModeDirect,
		URL:            streamURL,
		MimeType:       mimeType,
		FileName:       fileName,
		DurationSec:    float64(detail.RuntimeMinutes) * 60,
		CanDirectPlay:  true,
		AudioTracks:    []contracts.PlaybackAudioTrackDTO{},
		SubtitleTracks: []contracts.PlaybackSubtitleTrackDTO{},
	}
	if progress != nil && progress.PositionSec > 0 {
		dto.ResumePositionSec = progress.PositionSec
		if progress.DurationSec > 0 {
			dto.DurationSec = progress.DurationSec
		}
	}

	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) handleCreatePlaybackSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.playbackResolver == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "playback session manager not configured")
		return
	}
	movieID := strings.TrimSpace(r.PathValue("movieId"))
	if movieID == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "movieId is required")
		return
	}
	var body contracts.CreatePlaybackSessionRequest
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid json body")
			return
		}
	}
	if body.Mode == "" {
		body.Mode = contracts.PlaybackModeDirect
	}
	dto, err := h.playbackResolver.CreatePlaybackSession(r.Context(), movieID, body.Mode, body.StartPositionSec)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, storage.ErrMovieVideoNotFound) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie not found")
			return
		}
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, dto)
}

func (h *Handler) handleLaunchNativePlayback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.nativePlaybackLauncher == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "native playback is not configured")
		return
	}
	movieID := strings.TrimSpace(r.PathValue("movieId"))
	if movieID == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "movieId is required")
		return
	}
	var body contracts.NativePlaybackLaunchRequest
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid json body")
			return
		}
	}
	dto, err := h.nativePlaybackLauncher.LaunchNativePlayback(r.Context(), movieID, body.StartPositionSec)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, storage.ErrMovieVideoNotFound) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie not found")
			return
		}
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) handleGetPlaybackSessionFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.playbackResolver == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "playback session manager not configured")
		return
	}
	sessionID := strings.TrimSpace(r.PathValue("sessionId"))
	name := strings.TrimSpace(r.PathValue("file"))
	if sessionID == "" || name == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "sessionId and file are required")
		return
	}
	absPath, err := h.playbackResolver.ResolvePlaybackSessionFile(sessionID, name)
	if err != nil {
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "session file not found")
		return
	}
	ext := strings.ToLower(filepath.Ext(absPath))
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	switch ext {
	case ".m3u8":
		raw, err := os.ReadFile(absPath)
		if err != nil {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "session file not found")
			return
		}
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		w.WriteHeader(http.StatusOK)
		if r.Method == http.MethodHead {
			return
		}
		_, _ = w.Write(raw)
		return
	case ".ts":
		w.Header().Set("Content-Type", "video/mp2t")
	case ".m4s", ".mp4":
		w.Header().Set("Content-Type", "video/mp4")
	}

	http.ServeFile(w, r, absPath)
}

func (h *Handler) handleGetRecentPlaybackSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.playbackResolver == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "playback session manager not configured")
		return
	}
	limit := 20
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	dto, err := h.playbackResolver.ListRecentPlaybackSessions(r.Context(), limit)
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("list playback sessions failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to list playback sessions")
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) handleGetPlaybackSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.playbackResolver == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "playback session manager not configured")
		return
	}
	sessionID := strings.TrimSpace(r.PathValue("sessionId"))
	if sessionID == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "sessionId is required")
		return
	}
	dto, err := h.playbackResolver.GetPlaybackSession(r.Context(), sessionID)
	if err != nil {
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "playback session not found")
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) handleDeletePlaybackSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.playbackResolver == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "playback session manager not configured")
		return
	}
	sessionID := strings.TrimSpace(r.PathValue("sessionId"))
	if sessionID == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "sessionId is required")
		return
	}
	if err := h.playbackResolver.DeletePlaybackSession(sessionID); err != nil {
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "playback session not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleRevealMovieInFileManager(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	movieID := strings.TrimSpace(r.PathValue("movieId"))
	if movieID == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "movieId is required")
		return
	}

	absPath, err := h.store.ResolvePrimaryVideoPath(r.Context(), movieID)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrMovieVideoNotFound):
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie or video file not found")
		case errors.Is(err, storage.ErrMovieVideoNoLocation):
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie has no video file")
		case errors.Is(err, storage.ErrMovieVideoForbidden):
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie or video file not found")
		case errors.Is(err, storage.ErrMovieVideoNotFile):
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie or video file not found")
		default:
			if h.logger != nil {
				h.logger.Warn("resolve primary video for reveal failed", zap.Error(err))
			}
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to resolve video path")
		}
		return
	}

	// Use a detached timeout — not r.Context(): when the 204 is sent the request context
	// is often cancelled immediately and would kill the explorer child from exec.CommandContext.
	revealCtx, revealCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer revealCancel()
	if err := revealInFileManagerFn(revealCtx, absPath); err != nil {
		if h.logger != nil {
			h.logger.Warn("reveal in file manager failed", zap.Error(err), zap.String("path", absPath))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to open file manager")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleRevealLibraryPathInFileManager(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}

	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "id is required")
		return
	}

	dto, err := h.store.GetLibraryPath(r.Context(), id)
	if err != nil {
		if errors.Is(err, storage.ErrLibraryPathNotFound) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "library path not found")
			return
		}
		if h.logger != nil {
			h.logger.Warn("get library path for reveal failed", zap.Error(err), zap.String("id", id))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load library path")
		return
	}

	absDir := filepath.Clean(strings.TrimSpace(dto.Path))
	if absDir == "" || !filepath.IsAbs(absDir) {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "library path is invalid")
		return
	}

	info, err := os.Stat(absDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "library path directory not found")
			return
		}
		if h.logger != nil {
			h.logger.Warn("stat library path for reveal failed", zap.Error(err), zap.String("path", absDir))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to access library path")
		return
	}
	if !info.IsDir() {
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "library path directory not found")
		return
	}

	revealCtx, revealCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer revealCancel()
	if err := openDirectoryFn(revealCtx, absDir); err != nil {
		if h.logger != nil {
			h.logger.Warn("open library path in file manager failed", zap.Error(err), zap.String("path", absDir))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to open file manager")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleGetMovieComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	movieID := strings.TrimSpace(r.PathValue("movieId"))
	if movieID == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "movieId is required")
		return
	}
	ok, err := h.store.MovieRowExists(r.Context(), movieID)
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("movie comment get: exists check failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load movie")
		return
	}
	if !ok {
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie not found")
		return
	}
	dto, err := h.store.GetMovieComment(r.Context(), movieID)
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("get movie comment failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load comment")
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) handlePutMovieComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	movieID := strings.TrimSpace(r.PathValue("movieId"))
	if movieID == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "movieId is required")
		return
	}
	trashed, terr := h.store.IsMovieTrashed(r.Context(), movieID)
	if terr != nil {
		if h.logger != nil {
			h.logger.Warn("put movie comment: trashed check failed", zap.Error(terr))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load movie")
		return
	}
	if trashed {
		writeAppError(w, http.StatusConflict, contracts.ErrorCodeConflict, "movie is in trash")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
	if err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid body")
		return
	}
	var in contracts.PutMovieCommentBody
	if err := json.Unmarshal(body, &in); err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid JSON")
		return
	}
	dto, err := h.store.UpsertMovieComment(r.Context(), movieID, in.Body)
	if err != nil {
		if errors.Is(err, storage.ErrMovieNotFound) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie not found")
			return
		}
		if errors.Is(err, storage.ErrMovieCommentTooLong) {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "comment body too long")
			return
		}
		if h.logger != nil {
			h.logger.Warn("put movie comment failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to save comment")
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func parsePatchMovieInput(body []byte) (contracts.PatchMovieInput, error) {
	var in contracts.PatchMovieInput
	if len(bytes.TrimSpace(body)) == 0 {
		return in, errors.New("empty body")
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(body, &m); err != nil {
		return in, err
	}
	if raw, ok := m["isFavorite"]; ok {
		var b bool
		if err := json.Unmarshal(raw, &b); err != nil {
			return in, err
		}
		in.Favorite = &b
	}
	if raw, ok := m["rating"]; ok {
		in.UserRatingSet = true
		if string(bytes.TrimSpace(raw)) == "null" {
			in.UserRatingClear = true
		} else {
			var v float64
			if err := json.Unmarshal(raw, &v); err != nil {
				return in, err
			}
			in.UserRating = v
		}
	}
	if raw, ok := m["userTags"]; ok {
		in.UserTagsSet = true
		var tags []string
		if err := json.Unmarshal(raw, &tags); err != nil {
			return in, err
		}
		in.UserTags = tags
	}
	if raw, ok := m["metadataTags"]; ok {
		in.MetadataTagsSet = true
		var tags []string
		if err := json.Unmarshal(raw, &tags); err != nil {
			return in, err
		}
		in.MetadataTags = tags
	}
	if err := parsePatchOptionalStringField(m, "userTitle", &in.UserTitleSet, &in.UserTitleClear, &in.UserTitle); err != nil {
		return in, err
	}
	if err := parsePatchOptionalStringField(m, "userStudio", &in.UserStudioSet, &in.UserStudioClear, &in.UserStudio); err != nil {
		return in, err
	}
	if err := parsePatchOptionalStringField(m, "userSummary", &in.UserSummarySet, &in.UserSummaryClear, &in.UserSummary); err != nil {
		return in, err
	}
	if err := parsePatchOptionalStringField(m, "userReleaseDate", &in.UserReleaseDateSet, &in.UserReleaseDateClear, &in.UserReleaseDate); err != nil {
		return in, err
	}
	if raw, ok := m["userRuntimeMinutes"]; ok {
		in.UserRuntimeMinutesSet = true
		t := bytes.TrimSpace(raw)
		if len(t) == 0 || string(t) == "null" {
			in.UserRuntimeMinutesClear = true
		} else {
			var f float64
			if err := json.Unmarshal(raw, &f); err != nil {
				return in, err
			}
			if math.IsNaN(f) || math.IsInf(f, 0) || f < 0 || f > 99999 || f != math.Trunc(f) {
				return in, errors.New("invalid userRuntimeMinutes")
			}
			in.UserRuntimeMinutes = int(f)
		}
	}
	return in, nil
}

func parsePatchOptionalStringField(
	m map[string]json.RawMessage,
	key string,
	set *bool,
	clear *bool,
	out *string,
) error {
	raw, ok := m[key]
	if !ok {
		return nil
	}
	*set = true
	t := bytes.TrimSpace(raw)
	if len(t) == 0 || string(t) == "null" {
		*clear = true
		return nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return err
	}
	s = strings.TrimSpace(s)
	if s == "" {
		*clear = true
		return nil
	}
	*out = s
	return nil
}

var patchUserReleaseDateRx = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func validatePatchMovieDisplay(in contracts.PatchMovieInput) error {
	const maxSummary = 120_000
	if in.UserSummarySet && !in.UserSummaryClear && len(in.UserSummary) > maxSummary {
		return errors.New("userSummary too long")
	}
	if in.UserReleaseDateSet && !in.UserReleaseDateClear && in.UserReleaseDate != "" {
		if !patchUserReleaseDateRx.MatchString(in.UserReleaseDate) {
			return errors.New("userReleaseDate must be YYYY-MM-DD")
		}
	}
	if in.UserRuntimeMinutesSet && !in.UserRuntimeMinutesClear {
		if in.UserRuntimeMinutes < 0 || in.UserRuntimeMinutes > 99999 {
			return errors.New("userRuntimeMinutes out of range")
		}
	}
	return nil
}

func patchMovieInputHasFields(in contracts.PatchMovieInput) bool {
	if in.Favorite != nil {
		return true
	}
	if in.UserRatingSet {
		return true
	}
	if in.UserTagsSet {
		return true
	}
	if in.MetadataTagsSet {
		return true
	}
	if in.UserTitleSet || in.UserStudioSet || in.UserSummarySet || in.UserReleaseDateSet || in.UserRuntimeMinutesSet {
		return true
	}
	return false
}

func (h *Handler) handlePatchMovie(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	movieID := strings.TrimSpace(r.PathValue("movieId"))
	if movieID == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "movieId is required")
		return
	}

	trashed, terr := h.store.IsMovieTrashed(r.Context(), movieID)
	if terr != nil {
		if h.logger != nil {
			h.logger.Warn("patch movie: trashed check failed", zap.Error(terr))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load movie")
		return
	}
	if trashed {
		writeAppError(w, http.StatusConflict, contracts.ErrorCodeConflict, "movie is in trash")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if r.Body != nil {
		_ = r.Body.Close()
	}
	if err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "failed to read body")
		return
	}

	in, err := parsePatchMovieInput(body)
	if err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid json body")
		return
	}
	if !patchMovieInputHasFields(in) {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "no fields to update")
		return
	}
	if in.UserRatingSet && !in.UserRatingClear && (in.UserRating < 0 || in.UserRating > 5) {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, storage.ErrInvalidUserRating.Error())
		return
	}
	if err := validatePatchMovieDisplay(in); err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, err.Error())
		return
	}

	err = h.store.PatchMovieUserPrefs(r.Context(), movieID, in)
	if err != nil {
		if errors.Is(err, storage.ErrInvalidUserTags) {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, err.Error())
			return
		}
		if errors.Is(err, storage.ErrMovieNotFoundForPatch) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie not found")
			return
		}
		if errors.Is(err, storage.ErrInvalidUserRating) {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, err.Error())
			return
		}
		if h.logger != nil {
			h.logger.Warn("patch movie failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to patch movie")
		return
	}

	movie, err := h.store.GetMovieDetail(r.Context(), movieID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie not found")
			return
		}
		if h.logger != nil {
			h.logger.Warn("get movie after patch failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load movie")
		return
	}
	h.enrichMovieDetailLocalPosters(r.Context(), &movie)
	writeJSON(w, http.StatusOK, movie)
}

func (h *Handler) handleRefreshMovieMetadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.movieMetadataRefresher == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "metadata refresh not configured")
		return
	}
	movieID := strings.TrimSpace(r.PathValue("movieId"))
	if movieID == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "movieId is required")
		return
	}
	trashed, terr := h.store.IsMovieTrashed(r.Context(), movieID)
	if terr != nil {
		if h.logger != nil {
			h.logger.Warn("refresh metadata: trashed check failed", zap.Error(terr))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load movie")
		return
	}
	if trashed {
		writeAppError(w, http.StatusConflict, contracts.ErrorCodeConflict, "movie is in trash")
		return
	}

	task, err := h.movieMetadataRefresher.StartMovieMetadataRefresh(r.Context(), movieID)
	if err != nil {
		switch {
		case errors.Is(err, contracts.ErrScrapeMovieNotFound):
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie not found")
		case errors.Is(err, contracts.ErrScrapeMovieNoCode), errors.Is(err, contracts.ErrScrapeMovieNoLocation):
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, err.Error())
		default:
			if h.logger != nil {
				h.logger.Warn("start movie metadata refresh failed", zap.Error(err))
			}
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to start metadata refresh")
		}
		return
	}

	writeJSON(w, http.StatusAccepted, task)
}

func (h *Handler) handleMetadataScrapeByPaths(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.movieMetadataRefresher == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "metadata refresh not configured")
		return
	}

	var body contracts.StartMetadataRefreshByPathsRequest
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid json body")
			return
		}
	}
	if len(body.Paths) == 0 {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "paths must contain at least one entry")
		return
	}

	dto, err := h.movieMetadataRefresher.StartMetadataRefreshForLibraryPaths(r.Context(), body.Paths)
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("bulk metadata refresh failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to queue metadata refresh")
		return
	}

	writeJSON(w, http.StatusAccepted, dto)
}

// handleRestoreMovie clears trashed_at. 204 on success; 404 if missing; 400 if not in trash.
func (h *Handler) handleRestoreMovie(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	movieID := strings.TrimSpace(r.PathValue("movieId"))
	if movieID == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "movieId is required")
		return
	}
	err := h.store.RestoreMovie(r.Context(), movieID)
	if err != nil {
		if errors.Is(err, storage.ErrMovieNotFound) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie not found")
			return
		}
		if errors.Is(err, storage.ErrMovieNotInTrash) {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "movie is not in trash")
			return
		}
		if h.logger != nil {
			h.logger.Warn("restore movie failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to restore movie")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleDeleteMovie without ?permanent=true moves the movie to trash (204). With permanent=true,
// removes DB rows and on-disk files only if already trashed (see storage.DeleteMoviePermanently).
func (h *Handler) handleDeleteMovie(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	movieID := strings.TrimSpace(r.PathValue("movieId"))
	if movieID == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "movieId is required")
		return
	}
	permanent := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("permanent")), "true")
	if permanent {
		err := h.store.DeleteMoviePermanently(r.Context(), movieID, strings.TrimSpace(h.cfg.CacheDir))
		if err != nil {
			if errors.Is(err, storage.ErrMovieNotFound) {
				writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie not found")
				return
			}
			if errors.Is(err, storage.ErrMovieNotInTrash) {
				writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "movie must be in trash before permanent delete")
				return
			}
			if h.logger != nil {
				h.logger.Warn("permanent delete movie failed", zap.Error(err))
			}
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to delete movie")
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	err := h.store.TrashMovie(r.Context(), movieID)
	if err != nil {
		if errors.Is(err, storage.ErrMovieNotFound) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "movie not found")
			return
		}
		if h.logger != nil {
			h.logger.Warn("trash movie failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to move movie to trash")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) buildSettingsDTO(ctx context.Context) (contracts.SettingsDTO, error) {
	libraryPaths, err := h.store.ListLibraryPaths(ctx)
	if err != nil {
		return contracts.SettingsDTO{}, err
	}
	comicLibraryPaths, err := h.store.ListComicLibraryPaths(ctx)
	if err != nil {
		return contracts.SettingsDTO{}, err
	}
	photoLibraryPaths, err := h.store.ListPhotoLibraryPaths(ctx)
	if err != nil {
		return contracts.SettingsDTO{}, err
	}
	org := h.cfg.OrganizeLibrary
	if h.organizeLibraryCtl != nil {
		org = h.organizeLibraryCtl.OrganizeLibrary()
	}
	autoWatch := h.cfg.AutoLibraryWatch
	if h.autoLibraryWatchCtl != nil {
		autoWatch = h.autoLibraryWatchCtl.AutoLibraryWatch()
	}
	autoActorProfileScrape := h.cfg.AutoActorProfileScrape
	if h.autoActorProfileScrapeCtl != nil {
		autoActorProfileScrape = h.autoActorProfileScrapeCtl.AutoActorProfileScrape()
	}
	autoDownloadUpdates := h.cfg.AutoDownloadUpdates
	if h.autoDownloadUpdatesCtl != nil {
		autoDownloadUpdates = h.autoDownloadUpdatesCtl.AutoDownloadUpdates()
	}
	launchAtLogin := h.cfg.LaunchAtLogin
	launchAtLoginSupported := false
	if h.launchAtLoginCtl != nil {
		launchAtLogin = h.launchAtLoginCtl.LaunchAtLogin()
		launchAtLoginSupported = h.launchAtLoginCtl.LaunchAtLoginSupported()
	}
	curatedFrameExportFormat := config.NormalizeCuratedFrameExportFormat(h.cfg.CuratedFrameExportFormat)
	if h.curatedFrameExportFormatCtl != nil {
		curatedFrameExportFormat = config.NormalizeCuratedFrameExportFormat(h.curatedFrameExportFormatCtl.CuratedFrameExportFormat())
	}
	curatedFrameExportMode := config.NormalizeCuratedFrameExportMode(h.cfg.CuratedFrameExportMode)
	if h.curatedFrameExportModeCtl != nil {
		curatedFrameExportMode = config.NormalizeCuratedFrameExportMode(h.curatedFrameExportModeCtl.CuratedFrameExportMode())
	}
	defaultImportLibraryPathID := strings.TrimSpace(h.cfg.DefaultImportLibraryPathID)
	if h.defaultImportLibraryPathCtl != nil {
		defaultImportLibraryPathID = strings.TrimSpace(h.defaultImportLibraryPathCtl.DefaultImportLibraryPathID())
	}
	backupDirectory := strings.TrimSpace(h.cfg.BackupDirectory)
	if h.backupDirectoryCtl != nil {
		backupDirectory = strings.TrimSpace(h.backupDirectoryCtl.BackupDirectory())
	}
	comicLibraryEnabled := h.cfg.ComicLibraryEnabled
	autoComicLibraryWatch := h.cfg.AutoComicLibraryWatch
	defaultComicImportLibraryPathID := strings.TrimSpace(h.cfg.DefaultComicImportLibraryPathID)
	comicReader := comicReaderSettingsDTOFromConfig(h.cfg.ComicReader)
	comicCache := comicCacheSettingsDTOFromConfig(h.cfg.ComicCache)
	if h.comicSettingsCtl != nil {
		comicLibraryEnabled = h.comicSettingsCtl.ComicLibraryEnabled()
		autoComicLibraryWatch = h.comicSettingsCtl.AutoComicLibraryWatch()
		defaultComicImportLibraryPathID = strings.TrimSpace(h.comicSettingsCtl.DefaultComicImportLibraryPathID())
		comicReader = normalizeComicReaderSettingsDTO(h.comicSettingsCtl.ComicReaderSettings())
		comicCache = normalizeComicCacheSettingsDTO(h.comicSettingsCtl.ComicCacheSettings())
	}
	photoLibraryEnabled := h.cfg.PhotoLibraryEnabled
	autoPhotoLibraryWatch := h.cfg.AutoPhotoLibraryWatch
	defaultPhotoImportLibraryPathID := strings.TrimSpace(h.cfg.DefaultPhotoImportLibraryPathID)
	photoViewer := photoViewerSettingsDTOFromConfig(h.cfg.PhotoViewer)
	photoCache := photoCacheSettingsDTOFromConfig(h.cfg.PhotoCache)
	if h.photoSettingsCtl != nil {
		photoLibraryEnabled = h.photoSettingsCtl.PhotoLibraryEnabled()
		autoPhotoLibraryWatch = h.photoSettingsCtl.AutoPhotoLibraryWatch()
		defaultPhotoImportLibraryPathID = strings.TrimSpace(h.photoSettingsCtl.DefaultPhotoImportLibraryPathID())
		photoViewer = normalizePhotoViewerSettingsDTO(h.photoSettingsCtl.PhotoViewerSettings())
		photoCache = normalizePhotoCacheSettingsDTO(h.photoSettingsCtl.PhotoCacheSettings())
	}
	dto := contracts.SettingsDTO{
		LibraryPaths:                    libraryPaths,
		DefaultImportLibraryPathID:      defaultImportLibraryPathID,
		BackupDirectory:                 backupDirectory,
		ComicLibraryEnabled:             comicLibraryEnabled,
		AutoComicLibraryWatch:           autoComicLibraryWatch,
		ComicLibraryPaths:               comicLibraryPaths,
		DefaultComicImportLibraryPathID: defaultComicImportLibraryPathID,
		ComicReader:                     comicReader,
		ComicCache:                      comicCache,
		PhotoLibraryEnabled:             photoLibraryEnabled,
		AutoPhotoLibraryWatch:           autoPhotoLibraryWatch,
		PhotoLibraryPaths:               photoLibraryPaths,
		DefaultPhotoImportLibraryPathID: defaultPhotoImportLibraryPathID,
		PhotoViewer:                     photoViewer,
		PhotoCache:                      photoCache,
		Player: contracts.PlayerSettingsDTO{
			HardwareDecode:      h.cfg.Player.HardwareDecode,
			NativePlayerEnabled: h.cfg.Player.NativePlayerEnabled,
			NativePlayerCommand: h.cfg.Player.NativePlayerCommand,
			StreamPushEnabled:   h.cfg.Player.StreamPushEnabled,
			ForceStreamPush:     h.cfg.Player.ForceStreamPush,
			FFmpegCommand:       h.cfg.Player.FFmpegCommand,
			PreferNativePlayer:  h.cfg.Player.PreferNativePlayer,
			SeekForwardStepSec:  h.cfg.Player.SeekForwardStepSec,
			SeekBackwardStepSec: h.cfg.Player.SeekBackwardStepSec,
		},
		OrganizeLibrary:          org,
		AutoLibraryWatch:         autoWatch,
		AutoActorProfileScrape:   autoActorProfileScrape,
		AutoDownloadUpdates:      autoDownloadUpdates,
		LaunchAtLogin:            launchAtLogin,
		LaunchAtLoginSupported:   launchAtLoginSupported,
		CuratedFrameExportFormat: curatedFrameExportFormat,
		CuratedFrameExportMode:   curatedFrameExportMode,
		MetadataMovieProviders:   []string{},
	}
	if strings.TrimSpace(dto.Player.NativePlayerCommand) == "" {
		dto.Player.NativePlayerCommand = "mpv"
	}
	if strings.TrimSpace(dto.Player.FFmpegCommand) == "" {
		dto.Player.FFmpegCommand = "ffmpeg"
	}
	if dto.Player.SeekForwardStepSec <= 0 {
		dto.Player.SeekForwardStepSec = 10
	}
	if dto.Player.SeekBackwardStepSec <= 0 {
		dto.Player.SeekBackwardStepSec = 10
	}
	if h.playerSettingsCtl != nil {
		dto.Player = h.playerSettingsCtl.PlayerSettings()
	}
	if h.metadataScrapeCtl != nil {
		dto.MetadataMovieProvider = h.metadataScrapeCtl.MetadataMovieProvider()
		dto.MetadataMovieProviderChain = h.metadataScrapeCtl.MetadataMovieProviderChain()
		dto.MetadataMovieScrapeMode = h.metadataScrapeCtl.MetadataMovieScrapeMode()
		dto.MetadataMovieStrategy = h.metadataScrapeCtl.MetadataMovieStrategy()
		if list := h.metadataScrapeCtl.ListMetadataMovieProviders(); list != nil {
			dto.MetadataMovieProviders = list
		}
	}
	if h.proxyCtl != nil {
		p := h.proxyCtl.Proxy()
		dto.Proxy = contracts.ProxySettingsDTO{
			Enabled:  p.Enabled,
			URL:      p.URL,
			Username: p.Username,
			Password: p.Password,
		}
	}
	if h.aiSettingsCtl != nil {
		dto.AIProvider = h.aiSettingsCtl.AIProviderSettings()
	}
	if p, ok := h.aiChatProvider.(AIGovernanceProvider); ok {
		settings := p.AIGovernanceSettings()
		dto.AIGovernance = &settings
	}
	if h.backendLogCtl != nil {
		dto.BackendLog = h.backendLogCtl.BackendLogSettings()
	}
	return dto, nil
}

func patchBackendLogHasChanges(p *contracts.PatchBackendLogSettings) bool {
	if p == nil {
		return false
	}
	return p.LogDir != nil || p.LogFilePrefix != nil || p.LogMaxAgeDays != nil || p.LogLevel != nil
}

func movieProviderNameAllowed(want string, allowed []string) bool {
	if strings.TrimSpace(want) == "" {
		return true
	}
	lw := strings.ToLower(strings.TrimSpace(want))
	for _, a := range allowed {
		if strings.ToLower(strings.TrimSpace(a)) == lw {
			return true
		}
	}
	return false
}

type settingsPatchFailure struct {
	status  int
	code    string
	message func(error) string
}

func fixedSettingsPatchMessage(msg string) func(error) string {
	return func(error) string { return msg }
}

func (f settingsPatchFailure) userMessage(err error) string {
	if f.message != nil {
		return f.message(err)
	}
	if err != nil {
		return err.Error()
	}
	return "failed to update settings"
}

type settingsPatchOperation struct {
	name     string
	apply    func() error
	rollback func() error
	failure  settingsPatchFailure
}

func rollbackSettingsPatchOperations(ops []settingsPatchOperation, applied int) error {
	for i := applied - 1; i >= 0; i-- {
		if ops[i].rollback == nil {
			continue
		}
		if err := ops[i].rollback(); err != nil {
			return fmt.Errorf("%s rollback failed: %w", ops[i].name, err)
		}
	}
	return nil
}

func boolPtr(v bool) *bool       { return &v }
func stringPtr(v string) *string { return &v }
func intPtr(v int) *int          { return &v }

func backendLogPatchFromDTO(dto contracts.BackendLogSettingsDTO) contracts.PatchBackendLogSettings {
	return contracts.PatchBackendLogSettings{
		LogDir:        stringPtr(dto.LogDir),
		LogFilePrefix: stringPtr(dto.LogFilePrefix),
		LogMaxAgeDays: intPtr(dto.LogMaxAgeDays),
		LogLevel:      stringPtr(dto.LogLevel),
	}
}

func playerSettingsPatchFromDTO(dto contracts.PlayerSettingsDTO) contracts.PatchPlayerSettingsDTO {
	return contracts.PatchPlayerSettingsDTO{
		HardwareDecode:      boolPtr(dto.HardwareDecode),
		HardwareEncoder:     stringPtr(dto.HardwareEncoder),
		NativePlayerPreset:  stringPtr(dto.NativePlayerPreset),
		NativePlayerEnabled: boolPtr(dto.NativePlayerEnabled),
		NativePlayerCommand: stringPtr(dto.NativePlayerCommand),
		StreamPushEnabled:   boolPtr(dto.StreamPushEnabled),
		ForceStreamPush:     boolPtr(dto.ForceStreamPush),
		FFmpegCommand:       stringPtr(dto.FFmpegCommand),
		PreferNativePlayer:  boolPtr(dto.PreferNativePlayer),
		SeekForwardStepSec:  intPtr(dto.SeekForwardStepSec),
		SeekBackwardStepSec: intPtr(dto.SeekBackwardStepSec),
	}
}

func comicReaderSettingsDTOFromConfig(v config.ComicReaderConfig) contracts.ComicReaderSettingsDTO {
	n := config.NormalizeComicReaderConfig(v)
	return contracts.ComicReaderSettingsDTO{
		Mode:      n.Mode,
		Fit:       n.Fit,
		Direction: n.Direction,
	}
}

func comicCacheSettingsDTOFromConfig(v config.ComicCacheConfig) contracts.ComicCacheSettingsDTO {
	n := config.NormalizeComicCacheConfig(v)
	return contracts.ComicCacheSettingsDTO{MaxBytes: n.MaxBytes}
}

func normalizeComicReaderSettingsDTO(v contracts.ComicReaderSettingsDTO) contracts.ComicReaderSettingsDTO {
	return comicReaderSettingsDTOFromConfig(config.ComicReaderConfig{
		Mode:      v.Mode,
		Fit:       v.Fit,
		Direction: v.Direction,
	})
}

func normalizeComicCacheSettingsDTO(v contracts.ComicCacheSettingsDTO) contracts.ComicCacheSettingsDTO {
	return comicCacheSettingsDTOFromConfig(config.ComicCacheConfig{MaxBytes: v.MaxBytes})
}

func photoViewerSettingsDTOFromConfig(v config.PhotoViewerConfig) contracts.PhotoViewerSettingsDTO {
	n := config.NormalizePhotoViewerConfig(v)
	return contracts.PhotoViewerSettingsDTO{
		Mode:      n.Mode,
		Fit:       n.Fit,
		Direction: n.Direction,
	}
}

func photoCacheSettingsDTOFromConfig(v config.PhotoCacheConfig) contracts.PhotoCacheSettingsDTO {
	n := config.NormalizePhotoCacheConfig(v)
	return contracts.PhotoCacheSettingsDTO{MaxBytes: n.MaxBytes}
}

func normalizePhotoViewerSettingsDTO(v contracts.PhotoViewerSettingsDTO) contracts.PhotoViewerSettingsDTO {
	return photoViewerSettingsDTOFromConfig(config.PhotoViewerConfig{
		Mode:      v.Mode,
		Fit:       v.Fit,
		Direction: v.Direction,
	})
}

func normalizePhotoCacheSettingsDTO(v contracts.PhotoCacheSettingsDTO) contracts.PhotoCacheSettingsDTO {
	return photoCacheSettingsDTOFromConfig(config.PhotoCacheConfig{MaxBytes: v.MaxBytes})
}

func (h *Handler) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	dto, err := h.buildSettingsDTO(r.Context())
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("build settings dto failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to list library paths")
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) handlePatchSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.organizeLibraryCtl == nil && h.metadataScrapeCtl == nil && h.autoLibraryWatchCtl == nil && h.autoActorProfileScrapeCtl == nil && h.autoDownloadUpdatesCtl == nil && h.launchAtLoginCtl == nil && h.curatedFrameExportFormatCtl == nil && h.curatedFrameExportModeCtl == nil && h.defaultImportLibraryPathCtl == nil && h.backupDirectoryCtl == nil && h.proxyCtl == nil && h.backendLogCtl == nil && h.playerSettingsCtl == nil && h.aiSettingsCtl == nil && h.comicSettingsCtl == nil && h.photoSettingsCtl == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "settings runtime not available")
		return
	}

	var body contracts.PatchSettingsRequest
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid json body")
			return
		}
	}

	if body.OrganizeLibrary == nil && body.AutoLibraryWatch == nil && body.AutoActorProfileScrape == nil && body.AutoDownloadUpdates == nil && body.LaunchAtLogin == nil && body.CuratedFrameExportFormat == nil && body.CuratedFrameExportMode == nil && body.DefaultImportLibraryPathID == nil && body.BackupDirectory == nil && body.MetadataMovieProvider == nil && body.MetadataMovieProviderChain == nil && body.MetadataMovieScrapeMode == nil && body.MetadataMovieStrategy == nil && body.Proxy == nil && body.AIProvider == nil && !patchBackendLogHasChanges(body.BackendLog) && body.Player == nil && body.ComicLibraryEnabled == nil && body.AutoComicLibraryWatch == nil && body.DefaultComicImportLibraryPathID == nil && body.ComicReader == nil && body.ComicCache == nil && body.PhotoLibraryEnabled == nil && body.AutoPhotoLibraryWatch == nil && body.DefaultPhotoImportLibraryPathID == nil && body.PhotoViewer == nil && body.PhotoCache == nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "no supported fields to update")
		return
	}

	ops := make([]settingsPatchOperation, 0, 26)

	if body.OrganizeLibrary != nil {
		if h.organizeLibraryCtl == nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "organize library settings not available")
			return
		}
		prev := h.organizeLibraryCtl.OrganizeLibrary()
		target := *body.OrganizeLibrary
		ops = append(ops, settingsPatchOperation{
			name:     "organizeLibrary",
			apply:    func() error { return h.organizeLibraryCtl.SetOrganizeLibrary(target) },
			rollback: func() error { return h.organizeLibraryCtl.SetOrganizeLibrary(prev) },
			failure: settingsPatchFailure{
				status:  http.StatusInternalServerError,
				code:    contracts.ErrorCodeInternal,
				message: fixedSettingsPatchMessage("failed to save library settings"),
			},
		})
	}

	if body.AutoLibraryWatch != nil {
		if h.autoLibraryWatchCtl == nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "auto library watch settings not available")
			return
		}
		prev := h.autoLibraryWatchCtl.AutoLibraryWatch()
		target := *body.AutoLibraryWatch
		ops = append(ops, settingsPatchOperation{
			name:     "autoLibraryWatch",
			apply:    func() error { return h.autoLibraryWatchCtl.SetAutoLibraryWatch(target) },
			rollback: func() error { return h.autoLibraryWatchCtl.SetAutoLibraryWatch(prev) },
			failure: settingsPatchFailure{
				status:  http.StatusInternalServerError,
				code:    contracts.ErrorCodeInternal,
				message: fixedSettingsPatchMessage("failed to save library settings"),
			},
		})
	}

	if body.AutoActorProfileScrape != nil {
		if h.autoActorProfileScrapeCtl == nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "auto actor profile scrape settings not available")
			return
		}
		prev := h.autoActorProfileScrapeCtl.AutoActorProfileScrape()
		target := *body.AutoActorProfileScrape
		ops = append(ops, settingsPatchOperation{
			name:     "autoActorProfileScrape",
			apply:    func() error { return h.autoActorProfileScrapeCtl.SetAutoActorProfileScrape(target) },
			rollback: func() error { return h.autoActorProfileScrapeCtl.SetAutoActorProfileScrape(prev) },
			failure: settingsPatchFailure{
				status:  http.StatusInternalServerError,
				code:    contracts.ErrorCodeInternal,
				message: fixedSettingsPatchMessage("failed to save library settings"),
			},
		})
	}

	if body.AutoDownloadUpdates != nil {
		if h.autoDownloadUpdatesCtl == nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "auto download update settings not available")
			return
		}
		prev := h.autoDownloadUpdatesCtl.AutoDownloadUpdates()
		target := *body.AutoDownloadUpdates
		ops = append(ops, settingsPatchOperation{
			name:     "autoDownloadUpdates",
			apply:    func() error { return h.autoDownloadUpdatesCtl.SetAutoDownloadUpdates(target) },
			rollback: func() error { return h.autoDownloadUpdatesCtl.SetAutoDownloadUpdates(prev) },
			failure: settingsPatchFailure{
				status:  http.StatusInternalServerError,
				code:    contracts.ErrorCodeInternal,
				message: fixedSettingsPatchMessage("failed to save library settings"),
			},
		})
	}

	if body.LaunchAtLogin != nil {
		if h.launchAtLoginCtl == nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "launch at login settings not available")
			return
		}
		if *body.LaunchAtLogin && !h.launchAtLoginCtl.LaunchAtLoginSupported() {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "launch at login is not supported in this runtime")
			return
		}
		prev := h.launchAtLoginCtl.LaunchAtLogin()
		target := *body.LaunchAtLogin
		ops = append(ops, settingsPatchOperation{
			name:     "launchAtLogin",
			apply:    func() error { return h.launchAtLoginCtl.SetLaunchAtLogin(target) },
			rollback: func() error { return h.launchAtLoginCtl.SetLaunchAtLogin(prev) },
			failure: settingsPatchFailure{
				status:  http.StatusInternalServerError,
				code:    contracts.ErrorCodeInternal,
				message: fixedSettingsPatchMessage("failed to save library settings"),
			},
		})
	}

	if body.CuratedFrameExportFormat != nil {
		if h.curatedFrameExportFormatCtl == nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "curated frame export format settings not available")
			return
		}
		target := strings.ToLower(strings.TrimSpace(*body.CuratedFrameExportFormat))
		if target != "jpg" && target != "webp" && target != "png" {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, `curatedFrameExportFormat must be one of "jpg", "webp", or "png"`)
			return
		}
		prev := h.curatedFrameExportFormatCtl.CuratedFrameExportFormat()
		ops = append(ops, settingsPatchOperation{
			name:     "curatedFrameExportFormat",
			apply:    func() error { return h.curatedFrameExportFormatCtl.SetCuratedFrameExportFormat(target) },
			rollback: func() error { return h.curatedFrameExportFormatCtl.SetCuratedFrameExportFormat(prev) },
			failure: settingsPatchFailure{
				status:  http.StatusBadRequest,
				code:    contracts.ErrorCodeBadRequest,
				message: func(err error) string { return err.Error() },
			},
		})
	}

	if body.CuratedFrameExportMode != nil {
		if h.curatedFrameExportModeCtl == nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "curated frame export mode settings not available")
			return
		}
		target := strings.ToLower(strings.TrimSpace(*body.CuratedFrameExportMode))
		if target != "raw" && target != "watermarked" {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, `curatedFrameExportMode must be one of "raw" or "watermarked"`)
			return
		}
		prev := h.curatedFrameExportModeCtl.CuratedFrameExportMode()
		ops = append(ops, settingsPatchOperation{
			name:     "curatedFrameExportMode",
			apply:    func() error { return h.curatedFrameExportModeCtl.SetCuratedFrameExportMode(target) },
			rollback: func() error { return h.curatedFrameExportModeCtl.SetCuratedFrameExportMode(prev) },
			failure: settingsPatchFailure{
				status:  http.StatusBadRequest,
				code:    contracts.ErrorCodeBadRequest,
				message: func(err error) string { return err.Error() },
			},
		})
	}

	if body.DefaultImportLibraryPathID != nil {
		if h.defaultImportLibraryPathCtl == nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "default import library path settings not available")
			return
		}
		target := strings.TrimSpace(*body.DefaultImportLibraryPathID)
		if target != "" {
			if _, err := h.store.GetLibraryPath(r.Context(), target); err != nil {
				if errors.Is(err, storage.ErrLibraryPathNotFound) {
					writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "unknown defaultImportLibraryPathId")
					return
				}
				if h.logger != nil {
					h.logger.Warn("validate default import library path failed", zap.Error(err))
				}
				writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to validate default import library path")
				return
			}
		}
		prev := h.defaultImportLibraryPathCtl.DefaultImportLibraryPathID()
		ops = append(ops, settingsPatchOperation{
			name:     "defaultImportLibraryPathId",
			apply:    func() error { return h.defaultImportLibraryPathCtl.SetDefaultImportLibraryPathID(target) },
			rollback: func() error { return h.defaultImportLibraryPathCtl.SetDefaultImportLibraryPathID(prev) },
			failure: settingsPatchFailure{
				status:  http.StatusInternalServerError,
				code:    contracts.ErrorCodeInternal,
				message: fixedSettingsPatchMessage("failed to save library settings"),
			},
		})
	}

	if body.BackupDirectory != nil {
		if h.backupDirectoryCtl == nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "backup directory settings not available")
			return
		}
		target := strings.TrimSpace(*body.BackupDirectory)
		if target != "" && !isAbsoluteBackupPath(target) {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "backupDirectory must be an absolute path or empty")
			return
		}
		prev := h.backupDirectoryCtl.BackupDirectory()
		ops = append(ops, settingsPatchOperation{
			name:     "backupDirectory",
			apply:    func() error { return h.backupDirectoryCtl.SetBackupDirectory(target) },
			rollback: func() error { return h.backupDirectoryCtl.SetBackupDirectory(prev) },
			failure: settingsPatchFailure{
				status:  http.StatusInternalServerError,
				code:    contracts.ErrorCodeInternal,
				message: fixedSettingsPatchMessage("failed to save library settings"),
			},
		})
	}

	if body.ComicLibraryEnabled != nil {
		if h.comicSettingsCtl == nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "comic library settings not available")
			return
		}
		prev := h.comicSettingsCtl.ComicLibraryEnabled()
		target := *body.ComicLibraryEnabled
		// 开启 Beta 后再配置路径，扫描入口仍要求有效路径。

		ops = append(ops, settingsPatchOperation{
			name:     "comicLibraryEnabled",
			apply:    func() error { return h.comicSettingsCtl.SetComicLibraryEnabled(target) },
			rollback: func() error { return h.comicSettingsCtl.SetComicLibraryEnabled(prev) },
			failure: settingsPatchFailure{
				status:  http.StatusInternalServerError,
				code:    contracts.ErrorCodeInternal,
				message: fixedSettingsPatchMessage("failed to save comic library settings"),
			},
		})
	}

	if body.AutoComicLibraryWatch != nil {
		if h.comicSettingsCtl == nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "comic library settings not available")
			return
		}
		prev := h.comicSettingsCtl.AutoComicLibraryWatch()
		target := *body.AutoComicLibraryWatch
		ops = append(ops, settingsPatchOperation{
			name:     "autoComicLibraryWatch",
			apply:    func() error { return h.comicSettingsCtl.SetAutoComicLibraryWatch(target) },
			rollback: func() error { return h.comicSettingsCtl.SetAutoComicLibraryWatch(prev) },
			failure: settingsPatchFailure{
				status:  http.StatusInternalServerError,
				code:    contracts.ErrorCodeInternal,
				message: fixedSettingsPatchMessage("failed to save comic library settings"),
			},
		})
	}

	if body.DefaultComicImportLibraryPathID != nil {
		if h.comicSettingsCtl == nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "comic library settings not available")
			return
		}
		target := strings.TrimSpace(*body.DefaultComicImportLibraryPathID)
		if target != "" {
			if _, err := h.store.GetComicLibraryPath(r.Context(), target); err != nil {
				if errors.Is(err, storage.ErrComicLibraryPathNotFound) {
					writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeComicPathNotFound, "unknown defaultComicImportLibraryPathId")
					return
				}
				if h.logger != nil {
					h.logger.Warn("validate default comic import library path failed", zap.Error(err))
				}
				writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to validate default comic import library path")
				return
			}
		}
		prev := h.comicSettingsCtl.DefaultComicImportLibraryPathID()
		ops = append(ops, settingsPatchOperation{
			name:     "defaultComicImportLibraryPathId",
			apply:    func() error { return h.comicSettingsCtl.SetDefaultComicImportLibraryPathID(target) },
			rollback: func() error { return h.comicSettingsCtl.SetDefaultComicImportLibraryPathID(prev) },
			failure: settingsPatchFailure{
				status:  http.StatusInternalServerError,
				code:    contracts.ErrorCodeInternal,
				message: fixedSettingsPatchMessage("failed to save comic library settings"),
			},
		})
	}

	if body.ComicReader != nil {
		if h.comicSettingsCtl == nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "comic library settings not available")
			return
		}
		prev := normalizeComicReaderSettingsDTO(h.comicSettingsCtl.ComicReaderSettings())
		target := normalizeComicReaderSettingsDTO(*body.ComicReader)
		ops = append(ops, settingsPatchOperation{
			name:     "comicReader",
			apply:    func() error { return h.comicSettingsCtl.SetComicReaderSettings(target) },
			rollback: func() error { return h.comicSettingsCtl.SetComicReaderSettings(prev) },
			failure: settingsPatchFailure{
				status:  http.StatusBadRequest,
				code:    contracts.ErrorCodeBadRequest,
				message: func(err error) string { return err.Error() },
			},
		})
	}

	if body.ComicCache != nil {
		if h.comicSettingsCtl == nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "comic library settings not available")
			return
		}
		prev := normalizeComicCacheSettingsDTO(h.comicSettingsCtl.ComicCacheSettings())
		target := normalizeComicCacheSettingsDTO(*body.ComicCache)
		ops = append(ops, settingsPatchOperation{
			name:     "comicCache",
			apply:    func() error { return h.comicSettingsCtl.SetComicCacheSettings(target) },
			rollback: func() error { return h.comicSettingsCtl.SetComicCacheSettings(prev) },
			failure: settingsPatchFailure{
				status:  http.StatusBadRequest,
				code:    contracts.ErrorCodeBadRequest,
				message: func(err error) string { return err.Error() },
			},
		})
	}

	if body.PhotoLibraryEnabled != nil {
		if h.photoSettingsCtl == nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "photo library settings not available")
			return
		}
		prev := h.photoSettingsCtl.PhotoLibraryEnabled()
		target := *body.PhotoLibraryEnabled
		// 开启 Beta 后再配置路径，扫描入口仍要求有效路径。

		ops = append(ops, settingsPatchOperation{
			name:     "photoLibraryEnabled",
			apply:    func() error { return h.photoSettingsCtl.SetPhotoLibraryEnabled(target) },
			rollback: func() error { return h.photoSettingsCtl.SetPhotoLibraryEnabled(prev) },
			failure: settingsPatchFailure{
				status:  http.StatusInternalServerError,
				code:    contracts.ErrorCodeInternal,
				message: fixedSettingsPatchMessage("failed to save photo library settings"),
			},
		})
	}

	if body.AutoPhotoLibraryWatch != nil {
		if h.photoSettingsCtl == nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "photo library settings not available")
			return
		}
		prev := h.photoSettingsCtl.AutoPhotoLibraryWatch()
		target := *body.AutoPhotoLibraryWatch
		ops = append(ops, settingsPatchOperation{
			name:     "autoPhotoLibraryWatch",
			apply:    func() error { return h.photoSettingsCtl.SetAutoPhotoLibraryWatch(target) },
			rollback: func() error { return h.photoSettingsCtl.SetAutoPhotoLibraryWatch(prev) },
			failure: settingsPatchFailure{
				status:  http.StatusInternalServerError,
				code:    contracts.ErrorCodeInternal,
				message: fixedSettingsPatchMessage("failed to save photo library settings"),
			},
		})
	}

	if body.DefaultPhotoImportLibraryPathID != nil {
		if h.photoSettingsCtl == nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "photo library settings not available")
			return
		}
		prev := h.photoSettingsCtl.DefaultPhotoImportLibraryPathID()
		target := strings.TrimSpace(*body.DefaultPhotoImportLibraryPathID)
		if target != "" {
			if _, err := h.store.GetPhotoLibraryPath(r.Context(), target); err != nil {
				if errors.Is(err, storage.ErrPhotoLibraryPathNotFound) {
					writeAppError(w, http.StatusNotFound, contracts.ErrorCodePhotoPathNotFound, "photo library path not found")
					return
				}
				if h.logger != nil {
					h.logger.Warn("validate default photo import library path failed", zap.Error(err))
				}
				writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to validate default photo import library path")
				return
			}
		}
		ops = append(ops, settingsPatchOperation{
			name:     "defaultPhotoImportLibraryPathId",
			apply:    func() error { return h.photoSettingsCtl.SetDefaultPhotoImportLibraryPathID(target) },
			rollback: func() error { return h.photoSettingsCtl.SetDefaultPhotoImportLibraryPathID(prev) },
			failure: settingsPatchFailure{
				status:  http.StatusInternalServerError,
				code:    contracts.ErrorCodeInternal,
				message: fixedSettingsPatchMessage("failed to save photo library settings"),
			},
		})
	}

	if body.PhotoViewer != nil {
		if h.photoSettingsCtl == nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "photo library settings not available")
			return
		}
		prev := normalizePhotoViewerSettingsDTO(h.photoSettingsCtl.PhotoViewerSettings())
		target := normalizePhotoViewerSettingsDTO(*body.PhotoViewer)
		ops = append(ops, settingsPatchOperation{
			name:     "photoViewer",
			apply:    func() error { return h.photoSettingsCtl.SetPhotoViewerSettings(target) },
			rollback: func() error { return h.photoSettingsCtl.SetPhotoViewerSettings(prev) },
			failure: settingsPatchFailure{
				status:  http.StatusBadRequest,
				code:    contracts.ErrorCodeBadRequest,
				message: func(err error) string { return err.Error() },
			},
		})
	}

	if body.PhotoCache != nil {
		if h.photoSettingsCtl == nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "photo library settings not available")
			return
		}
		prev := normalizePhotoCacheSettingsDTO(h.photoSettingsCtl.PhotoCacheSettings())
		target := normalizePhotoCacheSettingsDTO(*body.PhotoCache)
		ops = append(ops, settingsPatchOperation{
			name:     "photoCache",
			apply:    func() error { return h.photoSettingsCtl.SetPhotoCacheSettings(target) },
			rollback: func() error { return h.photoSettingsCtl.SetPhotoCacheSettings(prev) },
			failure: settingsPatchFailure{
				status:  http.StatusBadRequest,
				code:    contracts.ErrorCodeBadRequest,
				message: func(err error) string { return err.Error() },
			},
		})
	}

	if body.MetadataMovieProvider != nil {
		if h.metadataScrapeCtl == nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "metadata scrape settings not available")
			return
		}
		name := strings.TrimSpace(*body.MetadataMovieProvider)
		if name != "" {
			allowed := h.metadataScrapeCtl.ListMetadataMovieProviders()
			if len(allowed) == 0 || !movieProviderNameAllowed(name, allowed) {
				writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "unknown metadataMovieProvider")
				return
			}
		}
		prev := h.metadataScrapeCtl.MetadataMovieProvider()
		target := name
		ops = append(ops, settingsPatchOperation{
			name:     "metadataMovieProvider",
			apply:    func() error { return h.metadataScrapeCtl.SetMetadataMovieProvider(target) },
			rollback: func() error { return h.metadataScrapeCtl.SetMetadataMovieProvider(prev) },
			failure: settingsPatchFailure{
				status:  http.StatusInternalServerError,
				code:    contracts.ErrorCodeInternal,
				message: fixedSettingsPatchMessage("failed to save library settings"),
			},
		})
	}

	if body.MetadataMovieProviderChain != nil {
		if h.metadataScrapeCtl == nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "metadata scrape settings not available")
			return
		}
		chain := *body.MetadataMovieProviderChain
		// Validate chain providers
		if len(chain) > 0 {
			allowed := h.metadataScrapeCtl.ListMetadataMovieProviders()
			for _, name := range chain {
				name = strings.TrimSpace(name)
				if name == "" {
					continue
				}
				if len(allowed) == 0 || !movieProviderNameAllowed(name, allowed) {
					writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "unknown provider in chain: "+name)
					return
				}
			}
		}
		prev := append([]string(nil), h.metadataScrapeCtl.MetadataMovieProviderChain()...)
		target := append([]string(nil), chain...)
		ops = append(ops, settingsPatchOperation{
			name:     "metadataMovieProviderChain",
			apply:    func() error { return h.metadataScrapeCtl.SetMetadataMovieProviderChain(target) },
			rollback: func() error { return h.metadataScrapeCtl.SetMetadataMovieProviderChain(prev) },
			failure: settingsPatchFailure{
				status:  http.StatusInternalServerError,
				code:    contracts.ErrorCodeInternal,
				message: fixedSettingsPatchMessage("failed to save library settings"),
			},
		})
	}

	if body.MetadataMovieScrapeMode != nil {
		if h.metadataScrapeCtl == nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "metadata scrape settings not available")
			return
		}
		mode := strings.TrimSpace(strings.ToLower(*body.MetadataMovieScrapeMode))
		if mode != "auto" && mode != "specified" && mode != "chain" {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid metadataMovieScrapeMode")
			return
		}
		prev := h.metadataScrapeCtl.MetadataMovieScrapeMode()
		target := mode
		ops = append(ops, settingsPatchOperation{
			name:     "metadataMovieScrapeMode",
			apply:    func() error { return h.metadataScrapeCtl.SetMetadataMovieScrapeMode(target) },
			rollback: func() error { return h.metadataScrapeCtl.SetMetadataMovieScrapeMode(prev) },
			failure: settingsPatchFailure{
				status:  http.StatusInternalServerError,
				code:    contracts.ErrorCodeInternal,
				message: fixedSettingsPatchMessage("failed to save library settings"),
			},
		})
	}

	if body.MetadataMovieStrategy != nil {
		if h.metadataScrapeCtl == nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "metadata scrape settings not available")
			return
		}
		strategy := strings.TrimSpace(strings.ToLower(*body.MetadataMovieStrategy))
		if strategy != "auto-global" && strategy != "auto-cn-friendly" && strategy != "custom-chain" && strategy != "specified" {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid metadataMovieStrategy")
			return
		}
		prev := h.metadataScrapeCtl.MetadataMovieStrategy()
		target := strategy
		ops = append(ops, settingsPatchOperation{
			name:     "metadataMovieStrategy",
			apply:    func() error { return h.metadataScrapeCtl.SetMetadataMovieStrategy(target) },
			rollback: func() error { return h.metadataScrapeCtl.SetMetadataMovieStrategy(prev) },
			failure: settingsPatchFailure{
				status:  http.StatusInternalServerError,
				code:    contracts.ErrorCodeInternal,
				message: fixedSettingsPatchMessage("failed to save library settings"),
			},
		})
	}

	if body.Proxy != nil {
		if h.proxyCtl == nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "proxy settings not available")
			return
		}
		cfg := config.ProxyConfig{
			Enabled:  body.Proxy.Enabled,
			URL:      strings.TrimSpace(body.Proxy.URL),
			Username: strings.TrimSpace(body.Proxy.Username),
			Password: body.Proxy.Password,
		}
		if cfg.Enabled && cfg.URL == "" {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "proxy URL is required when enabled")
			return
		}
		prev := h.proxyCtl.Proxy()
		target := cfg
		ops = append(ops, settingsPatchOperation{
			name:     "proxy",
			apply:    func() error { return h.proxyCtl.SetProxy(target) },
			rollback: func() error { return h.proxyCtl.SetProxy(prev) },
			failure: settingsPatchFailure{
				status:  http.StatusInternalServerError,
				code:    contracts.ErrorCodeInternal,
				message: fixedSettingsPatchMessage("failed to save proxy settings"),
			},
		})
	}

	if body.AIProvider != nil {
		if h.aiSettingsCtl == nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "ai provider settings not available")
			return
		}
		prev := h.aiSettingsCtl.AIProviderSettings()
		target := *body.AIProvider
		ops = append(ops, settingsPatchOperation{
			name:  "aiProvider",
			apply: func() error { return h.aiSettingsCtl.SetAIProviderSettingsPatch(target) },
			rollback: func() error {
				return h.aiSettingsCtl.SetAIProviderSettingsPatch(contracts.PatchAIProviderSettings{
					Kind:    &prev.Kind,
					BaseURL: &prev.BaseURL,
					APIKey:  &prev.APIKey,
					Model:   &prev.Model,
				})
			},
			failure: settingsPatchFailure{
				status:  http.StatusBadRequest,
				code:    contracts.ErrorCodeBadRequest,
				message: func(err error) string { return err.Error() },
			},
		})
	}

	if body.BackendLog != nil && patchBackendLogHasChanges(body.BackendLog) {
		if h.backendLogCtl == nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "backend log settings not available")
			return
		}
		prev := h.backendLogCtl.BackendLogSettings()
		target := *body.BackendLog
		ops = append(ops, settingsPatchOperation{
			name:     "backendLog",
			apply:    func() error { return h.backendLogCtl.SetBackendLogPatch(target) },
			rollback: func() error { return h.backendLogCtl.SetBackendLogPatch(backendLogPatchFromDTO(prev)) },
			failure: settingsPatchFailure{
				status:  http.StatusBadRequest,
				code:    contracts.ErrorCodeBadRequest,
				message: func(err error) string { return err.Error() },
			},
		})
	}

	if body.Player != nil {
		if h.playerSettingsCtl == nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "player settings not available")
			return
		}
		prev := h.playerSettingsCtl.PlayerSettings()
		target := *body.Player
		ops = append(ops, settingsPatchOperation{
			name:     "player",
			apply:    func() error { return h.playerSettingsCtl.SetPlayerSettingsPatch(target) },
			rollback: func() error { return h.playerSettingsCtl.SetPlayerSettingsPatch(playerSettingsPatchFromDTO(prev)) },
			failure: settingsPatchFailure{
				status:  http.StatusBadRequest,
				code:    contracts.ErrorCodeBadRequest,
				message: func(err error) string { return err.Error() },
			},
		})
	}

	for i, op := range ops {
		if err := op.apply(); err != nil {
			if h.logger != nil {
				h.logger.Warn("settings patch apply failed", zap.String("field", op.name), zap.Error(err))
			}
			if rollbackErr := rollbackSettingsPatchOperations(ops, i); rollbackErr != nil {
				if h.logger != nil {
					h.logger.Error("settings patch rollback failed", zap.Error(rollbackErr), zap.String("field", op.name))
				}
				writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to roll back settings after patch error")
				return
			}
			writeAppError(w, op.failure.status, op.failure.code, op.failure.userMessage(err))
			return
		}
	}

	dto, err := h.buildSettingsDTO(r.Context())
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("build settings dto failed after patch", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to list library paths")
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) handleImportMovies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.store == nil || h.tasks == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "movie import runtime not available")
		return
	}

	targetID := h.defaultImportLibraryPathID()
	if targetID == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeImportTargetNotConfigured, "default import library path is not configured")
		return
	}
	target, err := h.store.GetLibraryPath(r.Context(), targetID)
	if err != nil {
		if errors.Is(err, storage.ErrLibraryPathNotFound) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeImportTargetUnavailable, "default import library path was not found")
			return
		}
		if h.logger != nil {
			h.logger.Warn("load default import library path failed", zap.Error(err), zap.String("libraryPathId", targetID))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load default import library path")
		return
	}
	targetRoot := filepath.Clean(strings.TrimSpace(target.Path))
	if targetRoot == "" || targetRoot == "." {
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodeImportTargetUnavailable, "default import library path is invalid")
		return
	}
	if stat, err := os.Stat(targetRoot); err != nil || !stat.IsDir() {
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodeImportTargetUnavailable, "default import library path is unavailable")
		return
	}

	reader, err := r.MultipartReader()
	if err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "multipart form data is required")
		return
	}

	task := h.tasks.Create(contracts.TaskTypeImportMovies, map[string]any{
		"targetLibraryPathId": target.ID,
		"targetPath":          targetRoot,
		"stage":               "copying",
		"completedFiles":      0,
		"failedFiles":         0,
	})
	task = h.tasks.Start(task.TaskID, "Importing movies")
	h.saveTaskSnapshot(r.Context(), task)
	snapshotThrottle := newImportTaskSnapshotThrottle(time.Now(), importTaskSnapshotMinInterval, importTaskSnapshotMinBytes)

	var (
		totalFiles      int
		completedFiles  int
		failedFiles     int
		copiedBytes     int64
		declaredBytes   int64
		pendingRelPath  string
		errorItems      []map[string]any
		copiedAny       bool
		lastCurrentName string
	)

	progressPatch := func(extra map[string]any) map[string]any {
		patch := map[string]any{
			"targetLibraryPathId": target.ID,
			"targetPath":          targetRoot,
			"stage":               "copying",
			"totalFiles":          totalFiles,
			"completedFiles":      completedFiles,
			"failedFiles":         failedFiles,
			"copiedBytes":         copiedBytes,
			"totalBytes":          declaredBytes,
			"currentFileName":     lastCurrentName,
			"errorItems":          errorItems,
		}
		for k, v := range extra {
			patch[k] = v
		}
		return patch
	}

	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			task = h.tasks.ProgressWithMetadata(task.TaskID, task.Progress, "Failed to read import payload", progressPatch(nil))
			h.saveTaskSnapshot(r.Context(), task)
			task = h.tasks.Fail(task.TaskID, contracts.ErrorCodeImportSourceUnavailable, "failed to read import payload")
			h.saveTaskSnapshot(r.Context(), task)
			writeJSON(w, http.StatusAccepted, task)
			return
		}

		formName := part.FormName()
		fileName := strings.TrimSpace(part.FileName())
		if fileName == "" {
			value, _ := io.ReadAll(io.LimitReader(part, 64*1024))
			switch formName {
			case "relativePath":
				pendingRelPath = strings.TrimSpace(string(value))
			case "totalBytes":
				if n, err := strconv.ParseInt(strings.TrimSpace(string(value)), 10, 64); err == nil && n > 0 {
					declaredBytes = n
					if totalFiles == 0 {
						if err := ensureImportDiskSpace(targetRoot, declaredBytes); err != nil {
							task = h.tasks.Fail(task.TaskID, contracts.ErrorCodeImportNotEnoughSpace, "not enough disk space on the default import path")
							h.saveTaskSnapshot(r.Context(), task)
							writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeImportNotEnoughSpace, "not enough disk space on the default import path")
							return
						}
					}
				}
			}
			_ = part.Close()
			continue
		}

		totalFiles++
		lastCurrentName = fileName
		relPath := sanitizeImportRelativePath(pendingRelPath, fileName)
		pendingRelPath = ""
		if !isSupportedImportVideoPath(relPath) {
			failedFiles++
			errorItems = append(errorItems, importErrorItem(fileName, contracts.ErrorCodeImportSourceUnavailable, "unsupported video file type"))
			_, _ = io.Copy(io.Discard, part)
			_ = part.Close()
			task = h.tasks.ProgressWithMetadata(task.TaskID, importProgressPercent(copiedBytes, declaredBytes), "Skipped unsupported file "+fileName, progressPatch(nil))
			h.saveTaskSnapshot(r.Context(), task)
			continue
		}

		destPath, err := importDestinationPath(targetRoot, relPath)
		if err != nil {
			failedFiles++
			errorItems = append(errorItems, importErrorItem(fileName, contracts.ErrorCodeImportSourceUnavailable, "invalid destination path"))
			_, _ = io.Copy(io.Discard, part)
			_ = part.Close()
			task = h.tasks.ProgressWithMetadata(task.TaskID, importProgressPercent(copiedBytes, declaredBytes), "Skipped invalid path "+fileName, progressPatch(nil))
			h.saveTaskSnapshot(r.Context(), task)
			continue
		}

		if _, err := os.Stat(destPath); err == nil {
			failedFiles++
			errorItems = append(errorItems, importErrorItem(fileName, contracts.ErrorCodeImportConflict, "target file already exists"))
			_, _ = io.Copy(io.Discard, part)
			_ = part.Close()
			task = h.tasks.ProgressWithMetadata(task.TaskID, importProgressPercent(copiedBytes, declaredBytes), "Skipped existing file "+fileName, progressPatch(nil))
			h.saveTaskSnapshot(r.Context(), task)
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			failedFiles++
			code := classifyImportCopyError(err)
			errorItems = append(errorItems, importErrorItem(fileName, code, err.Error()))
			_, _ = io.Copy(io.Discard, part)
			_ = part.Close()
			task = h.tasks.ProgressWithMetadata(task.TaskID, importProgressPercent(copiedBytes, declaredBytes), "Failed to prepare "+fileName, progressPatch(nil))
			h.saveTaskSnapshot(r.Context(), task)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			failedFiles++
			code := classifyImportCopyError(err)
			errorItems = append(errorItems, importErrorItem(fileName, code, err.Error()))
			_, _ = io.Copy(io.Discard, part)
			_ = part.Close()
			task = h.tasks.ProgressWithMetadata(task.TaskID, importProgressPercent(copiedBytes, declaredBytes), "Failed to prepare "+fileName, progressPatch(nil))
			h.saveTaskSnapshot(r.Context(), task)
			continue
		}

		tempPath := destPath + "." + task.TaskID + ".tmp"
		n, err := copyImportPartToFile(tempPath, part, func(delta int64) {
			copiedBytes += delta
			task = h.tasks.ProgressWithMetadata(task.TaskID, importProgressPercent(copiedBytes, declaredBytes), "Copying "+fileName, progressPatch(nil))
			if snapshotThrottle.ShouldSave(time.Now(), copiedBytes, false) {
				h.saveTaskSnapshot(r.Context(), task)
			}
		})
		_ = part.Close()
		if err != nil {
			_ = os.Remove(tempPath)
			failedFiles++
			code := classifyImportCopyError(err)
			errorItems = append(errorItems, importErrorItem(fileName, code, err.Error()))
			task = h.tasks.ProgressWithMetadata(task.TaskID, importProgressPercent(copiedBytes, declaredBytes), "Failed to copy "+fileName, progressPatch(nil))
			h.saveTaskSnapshot(r.Context(), task)
			continue
		}
		if err := os.Rename(tempPath, destPath); err != nil {
			_ = os.Remove(tempPath)
			copiedBytes -= n
			if copiedBytes < 0 {
				copiedBytes = 0
			}
			failedFiles++
			code := classifyImportCopyError(err)
			errorItems = append(errorItems, importErrorItem(fileName, code, err.Error()))
			task = h.tasks.ProgressWithMetadata(task.TaskID, importProgressPercent(copiedBytes, declaredBytes), "Failed to finish "+fileName, progressPatch(nil))
			h.saveTaskSnapshot(r.Context(), task)
			continue
		}
		completedFiles++
		copiedAny = true
		task = h.tasks.ProgressWithMetadata(task.TaskID, importProgressPercent(copiedBytes, declaredBytes), "Copied "+fileName, progressPatch(nil))
		h.saveTaskSnapshot(r.Context(), task)
	}

	finalPatch := progressPatch(map[string]any{"stage": "completed"})
	if totalFiles == 0 {
		finalPatch["stage"] = "failed"
		task = h.tasks.ProgressWithMetadata(task.TaskID, 100, "No movie files were provided", finalPatch)
		h.saveTaskSnapshot(r.Context(), task)
		task = h.tasks.Fail(task.TaskID, contracts.ErrorCodeBadRequest, "no movie files were provided")
		h.saveTaskSnapshot(r.Context(), task)
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "no movie files were provided")
		return
	}

	var scanErr error
	if copiedAny && h.scanStarter != nil {
		var scanTask contracts.TaskDTO
		scanTask, scanErr = h.scanStarter.StartScan(r.Context(), []string{targetRoot})
		if scanErr == nil && scanTask.TaskID != "" {
			finalPatch["scanTaskId"] = scanTask.TaskID
		}
	}
	if scanErr != nil {
		finalPatch["scanError"] = scanErr.Error()
		if failedFiles == 0 {
			finalPatch["stage"] = "partial_failed"
			task = h.tasks.ProgressWithMetadata(task.TaskID, 100, "Movies copied, but scan could not start", finalPatch)
			h.saveTaskSnapshot(r.Context(), task)
			task = h.tasks.PartialFail(task.TaskID, contracts.ErrorCodeImportScanFailed, "movies copied, but scan could not start", finalPatch)
			h.saveTaskSnapshot(r.Context(), task)
			writeJSON(w, http.StatusAccepted, task)
			return
		}
	}

	if failedFiles > 0 {
		finalPatch["stage"] = "partial_failed"
		if completedFiles == 0 {
			finalPatch["stage"] = "failed"
			task = h.tasks.ProgressWithMetadata(task.TaskID, 100, "Movie import failed", finalPatch)
			h.saveTaskSnapshot(r.Context(), task)
			task = h.tasks.Fail(task.TaskID, contracts.ErrorCodeImportCopyFailed, "movie import failed")
			h.saveTaskSnapshot(r.Context(), task)
			writeJSON(w, http.StatusAccepted, task)
			return
		}
		task = h.tasks.PartialFail(task.TaskID, contracts.ErrorCodeImportCopyFailed, "movie import partially completed", finalPatch)
		h.saveTaskSnapshot(r.Context(), task)
		writeJSON(w, http.StatusAccepted, task)
		return
	}

	task = h.tasks.ProgressWithMetadata(task.TaskID, 100, "Movie import completed", finalPatch)
	h.saveTaskSnapshot(r.Context(), task)
	task = h.tasks.Complete(task.TaskID, "Movie import completed")
	h.saveTaskSnapshot(r.Context(), task)
	writeJSON(w, http.StatusAccepted, task)
}

func (h *Handler) defaultImportLibraryPathID() string {
	if h.defaultImportLibraryPathCtl != nil {
		return strings.TrimSpace(h.defaultImportLibraryPathCtl.DefaultImportLibraryPathID())
	}
	return strings.TrimSpace(h.cfg.DefaultImportLibraryPathID)
}

func (h *Handler) saveTaskSnapshot(ctx context.Context, task contracts.TaskDTO) {
	if h.store == nil {
		return
	}
	if err := h.store.SaveTask(ctx, task); err != nil && h.logger != nil {
		h.logger.Warn("save task snapshot failed", zap.Error(err), zap.String("taskId", task.TaskID))
	}
}

type importTaskSnapshotThrottle struct {
	lastSavedAt    time.Time
	lastSavedBytes int64
	minInterval    time.Duration
	minBytes       int64
}

func newImportTaskSnapshotThrottle(start time.Time, minInterval time.Duration, minBytes int64) *importTaskSnapshotThrottle {
	return &importTaskSnapshotThrottle{
		lastSavedAt: start,
		minInterval: minInterval,
		minBytes:    minBytes,
	}
}

func (t *importTaskSnapshotThrottle) ShouldSave(now time.Time, copiedBytes int64, force bool) bool {
	if force {
		t.markSaved(now, copiedBytes)
		return true
	}
	if t.minBytes > 0 && copiedBytes-t.lastSavedBytes >= t.minBytes {
		t.markSaved(now, copiedBytes)
		return true
	}
	if t.minInterval > 0 && now.Sub(t.lastSavedAt) >= t.minInterval {
		t.markSaved(now, copiedBytes)
		return true
	}
	return false
}

func (t *importTaskSnapshotThrottle) markSaved(now time.Time, copiedBytes int64) {
	t.lastSavedAt = now
	t.lastSavedBytes = copiedBytes
}

func mergeTaskMetadata(current map[string]any, patch map[string]any) map[string]any {
	if current == nil && patch == nil {
		return nil
	}
	out := make(map[string]any, len(current)+len(patch))
	for k, v := range current {
		out[k] = v
	}
	for k, v := range patch {
		out[k] = v
	}
	return out
}

func importErrorItem(fileName, code, message string) map[string]any {
	return map[string]any{
		"fileName": fileName,
		"code":     code,
		"message":  message,
	}
}

func sanitizeImportRelativePath(relativePath string, fallbackFileName string) string {
	raw := strings.TrimSpace(relativePath)
	if raw == "" {
		raw = strings.TrimSpace(fallbackFileName)
	}
	raw = strings.ReplaceAll(raw, "\\", "/")
	parts := strings.Split(raw, "/")
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		part = sanitizeImportPathSegment(part)
		if part == "" || part == "." || part == ".." {
			continue
		}
		clean = append(clean, part)
	}
	if len(clean) == 0 {
		name := sanitizeImportPathSegment(filepath.Base(strings.TrimSpace(fallbackFileName)))
		if name == "" {
			name = "movie"
		}
		clean = append(clean, name)
	}
	return filepath.Join(clean...)
}

func sanitizeImportPathSegment(segment string) string {
	segment = strings.TrimSpace(segment)
	segment = filepath.Base(segment)
	segment = strings.TrimSpace(segment)
	if segment == "" || segment == "." || segment == ".." {
		return ""
	}
	replacer := strings.NewReplacer(
		"<", "_",
		">", "_",
		":", "_",
		`"`, "_",
		"|", "_",
		"?", "_",
		"*", "_",
	)
	segment = replacer.Replace(segment)
	segment = strings.Map(func(r rune) rune {
		if r < 32 {
			return '_'
		}
		return r
	}, segment)
	return strings.Trim(segment, " .")
}

func importDestinationPath(root string, relativePath string) (string, error) {
	root = filepath.Clean(root)
	dest := filepath.Clean(filepath.Join(root, relativePath))
	rel, err := filepath.Rel(root, dest)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("destination escapes import root")
	}
	return dest, nil
}

func isSupportedImportVideoPath(path string) bool {
	return contracts.IsSupportedVideoExtension(filepath.Ext(strings.TrimSpace(path)))
}

// importDiskSpaceMarginBytes keeps a safety buffer on top of the declared import
// payload so a copy does not fill the target volume to the last byte.
const importDiskSpaceMarginBytes = 512 * 1024 * 1024

var errImportNotEnoughSpace = errors.New("target volume does not have enough free space for the import payload")

// ensureImportDiskSpace returns errImportNotEnoughSpace when targetRoot cannot
// hold requiredBytes plus the safety margin. The probe is advisory: on lookup
// failure the import proceeds and still relies on write-time error classification.
func ensureImportDiskSpace(targetRoot string, requiredBytes int64) error {
	if requiredBytes <= 0 {
		return nil
	}
	available, err := diskutil.AvailableDiskBytes(targetRoot)
	if err != nil {
		return nil
	}
	need := uint64(requiredBytes)
	if need > available || available-need < importDiskSpaceMarginBytes {
		return errImportNotEnoughSpace
	}
	return nil
}

func copyImportPartToFile(tempPath string, src io.Reader, onProgress func(delta int64)) (int64, error) {
	dst, err := os.OpenFile(tempPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, err
	}
	defer func() { _ = dst.Close() }()

	buf := make([]byte, importCopyBufferSize)
	var copied int64
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[:nr])
			if nw > 0 {
				copied += int64(nw)
				if onProgress != nil {
					onProgress(int64(nw))
				}
			}
			if ew != nil {
				return copied, ew
			}
			if nw != nr {
				return copied, io.ErrShortWrite
			}
		}
		if er != nil {
			if errors.Is(er, io.EOF) {
				break
			}
			return copied, er
		}
	}
	if err := dst.Sync(); err != nil {
		return copied, err
	}
	return copied, nil
}

func importProgressPercent(copiedBytes int64, declaredTotalBytes int64) int {
	if declaredTotalBytes <= 0 {
		return 0
	}
	p := int((float64(copiedBytes) / float64(declaredTotalBytes)) * 100)
	if p < 0 {
		return 0
	}
	if p > 99 {
		return 99
	}
	return p
}

func classifyImportCopyError(err error) string {
	if err == nil {
		return contracts.ErrorCodeImportCopyFailed
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "no space left") || strings.Contains(msg, "not enough space") || strings.Contains(msg, "disk full") {
		return contracts.ErrorCodeImportNotEnoughSpace
	}
	return contracts.ErrorCodeImportCopyFailed
}

func (h *Handler) handleAddLibraryPath(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}

	var body contracts.AddLibraryPathRequest
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid json body")
			return
		}
	}

	path := strings.TrimSpace(body.Path)
	if path == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "path is required")
		return
	}

	dto, err := h.store.AddLibraryPath(r.Context(), path, strings.TrimSpace(body.Title))
	if err != nil {
		if errors.Is(err, storage.ErrLibraryPathNotAbsolute) {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "library path must be an absolute path")
			return
		}
		if errors.Is(err, storage.ErrLibraryPathDuplicate) {
			writeAppError(w, http.StatusConflict, contracts.ErrorCodeConflict, "library path already exists")
			return
		}
		h.logger.Error("add library path failed", zap.Error(err))
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to add library path")
		return
	}

	h.reloadLibraryWatchIfAny(r.Context())

	resp := contracts.AddLibraryPathResponse{LibraryPathDTO: dto}
	if h.scanStarter != nil {
		if task, err := h.scanStarter.StartScan(r.Context(), []string{dto.Path}); err != nil {
			if errors.Is(err, contracts.ErrScanAlreadyRunning) {
				if h.logger != nil {
					h.logger.Warn("add library path: initial scan skipped (scan already in progress)", zap.String("path", dto.Path))
				}
			} else if h.logger != nil {
				h.logger.Warn("add library path: failed to start initial scan", zap.Error(err), zap.String("path", dto.Path))
			}
		} else {
			resp.ScanTask = &task
		}
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (h *Handler) handleGetLibraryPathStorageStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.libraryPathStorageStatus == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "library path storage status is not available")
		return
	}
	dto, err := h.libraryPathStorageStatus.ListLibraryPathStorageStatus(r.Context())
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to read library path storage status")
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) handleCheckLibraryPathStorageStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.libraryPathStorageStatus == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "library path storage status is not available")
		return
	}

	var body contracts.CheckLibraryPathStorageStatusRequest
	if r.Body != nil && r.Body != http.NoBody {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid storage status check request")
			return
		}
	}

	dto, err := h.libraryPathStorageStatus.CheckLibraryPathStorageStatus(r.Context(), body.LibraryPathIDs)
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to check library path storage status")
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) handleRebindLibraryPathStorage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.libraryPathStorageStatus == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "library path storage status is not available")
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "library path id is required")
		return
	}
	dto, err := h.libraryPathStorageStatus.RebindLibraryPathStorage(r.Context(), id)
	if err != nil {
		if errors.Is(err, storage.ErrLibraryPathNotFound) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "library path not found")
			return
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to rebind library path storage")
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) handleDeleteLibraryPath(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}

	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "id is required")
		return
	}

	pruned, err := h.store.DeleteLibraryPathAndPruneOrphanMovies(r.Context(), id)
	if err != nil {
		if errors.Is(err, storage.ErrLibraryPathNotFound) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "library path not found")
			return
		}
		h.logger.Error("delete library path failed", zap.Error(err))
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to delete library path")
		return
	}
	if pruned > 0 {
		h.logger.Info("library path removed; pruned orphan movies", zap.String("id", id), zap.Int("prunedMovies", pruned))
	}

	h.reloadLibraryWatchIfAny(r.Context())
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handlePatchLibraryPath(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}

	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "id is required")
		return
	}

	var body contracts.UpdateLibraryPathRequest
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid json body")
			return
		}
	}

	dto, err := h.store.UpdateLibraryPathTitle(r.Context(), id, body.Title)
	if err != nil {
		if errors.Is(err, storage.ErrLibraryPathNotFound) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "library path not found")
			return
		}
		h.logger.Error("update library path title failed", zap.Error(err))
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to update library path")
		return
	}

	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) handleStartScan(w http.ResponseWriter, r *http.Request) {
	var request contracts.StartScanRequest
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid json body")
			return
		}
	}

	task, err := h.scanStarter.StartScan(r.Context(), request.Paths)
	if err != nil {
		if errors.Is(err, contracts.ErrScanAlreadyRunning) {
			writeAppError(w, http.StatusConflict, contracts.ErrorCodeConflict, "scan already in progress")
			return
		}
		if h.logger != nil {
			h.logger.Warn("start scan failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to start scan")
		return
	}

	writeJSON(w, http.StatusAccepted, task)
}

func (h *Handler) handleGetTaskStatus(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskId")
	if taskID == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "taskId is required")
		return
	}

	task, ok := h.tasks.Get(taskID)
	if !ok {
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "task not found")
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (h *Handler) handleGetRecentTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	limit := 30
	if q := strings.TrimSpace(r.URL.Query().Get("limit")); q != "" {
		if n, err := strconv.Atoi(q); err == nil && n > 0 {
			limit = n
		}
	}
	tasks := h.tasks.ListRecentFinished(limit)
	writeJSON(w, http.StatusOK, contracts.RecentTasksDTO{Tasks: tasks})
}

func (h *Handler) reloadLibraryWatchIfAny(ctx context.Context) {
	if h.libraryWatchReloader == nil {
		return
	}
	if err := h.libraryWatchReloader.ReloadLibraryWatches(ctx); err != nil && h.logger != nil {
		h.logger.Warn("library watch reload failed", zap.Error(err))
	}
}

func (h *Handler) reloadComicLibraryWatchIfAny(ctx context.Context) {
	if h.comicLibraryWatchReloader == nil {
		return
	}
	if err := h.comicLibraryWatchReloader.ReloadComicLibraryWatches(ctx); err != nil && h.logger != nil {
		h.logger.Warn("comic library watch reload failed", zap.Error(err))
	}
}

func (h *Handler) reloadPhotoLibraryWatchIfAny(ctx context.Context) {
	if h.photoLibraryWatchReloader == nil {
		return
	}
	if err := h.photoLibraryWatchReloader.ReloadPhotoLibraryWatches(ctx); err != nil && h.logger != nil {
		h.logger.Warn("photo library watch reload failed", zap.Error(err))
	}
}

func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

// ListenAndServe starts an HTTP server on addr and gracefully shuts down when ctx is cancelled.
func ListenAndServe(ctx context.Context, addr string, handler http.Handler, logger *zap.Logger) error {
	return ListenAndServeWithReady(ctx, addr, handler, logger, nil)
}

// ListenAndServeWithReady starts an HTTP server, calls onReady after the TCP
// listener is bound, and gracefully shuts down when ctx is cancelled.
func ListenAndServeWithReady(ctx context.Context, addr string, handler http.Handler, logger *zap.Logger, onReady func(addr string)) error {
	srv := newHTTPServer(addr, handler)
	listener, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return err
	}
	boundAddr := listener.Addr().String()

	go func() {
		<-ctx.Done()
		if logger != nil {
			logger.Info("HTTP server shutdown initiated")
		}
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil && logger != nil {
			logger.Warn("HTTP server shutdown error", zap.Error(err))
		} else if logger != nil {
			logger.Info("HTTP server stopped")
		}
	}()

	if logger != nil {
		logger.Info("HTTP server listening", zap.String("addr", boundAddr), zap.String("configuredAddr", addr))
	}
	if onReady != nil {
		onReady(boundAddr)
	}
	if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeAppError(w http.ResponseWriter, status int, code string, message string) {
	writeAppErrorWithDetails(w, status, code, message, nil)
}

func writeAppErrorWithDetails(w http.ResponseWriter, status int, code string, message string, details map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(contracts.AppError{
		Code:      code,
		Message:   message,
		Retryable: status >= 500,
		Details:   details,
	})
}

func (h *Handler) handlePingProvider(w http.ResponseWriter, r *http.Request) {
	var req contracts.PingProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("invalid ping provider request body", zap.Error(err))
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "Invalid request body")
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "Provider name is required")
		return
	}

	// Verify provider exists
	found := false
	for _, p := range h.providerHealthChecker.ListProviders() {
		if strings.EqualFold(p, name) {
			found = true
			name = p // Use the canonical name from the list
			break
		}
	}
	if !found {
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodeProviderNotFound, "Provider not found: "+name)
		return
	}

	status, latencyMs, err := h.providerHealthChecker.CheckProviderHealth(r.Context(), name)
	if err != nil {
		h.logger.Warn("provider health check failed", zap.String("provider", name), zap.Error(err))
	}

	dto := contracts.ProviderHealthDTO{
		Name:      name,
		Status:    contracts.ProviderHealthStatus(status),
		LatencyMs: latencyMs,
	}
	if err != nil {
		dto.Message = err.Error()
		dto.ErrorCategory = classifyProviderHealthError(err)
	}
	h.enrichProviderHealthRuntime(&dto)

	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) handlePingAllProviders(w http.ResponseWriter, r *http.Request) {
	providers := h.providerHealthChecker.ListProviders()
	results := make([]contracts.ProviderHealthDTO, 0, len(providers))

	okCount := 0
	failCount := 0

	for _, name := range providers {
		status, latencyMs, err := h.providerHealthChecker.CheckProviderHealth(r.Context(), name)
		dto := contracts.ProviderHealthDTO{
			Name:      name,
			Status:    contracts.ProviderHealthStatus(status),
			LatencyMs: latencyMs,
		}
		if err != nil {
			dto.Message = err.Error()
			dto.ErrorCategory = classifyProviderHealthError(err)
			failCount++
		} else if status == "ok" {
			okCount++
		}
		h.enrichProviderHealthRuntime(&dto)
		results = append(results, dto)
	}

	writeJSON(w, http.StatusOK, contracts.PingAllProvidersResponse{
		Providers: results,
		Total:     len(providers),
		OK:        okCount,
		Fail:      failCount,
	})
}

func (h *Handler) enrichProviderHealthRuntime(dto *contracts.ProviderHealthDTO) {
	if dto == nil {
		return
	}
	checker, ok := h.providerHealthChecker.(*metatube.Service)
	if !ok {
		return
	}
	snap := checker.ProviderRuntimeHealth(dto.Name)
	if dto.ErrorCategory == "" {
		dto.ErrorCategory = snap.ErrorCategory
	}
	dto.CooldownUntil = snap.CooldownUntil
	dto.ConsecutiveFailures = snap.ConsecutiveFailures
	dto.AvgLatencyMs = snap.AvgLatencyMs
}

func classifyProviderHealthError(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "no such host"), strings.Contains(msg, "lookup "):
		return "dns_failure"
	case strings.Contains(msg, "timeout"):
		return "connect_timeout"
	case strings.Contains(msg, "tls"):
		return "tls_failure"
	case strings.Contains(msg, "restricted"), strings.Contains(msg, "geo"), strings.Contains(msg, "region"):
		return "region_restricted"
	case strings.Contains(msg, "forbidden"), strings.Contains(msg, "referer"), strings.Contains(msg, "hotlink"):
		return "hotlink_denied"
	case strings.Contains(msg, "no results"):
		return "provider_empty_result"
	default:
		return "provider_invalid_content"
	}
}

func (h *Handler) handleProxyPingJavbus(w http.ResponseWriter, r *http.Request) {
	h.handleProxyPingURL(w, r, "https://www.javbus.com/")
}

func (h *Handler) handleProxyPingGoogle(w http.ResponseWriter, r *http.Request) {
	h.handleProxyPingURL(w, r, "https://www.google.com/")
}

// handleProxyPingURL GETs targetURL using optional body.proxy or persisted proxy (same contract as ping-javbus).
func (h *Handler) handleProxyPingURL(w http.ResponseWriter, r *http.Request, targetURL string) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "Method not allowed")
		return
	}

	var req contracts.ProxyJavBusPingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		h.logger.Warn("invalid proxy outbound ping body", zap.Error(err))
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "Invalid request body")
		return
	}

	var cfg config.ProxyConfig
	if req.Proxy != nil {
		cfg = config.ProxyConfig{
			Enabled:  req.Proxy.Enabled,
			URL:      strings.TrimSpace(req.Proxy.URL),
			Username: strings.TrimSpace(req.Proxy.Username),
			Password: req.Proxy.Password,
		}
	} else {
		if h.proxyCtl == nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "proxy settings not available")
			return
		}
		cfg = h.proxyCtl.Proxy()
	}

	if cfg.Enabled && strings.TrimSpace(cfg.URL) == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "Proxy URL is required when proxy is enabled")
		return
	}

	// 与设置页「测试 JavBus / Google 连通」一致：超过 5s 即视为超时失败
	const proxyOutboundPingTimeout = 5 * time.Second
	client, err := proxyenv.NewHTTPClientForProxy(cfg, proxyOutboundPingTimeout)
	if err != nil {
		writeJSON(w, http.StatusOK, contracts.ProxyJavBusPingResponse{
			OK:      false,
			Message: err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), proxyOutboundPingTimeout)
	defer cancel()

	start := time.Now()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		writeJSON(w, http.StatusOK, contracts.ProxyJavBusPingResponse{OK: false, Message: err.Error()})
		return
	}
	httpReq.Header.Set("User-Agent", browserheaders.UserAgentChrome120)
	httpReq.Header.Set("Accept", browserheaders.AcceptLikeChrome)

	resp, err := client.Do(httpReq)
	latencyMs := time.Since(start).Milliseconds()
	if err != nil {
		writeJSON(w, http.StatusOK, contracts.ProxyJavBusPingResponse{
			OK:        false,
			LatencyMs: latencyMs,
			Message:   err.Error(),
		})
		return
	}
	defer resp.Body.Close()
	_, _ = io.CopyN(io.Discard, resp.Body, 64*1024)

	ok := resp.StatusCode < 500
	msg := ""
	if !ok {
		msg = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	writeJSON(w, http.StatusOK, contracts.ProxyJavBusPingResponse{
		OK:         ok,
		LatencyMs:  latencyMs,
		HTTPStatus: resp.StatusCode,
		Message:    msg,
	})
}

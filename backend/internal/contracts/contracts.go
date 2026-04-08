package contracts

import (
	"encoding/json"
	"errors"
	"strings"
)

// ErrScanAlreadyRunning is returned when a library scan is requested while another is in progress.
var ErrScanAlreadyRunning = errors.New("scan already running")

// ErrScrapeMovieNotFound is returned when a single-movie rescrape cannot find the movie in SQLite.
var ErrScrapeMovieNotFound = errors.New("movie not found")

// ErrScrapeMovieNoCode is returned when the movie row has an empty catalog code (番号).
var ErrScrapeMovieNoCode = errors.New("movie has no catalog code")

// ErrScrapeMovieNoLocation is returned when the movie row has an empty video path.
var ErrScrapeMovieNoLocation = errors.New("movie has no video path")

// ErrActorNotFound is returned when no actors row exists for the given display name.
var ErrActorNotFound = errors.New("actor not found")

type Command struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type Response struct {
	Kind      string    `json:"kind"`
	ID        string    `json:"id,omitempty"`
	OK        bool      `json:"ok"`
	Data      any       `json:"data,omitempty"`
	Error     *AppError `json:"error,omitempty"`
	Timestamp string    `json:"timestamp"`
}

type Event struct {
	Kind      string `json:"kind"`
	Type      string `json:"type"`
	Payload   any    `json:"payload,omitempty"`
	Timestamp string `json:"timestamp"`
}

type AppError struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	Retryable bool           `json:"retryable"`
	Details   map[string]any `json:"details,omitempty"`
}

type HealthDTO struct {
	Name         string `json:"name"`
	Version      string `json:"version"` // Build stamp YYYYMMDD.HHMMSS (UTC) or git.* / unknown; see Channel
	Channel      string `json:"channel"` // "dev" or "release" (-tags release)
	Transport    string `json:"transport"`
	DatabasePath string `json:"databasePath"`
}

type DevPerformanceSummaryDTO struct {
	Supported         bool    `json:"supported"`
	SampledAt         string  `json:"sampledAt,omitempty"`
	SystemCPUPercent  float64 `json:"systemCpuPercent,omitempty"`
	BackendCPUPercent float64 `json:"backendCpuPercent,omitempty"`
}

type ListMoviesRequest struct {
	Mode   string `json:"mode,omitempty"`
	Query  string `json:"query,omitempty"`
	Actor  string `json:"actor,omitempty"`
	Studio string `json:"studio,omitempty"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
}

// ActorProfileDTO is returned by GET /api/library/actors/profile.
type ActorProfileDTO struct {
	Name             string   `json:"name"`
	AvatarURL        string   `json:"avatarUrl,omitempty"`
	AvatarRemoteURL  string   `json:"avatarRemoteUrl,omitempty"`
	AvatarLocalURL   string   `json:"avatarLocalUrl,omitempty"`
	HasLocalAvatar   bool     `json:"hasLocalAvatar,omitempty"`
	Summary          string   `json:"summary,omitempty"`
	Homepage         string   `json:"homepage,omitempty"`
	Provider         string   `json:"provider,omitempty"`
	ProviderActorID  string   `json:"providerActorId,omitempty"`
	Height           int      `json:"height,omitempty"`
	Birthday         string   `json:"birthday,omitempty"`
	ProfileUpdatedAt string   `json:"profileUpdatedAt,omitempty"`
	UserTags         []string `json:"userTags,omitempty"`
}

// ActorListItemDTO is one row in GET /api/library/actors (library display name + stats + actor-only user tags).
type ActorListItemDTO struct {
	Name            string   `json:"name"`
	AvatarURL       string   `json:"avatarUrl,omitempty"`
	AvatarRemoteURL string   `json:"avatarRemoteUrl,omitempty"`
	AvatarLocalURL  string   `json:"avatarLocalUrl,omitempty"`
	HasLocalAvatar  bool     `json:"hasLocalAvatar,omitempty"`
	MovieCount      int      `json:"movieCount"`
	UserTags        []string `json:"userTags,omitempty"`
}

// ListActorsRequest is the query for GET /api/library/actors.
type ListActorsRequest struct {
	Q        string // substring match on actors.name or actor_user_tags.tag (case-insensitive)
	ActorTag string // exact match on actor_user_tags.tag
	Sort     string // "name" (default) or "movieCount"
	Limit    int
	Offset   int
}

// ListActorsResponse is returned by GET /api/library/actors.
type ListActorsResponse struct {
	Total  int                `json:"total"`
	Actors []ActorListItemDTO `json:"actors"`
}

// PatchActorUserTagsBody is the JSON body for PATCH /api/library/actors/tags?name=.
type PatchActorUserTagsBody struct {
	UserTags []string `json:"userTags"`
}

// MaxMovieCommentRunes is the maximum length (Unicode scalars) for PUT /library/movies/{id}/comment body.
const MaxMovieCommentRunes = 10000

// MovieCommentDTO is returned by GET /api/library/movies/{movieId}/comment (empty body when none saved).
type MovieCommentDTO struct {
	Body      string `json:"body"`
	UpdatedAt string `json:"updatedAt"`
}

// PutMovieCommentBody is the JSON body for PUT /api/library/movies/{movieId}/comment.
type PutMovieCommentBody struct {
	Body string `json:"body"`
}

type GetMovieDetailRequest struct {
	MovieID string `json:"movieId"`
}

type StartScanRequest struct {
	Paths []string `json:"paths,omitempty"`
}

// StartMetadataRefreshByPathsRequest is the body for POST /api/library/metadata-scrape.
// Each path must match a configured library root (after filepath.Clean); see MetadataRefreshQueuedDTO.invalidPaths.
type StartMetadataRefreshByPathsRequest struct {
	Paths []string `json:"paths"`
}

// MetadataRefreshQueuedDTO is returned when bulk metadata rescrape jobs are queued.
type MetadataRefreshQueuedDTO struct {
	Queued       int      `json:"queued"`
	Skipped      int      `json:"skipped"`
	InvalidPaths []string `json:"invalidPaths"`
}

type GetTaskStatusRequest struct {
	TaskID string `json:"taskId"`
}

type ScanFileResultDTO struct {
	TaskID       string `json:"taskId"`
	Path         string `json:"path"`
	FileName     string `json:"fileName"`
	Number       string `json:"number,omitempty"`
	MovieID      string `json:"movieId,omitempty"`
	Status       string `json:"status"`
	Reason       string `json:"reason,omitempty"`
	ImportLayout string `json:"importLayout,omitempty"` // loose | curated | external (extended first-scan only)
}

type ScanSummaryDTO struct {
	TaskID           string              `json:"taskId"`
	Paths            []string            `json:"paths"`
	FilesDiscovered  int                 `json:"filesDiscovered"`
	FilesImported    int                 `json:"filesImported"`
	FilesUpdated     int                 `json:"filesUpdated"`
	FilesSkipped     int                 `json:"filesSkipped"`
	RecognizedNumber int                 `json:"recognizedNumber"`
	Results          []ScanFileResultDTO `json:"results,omitempty"`
}

type MovieListItemDTO struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Code           string   `json:"code"`
	Studio         string   `json:"studio"`
	Actors         []string `json:"actors"`
	Tags           []string `json:"tags"`
	UserTags       []string `json:"userTags,omitempty"`
	RuntimeMinutes int      `json:"runtimeMinutes"`
	Rating         float64  `json:"rating"`
	IsFavorite     bool     `json:"isFavorite"`
	AddedAt        string   `json:"addedAt"`
	Location       string   `json:"location"`
	Resolution     string   `json:"resolution"`
	Year           int      `json:"year"`
	ReleaseDate    string   `json:"releaseDate,omitempty"`
	CoverURL       string   `json:"coverUrl,omitempty"`
	ThumbURL       string   `json:"thumbUrl,omitempty"`
	// TrashedAt is RFC3339 when the movie is in the recycle bin; omitted when active.
	TrashedAt string `json:"trashedAt,omitempty"`
}

type MovieDetailDTO struct {
	MovieListItemDTO
	Summary         string   `json:"summary"`
	PreviewImages   []string `json:"previewImages,omitempty"`
	PreviewVideoURL string   `json:"previewVideoUrl,omitempty"`
	// MetadataRating is the scraper/site score (movies.rating). UserRating is the local override (movies.user_rating).
	MetadataRating float64  `json:"metadataRating"`
	UserRating     *float64 `json:"userRating,omitempty"`
	// ActorAvatarURLs maps library actor display name (actors.name) -> actors.avatar URL after profile scrape.
	ActorAvatarURLs map[string]string `json:"actorAvatarUrls,omitempty"`
	// User*Override: in-memory seed only (json:"-"); SQLite applies overrides in SQL. EffectiveXXX for API = merge in EffectiveMovieDetailDTO.
	UserTitleOverride          *string `json:"-"`
	UserStudioOverride         *string `json:"-"`
	UserSummaryOverride        *string `json:"-"`
	UserReleaseDateOverride    *string `json:"-"`
	UserRuntimeMinutesOverride *int    `json:"-"`
}

// PatchMovieInput is the parsed body for PATCH /api/library/movies/{movieId}.
// Favorite: non-nil updates is_favorite.
// UserRatingSet: false = do not change user_rating; true + UserRatingClear = set NULL; true + !UserRatingClear = set UserRating (0–5).
// UserTagsSet: true replaces all user tags for the movie (UserTags may be empty to clear).
// MetadataTagsSet: true replaces all scraper/NFO (type=nfo) tags for the movie; does not touch user tags. Empty list clears NFO tags locally until next scrape.
// UserTitleSet etc.: JSON null or "" clears the user_* override (revert to scraped column). Non-empty string sets override.
// UserRuntimeMinutesSet + UserRuntimeMinutesClear (JSON null): clear runtime override. Otherwise set minutes (>= 0).
type PatchMovieInput struct {
	Favorite        *bool
	UserRatingSet   bool
	UserRatingClear bool
	UserRating      float64
	UserTagsSet     bool
	UserTags        []string
	MetadataTagsSet bool
	MetadataTags    []string

	UserTitleSet            bool
	UserTitleClear          bool
	UserTitle               string
	UserStudioSet           bool
	UserStudioClear         bool
	UserStudio              string
	UserSummarySet          bool
	UserSummaryClear        bool
	UserSummary             string
	UserReleaseDateSet      bool
	UserReleaseDateClear    bool
	UserReleaseDate         string
	UserRuntimeMinutesSet   bool
	UserRuntimeMinutesClear bool
	UserRuntimeMinutes      int
}

// EffectiveMovieDetailDTO returns a copy with title/studio/summary/release/runtime/year merged from User*Override (in-memory library).
func EffectiveMovieDetailDTO(m MovieDetailDTO) MovieDetailDTO {
	out := m
	if s := ptrTrimString(m.UserTitleOverride); s != "" {
		out.Title = s
	}
	if s := ptrTrimString(m.UserStudioOverride); s != "" {
		out.Studio = s
	}
	if s := ptrTrimString(m.UserSummaryOverride); s != "" {
		out.Summary = s
	}
	if s := ptrTrimString(m.UserReleaseDateOverride); s != "" {
		out.ReleaseDate = s
		if y, ok := yearPrefixFromReleaseDate(s); ok {
			out.Year = y
		}
	}
	if m.UserRuntimeMinutesOverride != nil {
		out.RuntimeMinutes = *m.UserRuntimeMinutesOverride
	}
	return out
}

func ptrTrimString(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

func yearPrefixFromReleaseDate(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if len(s) < 4 {
		return 0, false
	}
	y := 0
	for i := 0; i < 4; i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, false
		}
		y = y*10 + int(c-'0')
	}
	if y < 1800 || y > 3000 {
		return 0, false
	}
	return y, true
}

type MoviesPageDTO struct {
	Items  []MovieListItemDTO `json:"items"`
	Total  int                `json:"total"`
	Limit  int                `json:"limit"`
	Offset int                `json:"offset"`
}

type LibraryPathDTO struct {
	ID                      string `json:"id"`
	Path                    string `json:"path"`
	Title                   string `json:"title"`
	FirstLibraryScanPending bool   `json:"firstLibraryScanPending"`
}

// AddLibraryPathRequest is the body for POST /api/library/paths.
type AddLibraryPathRequest struct {
	Path  string `json:"path"`
	Title string `json:"title,omitempty"`
}

// AddLibraryPathResponse is returned by POST /api/library/paths (path fields + optional initial scan task).
type AddLibraryPathResponse struct {
	LibraryPathDTO
	ScanTask *TaskDTO `json:"scanTask,omitempty"`
}

// UpdateLibraryPathRequest is the body for PATCH /api/library/paths/{id}.
type UpdateLibraryPathRequest struct {
	Title string `json:"title"`
}

type SettingsDTO struct {
	LibraryPaths    []LibraryPathDTO  `json:"libraryPaths"`
	Player          PlayerSettingsDTO `json:"player"`
	OrganizeLibrary bool              `json:"organizeLibrary"`
	// ExtendedLibraryImport: when true, first scan under a newly added library root may classify curated/external layouts (library-config.cfg).
	ExtendedLibraryImport bool `json:"extendedLibraryImport"`
	// AutoLibraryWatch: when true, directory watching may queue debounced scans for new files under library roots (library-config.cfg).
	AutoLibraryWatch       bool     `json:"autoLibraryWatch"`
	MetadataMovieProvider  string   `json:"metadataMovieProvider"`
	MetadataMovieProviders []string `json:"metadataMovieProviders"`
	// MetadataMovieProviderChain: ordered provider priority list (may be non-empty while UI mode is auto/specified).
	MetadataMovieProviderChain []string `json:"metadataMovieProviderChain"`
	// MetadataMovieScrapeMode: auto | specified | chain — which strategy the backend uses for new scrapes.
	// Saved chain/specifier lists may remain in cfg when switching mode so the UI can restore them.
	MetadataMovieScrapeMode string `json:"metadataMovieScrapeMode"`
	// MetadataMovieStrategy refines provider scheduling while keeping legacy mode fields compatible.
	MetadataMovieStrategy string `json:"metadataMovieStrategy,omitempty"`
	// Proxy configuration for outbound HTTP requests (scraping, metadata fetch).
	Proxy ProxySettingsDTO `json:"proxy"`
	// BackendLog: file/console log settings persisted in library-config.cfg; restart backend to apply to Zap sinks.
	BackendLog BackendLogSettingsDTO `json:"backendLog"`
}

// BackendLogSettingsDTO mirrors config log fields exposed in settings (library-config.cfg).
type BackendLogSettingsDTO struct {
	LogDir        string `json:"logDir"`
	LogFilePrefix string `json:"logFilePrefix,omitempty"`
	LogMaxAgeDays int    `json:"logMaxAgeDays,omitempty"`
	LogLevel      string `json:"logLevel,omitempty"`
}

// PatchBackendLogSettings is a partial update for backendLog; nil pointer = leave unchanged.
type PatchBackendLogSettings struct {
	LogDir        *string `json:"logDir,omitempty"`
	LogFilePrefix *string `json:"logFilePrefix,omitempty"`
	LogMaxAgeDays *int    `json:"logMaxAgeDays,omitempty"`
	LogLevel      *string `json:"logLevel,omitempty"`
}

// ProxySettingsDTO is the proxy configuration for SettingsDTO.
type ProxySettingsDTO struct {
	Enabled  bool   `json:"enabled"`
	URL      string `json:"url,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// ProxyJavBusPingRequest is the body for POST /api/proxy/ping-javbus and POST /api/proxy/ping-google.
// When Proxy is nil, the server uses the currently persisted proxy config.
type ProxyJavBusPingRequest struct {
	Proxy *ProxySettingsDTO `json:"proxy,omitempty"`
}

// ProxyJavBusPingResponse reports outbound reachability for proxy ping endpoints (JavBus, Google, etc.).
type ProxyJavBusPingResponse struct {
	OK         bool   `json:"ok"`
	LatencyMs  int64  `json:"latencyMs"`
	HTTPStatus int    `json:"httpStatus,omitempty"`
	Message    string `json:"message,omitempty"`
}

// PatchSettingsRequest is the body for PATCH /api/settings (partial update).
type PatchSettingsRequest struct {
	OrganizeLibrary       *bool                   `json:"organizeLibrary,omitempty"`
	ExtendedLibraryImport *bool                   `json:"extendedLibraryImport,omitempty"`
	AutoLibraryWatch      *bool                   `json:"autoLibraryWatch,omitempty"`
	Player                *PatchPlayerSettingsDTO `json:"player,omitempty"`
	MetadataMovieProvider *string                 `json:"metadataMovieProvider,omitempty"`
	// MetadataMovieProviderChain: ordered list of providers to try in sequence; nil = no change; empty = clear (auto mode).
	MetadataMovieProviderChain *[]string `json:"metadataMovieProviderChain,omitempty"`
	// MetadataMovieScrapeMode: auto | specified | chain; switches active scrape strategy without necessarily clearing saved lists.
	MetadataMovieScrapeMode *string `json:"metadataMovieScrapeMode,omitempty"`
	// MetadataMovieStrategy: auto-global | auto-cn-friendly | custom-chain | specified.
	MetadataMovieStrategy *string `json:"metadataMovieStrategy,omitempty"`
	// Proxy: nil = no change; non-nil object replaces current proxy config.
	Proxy *ProxySettingsDTO `json:"proxy,omitempty"`
	// BackendLog: nil = no change; non-empty partial fields merge into current and persist.
	BackendLog *PatchBackendLogSettings `json:"backendLog,omitempty"`
}

type PlayerSettingsDTO struct {
	HardwareDecode      bool   `json:"hardwareDecode"`
	HardwareEncoder     string `json:"hardwareEncoder,omitempty"`
	NativePlayerPreset  string `json:"nativePlayerPreset,omitempty"`
	NativePlayerEnabled bool   `json:"nativePlayerEnabled"`
	NativePlayerCommand string `json:"nativePlayerCommand,omitempty"`
	StreamPushEnabled   bool   `json:"streamPushEnabled"`
	ForceStreamPush     bool   `json:"forceStreamPush,omitempty"`
	FFmpegCommand       string `json:"ffmpegCommand,omitempty"`
	PreferNativePlayer  bool   `json:"preferNativePlayer"`
	SeekForwardStepSec  int    `json:"seekForwardStepSec"`
	SeekBackwardStepSec int    `json:"seekBackwardStepSec"`
}

type PatchPlayerSettingsDTO struct {
	HardwareDecode      *bool   `json:"hardwareDecode,omitempty"`
	HardwareEncoder     *string `json:"hardwareEncoder,omitempty"`
	NativePlayerPreset  *string `json:"nativePlayerPreset,omitempty"`
	NativePlayerEnabled *bool   `json:"nativePlayerEnabled,omitempty"`
	NativePlayerCommand *string `json:"nativePlayerCommand,omitempty"`
	StreamPushEnabled   *bool   `json:"streamPushEnabled,omitempty"`
	ForceStreamPush     *bool   `json:"forceStreamPush,omitempty"`
	FFmpegCommand       *string `json:"ffmpegCommand,omitempty"`
	PreferNativePlayer  *bool   `json:"preferNativePlayer,omitempty"`
	SeekForwardStepSec  *int    `json:"seekForwardStepSec,omitempty"`
	SeekBackwardStepSec *int    `json:"seekBackwardStepSec,omitempty"`
}

// PlaybackProgressItemDTO is one row in GET /api/playback/progress.
type PlaybackProgressItemDTO struct {
	MovieID     string  `json:"movieId"`
	PositionSec float64 `json:"positionSec"`
	DurationSec float64 `json:"durationSec"`
	UpdatedAt   string  `json:"updatedAt"`
}

type PlaybackProgressListDTO struct {
	Items []PlaybackProgressItemDTO `json:"items"`
}

// PutPlaybackProgressBody is the JSON body for PUT /api/playback/progress/{movieId}.
type PutPlaybackProgressBody struct {
	PositionSec float64 `json:"positionSec"`
	DurationSec float64 `json:"durationSec"`
}

type PlaybackMode string

const (
	PlaybackModeDirect PlaybackMode = "direct"
	PlaybackModeHLS    PlaybackMode = "hls"
	PlaybackModeNative PlaybackMode = "native"
)

type PlaybackAudioTrackDTO struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Default bool   `json:"default"`
}

type PlaybackSubtitleTrackDTO struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Kind    string `json:"kind,omitempty"`
	Default bool   `json:"default"`
}

type PlaybackDescriptorDTO struct {
	MovieID           string                     `json:"movieId"`
	Mode              PlaybackMode               `json:"mode"`
	SessionID         string                     `json:"sessionId,omitempty"`
	SessionKind       string                     `json:"sessionKind,omitempty"`
	URL               string                     `json:"url"`
	MimeType          string                     `json:"mimeType,omitempty"`
	FileName          string                     `json:"fileName,omitempty"`
	TranscodeProfile  string                     `json:"transcodeProfile,omitempty"`
	DurationSec       float64                    `json:"durationSec,omitempty"`
	StartPositionSec  float64                    `json:"startPositionSec,omitempty"`
	ResumePositionSec float64                    `json:"resumePositionSec,omitempty"`
	CanDirectPlay     bool                       `json:"canDirectPlay"`
	Reason            string                     `json:"reason,omitempty"`
	ReasonCode        string                     `json:"reasonCode,omitempty"`
	ReasonMessage     string                     `json:"reasonMessage,omitempty"`
	SourceContainer   string                     `json:"sourceContainer,omitempty"`
	SourceVideoCodec  string                     `json:"sourceVideoCodec,omitempty"`
	SourceAudioCodec  string                     `json:"sourceAudioCodec,omitempty"`
	AudioTracks       []PlaybackAudioTrackDTO    `json:"audioTracks,omitempty"`
	SubtitleTracks    []PlaybackSubtitleTrackDTO `json:"subtitleTracks,omitempty"`
}

type CreatePlaybackSessionRequest struct {
	Mode             PlaybackMode `json:"mode,omitempty"`
	StartPositionSec float64      `json:"startPositionSec,omitempty"`
}

type PlaybackSessionStatusDTO struct {
	SessionID        string  `json:"sessionId"`
	MovieID          string  `json:"movieId"`
	SessionKind      string  `json:"sessionKind,omitempty"`
	TranscodeProfile string  `json:"transcodeProfile,omitempty"`
	StartPositionSec float64 `json:"startPositionSec,omitempty"`
	StartedAt        string  `json:"startedAt,omitempty"`
	LastAccessedAt   string  `json:"lastAccessedAt,omitempty"`
	ExpiresAt        string  `json:"expiresAt,omitempty"`
	FinishedAt       string  `json:"finishedAt,omitempty"`
	State            string  `json:"state,omitempty"`
	LastError        string  `json:"lastError,omitempty"`
}

type PlaybackSessionListDTO struct {
	Items []PlaybackSessionStatusDTO `json:"items"`
}

type NativePlaybackLaunchRequest struct {
	StartPositionSec float64 `json:"startPositionSec,omitempty"`
}

type NativePlaybackLaunchDTO struct {
	OK        bool   `json:"ok"`
	Command   string `json:"command,omitempty"`
	Target    string `json:"target,omitempty"`
	Mode      string `json:"mode,omitempty"`
	Message   string `json:"message,omitempty"`
	MovieID   string `json:"movieId,omitempty"`
	StartedAt string `json:"startedAt,omitempty"`
}

// CuratedFrameItemDTO is list metadata (no image); use GET /api/curated-frames/{id}/image for bytes.
type CuratedFrameItemDTO struct {
	ID          string   `json:"id"`
	MovieID     string   `json:"movieId"`
	Title       string   `json:"title"`
	Code        string   `json:"code"`
	Actors      []string `json:"actors"`
	PositionSec float64  `json:"positionSec"`
	CapturedAt  string   `json:"capturedAt"`
	Tags        []string `json:"tags"`
}

type CuratedFramesListDTO struct {
	Items []CuratedFrameItemDTO `json:"items"`
}

// CreateCuratedFrameBody is the JSON body for POST /api/curated-frames (image as standard base64, no data: prefix).
type CreateCuratedFrameBody struct {
	ID          string   `json:"id"`
	MovieID     string   `json:"movieId"`
	Title       string   `json:"title"`
	Code        string   `json:"code"`
	Actors      []string `json:"actors"`
	PositionSec float64  `json:"positionSec"`
	CapturedAt  string   `json:"capturedAt"`
	Tags        []string `json:"tags"`
	ImageBase64 string   `json:"imageBase64"`
}

// PatchCuratedFrameTagsBody is the JSON body for PATCH /api/curated-frames/{id}/tags.
type PatchCuratedFrameTagsBody struct {
	Tags []string `json:"tags"`
}

// PostCuratedFramesExportBody is the JSON body for POST /api/curated-frames/export.
type PostCuratedFramesExportBody struct {
	IDs       []string `json:"ids"`
	ActorName string   `json:"actorName,omitempty"`
	// Format is "webp" (default) or "png".
	Format string `json:"format,omitempty"`
}

// PlayedMoviesListDTO is returned by GET /api/library/played-movies.
type PlayedMoviesListDTO struct {
	MovieIDs []string `json:"movieIds"`
}

type TaskDTO struct {
	TaskID        string         `json:"taskId"`
	Type          string         `json:"type"`
	Status        string         `json:"status"`
	CreatedAt     string         `json:"createdAt"`
	StartedAt     string         `json:"startedAt,omitempty"`
	FinishedAt    string         `json:"finishedAt,omitempty"`
	Progress      int            `json:"progress"`
	Message       string         `json:"message,omitempty"`
	ErrorCode     string         `json:"errorCode,omitempty"`
	ErrorCategory string         `json:"errorCategory,omitempty"`
	ErrorMessage  string         `json:"errorMessage,omitempty"`
	Provider      string         `json:"provider,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

// RecentTasksDTO is returned by GET /api/tasks/recent (in-memory tasks only).
type RecentTasksDTO struct {
	Tasks []TaskDTO `json:"tasks"`
}

type TaskEventDTO struct {
	Task TaskDTO `json:"task"`
}

const (
	CommandSystemHealth  = "system.health"
	CommandLibraryList   = "library.list"
	CommandLibraryDetail = "library.detail"
	CommandSettingsGet   = "settings.get"
	CommandScanStart     = "scan.start"
	CommandScanStatus    = "scan.status"

	EventTaskStarted          = "task.started"
	EventTaskProgress         = "task.progress"
	EventTaskCompleted        = "task.completed"
	EventTaskFailed           = "task.failed"
	EventScanStarted          = "scan.started"
	EventScanProgress         = "scan.progress"
	EventScanFileSkipped      = "scan.file_skipped"
	EventScanFileImported     = "scan.file_imported"
	EventScanFileUpdated      = "scan.file_updated"
	EventScanCompleted        = "scan.completed"
	EventAssetDownloaded      = "asset.downloaded"
	EventAssetDownloadFailed  = "asset.download_failed"
	EventScraperMetadataSaved = "scraper.metadata_saved"
	EventScraperFailed        = "scraper.failed"

	TaskPending       = "pending"
	TaskRunning       = "running"
	TaskCompleted     = "completed"
	TaskPartialFailed = "partial_failed"
	TaskFailed        = "failed"
	TaskCancelled     = "cancelled"

	ErrorCodeBadRequest    = "COMMON_BAD_REQUEST"
	ErrorCodeNotFound      = "COMMON_NOT_FOUND"
	ErrorCodeInternal      = "COMMON_INTERNAL"
	ErrorCodeUnsupported   = "COMMON_UNSUPPORTED_COMMAND"
	ErrorCodeLibraryFetch  = "LIBRARY_FETCH_FAILED"
	ErrorCodeScanStart     = "SCAN_START_FAILED"
	ErrorCodeScanWalk      = "SCAN_WALK_FAILED"
	ErrorCodeScanCancelled = "SCAN_CANCELLED"
	ErrorCodeScraperInit   = "SCRAPER_INIT_FAILED"
	ErrorCodeScraperRun    = "SCRAPER_RUN_FAILED"
	ErrorCodeAssetDownload = "ASSET_DOWNLOAD_FAILED"
	ErrorCodeConflict      = "COMMON_CONFLICT"

	// Curated frames export
	ErrorCodeCuratedExportActorMismatch = "CURATED_EXPORT_ACTOR_MISMATCH"

	// Provider health check errors
	ErrorCodeProviderNotFound   = "PROVIDER_NOT_FOUND"
	ErrorCodeProviderPingFailed = "PROVIDER_PING_FAILED"
)

// ProviderHealthStatus indicates the availability status of a metadata provider.
type ProviderHealthStatus string

const (
	ProviderHealthOK       ProviderHealthStatus = "ok"
	ProviderHealthDegraded ProviderHealthStatus = "degraded"
	ProviderHealthFail     ProviderHealthStatus = "fail"
)

// ProviderHealthDTO is the result of pinging a single provider.
type ProviderHealthDTO struct {
	Name                string               `json:"name"`
	Status              ProviderHealthStatus `json:"status"`
	LatencyMs           int64                `json:"latencyMs"`
	Message             string               `json:"message,omitempty"`
	ErrorCategory       string               `json:"errorCategory,omitempty"`
	CooldownUntil       string               `json:"cooldownUntil,omitempty"`
	ConsecutiveFailures int                  `json:"consecutiveFailures,omitempty"`
	AvgLatencyMs        int64                `json:"avgLatencyMs,omitempty"`
}

// PingProviderRequest is the body for POST /api/providers/ping.
type PingProviderRequest struct {
	Name string `json:"name"`
}

// PingAllProvidersResponse is returned by POST /api/providers/ping-all.
type PingAllProvidersResponse struct {
	Providers []ProviderHealthDTO `json:"providers"`
	Total     int                 `json:"total"`
	OK        int                 `json:"ok"`
	Fail      int                 `json:"fail"`
}

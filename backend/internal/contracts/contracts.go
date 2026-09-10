// Package contracts defines DTOs, error codes, and shared types for the Curated backend API.
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

var (
	ErrRecommendationFeedbackInvalid        = errors.New("recommendation feedback invalid")
	ErrRecommendationFeedbackTargetNotFound = errors.New("recommendation feedback target not found")
	ErrRecommendationFeedbackLimitReached   = errors.New("recommendation feedback limit reached")
	ErrActorMergeInvalid                    = errors.New("actor merge invalid")
	ErrActorMergeNotFound                   = errors.New("actor merge actor not found")
	ErrActorMergeSelf                       = errors.New("actor merge source and target are identical")
	ErrActorMergeSourceAlias                = errors.New("actor merge source is an alias")
	ErrActorMergeConflict                   = errors.New("actor merge conflict")
	ErrActorMergeStalePreview               = errors.New("actor merge preview is stale")
	ErrActorMergeLinkLimit                  = errors.New("actor merge external link limit exceeded")
	ErrPersonalInsightsInvalidRange         = errors.New("personal insights range is invalid")
	ErrPersonalInsightsInvalidTimezone      = errors.New("personal insights timezone is invalid")
	ErrPersonalInsightsInvalidDimension     = errors.New("personal insights dimension is invalid")
	ErrPersonalInsightsInvalidLimit         = errors.New("personal insights limit is invalid")
)

// ErrBackupDestinationExists prevents accidental replacement of a backup package.
var ErrBackupDestinationExists = errors.New("backup destination already exists")

// Command represents a request message in the stdio JSONL transport.
type Command struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// Response answers a command, carrying ok/error status and optional data in stdio JSONL transport.
type Response struct {
	Kind      string    `json:"kind"`
	ID        string    `json:"id,omitempty"`
	OK        bool      `json:"ok"`
	Data      any       `json:"data,omitempty"`
	Error     *AppError `json:"error,omitempty"`
	Timestamp string    `json:"timestamp"`
}

// Event is an asynchronous notification pushed by the backend.
type Event struct {
	Kind      string `json:"kind"`
	Type      string `json:"type"`
	Payload   any    `json:"payload,omitempty"`
	Timestamp string `json:"timestamp"`
}

// AppError represents a structured API error with a stable machine-readable code.
type AppError struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	Retryable bool           `json:"retryable"`
	Details   map[string]any `json:"details,omitempty"`
}

// HealthDTO reports backend identity, version, and runtime information.
type HealthDTO struct {
	Name             string `json:"name"`
	Version          string `json:"version"` // Build stamp YYYYMMDD.HHMMSS (UTC) or git.* / unknown; see Channel
	Channel          string `json:"channel"` // "dev" or "release" (-tags release)
	InstallerVersion string `json:"installerVersion,omitempty"`
	Transport        string `json:"transport"`
	DatabasePath     string `json:"databasePath"`
}

// ConnectedClientDTO is one client that accessed the HTTP API during this backend process lifetime.
type ConnectedClientDTO struct {
	Key            string `json:"key"`
	IP             string `json:"ip"`
	Port           int    `json:"port,omitempty"`
	Hostname       string `json:"hostname,omitempty"`
	UserAgent      string `json:"userAgent,omitempty"`
	Browser        string `json:"browser"`
	BrowserVersion string `json:"browserVersion,omitempty"`
	OS             string `json:"os"`
	OSVersion      string `json:"osVersion,omitempty"`
	DeviceType     string `json:"deviceType"`
	AccessKind     string `json:"accessKind"`
	IsLocalMachine bool   `json:"isLocalMachine"`
	FirstSeen      string `json:"firstSeen"`
	LastSeen       string `json:"lastSeen"`
	RequestCount   int64  `json:"requestCount"`
}

// ConnectedClientsDTO summarizes clients that accessed the HTTP API during this backend process lifetime.
type ConnectedClientsDTO struct {
	Clients     []ConnectedClientDTO `json:"clients"`
	Total       int                  `json:"total"`
	LocalCount  int                  `json:"localCount"`
	RemoteCount int                  `json:"remoteCount"`
	SampledAt   string               `json:"sampledAt"`
}

// DevPerformanceSummaryDTO carries sampled CPU usage for the dev-only performance monitor bar.
type DevPerformanceSummaryDTO struct {
	Supported         bool    `json:"supported"`
	SampledAt         string  `json:"sampledAt,omitempty"`
	SystemCPUPercent  float64 `json:"systemCpuPercent,omitempty"`
	BackendCPUPercent float64 `json:"backendCpuPercent,omitempty"`
}

// AppUpdateStatusDTO reports the packaged-app update state vs the latest GitHub Release.
type AppUpdateStatusDTO struct {
	Supported            bool   `json:"supported"`
	Status               string `json:"status"`
	InstalledVersion     string `json:"installedVersion,omitempty"`
	LatestVersion        string `json:"latestVersion,omitempty"`
	HasUpdate            bool   `json:"hasUpdate,omitempty"`
	CheckedAt            string `json:"checkedAt,omitempty"`
	PublishedAt          string `json:"publishedAt,omitempty"`
	ReleaseName          string `json:"releaseName,omitempty"`
	ReleaseURL           string `json:"releaseUrl,omitempty"`
	InstallerDownloadURL string `json:"installerDownloadUrl,omitempty"`
	InstallerSHA256      string `json:"installerSha256,omitempty"`
	ArtifactStatus       string `json:"artifactStatus,omitempty"`
	DownloadedVersion    string `json:"downloadedVersion,omitempty"`
	DownloadedFileName   string `json:"downloadedFileName,omitempty"`
	DownloadedBytes      int64  `json:"downloadedBytes,omitempty"`
	TotalBytes           int64  `json:"totalBytes,omitempty"`
	DownloadProgress     int    `json:"downloadProgress,omitempty"`
	SignatureStatus      string `json:"signatureStatus,omitempty"`
	InstallReady         bool   `json:"installReady,omitempty"`
	LastInstallAttemptAt string `json:"lastInstallAttemptAt,omitempty"`
	LastInstallError     string `json:"lastInstallError,omitempty"`
	DownloadTaskID       string `json:"downloadTaskId,omitempty"`
	ReleaseNotesSnippet  string `json:"releaseNotesSnippet,omitempty"`
	Source               string `json:"source,omitempty"`
	ErrorMessage         string `json:"errorMessage,omitempty"`
}

// AppUpdateInstallRequest is the body for POST /api/app-update/install.
type AppUpdateInstallRequest struct {
	Mode string `json:"mode,omitempty"`
}

// BackupCreateRequest creates a new package at a server-side path.
type BackupCreateRequest struct {
	DestinationPath string `json:"destinationPath"`
}

// BackupPathRequest identifies an existing backup package.
type BackupPathRequest struct {
	BackupPath string `json:"backupPath"`
}

type BackupScopeDTO struct {
	DatabaseIncluded      bool `json:"databaseIncluded"`
	LibraryConfigIncluded bool `json:"libraryConfigIncluded"`
	UserAssetsIncluded    bool `json:"userAssetsIncluded"`
	MediaFilesIncluded    bool `json:"mediaFilesIncluded"`
}

type BackupFileDTO struct {
	Kind      string `json:"kind"`
	Path      string `json:"path"`
	SizeBytes int64  `json:"sizeBytes"`
	SHA256    string `json:"sha256"`
}

type BackupManifestDTO struct {
	Format           string          `json:"format"`
	FormatVersion    int             `json:"formatVersion"`
	CreatedAt        string          `json:"createdAt"`
	AppVersion       string          `json:"appVersion"`
	AppChannel       string          `json:"appChannel"`
	Scope            BackupScopeDTO  `json:"scope"`
	SchemaMigrations []string        `json:"schemaMigrations"`
	Files            []BackupFileDTO `json:"files"`
}

type BackupIntegrityDTO struct {
	QuickCheck           string `json:"quickCheck"`
	ForeignKeyViolations int    `json:"foreignKeyViolations"`
}

type BackupVerificationDTO struct {
	Valid             bool               `json:"valid"`
	CheckedAt         string             `json:"checkedAt"`
	Manifest          *BackupManifestDTO `json:"manifest,omitempty"`
	DatabaseIntegrity BackupIntegrityDTO `json:"databaseIntegrity"`
	Errors            []string           `json:"errors"`
	Warnings          []string           `json:"warnings"`
}

type BackupRestorePreflightDTO struct {
	CanRestore            bool                  `json:"canRestore"`
	CheckedAt             string                `json:"checkedAt"`
	Verification          BackupVerificationDTO `json:"verification"`
	TargetDatabase        string                `json:"targetDatabase"`
	TargetDatabaseExists  bool                  `json:"targetDatabaseExists"`
	TargetConfig          string                `json:"targetConfig,omitempty"`
	TargetConfigExists    bool                  `json:"targetConfigExists"`
	RequiredBytes         int64                 `json:"requiredBytes"`
	AvailableBytes        uint64                `json:"availableBytes"`
	AvailableBytesKnown   bool                  `json:"availableBytesKnown"`
	UnsupportedMigrations []string              `json:"unsupportedMigrations"`
	Errors                []string              `json:"errors"`
	Warnings              []string              `json:"warnings"`
}

// ListMoviesRequest filters the library movie listing using the same canonical
// semantics that Saved Views expose to the renderer.
type ListMoviesRequest struct {
	Mode  string `json:"mode,omitempty"`
	Query string `json:"query,omitempty"`
	// Tag / Tags are exact matches against metadata (nfo) or user tags; multiple values are AND.
	Tag  string   `json:"tag,omitempty"`
	Tags []string `json:"tags,omitempty"`
	// Actor is comma-separated exact actor names (AND; aliases resolve to canonical).
	Actor string `json:"actor,omitempty"`
	// Studio is comma-separated effective studio names (OR).
	Studio     string   `json:"studio,omitempty"`
	PlayState  string   `json:"playState,omitempty"`
	UserRating *float64 `json:"userRating,omitempty"`
	Unrated    bool     `json:"unrated,omitempty"`
	Resolution string   `json:"resolution,omitempty"`
	AddedAfter string   `json:"addedAfter,omitempty"`
	Year       string   `json:"year,omitempty"`
	Runtime    string   `json:"runtime,omitempty"`
	Catalog    string   `json:"catalog,omitempty"`
	Limit      int      `json:"limit,omitempty"`
	Offset     int      `json:"offset,omitempty"`
}

const SavedViewSchemaVersion = 1

// SavedViewFiltersV1 is the durable, versioned filter snapshot stored for a
// user-created library view. Navigation-only query state is deliberately absent.
type SavedViewFiltersV1 struct {
	SchemaVersion   int      `json:"schemaVersion"`
	Mode            string   `json:"mode,omitempty"`
	Query           string   `json:"q,omitempty"`
	Tag             string   `json:"tag,omitempty"`
	Actor           string   `json:"actor,omitempty"`
	Studio          string   `json:"studio,omitempty"`
	Tab             string   `json:"tab,omitempty"`
	PlayState       string   `json:"playState,omitempty"`
	UserRating      *float64 `json:"userRating,omitempty"`
	Unrated         bool     `json:"unrated,omitempty"`
	Resolution      string   `json:"resolution,omitempty"`
	AddedWithinDays int      `json:"addedWithinDays,omitempty"`
	Year            string   `json:"year,omitempty"`
	Runtime         string   `json:"runtime,omitempty"`
	Catalog         string   `json:"catalog,omitempty"`
	Sort            string   `json:"sort,omitempty"`
}

// SavedViewDTO is one ordered user-defined library view.
type SavedViewDTO struct {
	ID        string             `json:"id"`
	Name      string             `json:"name"`
	Filters   SavedViewFiltersV1 `json:"filters"`
	SortOrder int                `json:"sortOrder"`
	CreatedAt string             `json:"createdAt"`
	UpdatedAt string             `json:"updatedAt"`
}

type SavedViewsDTO struct {
	Items []SavedViewDTO `json:"items"`
}

type CreateSavedViewBody struct {
	Name    string             `json:"name"`
	Filters SavedViewFiltersV1 `json:"filters"`
}

type PatchSavedViewBody struct {
	Name    *string             `json:"name,omitempty"`
	Filters *SavedViewFiltersV1 `json:"filters,omitempty"`
}

type ReorderSavedViewsBody struct {
	IDs []string `json:"ids"`
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
	ExternalLinks    []string `json:"externalLinks,omitempty"`
	Aliases          []string `json:"aliases,omitempty"`
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

// PatchActorExternalLinksBody is the JSON body for PATCH /api/library/actors/external-links?name=.
type PatchActorExternalLinksBody struct {
	ExternalLinks []string `json:"externalLinks"`
}

type ActorMergeActorRefDTO struct {
	ID      int64    `json:"id"`
	Name    string   `json:"name"`
	Aliases []string `json:"aliases"`
}

type ActorMergeAssociationSummaryDTO struct {
	SourceCount    int `json:"sourceCount"`
	TargetCount    int `json:"targetCount"`
	DuplicateCount int `json:"duplicateCount"`
	ResultCount    int `json:"resultCount"`
}

type ActorMergeValuesSummaryDTO struct {
	Source []string `json:"source"`
	Target []string `json:"target"`
	Result []string `json:"result"`
}

type ActorMergeProfileFieldDTO struct {
	Field            string `json:"field"`
	SourceValue      string `json:"sourceValue"`
	TargetValue      string `json:"targetValue"`
	DefaultSelection string `json:"defaultSelection"`
	Conflict         bool   `json:"conflict"`
}

type ActorMergeBlockingReasonDTO struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ActorMergePreviewRequest struct {
	SourceName string `json:"sourceName"`
	TargetName string `json:"targetName"`
}

type ActorMergePreviewDTO struct {
	PreviewToken           string                          `json:"previewToken"`
	Source                 ActorMergeActorRefDTO           `json:"source"`
	Target                 ActorMergeActorRefDTO           `json:"target"`
	Movies                 ActorMergeAssociationSummaryDTO `json:"movies"`
	UserTags               ActorMergeValuesSummaryDTO      `json:"userTags"`
	ExternalLinks          ActorMergeValuesSummaryDTO      `json:"externalLinks"`
	RecommendationFeedback ActorMergeAssociationSummaryDTO `json:"recommendationFeedback"`
	CuratedFramesAffected  int                             `json:"curatedFramesAffected"`
	AliasesToMove          []string                        `json:"aliasesToMove"`
	ProfileFields          []ActorMergeProfileFieldDTO     `json:"profileFields"`
	CanApply               bool                            `json:"canApply"`
	BlockingReasons        []ActorMergeBlockingReasonDTO   `json:"blockingReasons"`
	RequiredDecisions      []string                        `json:"requiredDecisions"`
}

type ApplyActorMergeRequest struct {
	SourceName       string            `json:"sourceName"`
	TargetName       string            `json:"targetName"`
	PreviewToken     string            `json:"previewToken"`
	Confirm          bool              `json:"confirm"`
	ProfileDecisions map[string]string `json:"profileDecisions,omitempty"`
}

type ActorMergeAuditSummaryDTO struct {
	Movies                 ActorMergeAssociationSummaryDTO `json:"movies"`
	UserTags               []string                        `json:"userTags"`
	ExternalLinks          []string                        `json:"externalLinks"`
	Aliases                []string                        `json:"aliases"`
	RecommendationFeedback ActorMergeAssociationSummaryDTO `json:"recommendationFeedback"`
	CuratedFramesAffected  int                             `json:"curatedFramesAffected"`
	ProfileDecisions       map[string]string               `json:"profileDecisions"`
}

type ActorMergeAuditDTO struct {
	ID            string                    `json:"id"`
	SourceActorID int64                     `json:"sourceActorId"`
	TargetActorID int64                     `json:"targetActorId"`
	SourceName    string                    `json:"sourceName"`
	TargetName    string                    `json:"targetName"`
	PreviewToken  string                    `json:"previewToken"`
	Summary       ActorMergeAuditSummaryDTO `json:"summary"`
	AppliedAt     string                    `json:"appliedAt"`
}

type ActorMergeAuditListDTO struct {
	Items  []ActorMergeAuditDTO `json:"items"`
	Total  int                  `json:"total"`
	Limit  int                  `json:"limit"`
	Offset int                  `json:"offset"`
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

// GetMovieDetailRequest identifies a movie by its ID.
type GetMovieDetailRequest struct {
	MovieID string `json:"movieId"`
}

// StartScanRequest optionally restricts a library scan to specific paths.
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

// GetTaskStatusRequest identifies an async task by its ID.
type GetTaskStatusRequest struct {
	TaskID string `json:"taskId"`
}

// SupportedVideoExtensions lists the video container extensions recognized across
// the whole library pipeline (directory watch, library scan, movie import).
// Extensions are lowercase without a leading dot. Keep in sync with the frontend
// mirror in src/components/jav-library/MovieImportDialog.vue.
var SupportedVideoExtensions = []string{
	"mp4", "m4v", "mkv", "avi", "mov", "wmv", "webm", "ts", "m2ts",
	"flv", "mpeg", "mpg", "ogv", "rmvb", "iso",
}

var supportedVideoExtensionSet = func() map[string]struct{} {
	set := make(map[string]struct{}, len(SupportedVideoExtensions))
	for _, ext := range SupportedVideoExtensions {
		set[ext] = struct{}{}
	}
	return set
}()

// IsSupportedVideoExtension reports whether ext (case-insensitive, optional
// leading dot) is a supported video container extension.
func IsSupportedVideoExtension(ext string) bool {
	ext = strings.ToLower(strings.TrimSpace(ext))
	ext = strings.TrimPrefix(ext, ".")
	_, ok := supportedVideoExtensionSet[ext]
	return ok
}

// ScanFileResultDTO is a per-file scan outcome (imported, updated, or skipped).
type ScanFileResultDTO struct {
	TaskID       string `json:"taskId"`
	Path         string `json:"path"`
	FileName     string `json:"fileName"`
	Number       string `json:"number,omitempty"`
	MovieID      string `json:"movieId,omitempty"`
	Status       string `json:"status"`
	Reason       string `json:"reason,omitempty"`
	ImportLayout string `json:"importLayout,omitempty"` // deprecated layout hint field; no longer populated by scans
}

// ScanSummaryDTO aggregates scan results (files discovered, imported, updated, skipped).
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

// MovieListItemDTO is a compact movie row for library list views.
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
	UserRating     *float64 `json:"userRating,omitempty"`
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

// MovieDetailDTO extends MovieListItemDTO with full details for the detail page.
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
	// MetadataProvider is the Metatube movie provider that produced the current scraped metadata.
	// Empty when the title has not been scraped yet.
	MetadataProvider string `json:"metadataProvider,omitempty"`
	// Homepage is the scraped source-site URL (movies.homepage). Empty when not scraped.
	Homepage string `json:"homepage,omitempty"`
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
	// Internal optimistic concurrency preconditions; never accepted from JSON.
	ExpectedTitle   *string `json:"-"`
	ExpectedSummary *string `json:"-"`
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

// MoviesPageDTO is a paginated library movie listing.
type MoviesPageDTO struct {
	Items  []MovieListItemDTO `json:"items"`
	Total  int                `json:"total"`
	Limit  int                `json:"limit"`
	Offset int                `json:"offset"`
}

// LibraryPathDTO represents a configured library root path.
type LibraryPathDTO struct {
	ID                      string `json:"id"`
	Path                    string `json:"path"`
	Title                   string `json:"title"`
	FirstLibraryScanPending bool   `json:"firstLibraryScanPending"`
}

// LibraryPathStorageStatus describes whether a configured library path's backing storage is available.
type LibraryPathStorageStatus string

const (
	LibraryPathStorageStatusOnline           LibraryPathStorageStatus = "online"
	LibraryPathStorageStatusOffline          LibraryPathStorageStatus = "offline"
	LibraryPathStorageStatusVolumeMismatch   LibraryPathStorageStatus = "volume_mismatch"
	LibraryPathStorageStatusPathMissing      LibraryPathStorageStatus = "path_missing"
	LibraryPathStorageStatusPermissionDenied LibraryPathStorageStatus = "permission_denied"
	LibraryPathStorageStatusUnknown          LibraryPathStorageStatus = "unknown"
)

// LibraryPathStorageStatusDTO is the API-facing storage availability snapshot for one library path.
type LibraryPathStorageStatusDTO struct {
	LibraryPathID      string                   `json:"libraryPathId"`
	Path               string                   `json:"path"`
	Title              string                   `json:"title"`
	Status             LibraryPathStorageStatus `json:"status"`
	Message            string                   `json:"message"`
	CheckedAt          string                   `json:"checkedAt"`
	RootPath           string                   `json:"rootPath,omitempty"`
	DriveType          string                   `json:"driveType,omitempty"`
	VolumeLabel        string                   `json:"volumeLabel,omitempty"`
	FileSystem         string                   `json:"fileSystem,omitempty"`
	IdentityConfidence string                   `json:"identityConfidence,omitempty"`
	ExpectedVolumeID   string                   `json:"expectedVolumeId,omitempty"`
	CurrentVolumeID    string                   `json:"currentVolumeId,omitempty"`
	CanRescan          bool                     `json:"canRescan"`
	CanImport          bool                     `json:"canImport"`
}

// LibraryPathStorageStatusListDTO wraps storage status snapshots for configured library paths.
type LibraryPathStorageStatusListDTO struct {
	Items []LibraryPathStorageStatusDTO `json:"items"`
}

// LibraryHealthDatabaseDTO summarizes read-only SQLite integrity checks.
type LibraryHealthDatabaseDTO struct {
	QuickCheckOK       bool     `json:"quickCheckOk"`
	QuickCheckMessages []string `json:"quickCheckMessages"`
	ForeignKeyOK       bool     `json:"foreignKeyOk"`
	ForeignKeyCount    int      `json:"foreignKeyCount"`
}

// LibraryHealthFindingDTO is one stable, actionable health finding.
// RepairActions is advisory; the read-only scan never executes an action.
type LibraryHealthFindingDTO struct {
	ID            string         `json:"id"`
	Category      string         `json:"category"`
	Severity      string         `json:"severity"`
	EntityType    string         `json:"entityType"`
	EntityID      string         `json:"entityId,omitempty"`
	Label         string         `json:"label"`
	Path          string         `json:"path,omitempty"`
	Message       string         `json:"message"`
	Details       map[string]any `json:"details,omitempty"`
	RepairActions []string       `json:"repairActions,omitempty"`
}

// LibraryHealthSummaryDTO contains complete counts even when Findings is truncated.
type LibraryHealthSummaryDTO struct {
	TotalFindings       int            `json:"totalFindings"`
	CriticalFindings    int            `json:"criticalFindings"`
	WarningFindings     int            `json:"warningFindings"`
	InfoFindings        int            `json:"infoFindings"`
	SkippedOfflineFiles int            `json:"skippedOfflineFiles"`
	CategoryCounts      map[string]int `json:"categoryCounts"`
}

// LibraryHealthReportDTO is a fresh, read-only library health snapshot.
type LibraryHealthReportDTO struct {
	ScannedAt       string                        `json:"scannedAt"`
	Status          string                        `json:"status"`
	Database        LibraryHealthDatabaseDTO      `json:"database"`
	StorageStatuses []LibraryPathStorageStatusDTO `json:"storageStatuses"`
	Summary         LibraryHealthSummaryDTO       `json:"summary"`
	Findings        []LibraryHealthFindingDTO     `json:"findings"`
	Truncated       bool                          `json:"truncated"`
}

// StartLibraryHealthRepairRequest starts an explicitly confirmed bounded repair.
type StartLibraryHealthRepairRequest struct {
	Action     string   `json:"action"`
	Categories []string `json:"categories,omitempty"`
	FindingIDs []string `json:"findingIds,omitempty"`
	Limit      int      `json:"limit,omitempty"`
	Confirm    bool     `json:"confirm"`
}

// StartLibraryHealthActionRequest starts a confirmed cleanup for exact current finding IDs.
type StartLibraryHealthActionRequest struct {
	Action     string   `json:"action"`
	FindingIDs []string `json:"findingIds"`
	Confirm    bool     `json:"confirm"`
}

// LibraryHealthRepairItemDTO is one persisted per-finding repair outcome.
type LibraryHealthRepairItemDTO struct {
	Ordinal      int    `json:"ordinal"`
	FindingID    string `json:"findingId"`
	Category     string `json:"category"`
	MovieID      string `json:"movieId"`
	Label        string `json:"label"`
	Status       string `json:"status"`
	ChildTaskID  string `json:"childTaskId,omitempty"`
	ErrorCode    string `json:"errorCode,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	StartedAt    string `json:"startedAt,omitempty"`
	FinishedAt   string `json:"finishedAt,omitempty"`
}

// LibraryHealthRepairDTO exposes a persisted repair run and all per-item results.
type LibraryHealthRepairDTO struct {
	RepairID       string                       `json:"repairId"`
	TaskID         string                       `json:"taskId"`
	Action         string                       `json:"action"`
	Categories     []string                     `json:"categories"`
	Status         string                       `json:"status"`
	TotalItems     int                          `json:"totalItems"`
	CompletedItems int                          `json:"completedItems"`
	SucceededItems int                          `json:"succeededItems"`
	FailedItems    int                          `json:"failedItems"`
	CreatedAt      string                       `json:"createdAt"`
	StartedAt      string                       `json:"startedAt,omitempty"`
	FinishedAt     string                       `json:"finishedAt,omitempty"`
	Items          []LibraryHealthRepairItemDTO `json:"items"`
}

// CheckLibraryPathStorageStatusRequest optionally narrows a fresh storage check to specific library paths.
type CheckLibraryPathStorageStatusRequest struct {
	LibraryPathIDs []string `json:"libraryPathIds,omitempty"`
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

// AuthStatusDTO reports the current PIN app-lock state for the active browser session.
type AuthStatusDTO struct {
	PINEnabled        bool   `json:"pinEnabled"`
	Unlocked          bool   `json:"unlocked"`
	SetupRequired     bool   `json:"setupRequired"`
	PINLength         int    `json:"pinLength"`
	SessionExpiresAt  string `json:"sessionExpiresAt,omitempty"`
	TrustedForever    bool   `json:"trustedForever"`
	SessionTTLMinutes int    `json:"sessionTtlMinutes"`
	// LANRequiresPIN is a deprecated compatibility field. Curated now uses one global PIN lock.
	LANRequiresPIN bool `json:"lanRequiresPin"`
	LockOnRestart  bool `json:"lockOnRestart"`
}

// SetupPINRequest is the body for POST /api/auth/setup-pin.
type SetupPINRequest struct {
	PIN               string `json:"pin"`
	ConfirmPIN        string `json:"confirmPin"`
	SessionTTLMinutes *int   `json:"sessionTtlMinutes,omitempty"`
	LockOnRestart     *bool  `json:"lockOnRestart,omitempty"`
	TrustedForever    bool   `json:"trustedForever,omitempty"`
}

// UnlockPINRequest is the body for POST /api/auth/unlock.
type UnlockPINRequest struct {
	PIN            string `json:"pin"`
	TrustedForever bool   `json:"trustedForever,omitempty"`
}

// ChangePINRequest is the body for POST /api/auth/change-pin.
type ChangePINRequest struct {
	CurrentPIN string `json:"currentPin"`
	NewPIN     string `json:"newPin"`
	ConfirmPIN string `json:"confirmPin"`
}

// PatchAuthSettingsRequest is the body for PATCH /api/auth/settings.
type PatchAuthSettingsRequest struct {
	PINEnabled        *bool `json:"pinEnabled,omitempty"`
	SessionTTLMinutes *int  `json:"sessionTtlMinutes,omitempty"`
	LockOnRestart     *bool `json:"lockOnRestart,omitempty"`
}

// AuthSessionDTO is a safe, non-secret trusted-session summary.
type AuthSessionDTO struct {
	PublicID       string `json:"publicId"`
	UserAgent      string `json:"userAgent,omitempty"`
	IP             string `json:"ip,omitempty"`
	CreatedAt      string `json:"createdAt"`
	LastSeenAt     string `json:"lastSeenAt"`
	TrustedForever bool   `json:"trustedForever"`
	Current        bool   `json:"current"`
}

// AuthSessionsDTO lists active trusted-forever sessions.
type AuthSessionsDTO struct {
	Items []AuthSessionDTO `json:"items"`
}

// UpdateLibraryPathRequest is the body for PATCH /api/library/paths/{id}.
type UpdateLibraryPathRequest struct {
	Title string `json:"title"`
}

// SettingsDTO carries all application settings exposed to the frontend.
type SettingsDTO struct {
	LibraryPaths                    []LibraryPathDTO       `json:"libraryPaths"`
	DefaultImportLibraryPathID      string                 `json:"defaultImportLibraryPathId,omitempty"`
	BackupDirectory                 string                 `json:"backupDirectory"`
	Player                          PlayerSettingsDTO      `json:"player"`
	OrganizeLibrary                 bool                   `json:"organizeLibrary"`
	ComicLibraryEnabled             bool                   `json:"comicLibraryEnabled"`
	AutoComicLibraryWatch           bool                   `json:"autoComicLibraryWatch"`
	ComicLibraryPaths               []ComicLibraryPathDTO  `json:"comicLibraryPaths"`
	DefaultComicImportLibraryPathID string                 `json:"defaultComicImportLibraryPathId,omitempty"`
	ComicReader                     ComicReaderSettingsDTO `json:"comicReader"`
	ComicCache                      ComicCacheSettingsDTO  `json:"comicCache"`
	PhotoLibraryEnabled             bool                   `json:"photoLibraryEnabled"`
	AutoPhotoLibraryWatch           bool                   `json:"autoPhotoLibraryWatch"`
	PhotoLibraryPaths               []PhotoLibraryPathDTO  `json:"photoLibraryPaths"`
	DefaultPhotoImportLibraryPathID string                 `json:"defaultPhotoImportLibraryPathId,omitempty"`
	PhotoViewer                     PhotoViewerSettingsDTO `json:"photoViewer"`
	PhotoCache                      PhotoCacheSettingsDTO  `json:"photoCache"`
	// AutoLibraryWatch: when true, directory watching may queue debounced scans for new files under library roots (library-config.cfg).
	AutoLibraryWatch bool `json:"autoLibraryWatch"`
	// AutoActorProfileScrape: when true, movie scrapes and a bounded library sweep may enqueue missing actor profile scrapes (library-config.cfg).
	AutoActorProfileScrape bool `json:"autoActorProfileScrape"`
	// AutoDownloadUpdates: when true, startup update checks may automatically download and verify a newer installer.
	AutoDownloadUpdates bool `json:"autoDownloadUpdates"`
	// LaunchAtLogin: when true, Curated registers a current-user login autostart entry and starts silently in tray mode.
	LaunchAtLogin bool `json:"launchAtLogin"`
	// LaunchAtLoginSupported reports whether the current runtime can safely manage OS login autostart.
	LaunchAtLoginSupported   bool     `json:"launchAtLoginSupported"`
	CuratedFrameExportFormat string   `json:"curatedFrameExportFormat"`
	CuratedFrameExportMode   string   `json:"curatedFrameExportMode"`
	MetadataMovieProvider    string   `json:"metadataMovieProvider"`
	MetadataMovieProviders   []string `json:"metadataMovieProviders"`
	// MetadataMovieProviderChain: ordered provider priority list (may be non-empty while UI mode is auto/specified).
	MetadataMovieProviderChain []string `json:"metadataMovieProviderChain"`
	// MetadataMovieScrapeMode: auto | specified | chain — which strategy the backend uses for new scrapes.
	// Saved chain/specifier lists may remain in cfg when switching mode so the UI can restore them.
	MetadataMovieScrapeMode string `json:"metadataMovieScrapeMode"`
	// MetadataMovieStrategy refines provider scheduling while keeping legacy mode fields compatible.
	MetadataMovieStrategy string `json:"metadataMovieStrategy,omitempty"`
	// Proxy configuration for outbound HTTP requests (scraping, metadata fetch).
	Proxy ProxySettingsDTO `json:"proxy"`
	// AIProvider is the experimental agent LLM provider configuration (library-config.cfg).
	AIProvider   AIProviderSettingsDTO `json:"aiProvider"`
	AIGovernance *AIGovernanceDTO      `json:"aiGovernance,omitempty"`
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

// AIProviderSettingsDTO mirrors config.AIProviderConfig for the Settings experimental section.
// All fields may be empty, meaning the agent provider is not configured yet.
type AIProviderSettingsDTO struct {
	Kind    string `json:"kind"`
	BaseURL string `json:"baseUrl"`
	APIKey  string `json:"apiKey,omitempty"`
	Model   string `json:"model"`
}

// PatchAIProviderSettings is the partial update for aiProvider; nil pointer = leave unchanged.
type PatchAIProviderSettings struct {
	Kind    *string `json:"kind,omitempty"`
	BaseURL *string `json:"baseUrl,omitempty"`
	APIKey  *string `json:"apiKey,omitempty"`
	Model   *string `json:"model,omitempty"`
}

// AIProviderTestRequest is the body for POST /api/ai/provider/test. When Provider
// is nil the currently persisted provider config is tested (same contract as proxy pings).
type AIProviderTestRequest struct {
	Provider *AIProviderSettingsDTO `json:"provider,omitempty"`
}

// AIProviderTestResponse reports whether the provider answers a minimal chat completion.
type AIProviderTestResponse struct {
	OK        bool   `json:"ok"`
	LatencyMs int64  `json:"latencyMs"`
	Message   string `json:"message,omitempty"`
}

// AIChatMessage is one message of an experimental agent chat turn (system | user | assistant).
type AIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AIChatRequest is the body for POST /api/ai/chat (E2: agent loop with read tools).
type AIChatRequest struct {
	SessionID string          `json:"sessionId,omitempty"`
	Messages  []AIChatMessage `json:"messages"`
	Context   *AIChatContext  `json:"context,omitempty"`
	Locale    string          `json:"locale,omitempty"`
}

// AIChatContext is optional page context injected into the system prompt.
type AIChatContext struct {
	// ContextVersion is omitted by legacy clients. Version 1 adds the bounded,
	// explicit selection and filter projection below.
	ContextVersion   int                  `json:"contextVersion,omitempty"`
	Route            string               `json:"route,omitempty"`
	MovieID          string               `json:"movieId,omitempty"`
	ActorName        string               `json:"actorName,omitempty"`
	Query            string               `json:"query,omitempty"`
	Mentions         []AIChatMention      `json:"mentions,omitempty"`
	SelectedMovieIDs []string             `json:"selectedMovieIds,omitempty"`
	SelectedActors   []string             `json:"selectedActors,omitempty"`
	ActiveFilters    *AIChatActiveFilters `json:"activeFilters,omitempty"`
}

// AIChatMention is one user @-reference from the Agent composer.
type AIChatMention struct {
	Kind  string `json:"kind"`
	ID    string `json:"id"`
	Label string `json:"label"`
}

// AIChatActiveFilters is the small allowlisted subset of a current library
// view that can be shown to the model as one-turn context. It is not a saved
// view and must not gain navigation-only fields.
type AIChatActiveFilters struct {
	Query     string `json:"query,omitempty"`
	Tag       string `json:"tag,omitempty"`
	Actor     string `json:"actor,omitempty"`
	PlayState string `json:"playState,omitempty"`
	Runtime   string `json:"runtime,omitempty"`
}

// AIChatSessionDTO is one persisted agent conversation.
type AIChatSessionDTO struct {
	ID        string `json:"id"`
	Title     string `json:"title,omitempty"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type AIChatSessionListDTO struct {
	Items []AIChatSessionDTO `json:"items"`
}

type AIChatStoredMessageDTO struct {
	Events     []AIChatSSEEvent `json:"events,omitempty"`
	ID         string           `json:"id"`
	SessionID  string           `json:"sessionId"`
	Role       string           `json:"role"`
	Content    string           `json:"content"`
	ToolName   string           `json:"toolName,omitempty"`
	ToolCallID string           `json:"toolCallId,omitempty"`
	Seq        int              `json:"seq"`
	CreatedAt  string           `json:"createdAt"`
}

type AIChatSessionDetailDTO struct {
	AIChatSessionDTO
	NextCursor string                   `json:"nextCursor,omitempty"`
	Messages   []AIChatStoredMessageDTO `json:"messages"`
}

// AIAgentMovieCardDTO is a chat-ui projection of a movie already retrieved this turn.
type AIAgentMovieCardDTO struct {
	MovieID  string   `json:"movieId"`
	Title    string   `json:"title,omitempty"`
	Code     string   `json:"code,omitempty"`
	Actors   []string `json:"actors,omitempty"`
	CoverURL string   `json:"coverUrl,omitempty"`
	ThumbURL string   `json:"thumbUrl,omitempty"`
	Reason   string   `json:"reason,omitempty"`
}

// AIAgentProviderTitleDTO is a provider-search row reconciled against the
// local library. Rows with InLibrary=false never carry a MovieID.
type AIAgentProviderTitleDTO struct {
	Code      string  `json:"code,omitempty"`
	Title     string  `json:"title,omitempty"`
	Provider  string  `json:"provider,omitempty"`
	Score     float64 `json:"score,omitempty"`
	Homepage  string  `json:"homepage,omitempty"`
	InLibrary bool    `json:"inLibrary"`
	MovieID   string  `json:"movieId,omitempty"`
}

// AIEntityCandidateDTO is one safe, local-library entity candidate. Candidates
// are display-only until the user explicitly selects one in a later request.
type AIEntityCandidateDTO struct {
	Kind      string   `json:"kind"`
	MovieID   string   `json:"movieId,omitempty"`
	ActorName string   `json:"actorName,omitempty"`
	Title     string   `json:"title,omitempty"`
	Code      string   `json:"code,omitempty"`
	Aliases   []string `json:"aliases,omitempty"`
	Reason    string   `json:"reason,omitempty"`
}

// AIEntityResolutionDTO is the bounded result of resolving a user-provided
// movie code/title or actor canonical name/alias.
type AIEntityResolutionDTO struct {
	Query      string                 `json:"query"`
	Kind       string                 `json:"kind"`
	Status     string                 `json:"status"` // matched | ambiguous | unmatched
	Candidates []AIEntityCandidateDTO `json:"candidates"`
	Reason     string                 `json:"reason,omitempty"`
}

// AIEvidenceDTO tells the renderer where a tool result came from and the
// retrieval scope. It deliberately contains no raw source-page content.
type AIEvidenceDTO struct {
	Source      string            `json:"source"` // local | provider | source_page
	RetrievedAt string            `json:"retrievedAt"`
	Filters     map[string]string `json:"filters,omitempty"`
	Truncated   bool              `json:"truncated,omitempty"`
	NextCursor  string            `json:"nextCursor,omitempty"`
	Failed      bool              `json:"failed,omitempty"`
	ErrorCode   string            `json:"errorCode,omitempty"`
}

// AIChatOutcomeDTO is the explicit terminal state of one streamed turn.
type AIChatOutcomeDTO struct {
	ReasonCode string `json:"reasonCode,omitempty"` // Stable reason for localized status feedback.
	Status     string `json:"status"`               // completed | partial | needs_input | needs_confirmation | cancelled | failed
	Reason     string `json:"reason,omitempty"`
	Retryable  bool   `json:"retryable,omitempty"`
}

// AIConfirmChangeDTO is one field-level diff row on a write preview.
type AIConfirmChangeDTO struct {
	Path   string `json:"path"`
	Before any    `json:"before"`
	After  any    `json:"after"`
}

// AIActionRequest is the body for POST /api/ai/actions/{name}.
type AIActionRequest struct {
	MovieID      string `json:"movieId,omitempty"`
	Body         string `json:"body,omitempty"`
	Locale       string `json:"locale,omitempty"`
	TargetLocale string `json:"targetLocale,omitempty"`
	Range        string `json:"range,omitempty"`
	Timezone     string `json:"timezone,omitempty"`
}

// AIActionPreviewDTO is returned by a write-preview action (no persistence yet).
type AIActionPreviewDTO struct {
	Action       string               `json:"action"`
	Name         string               `json:"name"`
	SessionID    string               `json:"sessionId"`
	OriginalText string               `json:"originalText,omitempty"`
	ProposedText string               `json:"proposedText,omitempty"`
	Changes      []AIConfirmChangeDTO `json:"changes,omitempty"`
	ConfirmToken string               `json:"confirmToken,omitempty"`
	ExpiresAt    string               `json:"expiresAt,omitempty"`
	Arguments    json.RawMessage      `json:"arguments,omitempty"`
	Noop         bool                 `json:"noop,omitempty"`
}

// AIToolApplyRequest confirms a previously previewed write tool.
type AIToolApplyRequest struct {
	SessionID    string          `json:"sessionId"`
	Name         string          `json:"name"`
	Arguments    json.RawMessage `json:"arguments"`
	ConfirmToken string          `json:"confirmToken"`
}

// AIToolApplyDTO is returned after a confirmed write.
type AIToolApplyDTO struct {
	Replayed bool   `json:"replayed,omitempty"`
	OK       bool   `json:"ok"`
	Name     string `json:"name"`
	Data     any    `json:"data,omitempty"`
}

// AIChatSSEEvent is one server-sent event on POST /api/ai/chat.
// AIAnswerEvidenceDTO snapshots published facts, not reusable tool authority.
type AIAnswerEvidenceDTO struct {
	Version int                       `json:"version"`
	Items   []AIAnswerEvidenceItemDTO `json:"items"`
}

type AIAnswerEvidenceItemDTO struct {
	RefID       string         `json:"refId"`
	MovieID     string         `json:"movieId,omitempty"`
	Source      string         `json:"source"`
	Tool        string         `json:"tool"`
	RetrievedAt string         `json:"retrievedAt"`
	Truncated   bool           `json:"truncated,omitempty"`
	Fields      map[string]any `json:"fields"`
}

type AIChatSSEEvent struct {
	AnswerEvidence *AIAnswerEvidenceDTO      `json:"answerEvidence,omitempty"`
	ReceiptID      string                    `json:"receiptId,omitempty"`
	Applied        bool                      `json:"applied,omitempty"`
	Type           string                    `json:"type"`
	SessionID      string                    `json:"sessionId,omitempty"`
	MessageID      string                    `json:"messageId,omitempty"`
	Seq            int                       `json:"seq,omitempty"`
	Delta          string                    `json:"delta,omitempty"`
	ToolCallID     string                    `json:"toolCallId,omitempty"`
	Name           string                    `json:"name,omitempty"`
	OK             *bool                     `json:"ok,omitempty"`
	Summary        string                    `json:"summary,omitempty"`
	Truncated      bool                      `json:"truncated,omitempty"`
	Movies         []AIAgentMovieCardDTO     `json:"movies,omitempty"`
	ProviderRows   []AIAgentProviderTitleDTO `json:"providerRows,omitempty"`
	Resolution     *AIEntityResolutionDTO    `json:"resolution,omitempty"`
	Evidence       *AIEvidenceDTO            `json:"evidence,omitempty"`
	Outcome        *AIChatOutcomeDTO         `json:"outcome,omitempty"`
	ConfirmToken   string                    `json:"confirmToken,omitempty"`
	ExpiresAt      string                    `json:"expiresAt,omitempty"`
	Changes        []AIConfirmChangeDTO      `json:"changes,omitempty"`
	Arguments      json.RawMessage           `json:"arguments,omitempty"`
	Code           string                    `json:"code,omitempty"`
	Message        string                    `json:"message,omitempty"`
}

// PatchSettingsRequest is the body for PATCH /api/settings (partial update).
type PatchSettingsRequest struct {
	OrganizeLibrary                 *bool                   `json:"organizeLibrary,omitempty"`
	AutoLibraryWatch                *bool                   `json:"autoLibraryWatch,omitempty"`
	AutoActorProfileScrape          *bool                   `json:"autoActorProfileScrape,omitempty"`
	AutoDownloadUpdates             *bool                   `json:"autoDownloadUpdates,omitempty"`
	LaunchAtLogin                   *bool                   `json:"launchAtLogin,omitempty"`
	CuratedFrameExportFormat        *string                 `json:"curatedFrameExportFormat,omitempty"`
	CuratedFrameExportMode          *string                 `json:"curatedFrameExportMode,omitempty"`
	DefaultImportLibraryPathID      *string                 `json:"defaultImportLibraryPathId,omitempty"`
	BackupDirectory                 *string                 `json:"backupDirectory,omitempty"`
	Player                          *PatchPlayerSettingsDTO `json:"player,omitempty"`
	MetadataMovieProvider           *string                 `json:"metadataMovieProvider,omitempty"`
	ComicLibraryEnabled             *bool                   `json:"comicLibraryEnabled,omitempty"`
	AutoComicLibraryWatch           *bool                   `json:"autoComicLibraryWatch,omitempty"`
	DefaultComicImportLibraryPathID *string                 `json:"defaultComicImportLibraryPathId,omitempty"`
	ComicReader                     *ComicReaderSettingsDTO `json:"comicReader,omitempty"`
	ComicCache                      *ComicCacheSettingsDTO  `json:"comicCache,omitempty"`
	PhotoLibraryEnabled             *bool                   `json:"photoLibraryEnabled,omitempty"`
	AutoPhotoLibraryWatch           *bool                   `json:"autoPhotoLibraryWatch,omitempty"`
	DefaultPhotoImportLibraryPathID *string                 `json:"defaultPhotoImportLibraryPathId,omitempty"`
	PhotoViewer                     *PhotoViewerSettingsDTO `json:"photoViewer,omitempty"`
	PhotoCache                      *PhotoCacheSettingsDTO  `json:"photoCache,omitempty"`
	// MetadataMovieProviderChain: ordered list of providers to try in sequence; nil = no change; empty = clear (auto mode).
	MetadataMovieProviderChain *[]string `json:"metadataMovieProviderChain,omitempty"`
	// MetadataMovieScrapeMode: auto | specified | chain; switches active scrape strategy without necessarily clearing saved lists.
	MetadataMovieScrapeMode *string `json:"metadataMovieScrapeMode,omitempty"`
	// MetadataMovieStrategy: auto-global | auto-cn-friendly | custom-chain | specified.
	MetadataMovieStrategy *string `json:"metadataMovieStrategy,omitempty"`
	// Proxy: nil = no change; non-nil object replaces current proxy config.
	Proxy *ProxySettingsDTO `json:"proxy,omitempty"`
	// AIProvider: nil = no change; non-nil partial fields merge into the experimental agent provider config.
	AIProvider *PatchAIProviderSettings `json:"aiProvider,omitempty"`
	// BackendLog: nil = no change; non-empty partial fields merge into current and persist.
	BackendLog *PatchBackendLogSettings `json:"backendLog,omitempty"`
}

// MovieImportUploadFileManifest describes one file in a resumable movie import upload.
type MovieImportUploadFileManifest struct {
	RelativePath string `json:"relativePath"`
	Size         int64  `json:"size"`
	LastModified int64  `json:"lastModified,omitempty"`
}

// CreateMovieImportUploadRequest is the body for POST /api/import/movies/uploads.
type CreateMovieImportUploadRequest struct {
	Files []MovieImportUploadFileManifest `json:"files"`
}

// CheckImportMovieCodesRequest is the body for POST /api/import/movies/code-check.
type CheckImportMovieCodesRequest struct {
	Names []string `json:"names"`
}

// ImportMovieCodeMatchDTO is one library movie that matches an incoming filename.
type ImportMovieCodeMatchDTO struct {
	MovieID   string `json:"movieId"`
	Code      string `json:"code"`
	Title     string `json:"title"`
	MatchKind string `json:"matchKind"`
}

// ImportMovieCodeCheckItemDTO is the catalog-code check result for one filename.
type ImportMovieCodeCheckItemDTO struct {
	Name          string                    `json:"name"`
	ExtractedCode string                    `json:"extractedCode,omitempty"`
	Matches       []ImportMovieCodeMatchDTO `json:"matches"`
}

// ImportMovieCodeCheckDTO is the response for POST /api/import/movies/code-check.
type ImportMovieCodeCheckDTO struct {
	Items        []ImportMovieCodeCheckItemDTO `json:"items"`
	MatchedCount int                           `json:"matchedCount"`
}

// MovieImportUploadChunkDTO reports one persisted upload chunk range so clients
// can resume precisely instead of re-uploading completed chunks.
type MovieImportUploadChunkDTO struct {
	Index  int64 `json:"index"`
	Offset int64 `json:"offset"`
	Size   int64 `json:"size"`
}

// MovieImportUploadFileDTO reports server-side state for one resumable import file.
type MovieImportUploadFileDTO struct {
	FileID        string                      `json:"fileId"`
	RelativePath  string                      `json:"relativePath"`
	Size          int64                       `json:"size"`
	BytesReceived int64                       `json:"bytesReceived"`
	Complete      bool                        `json:"complete"`
	State         string                      `json:"state,omitempty"`
	Chunks        []MovieImportUploadChunkDTO `json:"chunks,omitempty"`
}

// MovieImportUploadDTO reports resumable import upload session state.
type MovieImportUploadDTO struct {
	UploadID       string                     `json:"uploadId"`
	TargetPath     string                     `json:"targetPath"`
	ChunkSize      int64                      `json:"chunkSize"`
	BytesReceived  int64                      `json:"bytesReceived"`
	TotalBytes     int64                      `json:"totalBytes"`
	State          string                     `json:"state"`
	ExpiresAt      string                     `json:"expiresAt,omitempty"`
	RecoveryStatus string                     `json:"recoveryStatus,omitempty"`
	RecoveryError  string                     `json:"recoveryError,omitempty"`
	Files          []MovieImportUploadFileDTO `json:"files"`
	Task           TaskDTO                    `json:"task"`
}

// CreateMovieImportUploadResponse is returned after creating a resumable movie import upload.
type CreateMovieImportUploadResponse = MovieImportUploadDTO

// PlayerSettingsDTO holds player/playback preferences exposed to the Settings UI.
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

// PatchPlayerSettingsDTO is the partial-update body for player settings.
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

// PlaybackProgressListDTO holds the full playback progress map.
type PlaybackProgressListDTO struct {
	Items []PlaybackProgressItemDTO `json:"items"`
}

// PutPlaybackProgressBody is the JSON body for PUT /api/playback/progress/{movieId}.
type PutPlaybackProgressBody struct {
	PositionSec float64 `json:"positionSec"`
	DurationSec float64 `json:"durationSec"`
}

// PlaybackWatchTimeDailyItemDTO is one day in the watch-time heatmap.
type PlaybackWatchTimeDailyItemDTO struct {
	DayKey     string  `json:"dayKey"`
	WatchedSec float64 `json:"watchedSec"`
}

// PlaybackWatchTimeDailyListDTO returns daily watch-time rows and summary metrics.
type PlaybackWatchTimeDailyListDTO struct {
	Items             []PlaybackWatchTimeDailyItemDTO `json:"items"`
	TotalWatchedSec   float64                         `json:"totalWatchedSec"`
	ActiveDays        int                             `json:"activeDays"`
	MaxDayWatchedSec  float64                         `json:"maxDayWatchedSec"`
	LongestStreakDays int                             `json:"longestStreakDays"`
}

// AddPlaybackWatchTimeBody records one bounded watch-time delta.
type AddPlaybackWatchTimeBody struct {
	MovieID    string  `json:"movieId"`
	DayKey     string  `json:"dayKey"`
	WatchedSec float64 `json:"watchedSec"`
}

type PersonalInsightsRange string

const (
	PersonalInsightsRange30Days  PersonalInsightsRange = "30d"
	PersonalInsightsRange90Days  PersonalInsightsRange = "90d"
	PersonalInsightsRange365Days PersonalInsightsRange = "365d"
	PersonalInsightsRangeAll     PersonalInsightsRange = "all"
)

type PersonalInsightsDimension string

const (
	PersonalInsightsDimensionActor  PersonalInsightsDimension = "actor"
	PersonalInsightsDimensionStudio PersonalInsightsDimension = "studio"
	PersonalInsightsDimensionTag    PersonalInsightsDimension = "tag"
)

type PersonalInsightsOverviewDTO struct {
	Range               PersonalInsightsRange `json:"range"`
	From                string                `json:"from"`
	To                  string                `json:"to"`
	Timezone            string                `json:"timezone"`
	GeneratedAt         string                `json:"generatedAt"`
	DataSince           *string               `json:"dataSince"`
	WatchedSeconds      float64               `json:"watchedSeconds"`
	StartedMovies       int                   `json:"startedMovies"`
	CompletedMovies     int                   `json:"completedMovies"`
	CompletionRate      *float64              `json:"completionRate"`
	CompletionThreshold float64               `json:"completionThreshold"`
	RatedMovies         int                   `json:"ratedMovies"`
	AverageUserRating   *float64              `json:"averageUserRating"`
}

type PersonalInsightsBreakdownItemDTO struct {
	Name           string  `json:"name"`
	WatchedSeconds float64 `json:"watchedSeconds"`
	MovieCount     int     `json:"movieCount"`
	ShareOfTotal   float64 `json:"shareOfTotal"`
}

type PersonalInsightsBreakdownDTO struct {
	Range               PersonalInsightsRange              `json:"range"`
	Dimension           PersonalInsightsDimension          `json:"dimension"`
	From                string                             `json:"from"`
	To                  string                             `json:"to"`
	Timezone            string                             `json:"timezone"`
	GeneratedAt         string                             `json:"generatedAt"`
	DataSince           *string                            `json:"dataSince"`
	TotalWatchedSeconds float64                            `json:"totalWatchedSeconds"`
	Attribution         string                             `json:"attribution"`
	Items               []PersonalInsightsBreakdownItemDTO `json:"items"`
	Limit               int                                `json:"limit"`
}

// PlaybackMode describes how a media file is delivered to the player.
type PlaybackMode string

// Delivery modes for media playback.
const (
	PlaybackModeDirect PlaybackMode = "direct"
	PlaybackModeHLS    PlaybackMode = "hls"
	PlaybackModeNative PlaybackMode = "native"
)

// PlaybackAudioTrackDTO describes an audio track for selection.
type PlaybackAudioTrackDTO struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Default bool   `json:"default"`
}

// PlaybackSubtitleTrackDTO describes a subtitle track for selection.
type PlaybackSubtitleTrackDTO struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Kind    string `json:"kind,omitempty"`
	Default bool   `json:"default"`
}

// PlaybackDescriptorDTO tells the frontend how to play a movie (direct stream, HLS session, or native player).
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

// CreatePlaybackSessionRequest allows the client to request a specific playback mode.
type CreatePlaybackSessionRequest struct {
	Mode             PlaybackMode `json:"mode,omitempty"`
	StartPositionSec float64      `json:"startPositionSec,omitempty"`
}

// PlaybackSessionStatusDTO is a snapshot of a playback session for diagnostics.
type PlaybackSessionStatusDTO struct {
	SessionID          string  `json:"sessionId"`
	MovieID            string  `json:"movieId"`
	SessionKind        string  `json:"sessionKind,omitempty"`
	TranscodeProfile   string  `json:"transcodeProfile,omitempty"`
	StartPositionSec   float64 `json:"startPositionSec,omitempty"`
	StartedAt          string  `json:"startedAt,omitempty"`
	LastAccessedAt     string  `json:"lastAccessedAt,omitempty"`
	ExpiresAt          string  `json:"expiresAt,omitempty"`
	FinishedAt         string  `json:"finishedAt,omitempty"`
	State              string  `json:"state,omitempty"`
	LastError          string  `json:"lastError,omitempty"`
	EncoderSpeed       string  `json:"encoderSpeed,omitempty"`
	WrittenDurationSec float64 `json:"writtenDurationSec,omitempty"`
	LastSeekKind       string  `json:"lastSeekKind,omitempty"`
}

// PlaybackSessionListDTO lists recent active or archived playback sessions.
type PlaybackSessionListDTO struct {
	Items []PlaybackSessionStatusDTO `json:"items"`
}

// NativePlaybackLaunchRequest optionally specifies a start position for native player launch.
type NativePlaybackLaunchRequest struct {
	StartPositionSec float64 `json:"startPositionSec,omitempty"`
}

// NativePlaybackLaunchDTO reports the result of launching an external player.
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
	ID          string                 `json:"id"`
	MovieID     string                 `json:"movieId"`
	Title       string                 `json:"title"`
	Code        string                 `json:"code"`
	Actors      []string               `json:"actors"`
	PositionSec float64                `json:"positionSec"`
	CapturedAt  string                 `json:"capturedAt"`
	Tags        []string               `json:"tags"`
	Motion      *CuratedFrameMotionDTO `json:"motion,omitempty"`
}

// CuratedFrameMotionDTO describes an app-owned motion artifact associated with a frame.
type CuratedFrameMotionDTO struct {
	Status       string  `json:"status"`
	ContentType  string  `json:"contentType"`
	DurationSec  float64 `json:"durationSec"`
	Width        int     `json:"width"`
	Height       int     `json:"height"`
	FPS          int     `json:"fps"`
	FileSize     int64   `json:"fileSize"`
	ArtifactURL  string  `json:"artifactUrl,omitempty"`
	ErrorMessage string  `json:"errorMessage,omitempty"`
	CreatedAt    string  `json:"createdAt,omitempty"`
	UpdatedAt    string  `json:"updatedAt,omitempty"`
}

// CuratedFramesListDTO is a paginated curated frame listing.
type CuratedFramesListDTO struct {
	NextCursor string                `json:"nextCursor,omitempty"`
	Items      []CuratedFrameItemDTO `json:"items"`
	Total      int                   `json:"total"`
	Limit      int                   `json:"limit"`
	Offset     int                   `json:"offset"`
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

// CuratedFrameStatsDTO reports the total curated frame count.
type CuratedFrameStatsDTO struct {
	Total int `json:"total"`
}

// CuratedFrameFacetItemDTO is one (name, count) facet row.
type CuratedFrameFacetItemDTO struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// CuratedFrameFacetListDTO holds a list of curated frame facet items.
type CuratedFrameFacetListDTO struct {
	Items []CuratedFrameFacetItemDTO `json:"items"`
}

// PatchCuratedFrameTagsBody is the JSON body for PATCH /api/curated-frames/{id}/tags.
type PatchCuratedFrameTagsBody struct {
	Tags []string `json:"tags"`
}

// PostCuratedFramesExportBody is the JSON body for POST /api/curated-frames/export.
type PostCuratedFramesExportBody struct {
	IDs       []string `json:"ids"`
	ActorName string   `json:"actorName,omitempty"`
	// Format is "jpg" (default), "webp", or "png". The export handler may also accept "jpeg" as a compatibility alias.
	Format string `json:"format,omitempty"`
}

// CreateMovieClipBody is the JSON body for POST /api/library/movies/{movieId}/clips.
type CreateMovieClipBody struct {
	StartSec       float64 `json:"startSec"`
	EndSec         float64 `json:"endSec"`
	Format         string  `json:"format,omitempty"`
	FPS            int     `json:"fps,omitempty"`
	Width          int     `json:"width,omitempty"`
	CuratedFrameID string  `json:"curatedFrameId,omitempty"`
}

// ExtractMovieFrameBody requests a source-file frame at an absolute media time.
type ExtractMovieFrameBody struct {
	PositionSec float64 `json:"positionSec"`
}

// PlayedMoviesListDTO is returned by GET /api/library/played-movies.
type PlayedMoviesListDTO struct {
	MovieIDs []string `json:"movieIds"`
}

// HomepageDailyRecommendationsDTO carries the hero and recommendation movie IDs for a UTC day.
type HomepageDailyRecommendationsDTO struct {
	DateUTC                string                          `json:"dateUtc"`
	GeneratedAt            string                          `json:"generatedAt"`
	GenerationVersion      string                          `json:"generationVersion,omitempty"`
	HeroMovieIDs           []string                        `json:"heroMovieIds"`
	RecommendationMovieIDs []string                        `json:"recommendationMovieIds"`
	Recommendations        []HomepageRecommendationItemDTO `json:"recommendations"`
}

type HomepageRecommendationReasonDTO struct {
	Code        string `json:"code"`
	EntityType  string `json:"entityType,omitempty"`
	EntityValue string `json:"entityValue,omitempty"`
}

type HomepageRecommendationFeedbackEffectDTO struct {
	FeedbackID  string `json:"feedbackId"`
	TargetType  string `json:"targetType"`
	TargetValue string `json:"targetValue"`
	Effect      string `json:"effect"`
}

type HomepageRecommendationItemDTO struct {
	MovieID         string                                    `json:"movieId"`
	Reasons         []HomepageRecommendationReasonDTO         `json:"reasons"`
	FeedbackEffects []HomepageRecommendationFeedbackEffectDTO `json:"feedbackEffects"`
}

// HomepageDailyRecommendationsRefreshOptions customizes a forced homepage snapshot refresh.
type HomepageDailyRecommendationsRefreshOptions struct {
	// PreserveHeroMovieIDs keeps the current homepage hero slate while regenerating today's recommendations.
	PreserveHeroMovieIDs []string `json:"preserveHeroMovieIds,omitempty"`
	// ExcludeRecommendationMovieIDs avoids returning the recommendation rail currently visible to the caller.
	ExcludeRecommendationMovieIDs []string `json:"excludeRecommendationMovieIds,omitempty"`
}

type RecommendationFeedbackDTO struct {
	ID            string `json:"id"`
	Action        string `json:"action"`
	TargetType    string `json:"targetType"`
	TargetValue   string `json:"targetValue"`
	SourceMovieID string `json:"sourceMovieId"`
	ExpiresAt     string `json:"expiresAt,omitempty"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
}

type RecommendationFeedbackListDTO struct {
	Items []RecommendationFeedbackDTO `json:"items"`
}

type CreateRecommendationFeedbackBody struct {
	Action        string `json:"action"`
	TargetType    string `json:"targetType"`
	TargetValue   string `json:"targetValue"`
	SourceMovieID string `json:"sourceMovieId"`
	DurationDays  int    `json:"durationDays,omitempty"`
}

// TaskDTO represents an async background task (scan, scrape, download).
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

// TaskEventDTO wraps a task status update as an event payload.
type TaskEventDTO struct {
	Task TaskDTO `json:"task"`
}

// Command and event type constants for the stdio JSONL transport.
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

	TaskTypeImportMovies         = "import.movies"
	TaskTypeLibraryHealthRepair  = "library.health.repair"
	TaskTypeLibraryHealthCleanup = "library.health.cleanup"
	TaskTypeScanComics           = "scan.comics"
	TaskTypeScanPhotos           = "scan.photos"
	TaskTypeImportComics         = "import.comics"
	TaskTypeImportPhotos         = "import.photos"
	TaskTypeComicCacheCleanup    = "comic.cache.cleanup"

	ErrorCodeBadRequest                           = "COMMON_BAD_REQUEST"
	ErrorCodeForbidden                            = "COMMON_FORBIDDEN"
	ErrorCodeNotFound                             = "COMMON_NOT_FOUND"
	ErrorCodeInternal                             = "COMMON_INTERNAL"
	ErrorCodeUnsupported                          = "COMMON_UNSUPPORTED_COMMAND"
	ErrorCodeLibraryFetch                         = "LIBRARY_FETCH_FAILED"
	ErrorCodeScanStart                            = "SCAN_START_FAILED"
	ErrorCodeScanWalk                             = "SCAN_WALK_FAILED"
	ErrorCodeScanCancelled                        = "SCAN_CANCELLED"
	ErrorCodeScraperInit                          = "SCRAPER_INIT_FAILED"
	ErrorCodeScraperRun                           = "SCRAPER_RUN_FAILED"
	ErrorCodeAssetDownload                        = "ASSET_DOWNLOAD_FAILED"
	ErrorCodeConflict                             = "COMMON_CONFLICT"
	ErrorCodeSavedViewInvalid                     = "SAVED_VIEW_INVALID"
	ErrorCodeSavedViewNameConflict                = "SAVED_VIEW_NAME_CONFLICT"
	ErrorCodeSavedViewLimit                       = "SAVED_VIEW_LIMIT_REACHED"
	ErrorCodeRecommendationFeedbackInvalid        = "RECOMMENDATION_FEEDBACK_INVALID"
	ErrorCodeRecommendationFeedbackTargetNotFound = "RECOMMENDATION_FEEDBACK_TARGET_NOT_FOUND"
	ErrorCodeAIProviderUnavailable                = "AI_PROVIDER_UNAVAILABLE"
	ErrorCodeAIChatFailed                         = "AI_CHAT_FAILED"
	ErrorCodeAIToolInvalidArgs                    = "AI_TOOL_INVALID_ARGS"
	ErrorCodeAIToolNotFound                       = "AI_TOOL_NOT_FOUND"
	ErrorCodeAIConfirmRequired                    = "AI_CONFIRM_REQUIRED"
	ErrorCodeAIConfirmExpired                     = "AI_CONFIRM_EXPIRED"
	ErrorCodeAIRateLimited                        = "AI_RATE_LIMITED"
	ErrorCodeRecommendationFeedbackLimit          = "RECOMMENDATION_FEEDBACK_LIMIT_REACHED"
	ErrorCodeActorMergeInvalid                    = "ACTOR_MERGE_INVALID"
	ErrorCodeActorMergeNotFound                   = "ACTOR_MERGE_NOT_FOUND"
	ErrorCodeActorMergeSelf                       = "ACTOR_MERGE_SELF"
	ErrorCodeActorMergeSourceAlias                = "ACTOR_MERGE_SOURCE_IS_ALIAS"
	ErrorCodeActorMergeConflict                   = "ACTOR_MERGE_CONFLICT"
	ErrorCodeActorMergeStalePreview               = "ACTOR_MERGE_STALE_PREVIEW"
	ErrorCodeActorMergeLinkLimit                  = "ACTOR_MERGE_LINK_LIMIT"
	ErrorCodePersonalInsightsInvalidRange         = "INSIGHTS_INVALID_RANGE"
	ErrorCodePersonalInsightsInvalidTimezone      = "INSIGHTS_INVALID_TIMEZONE"
	ErrorCodePersonalInsightsInvalidDimension     = "INSIGHTS_INVALID_DIMENSION"
	ErrorCodePersonalInsightsInvalidLimit         = "INSIGHTS_INVALID_LIMIT"

	ErrorCodeAuthLocked      = "AUTH_LOCKED"
	ErrorCodeAuthInvalidPIN  = "AUTH_INVALID_PIN"
	ErrorCodeAuthRateLimited = "AUTH_RATE_LIMITED"

	ErrorCodeImportSourceUnavailable   = "IMPORT_SOURCE_UNAVAILABLE"
	ErrorCodeImportTargetNotConfigured = "IMPORT_TARGET_NOT_CONFIGURED"
	ErrorCodeImportTargetUnavailable   = "IMPORT_TARGET_UNAVAILABLE"
	ErrorCodeImportNotEnoughSpace      = "IMPORT_NOT_ENOUGH_SPACE"
	ErrorCodeImportConflict            = "IMPORT_CONFLICT"
	ErrorCodeImportCopyFailed          = "IMPORT_COPY_FAILED"
	ErrorCodeImportCancelled           = "IMPORT_CANCELLED"
	ErrorCodeImportScanFailed          = "IMPORT_SCAN_FAILED"
	ErrorCodeImportUploadPersistFailed = "IMPORT_UPLOAD_PERSIST_FAILED"
	ErrorCodeImportUploadUnrecoverable = "IMPORT_UPLOAD_UNRECOVERABLE"
	ErrorCodeImportUploadExpired       = "IMPORT_UPLOAD_EXPIRED"

	ErrorCodeHealthRepairConfirmationRequired = "HEALTH_REPAIR_CONFIRMATION_REQUIRED"
	ErrorCodeHealthRepairNoFindings           = "HEALTH_REPAIR_NO_FINDINGS"
	ErrorCodeHealthRepairPersistFailed        = "HEALTH_REPAIR_PERSIST_FAILED"
	ErrorCodeHealthRepairInterrupted          = "HEALTH_REPAIR_INTERRUPTED"
	ErrorCodeHealthCleanupFailed              = "HEALTH_CLEANUP_FAILED"

	ErrorCodeComicLibraryDisabled     = "COMIC_LIBRARY_DISABLED"
	ErrorCodeComicPathNotConfigured   = "COMIC_PATH_NOT_CONFIGURED"
	ErrorCodeComicPathNotFound        = "COMIC_PATH_NOT_FOUND"
	ErrorCodeComicArchiveUnsupported  = "COMIC_ARCHIVE_UNSUPPORTED"
	ErrorCodeComicArchiveEmpty        = "COMIC_ARCHIVE_EMPTY"
	ErrorCodeComicArchiveReadFailed   = "COMIC_ARCHIVE_READ_FAILED"
	ErrorCodeComicBookNotFound        = "COMIC_BOOK_NOT_FOUND"
	ErrorCodeComicPageNotFound        = "COMIC_PAGE_NOT_FOUND"
	ErrorCodeComicImportTargetMissing = "COMIC_IMPORT_TARGET_MISSING"
	ErrorCodeComicImportConflict      = "COMIC_IMPORT_CONFLICT"
	ErrorCodeComicCacheCleanupFailed  = "COMIC_CACHE_CLEANUP_FAILED"

	ErrorCodePhotoPathNotConfigured   = "PHOTO_PATH_NOT_CONFIGURED"
	ErrorCodePhotoPathNotFound        = "PHOTO_PATH_NOT_FOUND"
	ErrorCodePhotoLibraryDisabled     = "PHOTO_LIBRARY_DISABLED"
	ErrorCodePhotoArchiveUnsupported  = "PHOTO_ARCHIVE_UNSUPPORTED"
	ErrorCodePhotoImportTargetMissing = "PHOTO_IMPORT_TARGET_MISSING"
	ErrorCodePhotoImportConflict      = "PHOTO_IMPORT_CONFLICT"
	ErrorCodePhotoArchiveEmpty        = "PHOTO_ARCHIVE_EMPTY"
	ErrorCodePhotoArchiveReadFailed   = "PHOTO_ARCHIVE_READ_FAILED"
	ErrorCodePhotoBookNotFound        = "PHOTO_BOOK_NOT_FOUND"
	ErrorCodePhotoPageNotFound        = "PHOTO_PAGE_NOT_FOUND"

	ErrorCodeAppUpdateDownloadFailed = "APP_UPDATE_DOWNLOAD_FAILED"
	ErrorCodeAppUpdateInstallFailed  = "APP_UPDATE_INSTALL_FAILED"

	ErrorCodeBackupInvalidRequest  = "BACKUP_INVALID_REQUEST"
	ErrorCodeBackupConflict        = "BACKUP_CONFLICT"
	ErrorCodeBackupCreateFailed    = "BACKUP_CREATE_FAILED"
	ErrorCodeBackupVerifyFailed    = "BACKUP_VERIFY_FAILED"
	ErrorCodeBackupPreflightFailed = "BACKUP_PREFLIGHT_FAILED"

	// Curated frames export
	ErrorCodeCuratedExportActorMismatch = "CURATED_EXPORT_ACTOR_MISMATCH"

	// Provider health check errors
	ErrorCodeProviderNotFound   = "PROVIDER_NOT_FOUND"
	ErrorCodeProviderPingFailed = "PROVIDER_PING_FAILED"
)

// ProviderHealthStatus indicates the availability status of a metadata provider.
type ProviderHealthStatus string

// Provider health status values.
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

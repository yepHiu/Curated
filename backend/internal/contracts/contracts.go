package contracts

import "encoding/json"

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
	Version      string `json:"version"`
	Transport    string `json:"transport"`
	DatabasePath string `json:"databasePath"`
}

type ListMoviesRequest struct {
	Mode   string `json:"mode,omitempty"`
	Query  string `json:"query,omitempty"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
}

type GetMovieDetailRequest struct {
	MovieID string `json:"movieId"`
}

type StartScanRequest struct {
	Paths []string `json:"paths,omitempty"`
}

type GetTaskStatusRequest struct {
	TaskID string `json:"taskId"`
}

type ScanFileResultDTO struct {
	TaskID   string `json:"taskId"`
	Path     string `json:"path"`
	FileName string `json:"fileName"`
	Number   string `json:"number,omitempty"`
	MovieID  string `json:"movieId,omitempty"`
	Status   string `json:"status"`
	Reason   string `json:"reason,omitempty"`
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
	RuntimeMinutes int      `json:"runtimeMinutes"`
	Rating         float64  `json:"rating"`
	IsFavorite     bool     `json:"isFavorite"`
	AddedAt        string   `json:"addedAt"`
	Location       string   `json:"location"`
	Resolution     string   `json:"resolution"`
	Year           int      `json:"year"`
}

type MovieDetailDTO struct {
	MovieListItemDTO
	Summary string `json:"summary"`
}

type MoviesPageDTO struct {
	Items  []MovieListItemDTO `json:"items"`
	Total  int                `json:"total"`
	Limit  int                `json:"limit"`
	Offset int                `json:"offset"`
}

type LibraryPathDTO struct {
	ID    string `json:"id"`
	Path  string `json:"path"`
	Title string `json:"title"`
}

type SettingsDTO struct {
	LibraryPaths        []LibraryPathDTO  `json:"libraryPaths"`
	ScanIntervalSeconds int               `json:"scanIntervalSeconds"`
	Player              PlayerSettingsDTO `json:"player"`
}

type PlayerSettingsDTO struct {
	HardwareDecode bool `json:"hardwareDecode"`
}

type TaskDTO struct {
	TaskID       string         `json:"taskId"`
	Type         string         `json:"type"`
	Status       string         `json:"status"`
	CreatedAt    string         `json:"createdAt"`
	StartedAt    string         `json:"startedAt,omitempty"`
	FinishedAt   string         `json:"finishedAt,omitempty"`
	Progress     int            `json:"progress"`
	Message      string         `json:"message,omitempty"`
	ErrorCode    string         `json:"errorCode,omitempty"`
	ErrorMessage string         `json:"errorMessage,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
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
	ErrorCodeScraperInit   = "SCRAPER_INIT_FAILED"
	ErrorCodeScraperRun    = "SCRAPER_RUN_FAILED"
	ErrorCodeAssetDownload = "ASSET_DOWNLOAD_FAILED"
)

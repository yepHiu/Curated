// Package core is the Agent tool gateway: registry, permission, confirm tokens,
// rate limits, projection, and audit. It must not import business packages.
package core

import (
	"context"
	"encoding/json"
	"time"
)

const (
	PermissionRead         = "read"
	PermissionWritePreview = "write-preview"
	PermissionWriteApply   = "write-apply"

	DomainQuery      = "query"
	DomainPresent    = "present"
	DomainUserWrite  = "user-write"
	DomainGovernance = "governance"
	DomainTask       = "task"

	PresentMoviesName               = "present_movies"
	PresentMoviesMaxItems           = 6
	SearchProviderTitlesName        = "search_provider_titles"
	GetSourcePageName               = "get_source_page"
	ProviderSearchDefaultLimit      = 15
	ProviderSearchMaxLimit          = 25
	MaxSourcePageBytes              = 32 * 1024
	SaveMovieCommentName            = "save_movie_comment"
	UpdateMovieDisplayOverridesName = "update_movie_display_overrides"
	CreateSavedViewName             = "create_saved_view"
	MaxMovieSummaryBytes            = 120_000

	ChannelChat   = "chat"
	ChannelAction = "action"
	ChannelMCP    = "mcp"

	ResultOK        = "ok"
	ResultError     = "error"
	ResultPreviewed = "previewed"
	ResultConfirmed = "confirmed"
	ResultRejected  = "rejected"

	SanitizeFull      = "full"
	SanitizeSanitized = "sanitized"
	SanitizeMinimal   = "minimal"

	DefaultListLimit   = 50
	DefaultStepLimit   = 0 // Unlimited unless the user opts into a limit.
	DefaultWritePerMin = 10
	DefaultBatchLimit  = 25
	ReadTimeout        = 10 * time.Second
	WriteTimeout       = 30 * time.Second
	ConfirmTTL         = 10 * time.Minute

	ToolsetVersion = "1.0.0"
)

// ToolDefinition is one registered atomic capability.
type ToolDefinition struct {
	Name         string
	Description  string
	ParamsSchema Schema
	Permission   string
	Domain       string
	Version      string
	Deprecated   bool
	Handler      Handler
	// Apply runs the write-apply branch. When set, a call with ConfirmTok
	// consumes the preview token and executes Apply instead of Handler.
	Apply Handler
	// NormalizeArgs rewrites model JSON before schema validation and
	// confirm hashing. Optional.
	NormalizeArgs func(json.RawMessage) (json.RawMessage, error)
}

// Handler executes a validated tool call. Write tools may return preview data
// without persisting when Permission is write-preview.
type Handler func(ctx context.Context, call Call) (Result, error)

// Call is one invocation entering the gateway.
type Call struct {
	// Preconditions come only from the server-issued preview ticket.
	Preconditions []Change
	Name          string
	Args          json.RawMessage
	SessionID     string
	Channel       string
	ConfirmTok    string
	Sanitize      string
}

// Change is one field-level preview row for write-preview.
type Change struct {
	Path   string `json:"path"`
	Before any    `json:"before"`
	After  any    `json:"after"`
}

// ToolError is a stable tool-level error surfaced to the model and audit log.
type ToolError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *ToolError) Error() string { return e.Message }

// Result is the unified tool envelope.
type Result struct {
	Answer        *AnswerSubmission `json:"-"`
	AnswerRefs    []AnswerRefHint   `json:"answerRefs,omitempty"`
	Preconditions []Change          `json:"-"`
	OK            bool              `json:"ok"`
	Data          any               `json:"data,omitempty"`
	Error         *ToolError        `json:"error,omitempty"`
	Truncated     bool              `json:"truncated,omitempty"`
	NextCursor    string            `json:"nextCursor,omitempty"`
	Changes       []Change          `json:"changes,omitempty"`
	ConfirmToken  string            `json:"confirmToken,omitempty"`
	ExpiresAt     string            `json:"expiresAt,omitempty"`
	// ConfirmArgs is the JSON hashed into the preview token. Not sent to the model.
	ConfirmArgs json.RawMessage `json:"-"`
}

// AuditRecord is an append-only invocation row.
type AuditRecord struct {
	Channel     string
	SessionID   string
	ToolName    string
	Permission  string
	ArgsSummary string
	Result      string
	ErrorCode   string
	DurationMs  int64
}

// AuditSink persists invocation records. Implementations must not mutate records.
type AuditSink interface {
	RecordInvocation(ctx context.Context, rec AuditRecord) error
}

// Settings controls permission overlay switches. Zero value is the safe default:
// writes disabled, preview still allowed, MCP read-only.
type Settings struct {
	Disabled           bool
	ReadOnly           bool
	GlobalWriteLimit   bool
	WriteEnabled       bool
	LowRiskDirectWrite bool
	MCPWriteExposed    bool
	StepLimit          int
	WritePerMinute     int
}

func (s Settings) stepLimit() int {
	if s.StepLimit > 0 {
		return s.StepLimit
	}
	return DefaultStepLimit
}

func (s Settings) writePerMinute() int {
	if s.WritePerMinute > 0 {
		return s.WritePerMinute
	}
	return DefaultWritePerMin
}

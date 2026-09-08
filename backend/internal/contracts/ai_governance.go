package contracts

type AIGovernanceDTO struct {
	Enabled        bool   `json:"enabled"`
	ReadOnly       bool   `json:"readOnly"`
	Privacy        string `json:"privacy"`
	StepLimit      int    `json:"stepLimit"` // 0 = unlimited; otherwise 1–30 tool calls per turn.
	WritePerMinute int    `json:"writePerMinute"`
	RetentionDays  int    `json:"retentionDays"`
}

type AIRunDTO struct {
	ID               string `json:"id"`
	StartedAt        string `json:"startedAt"`
	Channel          string `json:"channel"`
	Action           string `json:"action"`
	SessionID        string `json:"sessionId"`
	Provider         string `json:"provider"`
	Model            string `json:"model"`
	PromptVersion    string `json:"promptVersion"`
	Status           string `json:"status"`
	ErrorCode        string `json:"errorCode"`
	DurationMs       int64  `json:"durationMs"`
	FirstTextMs      *int64 `json:"firstTextMs"`
	ModelCalls       int    `json:"modelCalls"`
	UsageCalls       int    `json:"usageCalls"`
	ToolCalls        int    `json:"toolCalls"`
	PromptTokens     int64  `json:"promptTokens"`
	CompletionTokens int64  `json:"completionTokens"`
	TotalTokens      int64  `json:"totalTokens"`
}

type AIReportQuery struct {
	Days, Limit, Offset int
	Channel, Status     string
}
type AISummaryDTO struct {
	Runs             int      `json:"runs"`
	Failed           int      `json:"failed"`
	Partial          int      `json:"partial"`
	Cancelled        int      `json:"cancelled"`
	ModelCalls       int      `json:"modelCalls"`
	UsageCalls       int      `json:"usageCalls"`
	ToolCalls        int      `json:"toolCalls"`
	PromptTokens     int64    `json:"promptTokens"`
	CompletionTokens int64    `json:"completionTokens"`
	TotalTokens      int64    `json:"totalTokens"`
	AvgDurationMs    *float64 `json:"avgDurationMs"`
	AvgFirstTextMs   *float64 `json:"avgFirstTextMs"`
}
type AIReportDTO struct {
	Summary AISummaryDTO `json:"summary"`
	Items   []AIRunDTO   `json:"items"`
	Total   int          `json:"total"`
	Limit   int          `json:"limit"`
	Offset  int          `json:"offset"`
}
type AIAuditDTO struct {
	ID         string `json:"id"`
	CreatedAt  string `json:"createdAt"`
	Channel    string `json:"channel"`
	SessionID  string `json:"sessionId"`
	Tool       string `json:"tool"`
	Permission string `json:"permission"`
	Result     string `json:"result"`
	ErrorCode  string `json:"errorCode"`
	DurationMs int64  `json:"durationMs"`
}
type AIAuditPageDTO struct {
	Items  []AIAuditDTO `json:"items"`
	Total  int          `json:"total"`
	Limit  int          `json:"limit"`
	Offset int          `json:"offset"`
}
type AICleanupDTO struct {
	Runs     int64 `json:"runs"`
	Audit    int64 `json:"audit"`
	Receipts int64 `json:"receipts"`
}

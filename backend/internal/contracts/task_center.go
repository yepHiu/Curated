package contracts

// TaskCenterTask is a read-only projection. Business workers own all mutations.
type TaskCenterTask struct {
	Key          string                 `json:"key"`
	Kind         string                 `json:"kind"`
	ID           string                 `json:"id"`
	State        string                 `json:"state"`
	Stage        string                 `json:"stage"`
	Completed    int                    `json:"completed"`
	Total        int                    `json:"total"`
	Unit         string                 `json:"unit"`
	CreatedAt    string                 `json:"createdAt"`
	UpdatedAt    string                 `json:"updatedAt"`
	CanManage    bool                   `json:"canManage"`
	Subtitle     *SubtitleJob           `json:"subtitle,omitempty"`
	Organization *TagOrganizationJobDTO `json:"organization,omitempty"`
	AudioTrack   *int                   `json:"audioTrack,omitempty"`
	TextModel    string                 `json:"textModel,omitempty"`
}

type TaskCenterSummary struct {
	Total     int `json:"total"`
	Running   int `json:"running"`
	Queued    int `json:"queued"`
	Waiting   int `json:"waiting"`
	Attention int `json:"attention"`
	History   int `json:"history"`
}

type TaskCenterPage struct {
	Items      []TaskCenterTask  `json:"items"`
	NextCursor string            `json:"nextCursor"`
	Summary    TaskCenterSummary `json:"summary"`
}

// TaskCenterState keeps a rate-limit wait distinct from a user-action block.
func TaskCenterState(state, stage string) string {
	if state == "running" && stage == "waiting_quota" {
		return "waiting"
	}
	switch state {
	case "completed":
		return "succeeded"
	case "partial_failed":
		return "partial"
	case "queued", "running", "blocked", "failed", "cancelled":
		return state
	default:
		return "unknown"
	}
}

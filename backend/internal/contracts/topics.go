package contracts

// LibraryTopicDTO 表示由用户标签构成的稳定题材入口，不包含 NFO 写入能力。
type LibraryTopicDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MovieCount  int    `json:"movieCount"`
	Hidden      bool   `json:"hidden"`
}

// HomepageTopicGroupDTO 将题材和有界代表影片一起返回。
type HomepageTopicGroupDTO struct {
	Topic  LibraryTopicDTO    `json:"topic"`
	Movies []MovieListItemDTO `json:"movies"`
}

// TagOrganizationRequest 固定影片范围和标签语言，不接受元数据写字段。
type TagOrganizationRequest struct {
	Locale    string   `json:"locale,omitempty"`
	Scope     string   `json:"scope"`
	MovieIDs  []string `json:"movieIds,omitempty"`
	RequestID string   `json:"requestId"`
}

// TagOrganizationStatsDTO partitions active movies by source freshness.
// Unresolved is a subset of Organized: analyzed successfully without a match.
type TagOrganizationStatsDTO struct {
	Total       int `json:"total"`
	Organized   int `json:"organized"`
	Unorganized int `json:"unorganized"`
	Outdated    int `json:"outdated"`
	Unresolved  int `json:"unresolved"`
}

// TagOrganizationJobDTO 保存可恢复整理的真实进度和触发来源。
type TagOrganizationJobDTO struct {
	Locale              string `json:"locale"`
	VocabularyProcessed int    `json:"vocabularyProcessed"`
	VocabularyReady     bool   `json:"vocabularyReady"`
	ID                  string `json:"id"`
	TaskID              string `json:"taskId"`
	Status              string `json:"status"`
	Stage               string `json:"stage"`
	TriggerReason       string `json:"triggerReason"`
	Total               int    `json:"total"`
	Processed           int    `json:"processed"`
	Succeeded           int    `json:"succeeded"`
	Unresolved          int    `json:"unresolved"`
	Failed              int    `json:"failed"`
	Revision            int64  `json:"revision"`
	CreatedAt           string `json:"createdAt"`
	UpdatedAt           string `json:"updatedAt"`
	Error               string `json:"error,omitempty"`
}

// TopicEvidenceDTO 只提供已核对的原文片段，不暴露路径或完整模型输入。
type TopicEvidenceDTO struct {
	Topic string `json:"topic"`
	Field string `json:"field"`
	Quote string `json:"quote"`
}

// TagOrganizationItemDTO exposes only validated excerpts and the movie display identity.
type TagOrganizationItemDTO struct {
	Title    string             `json:"title"`
	Evidence []TopicEvidenceDTO `json:"evidence"`
	MovieID  string             `json:"movieId"`
	Status   string             `json:"status"`
	Reason   string             `json:"reason"`
}

// TagOrganizationUndoDTO 明确区分成功恢复和因新修改而跳过的影片。
type TagOrganizationUndoDTO struct {
	Restored  int `json:"restored"`
	Conflicts int `json:"conflicts"`
}

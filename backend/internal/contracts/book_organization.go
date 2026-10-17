package contracts

type BookOrganizationRequest struct {
	Kind      string   `json:"kind"`
	BookIDs   []string `json:"bookIds"`
	Locale    string   `json:"locale"`
	RequestID string   `json:"requestId"`
}
type BookOrganizationJob struct {
	ID            string `json:"id"`
	Kind          string `json:"kind"`
	Locale        string `json:"locale"`
	Status        string `json:"status"`
	Stage         string `json:"stage"`
	Total         int    `json:"total"`
	Processed     int    `json:"processed"`
	PendingReview int    `json:"pendingReview"`
	Applied       int    `json:"applied"`
	Failed        int    `json:"failed"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
}
type BookOrganizationItem struct {
	BookID   string            `json:"bookId"`
	Title    string            `json:"title"`
	Status   string            `json:"status"`
	Revision int               `json:"revision"`
	Before   BookMetadataState `json:"before"`
	Metadata BookMetadata      `json:"metadata"`
	Reason   string            `json:"reason"`
}
type BookOrganizationReview struct {
	BookID   string `json:"bookId"`
	Revision int    `json:"revision"`
	Decision string `json:"decision"`
}

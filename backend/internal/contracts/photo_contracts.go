package contracts

type PhotoLibraryPathDTO struct {
	ID                      string `json:"id"`
	Path                    string `json:"path"`
	Title                   string `json:"title"`
	FirstLibraryScanPending bool   `json:"firstLibraryScanPending"`
}

type AddPhotoLibraryPathRequest struct {
	Path  string `json:"path"`
	Title string `json:"title,omitempty"`
}

type AddPhotoLibraryPathResponse struct {
	PhotoLibraryPathDTO
	ScanTask *TaskDTO `json:"scanTask,omitempty"`
}

type UpdatePhotoLibraryPathRequest struct {
	Title string `json:"title"`
}

type PhotoBookListItemDTO struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	Tags             []string `json:"tags"`
	Rating           *float64 `json:"rating,omitempty"`
	IsFavorite       bool     `json:"isFavorite"`
	PageCount        int      `json:"pageCount"`
	CurrentPageIndex int      `json:"currentPageIndex"`
	CoverURL         string   `json:"coverUrl,omitempty"`
	SourceFileName   string   `json:"sourceFileName"`
	Location         string   `json:"location"`
	AddedAt          string   `json:"addedAt"`
	UpdatedAt        string   `json:"updatedAt"`
	LastViewedAt     string   `json:"lastViewedAt,omitempty"`
	CompletedAt      string   `json:"completedAt,omitempty"`
}

type PhotoBookDetailDTO struct {
	PhotoBookListItemDTO
	Pages []PhotoPageDTO `json:"pages"`
}

type ListPhotoBooksRequest struct {
	Query    string `json:"query,omitempty"`
	Tag      string `json:"tag,omitempty"`
	Favorite *bool  `json:"favorite,omitempty"`
	Limit    int    `json:"limit,omitempty"`
	Offset   int    `json:"offset,omitempty"`
}

type PhotoBooksPageDTO struct {
	Items  []PhotoBookListItemDTO `json:"items"`
	Total  int                    `json:"total"`
	Limit  int                    `json:"limit"`
	Offset int                    `json:"offset"`
}

type PhotoPageDTO struct {
	PhotoID   string `json:"photoId"`
	Index     int    `json:"index"`
	EntryPath string `json:"entryPath"`
	FileName  string `json:"fileName"`
	ImageExt  string `json:"imageExt,omitempty"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
	ImageURL  string `json:"imageUrl,omitempty"`
	ThumbURL  string `json:"thumbUrl,omitempty"`
}

type PhotoViewerSettingsDTO struct {
	Mode      string `json:"mode"`
	Fit       string `json:"fit"`
	Direction string `json:"direction"`
}

type PhotoCacheSettingsDTO struct {
	MaxBytes int64 `json:"maxBytes"`
}

package contracts

type ComicLibraryPathDTO struct {
	ID                      string `json:"id"`
	Path                    string `json:"path"`
	Title                   string `json:"title"`
	FirstLibraryScanPending bool   `json:"firstLibraryScanPending"`
}

type ComicBookListItemDTO struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	Tags             []string `json:"tags"`
	Rating           *float64 `json:"rating,omitempty"`
	IsFavorite       bool     `json:"isFavorite"`
	ReadStatus       string   `json:"readStatus"`
	PageCount        int      `json:"pageCount"`
	CurrentPageIndex int      `json:"currentPageIndex"`
	CoverURL         string   `json:"coverUrl,omitempty"`
	SourceFileName   string   `json:"sourceFileName"`
	Location         string   `json:"location"`
	AddedAt          string   `json:"addedAt"`
	UpdatedAt        string   `json:"updatedAt"`
	LastReadAt       string   `json:"lastReadAt,omitempty"`
	CompletedAt      string   `json:"completedAt,omitempty"`
}

type ComicBookDetailDTO struct {
	ComicBookListItemDTO
	Pages []ComicPageDTO `json:"pages"`
}

type ComicPageDTO struct {
	ComicID   string `json:"comicId"`
	Index     int    `json:"index"`
	EntryPath string `json:"entryPath"`
	FileName  string `json:"fileName"`
	ImageExt  string `json:"imageExt,omitempty"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
	ImageURL  string `json:"imageUrl,omitempty"`
	ThumbURL  string `json:"thumbUrl,omitempty"`
}

type ComicReaderSettingsDTO struct {
	Mode      string `json:"mode"`
	Fit       string `json:"fit"`
	Direction string `json:"direction"`
}

type ComicCacheSettingsDTO struct {
	MaxBytes int64 `json:"maxBytes"`
}

type ComicCacheStatusDTO struct {
	MaxBytes   int64 `json:"maxBytes"`
	UsedBytes  int64 `json:"usedBytes"`
	EntryCount int   `json:"entryCount"`
}

type PatchComicBookRequest struct {
	Title       *string   `json:"title,omitempty"`
	Tags        *[]string `json:"tags,omitempty"`
	Favorite    *bool     `json:"favorite,omitempty"`
	RatingSet   bool      `json:"ratingSet,omitempty"`
	RatingClear bool      `json:"ratingClear,omitempty"`
	Rating      *float64  `json:"rating,omitempty"`
}

type PutComicProgressRequest struct {
	PageIndex int  `json:"pageIndex"`
	Completed bool `json:"completed"`
}

type PutComicReadingPreferencesRequest struct {
	Mode      *string `json:"mode,omitempty"`
	Fit       *string `json:"fit,omitempty"`
	Direction *string `json:"direction,omitempty"`
}

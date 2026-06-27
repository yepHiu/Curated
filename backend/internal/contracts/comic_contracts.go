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

type ListComicBooksRequest struct {
	Query      string `json:"query,omitempty"`
	Tag        string `json:"tag,omitempty"`
	Favorite   *bool  `json:"favorite,omitempty"`
	ReadStatus string `json:"readStatus,omitempty"`
	Limit      int    `json:"limit,omitempty"`
	Offset     int    `json:"offset,omitempty"`
}

type ComicBooksPageDTO struct {
	Items  []ComicBookListItemDTO `json:"items"`
	Total  int                    `json:"total"`
	Limit  int                    `json:"limit"`
	Offset int                    `json:"offset"`
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

type ComicReadingProgressDTO struct {
	ComicID   string `json:"comicId"`
	PageIndex int    `json:"pageIndex"`
	Completed bool   `json:"completed"`
	UpdatedAt string `json:"updatedAt"`
}

type PutComicReadingPreferencesRequest struct {
	Mode      *string `json:"mode,omitempty"`
	Fit       *string `json:"fit,omitempty"`
	Direction *string `json:"direction,omitempty"`
}

type ComicReadingPreferencesDTO struct {
	ComicID   string `json:"comicId,omitempty"`
	Mode      string `json:"mode"`
	Fit       string `json:"fit"`
	Direction string `json:"direction"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

type ComicCacheEntryDTO struct {
	CacheKey       string `json:"cacheKey"`
	ComicID        string `json:"comicId"`
	Kind           string `json:"kind"`
	PageIndex      int    `json:"pageIndex"`
	Path           string `json:"path"`
	SizeBytes      int64  `json:"sizeBytes"`
	CreatedAt      string `json:"createdAt"`
	LastAccessedAt string `json:"lastAccessedAt"`
}

package contracts

// MovieCardDTO contains only fields needed to render a browsing card.
type MovieCardDTO struct {
	CoverVariants  []ImageVariantDTO `json:"coverVariants,omitempty"`
	ThumbVariants  []ImageVariantDTO `json:"thumbVariants,omitempty"`
	Resolution     string            `json:"resolution"`
	ID             string            `json:"id"`
	Title          string            `json:"title"`
	Code           string            `json:"code"`
	Studio         string            `json:"studio"`
	Actors         []string          `json:"actors"`
	RuntimeMinutes int               `json:"runtimeMinutes"`
	Rating         float64           `json:"rating"`
	IsFavorite     bool              `json:"isFavorite"`
	Year           int               `json:"year"`
	CoverURL       string            `json:"coverUrl,omitempty"`
	ThumbURL       string            `json:"thumbUrl,omitempty"`
}

type MovieCardsDTO struct {
	Items      []MovieCardDTO `json:"items"`
	MissingIDs []string       `json:"missingIds"`
}

const MaxMovieCardIDs = 50

type ImageVariantDTO struct {
	Width int    `json:"width"`
	URL   string `json:"url"`
}

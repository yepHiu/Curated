package scraper

import "context"

type Metadata struct {
	MovieID         string
	Number          string
	Title           string
	Summary         string
	Provider        string
	Homepage        string
	Director        string
	Studio          string
	Label           string
	Series          string
	Actors          []string
	Tags            []string
	RuntimeMinutes  int
	Rating          float64
	ReleaseDate     string
	CoverURL        string
	ThumbURL        string
	PreviewVideoURL string
	PreviewImages   []string
}

// ActorProfile is merged into the library actors row after a successful actor scrape.
type ActorProfile struct {
	DisplayName     string
	AvatarURL       string
	Summary         string
	Homepage        string
	Provider        string
	ProviderActorID string
	Height          int
	Birthday        string
}

// MovieScrapeOptions controls movie metadata scraping. Provider empty means all registered sources (Metatube SearchMovieAll),
// except FC2 番号 (common/number.IsFC2) which is limited to FC2 + fc2hub providers only.
type MovieScrapeOptions struct {
	Provider string
}

type Service interface {
	Scrape(ctx context.Context, movieID string, number string, opts MovieScrapeOptions) (Metadata, error)
	ScrapeActor(ctx context.Context, displayName string) (ActorProfile, error)
}

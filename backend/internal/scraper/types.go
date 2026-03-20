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

type Service interface {
	Scrape(ctx context.Context, movieID string, number string) (Metadata, error)
}

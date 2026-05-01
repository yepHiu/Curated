// Package library manages the in-memory movie catalogue: ingest, search, filter, and metadata application.
package library

import (
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"curated-backend/internal/contracts"
	"curated-backend/internal/scraper"
)

var errMovieNotFound = errors.New("movie not found")

// Service is the thread-safe in-memory movie store with search and metadata application.
type Service struct {
	mu          sync.RWMutex
	movies      []contracts.MovieDetailDTO
	searchTexts map[string]string // movieID → pre-computed searchable text
}

// NewService creates an empty movie Service.
func NewService() *Service {
	return &Service{
		movies:      nil,
		searchTexts: make(map[string]string),
	}
}

// ListMovies filters, sorts, and paginates the in-memory movie list by mode, query, actor, and studio.
func (s *Service) ListMovies(request contracts.ListMoviesRequest) contracts.MoviesPageDTO {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filtered := make([]contracts.MovieDetailDTO, 0, len(s.movies))
	query := strings.TrimSpace(strings.ToLower(request.Query))

	actorExact := strings.TrimSpace(request.Actor)
	studioExact := strings.TrimSpace(request.Studio)
	for _, movie := range s.movies {
		if request.Mode == "favorites" && !movie.IsFavorite {
			continue
		}

		eff := contracts.EffectiveMovieDetailDTO(movie)
		if query != "" {
			searchText := s.searchTexts[movie.ID]
			if !matchesQuery(eff, searchText, query) {
				continue
			}
		}

		if actorExact != "" && !slices.Contains(movie.Actors, actorExact) {
			continue
		}

		if studioExact != "" && strings.TrimSpace(eff.Studio) != studioExact {
			continue
		}

		filtered = append(filtered, movie)
	}

	slices.SortFunc(filtered, func(a, b contracts.MovieDetailDTO) int {
		switch {
		case a.AddedAt > b.AddedAt:
			return -1
		case a.AddedAt < b.AddedAt:
			return 1
		default:
			return strings.Compare(a.ID, b.ID)
		}
	})

	limit := request.Limit
	if limit <= 0 {
		limit = 24
	}

	offset := request.Offset
	if offset < 0 {
		offset = 0
	}

	total := len(filtered)
	if offset > total {
		offset = total
	}

	end := offset + limit
	if end > total {
		end = total
	}

	items := make([]contracts.MovieListItemDTO, 0, end-offset)
	for _, movie := range filtered[offset:end] {
		m := contracts.EffectiveMovieDetailDTO(movie)
		syncEffectiveRating(&m)
		items = append(items, m.MovieListItemDTO)
	}

	return contracts.MoviesPageDTO{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}
}

// GetMovie returns the movie with the given ID, or errMovieNotFound.
func (s *Service) GetMovie(movieID string) (contracts.MovieDetailDTO, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, movie := range s.movies {
		if movie.ID == movieID {
			m := contracts.EffectiveMovieDetailDTO(movie)
			syncEffectiveRating(&m)
			return m, nil
		}
	}
	return contracts.MovieDetailDTO{}, errMovieNotFound
}

// IsNotFound reports whether err is errMovieNotFound.
func IsNotFound(err error) bool {
	return errors.Is(err, errMovieNotFound)
}

// UpsertScannedMovie inserts or updates a movie record from a scan result, deduplicating by ID, code, or file location.
func (s *Service) UpsertScannedMovie(result contracts.ScanFileResultDTO) {
	if result.MovieID == "" || result.Number == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for index, movie := range s.movies {
		if movie.ID == result.MovieID || movie.Code == result.Number || movie.Location == result.Path {
			movie.ID = result.MovieID
			movie.Code = result.Number
			movie.Title = result.Number
			movie.Location = result.Path
			if movie.AddedAt == "" {
				movie.AddedAt = time.Now().UTC().Format("2006-01-02")
			}
			if movie.Resolution == "" {
				movie.Resolution = strings.TrimPrefix(strings.ToLower(filepath.Ext(result.Path)), ".")
			}
			syncEffectiveRating(&movie)
			s.movies[index] = movie
			return
		}
	}

	s.movies = append(s.movies, contracts.MovieDetailDTO{
		MovieListItemDTO: contracts.MovieListItemDTO{
			ID:             result.MovieID,
			Title:          result.Number,
			Code:           result.Number,
			Studio:         "Unknown",
			Actors:         []string{},
			Tags:           []string{"Scanned"},
			RuntimeMinutes: 0,
			Rating:         0,
			IsFavorite:     false,
			AddedAt:        time.Now().UTC().Format("2006-01-02"),
			Location:       result.Path,
				Resolution:     strings.TrimPrefix(strings.ToLower(filepath.Ext(result.Path)), "."),
				Year: 0,
			},
		Summary:        "Metadata pending scrape.",
		MetadataRating: 0,
		UserRating:     nil,
	})
}

// ApplyScrapedMetadata merges scraped metadata fields into the matching movie, preserving local-only fields like UserTags.
func (s *Service) ApplyScrapedMetadata(metadata scraper.Metadata) {
	if metadata.MovieID == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for index, movie := range s.movies {
		if movie.ID != metadata.MovieID {
			continue
		}

		if metadata.Title != "" {
			movie.Title = metadata.Title
		}
		if metadata.Number != "" {
			movie.Code = metadata.Number
		}
		if metadata.Studio != "" {
			movie.Studio = metadata.Studio
		}
		if metadata.RuntimeMinutes > 0 {
			movie.RuntimeMinutes = metadata.RuntimeMinutes
		}
		if metadata.Rating > 0 {
			movie.MetadataRating = metadata.Rating
		}
		if metadata.ReleaseDate != "" && len(metadata.ReleaseDate) >= 4 {
			if year := metadata.ReleaseDate[:4]; len(year) == 4 {
				if parsedYear := parseYear(year); parsedYear > 0 {
					movie.Year = parsedYear
				}
			}
		}
		movie.Actors = append([]string{}, metadata.Actors...)
		movie.Tags = append([]string{}, metadata.Tags...)
		// UserTags are local-only; scraper must not overwrite them.
		movie.Summary = coalesceSummary(metadata.Summary)
		if metadata.CoverURL != "" {
			movie.CoverURL = metadata.CoverURL
		}
		if metadata.ThumbURL != "" {
			movie.ThumbURL = metadata.ThumbURL
		}
		if len(metadata.PreviewImages) > 0 {
			movie.PreviewImages = append([]string{}, metadata.PreviewImages...)
		}
		syncEffectiveRating(&movie)
		s.movies[index] = movie
		return
	}
}

func parseYear(value string) int {
	year := 0
	for _, r := range value {
		if r < '0' || r > '9' {
			return 0
		}
		year = year*10 + int(r-'0')
	}
	return year
}

func coalesceSummary(summary string) string {
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return "Metadata pending scrape."
	}
	return summary
}

func matchesQuery(movie contracts.MovieDetailDTO, searchText, query string) bool {
	// Static fields that never change at runtime.
	fields := []string{
		movie.Title,
		movie.Code,
		movie.Studio,
		movie.Summary,
	}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), query) {
			return true
		}
	}
	// Pre-computed text for slice-based fields (Actors, Tags, UserTags).
	if searchText != "" && strings.Contains(strings.ToLower(searchText), query) {
		return true
	}
	return false
}

func buildSearchText(m contracts.MovieListItemDTO) string {
	parts := make([]string, 0, len(m.Actors)+len(m.Tags)+len(m.UserTags))
	parts = append(parts, m.Actors...)
	parts = append(parts, m.Tags...)
	parts = append(parts, m.UserTags...)
	return strings.Join(parts, " ")
}

// RebuildSearchTexts recomputes the cached search text for every movie in the service.
// Call this after bulk-loading movies (e.g. from storage on startup) before any ListMovies call.
func (s *Service) RebuildSearchTexts() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.searchTexts == nil {
		s.searchTexts = make(map[string]string, len(s.movies))
	}
	for i := range s.movies {
		s.searchTexts[s.movies[i].ID] = buildSearchText(s.movies[i].MovieListItemDTO)
	}
}

func syncEffectiveRating(m *contracts.MovieDetailDTO) {
	if m.UserRating != nil {
		m.Rating = *m.UserRating
		return
	}
	m.Rating = m.MetadataRating
}

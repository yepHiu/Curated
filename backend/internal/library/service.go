package library

import (
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"jav-shadcn/backend/internal/contracts"
	"jav-shadcn/backend/internal/scraper"
)

var errMovieNotFound = errors.New("movie not found")

type Service struct {
	mu     sync.RWMutex
	movies []contracts.MovieDetailDTO
}

func NewService() *Service {
	return &Service{
		movies: seedMovies(),
	}
}

func (s *Service) ListMovies(request contracts.ListMoviesRequest) contracts.MoviesPageDTO {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filtered := make([]contracts.MovieDetailDTO, 0, len(s.movies))
	query := strings.TrimSpace(strings.ToLower(request.Query))

	for _, movie := range s.movies {
		if request.Mode == "favorites" && !movie.IsFavorite {
			continue
		}

		if query != "" && !matchesQuery(movie, query) {
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
		m := movie
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

func (s *Service) GetMovie(movieID string) (contracts.MovieDetailDTO, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, movie := range s.movies {
		if movie.ID == movieID {
			m := movie
			syncEffectiveRating(&m)
			return m, nil
		}
	}
	return contracts.MovieDetailDTO{}, errMovieNotFound
}

// PatchMovie updates in-memory seed movies (used when SQLite has no rows).
func (s *Service) PatchMovie(movieID string, in contracts.PatchMovieInput) (contracts.MovieDetailDTO, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.movies {
		if s.movies[i].ID != movieID {
			continue
		}
		m := &s.movies[i]
		if in.Favorite != nil {
			m.IsFavorite = *in.Favorite
		}
		if in.UserRatingSet {
			if in.UserRatingClear {
				m.UserRating = nil
			} else {
				v := in.UserRating
				m.UserRating = &v
			}
		}
		syncEffectiveRating(m)
		return *m, nil
	}
	return contracts.MovieDetailDTO{}, errMovieNotFound
}

func IsNotFound(err error) bool {
	return errors.Is(err, errMovieNotFound)
}

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
			Year:           0,
		},
		Summary:        "Metadata pending scrape.",
		MetadataRating: 0,
		UserRating:     nil,
	})
}

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

func matchesQuery(movie contracts.MovieDetailDTO, query string) bool {
	fields := []string{
		movie.Title,
		movie.Code,
		movie.Studio,
		movie.Summary,
		strings.Join(movie.Actors, " "),
		strings.Join(movie.Tags, " "),
	}

	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), query) {
			return true
		}
	}

	return false
}

func seedMovies() []contracts.MovieDetailDTO {
	return []contracts.MovieDetailDTO{
		{
			MovieListItemDTO: contracts.MovieListItemDTO{
				ID:             "mkb-100",
				Title:          "Midnight Kiss Broadcast",
				Code:           "MKB-100",
				Studio:         "Velvet North",
				Actors:         []string{"Mina Kaze", "Rin Asuka"},
				Tags:           []string{"Romance", "4K", "Late Night"},
				RuntimeMinutes: 134,
				Rating:         4.8,
				IsFavorite:     true,
				AddedAt:        "2026-01-03",
				Location:       "D:/Media/JAV/Main/MKB-100.mkv",
				Resolution:     "2160p",
				Year:           2025,
			},
			Summary:        "A polished late-night feature with strong cast chemistry and metadata-rich presentation.",
			MetadataRating: 4.8,
			UserRating:     nil,
		},
		{
			MovieListItemDTO: contracts.MovieListItemDTO{
				ID:             "sld-101",
				Title:          "Silk Line Directive",
				Code:           "SLD-101",
				Studio:         "Studio Garnet",
				Actors:         []string{"Airi Sena"},
				Tags:           []string{"Drama", "Office", "High Rating"},
				RuntimeMinutes: 126,
				Rating:         4.7,
				IsFavorite:     true,
				AddedAt:        "2026-01-14",
				Location:       "E:/Vault/JAV/New/SLD-101.mp4",
				Resolution:     "1080p",
				Year:           2025,
			},
			Summary:        "An elegant office-set release used to validate detail views, favorites, and list filtering.",
			MetadataRating: 4.7,
			UserRating:     nil,
		},
		{
			MovieListItemDTO: contracts.MovieListItemDTO{
				ID:             "nva-102",
				Title:          "Neon Velvet Archive",
				Code:           "NVA-102",
				Studio:         "Moonlight Works",
				Actors:         []string{"Yua Mori", "Nao Shin"},
				Tags:           []string{"Sci-Fi", "Stylized", "New"},
				RuntimeMinutes: 118,
				Rating:         4.5,
				IsFavorite:     false,
				AddedAt:        "2026-02-11",
				Location:       "D:/Media/JAV/Main/NVA-102.mkv",
				Resolution:     "2160p",
				Year:           2026,
			},
			Summary:        "A stylized catalog entry that exercises tag-heavy filtering and recent import ordering.",
			MetadataRating: 4.5,
			UserRating:     nil,
		},
		{
			MovieListItemDTO: contracts.MovieListItemDTO{
				ID:             "prm-103",
				Title:          "Private Room Memoir",
				Code:           "PRM-103",
				Studio:         "Golden Frame",
				Actors:         []string{"Sora Minami", "Miu Arata"},
				Tags:           []string{"Character", "Favorites", "Longform"},
				RuntimeMinutes: 151,
				Rating:         4.9,
				IsFavorite:     true,
				AddedAt:        "2026-03-08",
				Location:       "F:/Offline/Collections/PRM-103.mp4",
				Resolution:     "2160p",
				Year:           2024,
			},
			Summary:        "A high-rated longform title kept here as the default rich detail fixture for the backend scaffold.",
			MetadataRating: 4.9,
			UserRating:     nil,
		},
	}
}

func syncEffectiveRating(m *contracts.MovieDetailDTO) {
	if m.UserRating != nil {
		m.Rating = *m.UserRating
		return
	}
	m.Rating = m.MetadataRating
}

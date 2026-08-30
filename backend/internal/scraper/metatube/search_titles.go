package metatube

import (
	"context"
	"fmt"
	"strings"

	mtnum "github.com/metatube-community/metatube-sdk-go/common/number"
	"github.com/metatube-community/metatube-sdk-go/model"

	"curated-backend/internal/scraper"
)

// SearchTitles returns a bounded Metatube movie search without fetching full info or enqueueing scrape.movie.
func (s *Service) SearchTitles(ctx context.Context, keyword string, limit int) ([]scraper.TitleHit, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil, fmt.Errorf("empty search keyword")
	}
	if limit <= 0 {
		limit = 15
	}
	if limit > 25 {
		limit = 25
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	if s == nil || s.engine == nil {
		return nil, fmt.Errorf("metadata search is unavailable")
	}

	var (
		results []*model.MovieSearchResult
		err     error
	)
	if mtnum.IsFC2(mtnum.Trim(keyword)) {
		results, err = s.searchMovieFC2Providers(ctx, keyword)
	} else {
		results, err = s.engine.SearchMovieAll(keyword, false)
	}
	if err != nil {
		return nil, err
	}
	ranked := s.rankMovieSearchResults(mtnum.Trim(keyword), results)
	out := make([]scraper.TitleHit, 0, limit)
	seen := map[string]struct{}{}
	for _, result := range ranked {
		if result == nil || !result.IsValid() {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(result.Provider) + "|" + strings.TrimSpace(result.Number))
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, scraper.TitleHit{
			Number:   strings.TrimSpace(result.Number),
			Title:    strings.TrimSpace(result.Title),
			Provider: strings.TrimSpace(result.Provider),
			Homepage: strings.TrimSpace(result.Homepage),
			Score:    result.Score,
		})
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

package metatube

import (
	"context"
	"curated-backend/internal/library/moviecode"
	"curated-backend/internal/scraper"
	fc2source "curated-backend/internal/scraper/fc2"
	"errors"
	"fmt"
	mtnum "github.com/metatube-community/metatube-sdk-go/common/number"
	"github.com/metatube-community/metatube-sdk-go/model"
)

// exactWishlistResult 只接受同一规范番号，同 provider 多个不同记录视为歧义。
func exactWishlistResult(number string, results []*model.MovieSearchResult) (*model.MovieSearchResult, error) {
	_, key, err := moviecode.WishlistIdentity(number)
	if err != nil {
		return nil, err
	}
	var selected *model.MovieSearchResult
	seen := map[string]string{}
	for _, r := range results {
		if r == nil || !r.IsValid() {
			continue
		}
		_, candidate, e := moviecode.WishlistIdentity(r.Number)
		if e != nil || candidate != key {
			continue
		}
		if id, ok := seen[r.Provider]; ok && id != r.ID {
			return nil, errors.New("WISHLIST_AMBIGUOUS")
		}
		seen[r.Provider] = r.ID
		if selected == nil {
			selected = r
		}
	}
	if selected == nil {
		return nil, errors.New("WISHLIST_NOT_FOUND")
	}
	return selected, nil
}

// ScrapeWishlist 复用普通来源策略与 FC2 多源补全，对愿望身份冲突保留人工复核标识。
func (s *Service) ScrapeWishlist(ctx context.Context, id, number string, opts scraper.MovieScrapeOptions) (scraper.Metadata, error) {
	if mtnum.IsFC2(mtnum.Trim(number)) {
		metadata, err := s.scrapeFC2(ctx, id, number, opts)
		if errors.Is(err, fc2source.ErrIdentity) {
			return scraper.Metadata{}, fmt.Errorf("WISHLIST_IDENTITY_MISMATCH: %w", err)
		}
		return metadata, err
	}
	chain := s.resolveProviderChain(opts, mtnum.IsFC2(mtnum.Trim(number)))
	var results []*model.MovieSearchResult
	var err error
	if len(chain) == 0 {
		results, err = s.engine.SearchMovieAll(number, false)
		if err != nil {
			return scraper.Metadata{}, err
		}
		hit, e := exactWishlistResult(number, results)
		if e != nil {
			return scraper.Metadata{}, e
		}
		return s.fetchWishlistInfo(ctx, id, number, hit)
	}
	var last error
	for _, provider := range chain {
		if err = ctx.Err(); err != nil {
			return scraper.Metadata{}, err
		}
		results, err = s.engine.SearchMovie(number, provider, false)
		if err != nil {
			last = err
			continue
		}
		hit, e := exactWishlistResult(number, results)
		if e != nil {
			last = e
			continue
		}
		metadata, e := s.fetchWishlistInfo(ctx, id, number, hit)
		if e == nil {
			return metadata, nil
		}
		last = e
	}
	if last == nil {
		last = errors.New("WISHLIST_NOT_FOUND")
	}
	return scraper.Metadata{}, fmt.Errorf("wishlist providers: %w", last)
}

// fetchWishlistInfo 标记上下文，详情接口再次检查作品番号，防止搜索命中与详情错位。
func (s *Service) fetchWishlistInfo(ctx context.Context, id, number string, hit *model.MovieSearchResult) (scraper.Metadata, error) {
	return s.fetchMovieInfo(ctx, id, number, hit, true)
}

package metatube

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/metatube-community/metatube-sdk-go/database"
	"github.com/metatube-community/metatube-sdk-go/engine"
	"github.com/metatube-community/metatube-sdk-go/engine/providerid"
	"go.uber.org/zap"

	"jav-shadcn/backend/internal/scraper"
)

type Service struct {
	logger *zap.Logger
	engine *engine.Engine
}

func NewService(logger *zap.Logger, requestTimeout time.Duration) (*Service, error) {
	if requestTimeout <= 0 {
		requestTimeout = 45 * time.Second
	}

	db, err := database.Open(&database.Config{
		DSN:                  "",
		DisableAutomaticPing: true,
	})
	if err != nil {
		return nil, fmt.Errorf("metatube database open: %w", err)
	}

	eng := engine.New(db, engine.WithRequestTimeout(requestTimeout))
	if err := eng.DBAutoMigrate(true); err != nil {
		return nil, fmt.Errorf("metatube schema migrate: %w", err)
	}

	return &Service{
		logger: logger,
		engine: eng,
	}, nil
}

func (s *Service) Engine() *engine.Engine {
	return s.engine
}

func (s *Service) Scrape(ctx context.Context, movieID string, number string) (scraper.Metadata, error) {
	select {
	case <-ctx.Done():
		return scraper.Metadata{}, ctx.Err()
	default:
	}

	s.logger.Info("scraping metadata", zap.String("number", number), zap.String("movieId", movieID))

	results, err := s.engine.SearchMovieAll(number, false)
	if err != nil {
		return scraper.Metadata{}, fmt.Errorf("search failed for %s: %w", number, err)
	}
	if len(results) == 0 {
		return scraper.Metadata{}, fmt.Errorf("no results found for %s", number)
	}

	first := results[0]
	s.logger.Info("search result selected",
		zap.String("number", number),
		zap.String("provider", first.Provider),
		zap.String("providerMovieId", first.ID),
		zap.String("title", first.Title),
		zap.Int("totalResults", len(results)),
	)

	pid, err := providerid.New(first.Provider, first.ID)
	if err != nil {
		return scraper.Metadata{}, fmt.Errorf("invalid provider result for %s: provider=%s id=%s: %w", number, first.Provider, first.ID, err)
	}

	select {
	case <-ctx.Done():
		return scraper.Metadata{}, ctx.Err()
	default:
	}

	info, err := s.engine.GetMovieInfoByProviderID(pid, true)
	if err != nil {
		return scraper.Metadata{}, fmt.Errorf("get movie info failed for %s (provider=%s): %w", number, first.Provider, err)
	}

	s.logger.Info("metadata fetched",
		zap.String("number", number),
		zap.String("title", info.Title),
		zap.String("maker", info.Maker),
		zap.Int("actors", len(info.Actors)),
		zap.Int("genres", len(info.Genres)),
		zap.Int("previewImages", len(info.PreviewImages)),
		zap.String("coverURL", info.CoverURL),
	)

	return scraper.Metadata{
		MovieID:         movieID,
		Number:          number,
		Title:           info.Title,
		Summary:         info.Summary,
		Provider:        info.Provider,
		Homepage:        info.Homepage,
		Director:        info.Director,
		Studio:          info.Maker,
		Label:           info.Label,
		Series:          info.Series,
		Actors:          cleanStrings(info.Actors),
		Tags:            cleanStrings(info.Genres),
		RuntimeMinutes:  info.Runtime,
		Rating:          info.Score,
		ReleaseDate:     formatReleaseDate(info.ReleaseDate),
		CoverURL:        info.CoverURL,
		ThumbURL:        info.ThumbURL,
		PreviewVideoURL: info.PreviewVideoURL,
		PreviewImages:   cleanStrings(info.PreviewImages),
	}, nil
}

func formatReleaseDate(value any) string {
	formatted := strings.TrimSpace(fmt.Sprint(value))
	switch formatted {
	case "", "0001-01-01 00:00:00 +0000 UTC":
		return ""
	default:
		return formatted
	}
}

func cleanStrings(values []string) []string {
	cleaned := make([]string, 0, len(values))
	seen := make(map[string]struct{})

	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		cleaned = append(cleaned, value)
	}

	return cleaned
}

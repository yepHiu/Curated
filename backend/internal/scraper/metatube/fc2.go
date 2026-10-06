package metatube

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/metatube-community/metatube-sdk-go/model"
	"go.uber.org/zap"

	"curated-backend/internal/scraper"
	fc2source "curated-backend/internal/scraper/fc2"
)

// fc2Providers 尊重指定来源与自定义次序；普通影片链没有 FC2 源时沿用自动专用来源。
func fc2Providers(opts scraper.MovieScrapeOptions) ([]string, error) {
	if opts.Automatic {
		return append([]string(nil), fc2MovieProviderNames...), nil
	}
	if len(opts.ProviderChain) == 0 && strings.TrimSpace(opts.Provider) != "" {
		name := strings.TrimSpace(opts.Provider)
		if !fc2source.Supports(name) {
			return nil, fmt.Errorf("specified provider %q does not support FC2 metadata", name)
		}
		return []string{name}, nil
	}
	var providers []string
	seen := map[string]bool{}
	for _, name := range opts.ProviderChain {
		name = strings.TrimSpace(name)
		if fc2source.Supports(name) && !seen[name] {
			providers = append(providers, name)
			seen[name] = true
		}
	}
	if len(providers) == 0 {
		providers = append(providers, fc2MovieProviderNames...)
	}
	return providers, nil
}

// collectFC2 获取最多两路并发、45 秒总预算的精确结果，保留逐源失败原因。
func (s *Service) collectFC2(ctx context.Context, movieID, number string, providers []string, explicit bool) ([]scraper.Metadata, error) {
	budget, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	results := make([]scraper.Metadata, len(providers))
	failures := make([]error, len(providers))
	var wg sync.WaitGroup
	slots := make(chan struct{}, 2)
	for index, provider := range providers {
		wg.Add(1)
		go func(index int, provider string) {
			// 每个来源独占一个结果位置；等待槽位时也响应取消。
			defer wg.Done()
			select {
			case slots <- struct{}{}:
			case <-budget.Done():
				failures[index] = budget.Err()
				return
			}
			defer func() { // 释放本来源的并发槽位。
				<-slots
			}()
			if !explicit {
				until, _ := time.Parse(time.RFC3339, s.ProviderRuntimeHealth(provider).CooldownUntil)
				if time.Now().Before(until) {
					failures[index] = fmt.Errorf("%s forbidden: provider cooldown until %s", provider, until.Format(time.RFC3339))
					return
				}
			}
			start := time.Now()
			metadata, err := s.fc2Lookup(budget, provider, movieID, number)
			if err == nil && (fc2source.Digits(metadata.Number) != fc2source.Digits(number) || metadata.Title == "" || metadata.Provider != provider) {
				err = fmt.Errorf("%s: %w", provider, fc2source.ErrIdentity)
			}
			latency := time.Since(start).Milliseconds()
			if err != nil {
				failures[index] = fmt.Errorf("%s: %w", provider, err)
				if errors.Is(err, fc2source.ErrNotFound) {
					s.recordProviderSuccess(provider, latency)
				} else if budget.Err() == nil {
					s.recordProviderFailure(provider, latency, err)
					if strings.Contains(strings.ToLower(err.Error()), "forbidden") {
						s.cooldownFC2Provider(provider)
					}
				}
				s.logger.Warn("FC2 metadata source failed", zap.String("number", number), zap.String("provider", provider), zap.Error(err))
				return
			}
			s.recordProviderSuccess(provider, latency)
			results[index] = metadata
			s.logger.Info("FC2 metadata source succeeded", zap.String("number", number), zap.String("provider", provider), zap.Int("actors", len(metadata.Actors)))
		}(index, provider)
	}
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var successful []scraper.Metadata
	for _, metadata := range results {
		if metadata.Title != "" {
			successful = append(successful, metadata)
		}
	}
	if len(successful) == 0 {
		return nil, fmt.Errorf("FC2 metadata sources failed for %s: %w", number, errors.Join(failures...))
	}
	return successful, nil
}

// cooldownFC2Provider 对访问验证阻断启用五分钟冷却；指定来源可显式重试。
func (s *Service) cooldownFC2Provider(provider string) {
	s.healthMu.Lock()
	defer s.healthMu.Unlock()
	health := s.health[provider]
	health.CooldownUntil = time.Now().UTC().Add(5 * time.Minute).Format(time.RFC3339)
	s.health[provider] = health
}

// scrapeFC2 在选定来源内补全字段，不让一个站点的失败抹掉其它有效结果。
func (s *Service) scrapeFC2(ctx context.Context, movieID, number string, opts scraper.MovieScrapeOptions) (scraper.Metadata, error) {
	providers, err := fc2Providers(opts)
	if err != nil {
		return scraper.Metadata{}, err
	}
	results, err := s.collectFC2(ctx, movieID, number, providers, opts.Provider != "" && len(opts.ProviderChain) == 0)
	if err != nil {
		return scraper.Metadata{}, err
	}
	metadata := mergeFC2Metadata(results)
	s.logger.Info("FC2 metadata enriched", zap.String("number", number), zap.String("provider", metadata.Provider), zap.Int("sources", len(results)), zap.Int("actors", len(metadata.Actors)))
	return metadata, nil
}

// mergeFC2Metadata 以优先来源为主，缺失字段从后续精确来源补齐并保留图片来源。
func mergeFC2Metadata(results []scraper.Metadata) scraper.Metadata {
	out := results[0]
	out.AssetSources = make(map[string]scraper.AssetSource)
	for _, source := range results {
		if out.Title == "" {
			out.Title = source.Title
		}
		if len(strings.TrimSpace(source.Summary)) > len(strings.TrimSpace(out.Summary)) {
			out.Summary = source.Summary
		}
		if out.Studio == "" {
			out.Studio = source.Studio
		}
		if out.Director == "" {
			out.Director = source.Director
		}
		if out.Label == "" {
			out.Label = source.Label
		}
		if out.Series == "" {
			out.Series = source.Series
		}
		if len(out.Actors) == 0 {
			out.Actors = source.Actors
		}
		if len(out.Tags) == 0 {
			out.Tags = source.Tags
		}
		if out.RuntimeMinutes == 0 {
			out.RuntimeMinutes = source.RuntimeMinutes
		}
		if out.Rating == 0 {
			out.Rating = source.Rating
		}
		if out.ReleaseDate == "" {
			out.ReleaseDate = source.ReleaseDate
		}
		if out.CoverURL == "" {
			out.CoverURL = source.CoverURL
		}
		if out.ThumbURL == "" {
			out.ThumbURL = source.ThumbURL
		}
		if out.PreviewVideoURL == "" {
			out.PreviewVideoURL = source.PreviewVideoURL
		}
		if len(out.PreviewImages) == 0 {
			out.PreviewImages = source.PreviewImages
		}
		for _, resource := range append([]string{source.CoverURL, source.ThumbURL}, source.PreviewImages...) {
			if resource != "" {
				if _, exists := out.AssetSources[resource]; !exists {
					out.AssetSources[resource] = scraper.AssetSource{Provider: source.Provider, Homepage: source.Homepage}
				}
			}
		}
	}
	return out
}

// searchMovieFC2Providers 为影片搜索和 Agent 返回同一组精确来源结果。
func (s *Service) searchMovieFC2Providers(ctx context.Context, keyword string) ([]*model.MovieSearchResult, error) {
	metadata, err := s.collectFC2(ctx, "", keyword, fc2MovieProviderNames, false)
	if err != nil {
		return nil, err
	}
	var results []*model.MovieSearchResult
	for _, item := range metadata {
		results = append(results, &model.MovieSearchResult{ID: fc2source.Digits(item.Number), Number: item.Number, Title: item.Title, Provider: item.Provider, Homepage: item.Homepage, CoverURL: item.CoverURL, ThumbURL: item.ThumbURL, Score: item.Rating})
	}
	return results, nil
}

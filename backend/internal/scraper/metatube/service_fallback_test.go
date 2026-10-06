package metatube

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"curated-backend/internal/scraper"
	"github.com/metatube-community/metatube-sdk-go/engine/providerid"
	"github.com/metatube-community/metatube-sdk-go/model"
	"go.uber.org/zap"
)

// TestProviderChainDetailFallback 验证搜索成功、详情失败后继续，并且只有完整成功才记健康。
func TestProviderChainDetailFallback(t *testing.T) {
	service, err := NewService(zap.NewNop(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	service.movieSearch = func(number, provider string, fallback bool) ([]*model.MovieSearchResult, error) { // 两个站点均返回搜索命中。
		return []*model.MovieSearchResult{{ID: "id", Number: number, Provider: provider, Title: "Search", Homepage: "https://example/movie"}}, nil
	}
	service.movieInfo = func(id providerid.ProviderID, lazy bool) (*model.MovieInfo, error) { // 第一来源仅在详情阶段失败。
		if id.Provider == "JavBus" {
			return nil, errors.New("HTTP 403 forbidden")
		}
		return &model.MovieInfo{ID: "id", Number: "ABC-123", Title: "Detail", Provider: id.Provider, Homepage: "https://example/movie", CoverURL: "https://example/cover"}, nil
	}
	metadata, err := service.Scrape(context.Background(), "movie", "ABC-123", scraper.MovieScrapeOptions{ProviderChain: []string{"JavBus", "JAV321"}})
	if err != nil || metadata.Provider != "JAV321" {
		t.Fatalf("fallback failed: %+v %v", metadata, err)
	}
	first := service.ProviderRuntimeHealth("JavBus")
	if first.LastOKAt != "" || first.ConsecutiveFailures != 1 || service.ProviderRuntimeHealth("JAV321").LastOKAt == "" {
		t.Fatalf("premature success health: %+v", first)
	}
}

// TestProviderChainRetainsEveryFailure 验证后续空结果不会抹掉前一来源的详情阻断。
func TestProviderChainRetainsEveryFailure(t *testing.T) {
	service, err := NewService(zap.NewNop(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	service.movieSearch = func(number, provider string, fallback bool) ([]*model.MovieSearchResult, error) { // 后一来源无结果，前一来源只在详情阶段失败。
		if provider == "JAV321" {
			return nil, nil
		}
		return []*model.MovieSearchResult{{ID: "id", Number: number, Provider: provider, Title: "Search", Homepage: "https://example/movie"}}, nil
	}
	service.movieInfo = func(id providerid.ProviderID, lazy bool) (*model.MovieInfo, error) { // 模拟详情页验证失败。
		return nil, errors.New("HTTP 403 forbidden")
	}
	_, err = service.Scrape(context.Background(), "movie", "ABC-123", scraper.MovieScrapeOptions{ProviderChain: []string{"JavBus", "JAV321"}})
	if err == nil || !strings.Contains(err.Error(), "JavBus detail failed") || !strings.Contains(err.Error(), "HTTP 403") || !strings.Contains(err.Error(), "no results from JAV321") {
		t.Fatalf("source errors lost: %v", err)
	}
}

// TestProviderChainRejectsEmptyDetail 验证 SDK 返回 nil 无错误时，仍继续后续来源且不会崩溃。
func TestProviderChainRejectsEmptyDetail(t *testing.T) {
	service, err := NewService(zap.NewNop(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	service.movieSearch = func(number, provider string, fallback bool) ([]*model.MovieSearchResult, error) { // 两源均有效命中。
		return []*model.MovieSearchResult{{ID: "id", Number: number, Provider: provider, Title: "Search", Homepage: "https://example/movie"}}, nil
	}
	service.movieInfo = func(id providerid.ProviderID, lazy bool) (*model.MovieInfo, error) { // 空详情被拒绝，随后来源成功。
		if id.Provider == "JavBus" {
			return nil, nil
		}
		return &model.MovieInfo{ID: "id", Number: "ABC-123", Title: "Detail", Provider: id.Provider, Homepage: "https://example/movie"}, nil
	}
	metadata, err := service.Scrape(context.Background(), "movie", "ABC-123", scraper.MovieScrapeOptions{ProviderChain: []string{"JavBus", "JAV321"}})
	if err != nil || metadata.Provider != "JAV321" {
		t.Fatalf("empty detail fallback failed: %+v %v", metadata, err)
	}
}

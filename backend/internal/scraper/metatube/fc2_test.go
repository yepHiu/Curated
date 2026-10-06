package metatube

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"curated-backend/internal/scraper"
	fc2source "curated-backend/internal/scraper/fc2"
	"go.uber.org/zap"
)

// TestFC2Enrichment 验证失败来源不阻断有效字段，查询最多两路且图片来源独立。
func TestFC2Enrichment(t *testing.T) {
	var active, peak atomic.Int32
	service := &Service{logger: zap.NewNop(), health: map[string]ProviderRuntimeHealth{}}
	service.fc2Lookup = func(ctx context.Context, provider, id, number string) (scraper.Metadata, error) {
		// 两路并发中保留一段重叠时间，验证全局槽位上限。
		current := active.Add(1)
		defer active.Add(-1)
		for old := peak.Load(); current > old && !peak.CompareAndSwap(old, current); old = peak.Load() {
		}
		time.Sleep(10 * time.Millisecond)
		if provider == "fc2hub" {
			return scraper.Metadata{}, errors.New("HTTP 403 forbidden: browser verification")
		}
		metadata := scraper.Metadata{MovieID: id, Number: number, Title: provider + " title", Provider: provider, Homepage: "https://" + provider + "/movie"}
		if provider == "FC2" {
			metadata.CoverURL = "https://images/official.jpg"
		}
		if provider == "JavDB" {
			metadata.Actors = []string{"Actor"}
			metadata.RuntimeMinutes = 80
		}
		return metadata, nil
	}
	metadata, err := service.Scrape(context.Background(), "movie", "FC2-3977618", scraper.MovieScrapeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if peak.Load() > 2 || metadata.Provider != "FC2" || len(metadata.Actors) != 1 || metadata.RuntimeMinutes != 80 {
		t.Fatalf("bad enrichment %+v peak=%d", metadata, peak.Load())
	}
	if metadata.AssetSources[metadata.CoverURL].Provider != "FC2" {
		t.Fatalf("image source lost %+v", metadata.AssetSources)
	}
	if service.ProviderRuntimeHealth("fc2hub").CooldownUntil == "" {
		t.Fatal("403 did not trigger cooldown")
	}
}

// TestFC2ProviderExposure 验证专用来源可选择，但未注册的来源不进入普通影片自动链。
func TestFC2ProviderExposure(t *testing.T) {
	service, err := NewService(zap.NewNop(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	names := strings.Join(service.ListMovieProviderNames(), ",")
	for _, name := range []string{"JavDB", "PPVDataBank"} {
		if !strings.Contains(names, name) {
			t.Fatalf("missing FC2 provider %s", name)
		}
		for _, candidate := range service.PreferredMovieProviderChain("auto-cn-friendly") {
			if candidate == name {
				t.Fatalf("FC2-only adapter in ordinary chain: %s", name)
			}
		}
	}
}

// TestFC2WishlistIdentityError 验证错番号保持愿望清单的人工复核错误标识。
func TestFC2WishlistIdentityError(t *testing.T) {
	service := &Service{logger: zap.NewNop(), health: map[string]ProviderRuntimeHealth{}}
	service.fc2Lookup = func(ctx context.Context, provider, id, number string) (scraper.Metadata, error) { // 模拟详情身份冲突。
		return scraper.Metadata{}, fc2source.ErrIdentity
	}
	_, err := service.ScrapeWishlist(context.Background(), "wishlist", "FC2-3977618", scraper.MovieScrapeOptions{Provider: "FC2"})
	if err == nil || !strings.Contains(err.Error(), "WISHLIST_IDENTITY_MISMATCH") || !errors.Is(err, fc2source.ErrIdentity) {
		t.Fatalf("wishlist mismatch lost: %v", err)
	}
}

// TestFC2CooldownRespectsExplicitRetry 验证自动查询避开验证页，单源重试仍可再次获取。
func TestFC2CooldownRespectsExplicitRetry(t *testing.T) {
	var calls atomic.Int32
	service := &Service{logger: zap.NewNop(), health: map[string]ProviderRuntimeHealth{}}
	service.fc2Lookup = func(ctx context.Context, provider, id, number string) (scraper.Metadata, error) { // 只统计触达站点的次数。
		calls.Add(1)
		return scraper.Metadata{}, errors.New("HTTP 403 forbidden")
	}
	providers := []string{"fc2hub"}
	service.collectFC2(context.Background(), "movie", "FC2-3977618", providers, false)
	service.collectFC2(context.Background(), "movie", "FC2-3977618", providers, false)
	service.Scrape(context.Background(), "movie", "FC2-3977618", scraper.MovieScrapeOptions{Provider: "fc2hub"})
	if calls.Load() != 2 {
		t.Fatalf("cooldown/explicit retry calls=%d", calls.Load())
	}
}

// TestFC2SpecifiedSource 验证单源和自动生成链遵循各自语义。
func TestFC2SpecifiedSource(t *testing.T) {
	for _, test := range []struct {
		options scraper.MovieScrapeOptions
		want    int
	}{
		{scraper.MovieScrapeOptions{Provider: "FC2"}, 1},
		{scraper.MovieScrapeOptions{ProviderChain: []string{"JavBus", "FC2"}}, 1},
		{scraper.MovieScrapeOptions{ProviderChain: []string{"JavBus"}, Automatic: true}, 4},
	} {
		t.Run(strings.Join(test.options.ProviderChain, ",")+test.options.Provider, func(t *testing.T) { // 每次独立统计允许访问的来源数。
			var calls atomic.Int32
			service := &Service{logger: zap.NewNop(), health: map[string]ProviderRuntimeHealth{}}
			service.fc2Lookup = func(ctx context.Context, provider, id, number string) (scraper.Metadata, error) { // 返回精确番号，统计是否扩大来源。
				calls.Add(1)
				return scraper.Metadata{MovieID: id, Number: number, Provider: provider, Title: "Title", Homepage: "https://example/movie"}, nil
			}
			if _, err := service.Scrape(context.Background(), "movie", "FC2-3977618", test.options); err != nil {
				t.Fatal(err)
			}
			if int(calls.Load()) != test.want {
				t.Fatalf("calls=%d want=%d", calls.Load(), test.want)
			}
		})
	}
	if _, err := fc2Providers(scraper.MovieScrapeOptions{Provider: "JavBus"}); err == nil {
		t.Fatal("unsupported specified source accepted")
	}
}

// TestFC2FailureDetails 验证所有来源失败时仍能定位访问阻断。
func TestFC2FailureDetails(t *testing.T) {
	service := &Service{logger: zap.NewNop(), health: map[string]ProviderRuntimeHealth{}}
	service.fc2Lookup = func(ctx context.Context, provider, id, number string) (scraper.Metadata, error) { // 模拟站点返回验证阻断。
		if provider == "FC2" {
			return scraper.Metadata{}, fc2source.ErrNotFound
		}
		return scraper.Metadata{}, errors.New("HTTP 403 forbidden: browser verification")
	}
	_, err := service.Scrape(context.Background(), "movie", "FC2-3977618", scraper.MovieScrapeOptions{})
	if err == nil || !strings.Contains(err.Error(), "fc2hub") || !strings.Contains(err.Error(), "HTTP 403") || classifyProviderError(err) != "hotlink_denied" {
		t.Fatalf("failure details lost: %v", err)
	}
}

// TestMergeFC2AssetSource 验证主来源没有图片时，补全图片不会继承错误 Referer。
func TestMergeFC2AssetSource(t *testing.T) {
	metadata := mergeFC2Metadata([]scraper.Metadata{{Title: "Javten", Provider: "fc2hub", Homepage: "https://javten/movie"}, {Title: "Official", Provider: "FC2", Homepage: "https://fc2/article", CoverURL: "https://images/cover", PreviewImages: []string{"https://images/sample"}}})
	if metadata.Provider != "fc2hub" || metadata.AssetSources[metadata.CoverURL].Homepage != "https://fc2/article" || metadata.AssetSources[metadata.PreviewImages[0]].Provider != "FC2" {
		t.Fatalf("mixed source image context lost %+v", metadata)
	}
}

package metatube

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/metatube-community/metatube-sdk-go/collection/sets"
	"github.com/metatube-community/metatube-sdk-go/collection/slices"
	"github.com/metatube-community/metatube-sdk-go/common/comparer"
	mtnum "github.com/metatube-community/metatube-sdk-go/common/number"
	"github.com/metatube-community/metatube-sdk-go/database"
	"github.com/metatube-community/metatube-sdk-go/engine"
	"github.com/metatube-community/metatube-sdk-go/engine/providerid"
	"github.com/metatube-community/metatube-sdk-go/model"
	"go.uber.org/zap"
	"gorm.io/datatypes"

	"jav-shadcn/backend/internal/scraper"
)

// Metatube movie providers dedicated to FC2 PPV IDs (order: higher official priority first).
// Keep in sync with github.com/metatube-community/metatube-sdk-go/engine/register.go for your version.
var fc2MovieProviderNames = []string{"FC2", "fc2hub"}

// providerHealthMaxLatencyMs: health probe round-trip above this is reported as fail (not merely slow).
const providerHealthMaxLatencyMs int64 = 5000

func isFC2MovieProviderName(name string) bool {
	switch strings.TrimSpace(name) {
	case "FC2", "fc2hub":
		return true
	default:
		return false
	}
}

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

// ListMovieProviderNames returns sorted Metatube movie provider names registered in this engine build.
func (s *Service) ListMovieProviderNames() []string {
	m := s.engine.GetMovieProviders()
	out := make([]string, 0, len(m))
	for name := range m {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func firstValidMovieSearchResult(results []*model.MovieSearchResult) *model.MovieSearchResult {
	for _, r := range results {
		if r != nil && r.IsValid() {
			return r
		}
	}
	return nil
}

// rankMovieSearchResults dedupes and sorts like engine.SearchMovieAll post-processing.
func (s *Service) rankMovieSearchResults(keyword string, results []*model.MovieSearchResult) []*model.MovieSearchResult {
	msr := sets.NewOrderedSetWithHash(func(v *model.MovieSearchResult) string { return v.Provider + v.ID })
	msr.Add(results...)
	out := msr.AsSlice()
	ps := new(slices.WeightedSlice[*model.MovieSearchResult, float64])
	for _, result := range out {
		if !result.IsValid() {
			continue
		}
		if _, err := s.engine.GetMovieProviderByName(result.Provider); err != nil {
			continue
		}
		priority := comparer.Compare(keyword, result.Number) * s.engine.MustGetMovieProviderByName(result.Provider).Priority()
		ps.Append(result, priority)
	}
	return ps.SortFunc(sort.Stable).Slice()
}

// searchMovieFC2Providers queries only FC2-dedicated Metatube providers (no fanza/javbus etc.).
func (s *Service) searchMovieFC2Providers(ctx context.Context, keyword string) ([]*model.MovieSearchResult, error) {
	var merged []*model.MovieSearchResult
	for _, name := range fc2MovieProviderNames {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		if _, err := s.engine.GetMovieProviderByName(name); err != nil {
			continue
		}
		r, err := s.engine.SearchMovie(keyword, name, false)
		if err != nil || len(r) == 0 {
			continue
		}
		merged = append(merged, r...)
	}
	if len(merged) == 0 {
		return nil, fmt.Errorf("no results from FC2 metadata providers for %s", keyword)
	}
	return s.rankMovieSearchResults(mtnum.Trim(keyword), merged), nil
}

func (s *Service) Scrape(ctx context.Context, movieID string, number string, opts scraper.MovieScrapeOptions) (scraper.Metadata, error) {
	select {
	case <-ctx.Done():
		return scraper.Metadata{}, ctx.Err()
	default:
	}

	isFC2 := mtnum.IsFC2(mtnum.Trim(number))

	// Determine provider chain to use
	chain := s.resolveProviderChain(opts, isFC2)
	hasChain := len(chain) > 0

	// Log scraping strategy
	if hasChain {
		s.logger.Info("scraping metadata with provider chain",
			zap.String("number", number),
			zap.String("movieId", movieID),
			zap.Strings("chain", chain),
			zap.Bool("fc2", isFC2),
		)
	} else if isFC2 {
		s.logger.Info("scraping metadata",
			zap.String("number", number),
			zap.String("movieId", movieID),
			zap.String("metadataProvider", "fc2-providers"),
		)
	} else {
		s.logger.Info("scraping metadata", zap.String("number", number), zap.String("movieId", movieID), zap.String("metadataProvider", "auto"))
	}

	// Try providers in sequence if chain is specified
	if hasChain {
		return s.scrapeWithChain(ctx, movieID, number, chain)
	}

	// Fall back to original logic (single provider or auto)
	return s.scrapeSingleOrAuto(ctx, movieID, number, opts.Provider, isFC2)
}

// resolveProviderChain determines the effective provider chain from options and FC2 status.
func (s *Service) resolveProviderChain(opts scraper.MovieScrapeOptions, isFC2 bool) []string {
	// Clean and filter chain
	var chain []string
	for _, p := range opts.ProviderChain {
		if name := strings.TrimSpace(p); name != "" {
			chain = append(chain, name)
		}
	}

	if len(chain) == 0 {
		// Fall back to single provider if specified
		if single := strings.TrimSpace(opts.Provider); single != "" {
			chain = []string{single}
		}
	}

	// For FC2 content, filter to FC2-only providers
	if isFC2 && len(chain) > 0 {
		var fc2Chain []string
		for _, name := range chain {
			if isFC2MovieProviderName(name) {
				fc2Chain = append(fc2Chain, name)
			}
		}
		if len(fc2Chain) == 0 {
			// Chain has no FC2 providers, ignore chain and use default FC2 providers
			return nil
		}
		return fc2Chain
	}

	return chain
}

// scrapeWithChain tries each provider in the chain sequentially until one succeeds.
func (s *Service) scrapeWithChain(ctx context.Context, movieID, number string, chain []string) (scraper.Metadata, error) {
	var lastErr error
	for i, providerName := range chain {
		select {
		case <-ctx.Done():
			return scraper.Metadata{}, ctx.Err()
		default:
		}

		if _, err := s.engine.GetMovieProviderByName(providerName); err != nil {
			lastErr = fmt.Errorf("unknown provider %q", providerName)
			s.logger.Warn("skipping unknown provider in chain",
				zap.String("provider", providerName),
				zap.Int("index", i),
			)
			continue
		}

		s.logger.Info("trying provider in chain",
			zap.String("number", number),
			zap.String("provider", providerName),
			zap.Int("index", i),
			zap.Int("total", len(chain)),
		)

		results, err := s.engine.SearchMovie(number, providerName, false)
		if err != nil {
			lastErr = err
			s.logger.Warn("search failed in chain",
				zap.String("number", number),
				zap.String("provider", providerName),
				zap.Error(err),
			)
			continue
		}

		first := firstValidMovieSearchResult(results)
		if first == nil {
			lastErr = fmt.Errorf("no results from %s", providerName)
			s.logger.Warn("no results from provider in chain",
				zap.String("number", number),
				zap.String("provider", providerName),
			)
			continue
		}

		// Found a valid result
		s.logger.Info("search result selected from chain",
			zap.String("number", number),
			zap.String("provider", first.Provider),
			zap.String("providerMovieId", first.ID),
			zap.String("title", first.Title),
			zap.Int("tried", i+1),
		)

		return s.fetchMovieInfo(ctx, movieID, number, first)
	}

	// All providers in chain failed
	return scraper.Metadata{}, fmt.Errorf("all providers in chain failed for %s: %w", number, lastErr)
}

// scrapeSingleOrAuto handles single provider or automatic (all sources) scraping.
func (s *Service) scrapeSingleOrAuto(ctx context.Context, movieID, number, prefer string, isFC2 bool) (scraper.Metadata, error) {
	var (
		results []*model.MovieSearchResult
		err     error
	)
	if prefer != "" {
		if _, gerr := s.engine.GetMovieProviderByName(prefer); gerr != nil {
			return scraper.Metadata{}, fmt.Errorf("unknown movie metadata provider %q", prefer)
		}
		results, err = s.engine.SearchMovie(number, prefer, false)
		if err != nil {
			return scraper.Metadata{}, fmt.Errorf("search failed for %s on provider %q: %w", number, prefer, err)
		}
	} else if isFC2 {
		results, err = s.searchMovieFC2Providers(ctx, number)
		if err != nil {
			return scraper.Metadata{}, fmt.Errorf("search failed for %s: %w", number, err)
		}
	} else {
		results, err = s.engine.SearchMovieAll(number, false)
		if err != nil {
			return scraper.Metadata{}, fmt.Errorf("search failed for %s: %w", number, err)
		}
	}

	first := firstValidMovieSearchResult(results)
	if first == nil {
		if prefer != "" {
			return scraper.Metadata{}, fmt.Errorf("no results found for %s on metadata provider %q (try automatic provider selection in settings)", number, prefer)
		}
		return scraper.Metadata{}, fmt.Errorf("no results found for %s", number)
	}
	s.logger.Info("search result selected",
		zap.String("number", number),
		zap.String("provider", first.Provider),
		zap.String("providerMovieId", first.ID),
		zap.String("title", first.Title),
		zap.Int("totalResults", len(results)),
	)

	return s.fetchMovieInfo(ctx, movieID, number, first)
}

// effectivePosterURLs merges Metatube's thumb/cover vs big_* fields (some providers only fill the latter).
func effectivePosterURLs(info *model.MovieInfo) (coverURL, thumbURL string) {
	if info == nil {
		return "", ""
	}
	cover := strings.TrimSpace(info.CoverURL)
	if cover == "" {
		cover = strings.TrimSpace(info.BigCoverURL)
	}
	thumb := strings.TrimSpace(info.ThumbURL)
	if thumb == "" {
		thumb = strings.TrimSpace(info.BigThumbURL)
	}
	switch {
	case thumb == "" && cover != "":
		thumb = cover
	case cover == "" && thumb != "":
		cover = thumb
	}
	return cover, thumb
}

// fetchMovieInfo fetches detailed movie info from a search result.
func (s *Service) fetchMovieInfo(ctx context.Context, movieID, number string, result *model.MovieSearchResult) (scraper.Metadata, error) {
	pid, err := providerid.New(result.Provider, result.ID)
	if err != nil {
		return scraper.Metadata{}, fmt.Errorf("invalid provider result for %s: provider=%s id=%s: %w", number, result.Provider, result.ID, err)
	}

	select {
	case <-ctx.Done():
		return scraper.Metadata{}, ctx.Err()
	default:
	}

	info, err := s.engine.GetMovieInfoByProviderID(pid, true)
	if err != nil {
		return scraper.Metadata{}, fmt.Errorf("get movie info failed for %s (provider=%s): %w", number, result.Provider, err)
	}

	coverURL, thumbURL := effectivePosterURLs(info)
	s.logger.Info("metadata fetched",
		zap.String("number", number),
		zap.String("title", info.Title),
		zap.String("maker", info.Maker),
		zap.Int("actors", len(info.Actors)),
		zap.Int("genres", len(info.Genres)),
		zap.Int("previewImages", len(info.PreviewImages)),
		zap.String("coverURL", coverURL),
		zap.String("thumbURL", thumbURL),
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
		CoverURL:        coverURL,
		ThumbURL:        thumbURL,
		PreviewVideoURL: info.PreviewVideoURL,
		PreviewImages:   cleanStrings(info.PreviewImages),
	}, nil
}

func (s *Service) ScrapeActor(ctx context.Context, displayName string) (scraper.ActorProfile, error) {
	out := scraper.ActorProfile{DisplayName: strings.TrimSpace(displayName)}
	if out.DisplayName == "" {
		return out, fmt.Errorf("empty actor name")
	}

	select {
	case <-ctx.Done():
		return out, ctx.Err()
	default:
	}

	var (
		results []*model.ActorSearchResult
		lastErr error
	)
	for _, kw := range actorSearchKeywords(out.DisplayName) {
		rs, err := s.engine.SearchActorAll(kw, true)
		if err != nil {
			lastErr = err
			s.logger.Info("actor search keyword skipped",
				zap.String("displayName", out.DisplayName),
				zap.String("keyword", kw),
				zap.Error(err),
			)
			continue
		}
		if len(rs) > 0 {
			results = rs
			s.logger.Info("actor search hit",
				zap.String("displayName", out.DisplayName),
				zap.String("keyword", kw),
				zap.Int("results", len(rs)),
			)
			break
		}
	}
	if len(results) == 0 {
		if lastErr != nil {
			return out, fmt.Errorf("actor search failed for %q: %w", out.DisplayName, lastErr)
		}
		return out, fmt.Errorf("no actor search results for %q", out.DisplayName)
	}

	best := pickBestActorSearchResult(results, out.DisplayName)
	if best == nil {
		return out, fmt.Errorf("no valid actor search results for %q", out.DisplayName)
	}
	pid, err := providerid.New(best.Provider, best.ID)
	if err != nil {
		return out, fmt.Errorf("invalid actor provider result for %q: provider=%s id=%s: %w", out.DisplayName, best.Provider, best.ID, err)
	}

	select {
	case <-ctx.Done():
		return out, ctx.Err()
	default:
	}

	// lazy=false：避免引擎本地 actor_metadata 里不完整旧记录被当成有效结果，从而跳过站点拉取与 Gfriends 头像注入。
	info, err := s.engine.GetActorInfoByProviderID(pid, false)
	if err != nil {
		return out, fmt.Errorf("get actor info failed for %q (provider=%s): %w", out.DisplayName, best.Provider, err)
	}

	s.logger.Info("actor metadata fetched",
		zap.String("displayName", out.DisplayName),
		zap.String("provider", info.Provider),
		zap.String("providerActorId", info.ID),
		zap.Int("images", len(info.Images)),
	)

	var avatar string
	for _, u := range info.Images {
		u = strings.TrimSpace(u)
		if u != "" {
			avatar = u
			break
		}
	}

	return scraper.ActorProfile{
		DisplayName:     out.DisplayName,
		AvatarURL:       avatar,
		Summary:         strings.TrimSpace(info.Summary),
		Homepage:        strings.TrimSpace(info.Homepage),
		Provider:        strings.TrimSpace(info.Provider),
		ProviderActorID: strings.TrimSpace(info.ID),
		Height:          info.Height,
		Birthday:        actorBirthdayString(info.Birthday),
	}, nil
}

// actorSearchKeywords 尝试多种写法，提高与刮削元数据里演员名的匹配率（空格、全角空格等）。
func actorSearchKeywords(displayName string) []string {
	s := strings.TrimSpace(displayName)
	if s == "" {
		return nil
	}
	seen := make(map[string]struct{})
	var out []string
	add := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		if _, ok := seen[v]; ok {
			return
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	add(s)
	norm := strings.Join(strings.Fields(s), " ")
	add(norm)
	noSpace := strings.ReplaceAll(s, " ", "")
	noSpace = strings.ReplaceAll(noSpace, "\u3000", "")
	add(noSpace)
	return out
}

func pickBestActorSearchResult(results []*model.ActorSearchResult, displayName string) *model.ActorSearchResult {
	var best *model.ActorSearchResult
	bestScore := -1.0
	for _, r := range results {
		if r == nil || !r.IsValid() {
			continue
		}
		sc := comparer.Compare(strings.TrimSpace(r.Name), displayName)
		if best == nil || sc > bestScore {
			best = r
			bestScore = sc
		}
	}
	return best
}

func actorBirthdayString(d datatypes.Date) string {
	t := time.Time(d)
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("2006-01-02")
}

// formatReleaseDate 将 Metatube 的 datatypes.Date（底层 time.Time）规范为 UTC 日历日 YYYY-MM-DD。
// 原先使用 fmt.Sprint 会得到 "2006-01-02 00:00:00 +0000 UTC" 等字符串，入库/展示容易误判为「日期错了」。
func formatReleaseDate(value any) string {
	switch v := value.(type) {
	case datatypes.Date:
		return releaseDateFromTime(time.Time(v))
	case time.Time:
		return releaseDateFromTime(v)
	default:
		s := strings.TrimSpace(fmt.Sprint(value))
		if s == "" || strings.HasPrefix(s, "0001-01-01") {
			return ""
		}
		// 兼容历史或异常路径里已存在的 "YYYY-MM-DD ..." 前缀
		if i := strings.IndexByte(s, ' '); i == 10 {
			head := s[:i]
			if _, err := time.Parse("2006-01-02", head); err == nil {
				return head
			}
		}
		if len(s) == 10 {
			if _, err := time.Parse("2006-01-02", s); err == nil {
				return s
			}
		}
		return s
	}
}

func releaseDateFromTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("2006-01-02")
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

// ListProviders returns sorted Metatube movie provider names.
func (s *Service) ListProviders() []string {
	return s.ListMovieProviderNames()
}

// isTransientHealthProbeErr matches errors that often succeed on retry (e.g. AVBASE GetBuildID GET https://www.avbase.net/).
func isTransientHealthProbeErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "eof") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "i/o timeout") ||
		strings.Contains(msg, "timeout") && strings.Contains(msg, "client")
}

// CheckProviderHealth pings a single provider and returns its health status.
// It performs a lightweight search to verify the provider is responsive.
func (s *Service) CheckProviderHealth(ctx context.Context, name string) (status string, latencyMs int64, err error) {
	start := time.Now()

	_, err = s.engine.GetMovieProviderByName(name)
	if err != nil {
		return "fail", time.Since(start).Milliseconds(), fmt.Errorf("provider not found: %w", err)
	}

	// Use a lightweight search to check provider health
	// Search for a known popular ID that should exist on most providers
	testKeyword := "SSIS"

	const maxAttempts = 3
	var results []*model.MovieSearchResult
	var searchErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return "fail", time.Since(start).Milliseconds(), ctx.Err()
			case <-time.After(400 * time.Millisecond):
			}
		}
		results, searchErr = s.engine.SearchMovie(testKeyword, name, false)
		if searchErr == nil {
			break
		}
		if !isTransientHealthProbeErr(searchErr) || attempt == maxAttempts-1 {
			break
		}
	}
	latency := time.Since(start).Milliseconds()

	if searchErr != nil {
		return "fail", latency, searchErr
	}

	if latency > providerHealthMaxLatencyMs {
		return "fail", latency, fmt.Errorf("latency %dms exceeds %dms threshold", latency, providerHealthMaxLatencyMs)
	}

	// If we got any valid results, consider the provider healthy
	if len(results) > 0 {
		return "ok", latency, nil
	}

	// No results but no error - provider is responsive but maybe the test keyword returned nothing
	return "ok", latency, nil
}

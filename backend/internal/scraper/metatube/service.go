package metatube

import (
	"context"
	"fmt"
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

	prefer := strings.TrimSpace(opts.Provider)
	isFC2 := mtnum.IsFC2(mtnum.Trim(number))
	if isFC2 && prefer != "" && !isFC2MovieProviderName(prefer) {
		s.logger.Warn("FC2 number: ignoring non-FC2 metadata provider; using FC2-only sources",
			zap.String("number", number),
			zap.String("movieId", movieID),
			zap.String("ignoredProvider", prefer),
		)
		prefer = ""
	}

	if prefer != "" {
		s.logger.Info("scraping metadata",
			zap.String("number", number),
			zap.String("movieId", movieID),
			zap.String("metadataProvider", prefer),
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

package app

import (
	"context"
	"hash/fnv"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"

	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

const homepageDailyRecommendationGenerationVersion = "v4"

const (
	homepageDailyExposureLookbackDays   = 14
	homepageDailyExposurePenaltyFloor   = 0.75
	homepageDailyExposurePenaltyScale   = 3.0
	homepageDailyActorDiversityPenalty  = 2.25
	homepageDailyStudioDiversityPenalty = 1.75
)

var homepageDailyRecentExclusionWindows = []int{7, 5, 3, 1, 0}

type homepageDailyCandidate struct {
	movie     contracts.MovieListItemDTO
	score     float64
	hashScore uint64
}

type homepageDailySelectionState struct {
	actorCounts  map[string]int
	studioCounts map[string]int
}

type homepageDailyExclusionPolicy struct {
	windowDays int
	movieIDs   map[string]struct{}
}

func (a *App) GetOrCreateHomepageDailyRecommendations(ctx context.Context, dateUTC string) (contracts.HomepageDailyRecommendationsDTO, error) {
	if strings.TrimSpace(dateUTC) == "" {
		dateUTC = time.Now().UTC().Format("2006-01-02")
	}

	if snapshot, ok, err := a.store.GetHomepageDailyRecommendationSnapshot(ctx, dateUTC); err != nil {
		return contracts.HomepageDailyRecommendationsDTO{}, err
	} else if ok {
		return contracts.HomepageDailyRecommendationsDTO{
			DateUTC:                snapshot.DateUTC,
			GeneratedAt:            snapshot.GeneratedAt,
			GenerationVersion:      snapshot.GenerationVersion,
			HeroMovieIDs:           append([]string(nil), snapshot.HeroMovieIDs...),
			RecommendationMovieIDs: append([]string(nil), snapshot.RecommendationMovieIDs...),
		}, nil
	}

	dto, err := a.generateHomepageDailyRecommendations(ctx, dateUTC)
	if err != nil {
		return contracts.HomepageDailyRecommendationsDTO{}, err
	}

	if err := a.store.UpsertHomepageDailyRecommendationSnapshot(ctx, storage.HomepageDailyRecommendationSnapshot{
		DateUTC:                dto.DateUTC,
		HeroMovieIDs:           dto.HeroMovieIDs,
		RecommendationMovieIDs: dto.RecommendationMovieIDs,
		GeneratedAt:            dto.GeneratedAt,
		GenerationVersion:      dto.GenerationVersion,
	}); err != nil {
		return contracts.HomepageDailyRecommendationsDTO{}, err
	}

	return dto, nil
}

func (a *App) RegenerateHomepageDailyRecommendations(ctx context.Context, dateUTC string) (contracts.HomepageDailyRecommendationsDTO, error) {
	if strings.TrimSpace(dateUTC) == "" {
		dateUTC = time.Now().UTC().Format("2006-01-02")
	}

	dto, err := a.generateHomepageDailyRecommendations(ctx, dateUTC)
	if err != nil {
		return contracts.HomepageDailyRecommendationsDTO{}, err
	}

	if err := a.store.UpsertHomepageDailyRecommendationSnapshot(ctx, storage.HomepageDailyRecommendationSnapshot{
		DateUTC:                dto.DateUTC,
		HeroMovieIDs:           dto.HeroMovieIDs,
		RecommendationMovieIDs: dto.RecommendationMovieIDs,
		GeneratedAt:            dto.GeneratedAt,
		GenerationVersion:      dto.GenerationVersion,
	}); err != nil {
		return contracts.HomepageDailyRecommendationsDTO{}, err
	}

	return dto, nil
}

func (a *App) generateHomepageDailyRecommendations(ctx context.Context, dateUTC string) (contracts.HomepageDailyRecommendationsDTO, error) {
	page, err := a.store.ListMovies(ctx, contracts.ListMoviesRequest{
		Limit:  10000,
		Offset: 0,
	})
	if err != nil {
		return contracts.HomepageDailyRecommendationsDTO{}, err
	}

	exposurePenaltyByMovieID, err := a.buildHomepageExposurePenaltyMap(ctx, dateUTC)
	if err != nil {
		return contracts.HomepageDailyRecommendationsDTO{}, err
	}

	recentSnapshots, err := a.listHomepageRecentSnapshots(ctx, dateUTC, homepageDailyExposureLookbackDays)
	if err != nil {
		return contracts.HomepageDailyRecommendationsDTO{}, err
	}
	exclusionLadder, err := buildHomepageDailyExclusionLadder(dateUTC, recentSnapshots, homepageDailyRecentExclusionWindows)
	if err != nil {
		return contracts.HomepageDailyRecommendationsDTO{}, err
	}

	allCandidates := rankHomepageDailyCandidates(page.Items, dateUTC, exposurePenaltyByMovieID)
	selected := make(map[string]struct{})
	selectionState := homepageDailySelectionState{
		actorCounts:  make(map[string]int),
		studioCounts: make(map[string]int),
	}

	heroIDs, heroExclusionWindowUsed := selectHomepageDailyIDs(
		allCandidates,
		selected,
		selectionState,
		8,
		true,
		exclusionLadder,
	)
	recommendationIDs, recommendationExclusionWindowUsed := selectHomepageDailyIDs(
		allCandidates,
		selected,
		selectionState,
		6,
		false,
		exclusionLadder,
	)

	dto := contracts.HomepageDailyRecommendationsDTO{
		DateUTC:                dateUTC,
		GeneratedAt:            time.Now().UTC().Format(time.RFC3339),
		GenerationVersion:      homepageDailyRecommendationGenerationVersion,
		HeroMovieIDs:           heroIDs,
		RecommendationMovieIDs: recommendationIDs,
	}

	if a.logger != nil {
		a.logger.Info("homepage daily recommendations generated",
			zap.String("dateUTC", dateUTC),
			zap.Int("candidateCount", len(allCandidates)),
			zap.Int("historyPenaltyMovieCount", len(exposurePenaltyByMovieID)),
			zap.Int("recentSnapshotCount", len(recentSnapshots)),
			zap.Int("heroExclusionWindowDays", heroExclusionWindowUsed),
			zap.Int("recommendationExclusionWindowDays", recommendationExclusionWindowUsed),
			zap.Strings("heroMovieIDs", heroIDs),
			zap.Strings("recommendationMovieIDs", recommendationIDs),
		)
	}

	return dto, nil
}

func (a *App) buildHomepageExposurePenaltyMap(ctx context.Context, dateUTC string) (map[string]float64, error) {
	currentDate, err := time.Parse("2006-01-02", dateUTC)
	if err != nil {
		return nil, err
	}

	startDateUTC := currentDate.AddDate(0, 0, -homepageDailyExposureLookbackDays).Format("2006-01-02")
	endDateUTC := currentDate.AddDate(0, 0, -1).Format("2006-01-02")
	if startDateUTC > endDateUTC {
		return map[string]float64{}, nil
	}

	snapshots, err := a.store.ListHomepageDailyRecommendationSnapshotsInRange(ctx, startDateUTC, endDateUTC)
	if err != nil {
		return nil, err
	}

	penaltyByMovieID := make(map[string]float64)
	for _, snapshot := range snapshots {
		snapshotDate, err := time.Parse("2006-01-02", snapshot.DateUTC)
		if err != nil {
			continue
		}
		daysAgo := int(currentDate.Sub(snapshotDate).Hours() / 24)
		if daysAgo <= 0 || daysAgo > homepageDailyExposureLookbackDays {
			continue
		}

		seenToday := make(map[string]struct{})
		for _, movieID := range append(append([]string{}, snapshot.HeroMovieIDs...), snapshot.RecommendationMovieIDs...) {
			normalizedMovieID := strings.TrimSpace(movieID)
			if normalizedMovieID == "" {
				continue
			}
			if _, ok := seenToday[normalizedMovieID]; ok {
				continue
			}
			seenToday[normalizedMovieID] = struct{}{}
			penaltyByMovieID[normalizedMovieID] += homepageExposurePenalty(daysAgo)
		}
	}

	return penaltyByMovieID, nil
}

func (a *App) listHomepageRecentSnapshots(
	ctx context.Context,
	dateUTC string,
	lookbackDays int,
) ([]storage.HomepageDailyRecommendationSnapshot, error) {
	currentDate, err := time.Parse("2006-01-02", dateUTC)
	if err != nil {
		return nil, err
	}
	if lookbackDays <= 0 {
		return []storage.HomepageDailyRecommendationSnapshot{}, nil
	}

	startDateUTC := currentDate.AddDate(0, 0, -lookbackDays).Format("2006-01-02")
	endDateUTC := currentDate.AddDate(0, 0, -1).Format("2006-01-02")
	if startDateUTC > endDateUTC {
		return []storage.HomepageDailyRecommendationSnapshot{}, nil
	}

	return a.store.ListHomepageDailyRecommendationSnapshotsInRange(ctx, startDateUTC, endDateUTC)
}

func buildHomepageDailyExclusionLadder(
	dateUTC string,
	snapshots []storage.HomepageDailyRecommendationSnapshot,
	windows []int,
) ([]homepageDailyExclusionPolicy, error) {
	currentDate, err := time.Parse("2006-01-02", dateUTC)
	if err != nil {
		return nil, err
	}

	policies := make([]homepageDailyExclusionPolicy, 0, len(windows))
	for _, windowDays := range windows {
		policies = append(policies, homepageDailyExclusionPolicy{
			windowDays: windowDays,
			movieIDs:   buildHomepageRecentExclusionSet(currentDate, snapshots, windowDays),
		})
	}
	return policies, nil
}

func buildHomepageRecentExclusionSet(
	currentDate time.Time,
	snapshots []storage.HomepageDailyRecommendationSnapshot,
	windowDays int,
) map[string]struct{} {
	excluded := make(map[string]struct{})
	if windowDays <= 0 {
		return excluded
	}

	for _, snapshot := range snapshots {
		snapshotDate, err := time.Parse("2006-01-02", snapshot.DateUTC)
		if err != nil {
			continue
		}
		daysAgo := int(currentDate.Sub(snapshotDate).Hours() / 24)
		if daysAgo <= 0 || daysAgo > windowDays {
			continue
		}
		for _, movieID := range append(snapshot.HeroMovieIDs, snapshot.RecommendationMovieIDs...) {
			normalizedMovieID := strings.TrimSpace(movieID)
			if normalizedMovieID == "" {
				continue
			}
			excluded[normalizedMovieID] = struct{}{}
		}
	}

	return excluded
}

func rankHomepageDailyCandidates(items []contracts.MovieListItemDTO, dateUTC string, extraPenalty map[string]float64) []homepageDailyCandidate {
	candidates := make([]homepageDailyCandidate, 0, len(items))
	for _, movie := range items {
		if strings.TrimSpace(movie.ID) == "" {
			continue
		}
		if strings.TrimSpace(movie.TrashedAt) != "" {
			continue
		}

		score := movie.Rating * 10
		if movie.IsFavorite {
			score += 24
		}
		if strings.TrimSpace(movie.CoverURL) != "" || strings.TrimSpace(movie.ThumbURL) != "" {
			score += 6
		}
		if addedAt := strings.TrimSpace(movie.AddedAt); addedAt != "" {
			score += homepageFreshnessBoost(addedAt, dateUTC)
		}
		if extraPenalty != nil {
			score -= extraPenalty[movie.ID]
		}

		hashScore := homepageDailyHash(dateUTC + "|" + movie.ID)
		candidates = append(candidates, homepageDailyCandidate{
			movie:     movie,
			score:     score,
			hashScore: hashScore,
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		if candidates[i].hashScore != candidates[j].hashScore {
			return candidates[i].hashScore < candidates[j].hashScore
		}
		return candidates[i].movie.ID < candidates[j].movie.ID
	})

	return candidates
}

func selectHomepageDailyIDs(
	candidates []homepageDailyCandidate,
	selected map[string]struct{},
	selectionState homepageDailySelectionState,
	limit int,
	heroOnly bool,
	exclusionLadder []homepageDailyExclusionPolicy,
) ([]string, int) {
	if limit <= 0 {
		return nil, 0
	}

	out := make([]string, 0, limit)
	exclusionWindowUsed := 0

	appendFromPool := func(excludedMovieIDs map[string]struct{}) {
		for len(out) < limit {
			bestIndex := -1
			bestScore := 0.0
			var bestHash uint64
			for index, candidate := range candidates {
				movieID := strings.TrimSpace(candidate.movie.ID)
				if movieID == "" {
					continue
				}
				if _, ok := selected[movieID]; ok {
					continue
				}
				if heroOnly && isFC2MovieCode(candidate.movie.Code) {
					continue
				}
				if _, excluded := excludedMovieIDs[movieID]; excluded {
					continue
				}

				score := candidate.score - homepageDiversityPenalty(candidate.movie, selectionState)
				if bestIndex == -1 || score > bestScore || (score == bestScore && candidate.hashScore < bestHash) {
					bestIndex = index
					bestScore = score
					bestHash = candidate.hashScore
				}
			}
			if bestIndex < 0 {
				return
			}

			chosen := candidates[bestIndex].movie
			movieID := strings.TrimSpace(chosen.ID)
			selected[movieID] = struct{}{}
			accumulateHomepageDiversity(selectionState, chosen)
			out = append(out, movieID)
		}
	}

	for _, policy := range exclusionLadder {
		if len(out) >= limit {
			break
		}
		beforeCount := len(out)
		appendFromPool(policy.movieIDs)
		if len(out) > beforeCount {
			exclusionWindowUsed = policy.windowDays
		}
	}

	return out, exclusionWindowUsed
}

func homepageDiversityPenalty(
	movie contracts.MovieListItemDTO,
	selectionState homepageDailySelectionState,
) float64 {
	penalty := 0.0

	seenActors := make(map[string]struct{})
	for _, actor := range movie.Actors {
		normalizedActor := strings.TrimSpace(actor)
		if normalizedActor == "" {
			continue
		}
		if _, ok := seenActors[normalizedActor]; ok {
			continue
		}
		seenActors[normalizedActor] = struct{}{}
		penalty += float64(selectionState.actorCounts[normalizedActor]) * homepageDailyActorDiversityPenalty
	}

	normalizedStudio := strings.TrimSpace(movie.Studio)
	if normalizedStudio != "" {
		penalty += float64(selectionState.studioCounts[normalizedStudio]) * homepageDailyStudioDiversityPenalty
	}

	return penalty
}

func accumulateHomepageDiversity(selectionState homepageDailySelectionState, movie contracts.MovieListItemDTO) {
	seenActors := make(map[string]struct{})
	for _, actor := range movie.Actors {
		normalizedActor := strings.TrimSpace(actor)
		if normalizedActor == "" {
			continue
		}
		if _, ok := seenActors[normalizedActor]; ok {
			continue
		}
		seenActors[normalizedActor] = struct{}{}
		selectionState.actorCounts[normalizedActor]++
	}

	normalizedStudio := strings.TrimSpace(movie.Studio)
	if normalizedStudio != "" {
		selectionState.studioCounts[normalizedStudio]++
	}
}

func homepageFreshnessBoost(addedAt string, dateUTC string) float64 {
	addedDate, err := time.Parse("2006-01-02", addedAt)
	if err != nil {
		return 0
	}
	currentDate, err := time.Parse("2006-01-02", dateUTC)
	if err != nil {
		return 0
	}
	days := int(currentDate.Sub(addedDate).Hours() / 24)
	switch {
	case days < 0:
		return 0
	case days <= 7:
		return 10
	case days <= 30:
		return 6
	case days <= 90:
		return 3
	default:
		return 0
	}
}

func homepageDailyHash(seed string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(seed))
	return h.Sum64()
}

func homepageExposurePenalty(daysAgo int) float64 {
	if daysAgo <= 0 {
		return 0
	}
	if daysAgo > homepageDailyExposureLookbackDays {
		return 0
	}

	recencyWeight := float64(homepageDailyExposureLookbackDays-daysAgo+1) / float64(homepageDailyExposureLookbackDays)
	return homepageDailyExposurePenaltyFloor + recencyWeight*homepageDailyExposurePenaltyScale
}

func isFC2MovieCode(code string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(code))
	normalized = strings.ReplaceAll(normalized, " ", "")
	normalized = strings.ReplaceAll(normalized, "-", "")
	normalized = strings.ReplaceAll(normalized, "_", "")
	return strings.HasPrefix(normalized, "FC2")
}

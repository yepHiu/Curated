package app

import (
	"context"
	"hash/fnv"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"

	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

const homepageDailyRecommendationGenerationVersion = "v6"

const (
	homepageDailyExposureLookbackDays   = 28
	homepageDailyExposurePenaltyFloor   = 0.75
	homepageDailyExposurePenaltyScale   = 3.0
	homepageDailyActorDiversityPenalty  = 12.0
	homepageDailyStudioDiversityPenalty = 8.0
	homepageDailyCoolingDays            = 14
	homepageDailyHardCoolingDays        = 3
)

var homepageDailyRecentExclusionWindows = []int{14, 10, 7, 5, 3, 1, 0}

type homepageDailyCandidate struct {
	movie     contracts.MovieListItemDTO
	score     float64
	hashScore uint64
	state     storage.HomepageRecommendationState

	// Pre-computed normalized fields to avoid per-iteration allocations.
	uniqueActors     []string
	normalizedStudio string
	recencyFactor    float64 // pre-computed from homepageRecommendationWeight static parts
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
	} else if ok && snapshot.GenerationVersion == homepageDailyRecommendationGenerationVersion {
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
	if err := a.persistHomepageRecommendationStates(ctx, dto); err != nil {
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
	if err := a.persistHomepageRecommendationStates(ctx, dto); err != nil {
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

	recommendationStates, err := a.listHomepageRecommendationStateMap(ctx)
	if err != nil {
		return contracts.HomepageDailyRecommendationsDTO{}, err
	}

	allCandidates := rankHomepageDailyCandidates(page.Items, dateUTC, exposurePenaltyByMovieID, recommendationStates)
	selected := make(map[string]struct{})
	selectionState := homepageDailySelectionState{
		actorCounts:  make(map[string]int),
		studioCounts: make(map[string]int),
	}
	random := rand.New(rand.NewSource(int64(homepageDailyHash(dateUTC + "|" + homepageDailyRecommendationGenerationVersion))))

	heroIDs, heroExclusionWindowUsed := selectHomepageDailyIDs(
		allCandidates,
		selected,
		selectionState,
		8,
		true,
		exclusionLadder,
		random,
	)
	recommendationIDs, recommendationExclusionWindowUsed := selectHomepageDailyIDs(
		allCandidates,
		selected,
		selectionState,
		6,
		false,
		exclusionLadder,
		random,
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
			zap.Int("recommendationStateCount", len(recommendationStates)),
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

func (a *App) listHomepageRecommendationStateMap(ctx context.Context) (map[string]storage.HomepageRecommendationState, error) {
	states, err := a.store.ListHomepageRecommendationStates(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]storage.HomepageRecommendationState, len(states))
	for _, state := range states {
		movieID := strings.TrimSpace(state.MovieID)
		if movieID == "" {
			continue
		}
		out[movieID] = state
	}
	return out, nil
}

func (a *App) persistHomepageRecommendationStates(ctx context.Context, dto contracts.HomepageDailyRecommendationsDTO) error {
	currentStates, err := a.listHomepageRecommendationStateMap(ctx)
	if err != nil {
		return err
	}

	generatedAt := strings.TrimSpace(dto.GeneratedAt)
	if generatedAt == "" {
		generatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	generatedTime, err := time.Parse(time.RFC3339, generatedAt)
	if err != nil {
		generatedTime = time.Now().UTC()
		generatedAt = generatedTime.Format(time.RFC3339)
	}
	recommendedTime := generatedTime
	if parsedDateUTC, err := time.Parse("2006-01-02", dto.DateUTC); err == nil {
		recommendedTime = parsedDateUTC
	}
	recommendedAt := recommendedTime.Format(time.RFC3339)
	skipUntil := recommendedTime.AddDate(0, 0, homepageDailyHardCoolingDays).Format(time.RFC3339)

	selected := append(append([]string{}, dto.HeroMovieIDs...), dto.RecommendationMovieIDs...)
	updates := make([]storage.HomepageRecommendationState, 0, len(selected))
	seen := make(map[string]struct{}, len(selected))
	for _, movieID := range selected {
		normalizedMovieID := strings.TrimSpace(movieID)
		if normalizedMovieID == "" {
			continue
		}
		if _, ok := seen[normalizedMovieID]; ok {
			continue
		}
		seen[normalizedMovieID] = struct{}{}
		state := currentStates[normalizedMovieID]
		state.MovieID = normalizedMovieID
		state.LastRecommendedAt = recommendedAt
		state.RecommendCount++
		state.SkipUntil = skipUntil
		state.UpdatedAt = generatedAt
		updates = append(updates, state)
	}
	return a.store.UpsertHomepageRecommendationStates(ctx, updates)
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

func rankHomepageDailyCandidates(
	items []contracts.MovieListItemDTO,
	dateUTC string,
	extraPenalty map[string]float64,
	states map[string]storage.HomepageRecommendationState,
) []homepageDailyCandidate {
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
			movie:            movie,
			score:            score,
			hashScore:        hashScore,
			state:            states[movie.ID],
			uniqueActors:     normalizeUniqueActors(movie.Actors),
			normalizedStudio: strings.TrimSpace(movie.Studio),
			recencyFactor:    homepageRecencyFactor(states[movie.ID], dateUTC),
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
	random *rand.Rand,
) ([]string, int) {
	if limit <= 0 {
		return nil, 0
	}

	out := make([]string, 0, limit)
	exclusionWindowUsed := 0

	appendFromPool := func(excludedMovieIDs map[string]struct{}) {
		if len(out) >= limit {
			return
		}

		// Build initial active set: candidates not yet selected and not excluded.
		active := make([]int, 0, len(candidates))
		activeSet := make(map[int]struct{}, len(candidates))
		weights := make([]float64, len(candidates))

		// Inverted indexes: actor/studio → candidate indices sharing that attribute.
		actorToCandidates := make(map[string][]int)
		studioToCandidates := make(map[string][]int)

		totalWeight := 0.0

		for idx, candidate := range candidates {
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

			score := candidate.score - diversityPenaltyFast(candidate, selectionState)
			w := homepageCandidateWeight(score, candidate.recencyFactor)
			if w <= 0 {
				continue
			}

			active = append(active, idx)
			activeSet[idx] = struct{}{}
			weights[idx] = w
			totalWeight += w

			for _, actor := range candidate.uniqueActors {
				actorToCandidates[actor] = append(actorToCandidates[actor], idx)
			}
			if candidate.normalizedStudio != "" {
				studioToCandidates[candidate.normalizedStudio] = append(studioToCandidates[candidate.normalizedStudio], idx)
			}
		}

		if len(active) == 0 || totalWeight <= 0 {
			return
		}

		for len(out) < limit {
			if len(active) == 0 || totalWeight <= 0 {
				return
			}

			// Weighted random selection.
			threshold := random.Float64() * totalWeight
			bestIdx := active[len(active)-1]
			for _, idx := range active {
				threshold -= weights[idx]
				if threshold <= 0 {
					bestIdx = idx
					break
				}
			}

			chosen := candidates[bestIdx]
			movieID := strings.TrimSpace(chosen.movie.ID)
			selected[movieID] = struct{}{}
			accumulateHomepageDiversityFast(selectionState, chosen)
			out = append(out, movieID)

			// Remove chosen from active set.
			delete(activeSet, bestIdx)
			totalWeight -= weights[bestIdx]

			// Rebuild active slice without the chosen index.
			newActive := make([]int, 0, len(active)-1)
			for _, idx := range active {
				if idx != bestIdx {
					newActive = append(newActive, idx)
				}
			}
			active = newActive

			// Incrementally update weights for remaining candidates that share
			// actors or studio with the chosen one.
			affected := make(map[int]struct{})
			for _, actor := range chosen.uniqueActors {
				for _, idx := range actorToCandidates[actor] {
					if _, ok := activeSet[idx]; ok {
						affected[idx] = struct{}{}
					}
				}
			}
			if chosen.normalizedStudio != "" {
				for _, idx := range studioToCandidates[chosen.normalizedStudio] {
					if _, ok := activeSet[idx]; ok {
						affected[idx] = struct{}{}
					}
				}
			}

			for idx := range affected {
				if _, ok := activeSet[idx]; !ok {
					continue
				}
				oldWeight := weights[idx]
				score := candidates[idx].score - diversityPenaltyFast(candidates[idx], selectionState)
				newWeight := homepageCandidateWeight(score, candidates[idx].recencyFactor)
				weights[idx] = newWeight
				totalWeight = totalWeight - oldWeight + newWeight
			}
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

// diversityPenaltyFast uses pre-computed normalized actors/studio from the candidate.
func diversityPenaltyFast(
	candidate homepageDailyCandidate,
	selectionState homepageDailySelectionState,
) float64 {
	penalty := 0.0
	for _, actor := range candidate.uniqueActors {
		penalty += float64(selectionState.actorCounts[actor]) * homepageDailyActorDiversityPenalty
	}
	if candidate.normalizedStudio != "" {
		penalty += float64(selectionState.studioCounts[candidate.normalizedStudio]) * homepageDailyStudioDiversityPenalty
	}
	return penalty
}

// accumulateHomepageDiversityFast uses pre-computed normalized actors/studio.
func accumulateHomepageDiversityFast(selectionState homepageDailySelectionState, candidate homepageDailyCandidate) {
	for _, actor := range candidate.uniqueActors {
		selectionState.actorCounts[actor]++
	}
	if candidate.normalizedStudio != "" {
		selectionState.studioCounts[candidate.normalizedStudio]++
	}
}

func homepageRecommendationWeight(score float64, state storage.HomepageRecommendationState, dateUTC string) float64 {
	currentDate, err := time.Parse("2006-01-02", dateUTC)
	if err != nil {
		return 0
	}
	currentTime := currentDate

	if skipUntil := strings.TrimSpace(state.SkipUntil); skipUntil != "" {
		if parsedSkipUntil, err := time.Parse(time.RFC3339, skipUntil); err == nil && currentTime.Before(parsedSkipUntil) {
			return 0
		}
	}

	daysSinceLastRecommendation := 90
	if lastRecommendedAt := strings.TrimSpace(state.LastRecommendedAt); lastRecommendedAt != "" {
		if parsedLastRecommendedAt, err := time.Parse(time.RFC3339, lastRecommendedAt); err == nil {
			daysSinceLastRecommendation = int(currentTime.Sub(parsedLastRecommendedAt).Hours() / 24)
		}
	}
	if daysSinceLastRecommendation >= 0 && daysSinceLastRecommendation < homepageDailyHardCoolingDays {
		return 0
	}
	if daysSinceLastRecommendation < 0 {
		daysSinceLastRecommendation = 0
	}

	positiveScore := math.Max(score, 0.1)
	recencyDays := math.Min(float64(daysSinceLastRecommendation), 90)
	recencyWeight := math.Pow(recencyDays+1, 1.5)
	countPenalty := math.Log2(float64(state.RecommendCount) + 2)
	if countPenalty < 1 {
		countPenalty = 1
	}

	weight := positiveScore * recencyWeight / countPenalty
	if daysSinceLastRecommendation < homepageDailyCoolingDays {
		weight *= float64(daysSinceLastRecommendation) / float64(homepageDailyCoolingDays)
	}
	return weight
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

func normalizeUniqueActors(actors []string) []string {
	if len(actors) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(actors))
	out := make([]string, 0, len(actors))
	for _, a := range actors {
		na := strings.TrimSpace(a)
		if na == "" {
			continue
		}
		if _, ok := seen[na]; ok {
			continue
		}
		seen[na] = struct{}{}
		out = append(out, na)
	}
	return out
}

// homepageRecencyFactor pre-computes the static parts of homepageRecommendationWeight
// that do not depend on the diversity-adjusted score.
func homepageRecencyFactor(state storage.HomepageRecommendationState, dateUTC string) float64 {
	currentDate, err := time.Parse("2006-01-02", dateUTC)
	if err != nil {
		return 0
	}

	if skipUntil := strings.TrimSpace(state.SkipUntil); skipUntil != "" {
		if parsedSkipUntil, err := time.Parse(time.RFC3339, skipUntil); err == nil && currentDate.Before(parsedSkipUntil) {
			return 0
		}
	}

	daysSinceLastRecommendation := 90
	if lastRecommendedAt := strings.TrimSpace(state.LastRecommendedAt); lastRecommendedAt != "" {
		if parsedLastRecommendedAt, err := time.Parse(time.RFC3339, lastRecommendedAt); err == nil {
			daysSinceLastRecommendation = int(currentDate.Sub(parsedLastRecommendedAt).Hours() / 24)
		}
	}
	if daysSinceLastRecommendation >= 0 && daysSinceLastRecommendation < homepageDailyHardCoolingDays {
		return 0
	}
	if daysSinceLastRecommendation < 0 {
		daysSinceLastRecommendation = 0
	}

	recencyDays := math.Min(float64(daysSinceLastRecommendation), 90)
	recencyWeight := math.Pow(recencyDays+1, 1.5)
	countPenalty := math.Log2(float64(state.RecommendCount) + 2)
	if countPenalty < 1 {
		countPenalty = 1
	}
	factor := recencyWeight / countPenalty
	if daysSinceLastRecommendation < homepageDailyCoolingDays {
		factor *= float64(daysSinceLastRecommendation) / float64(homepageDailyCoolingDays)
	}
	return factor
}

// homepageCandidateWeight computes the final weight from a score and pre-computed recencyFactor.
func homepageCandidateWeight(score float64, recencyFactor float64) float64 {
	if recencyFactor <= 0 {
		return 0
	}
	positiveScore := math.Max(score, 0.1)
	return positiveScore * recencyFactor
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

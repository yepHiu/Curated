package app

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"curated-backend/internal/contracts"
)

func TestHomepageRecommendationsExposePersistedReasons(t *testing.T) {
	t.Parallel()
	fixture := newHomepageRecommendationFixture(t, 20)
	ctx := context.Background()

	first, err := fixture.app.GetOrCreateHomepageDailyRecommendations(ctx, "2026-07-21")
	if err != nil {
		t.Fatalf("generate recommendations: %v", err)
	}
	if len(first.Recommendations) != len(first.RecommendationMovieIDs) {
		t.Fatalf("items=%d ids=%d", len(first.Recommendations), len(first.RecommendationMovieIDs))
	}
	for index, item := range first.Recommendations {
		if item.MovieID != first.RecommendationMovieIDs[index] || len(item.Reasons) == 0 {
			t.Fatalf("recommendation[%d] = %#v", index, item)
		}
	}

	cached, err := fixture.app.GetOrCreateHomepageDailyRecommendations(ctx, "2026-07-21")
	if err != nil {
		t.Fatalf("load cached recommendations: %v", err)
	}
	if len(cached.Recommendations) != len(first.Recommendations) {
		t.Fatalf("cached items=%d first=%d", len(cached.Recommendations), len(first.Recommendations))
	}
	for index := range first.Recommendations {
		if cached.Recommendations[index].MovieID != first.Recommendations[index].MovieID ||
			!slices.Equal(cached.Recommendations[index].Reasons, first.Recommendations[index].Reasons) {
			t.Fatalf("cached recommendation[%d] = %#v, want %#v", index, cached.Recommendations[index], first.Recommendations[index])
		}
	}
}

func TestHomepageRecommendationFeedbackChangesGenerationWithoutRefreshSideEffects(t *testing.T) {
	t.Parallel()
	fixture := newHomepageRecommendationFixture(t, 15)
	ctx := context.Background()

	if _, err := fixture.app.RegenerateHomepageDailyRecommendations(ctx, "2026-07-21"); err != nil {
		t.Fatalf("initial refresh: %v", err)
	}
	feedbackBefore, err := fixture.app.ListHomepageRecommendationFeedback(ctx)
	if err != nil || len(feedbackBefore.Items) != 0 {
		t.Fatalf("refresh created feedback: %#v, err=%v", feedbackBefore, err)
	}

	if _, err := fixture.app.CreateHomepageRecommendationFeedback(ctx, contracts.CreateRecommendationFeedbackBody{
		Action:        "not_interested",
		TargetType:    "movie",
		TargetValue:   "m01",
		SourceMovieID: "m01",
	}); err != nil {
		t.Fatalf("create not interested: %v", err)
	}
	dto, err := fixture.app.RegenerateHomepageDailyRecommendations(ctx, "2026-07-22")
	if err != nil {
		t.Fatalf("regenerate after feedback: %v", err)
	}
	allSelected := append(append([]string{}, dto.HeroMovieIDs...), dto.RecommendationMovieIDs...)
	if slices.Contains(allSelected, "m01") {
		t.Fatalf("not-interested movie remained selected: %#v", allSelected)
	}
}

func TestHomepageRecommendationLessFeedbackKeepsExplorationAndExposesEffect(t *testing.T) {
	t.Parallel()
	fixture := newHomepageRecommendationFixture(t, 14)
	fixture.mustResaveMovieMetadataWithProfile(t, "m01", 4.5, "Studio B", "Actor B")
	ctx := context.Background()

	feedback, err := fixture.app.CreateHomepageRecommendationFeedback(ctx, contracts.CreateRecommendationFeedbackBody{
		Action:        "less",
		TargetType:    "actor",
		TargetValue:   "Actor B",
		SourceMovieID: "m01",
	})
	if err != nil {
		t.Fatalf("create less feedback: %v", err)
	}
	dto, err := fixture.app.RegenerateHomepageDailyRecommendations(
		ctx,
		"2026-07-21",
		contracts.HomepageDailyRecommendationsRefreshOptions{
			PreserveHeroMovieIDs: []string{"m02", "m03", "m04", "m05", "m06", "m07", "m08", "m09"},
		},
	)
	if err != nil {
		t.Fatalf("regenerate: %v", err)
	}

	var found bool
	for _, item := range dto.Recommendations {
		if item.MovieID != "m01" {
			continue
		}
		found = true
		if len(item.FeedbackEffects) != 1 || item.FeedbackEffects[0].FeedbackID != feedback.ID || item.FeedbackEffects[0].Effect != "weight_reduced" {
			t.Fatalf("effects = %#v", item.FeedbackEffects)
		}
	}
	if !found {
		t.Fatal("less-feedback movie was fully excluded; bounded exploration must retain eligibility")
	}
}

func TestHomepageRecommendationFeedbackValidatesPersistsAndDeletes(t *testing.T) {
	t.Parallel()
	fixture := newHomepageRecommendationFixture(t, 2)
	ctx := context.Background()

	actorFeedback, err := fixture.app.CreateHomepageRecommendationFeedback(ctx, contracts.CreateRecommendationFeedbackBody{
		Action:        "less",
		TargetType:    "actor",
		TargetValue:   "actor a",
		SourceMovieID: "m01",
	})
	if err != nil {
		t.Fatalf("create actor feedback: %v", err)
	}
	if actorFeedback.TargetValue != "Actor A" || actorFeedback.Action != "less" || actorFeedback.TargetType != "actor" {
		t.Fatalf("actor feedback = %#v", actorFeedback)
	}

	snooze, err := fixture.app.CreateHomepageRecommendationFeedback(ctx, contracts.CreateRecommendationFeedbackBody{
		Action:        "snooze",
		TargetType:    "movie",
		TargetValue:   "M02",
		SourceMovieID: "m02",
		DurationDays:  7,
	})
	if err != nil {
		t.Fatalf("create snooze: %v", err)
	}
	if _, err := time.Parse(time.RFC3339Nano, snooze.ExpiresAt); err != nil {
		t.Fatalf("ExpiresAt = %q: %v", snooze.ExpiresAt, err)
	}

	_, err = fixture.app.CreateHomepageRecommendationFeedback(ctx, contracts.CreateRecommendationFeedbackBody{
		Action:        "less",
		TargetType:    "actor",
		TargetValue:   "invented actor",
		SourceMovieID: "m01",
	})
	if !errors.Is(err, contracts.ErrRecommendationFeedbackTargetNotFound) {
		t.Fatalf("invalid target error = %v", err)
	}

	list, err := fixture.app.ListHomepageRecommendationFeedback(ctx)
	if err != nil {
		t.Fatalf("list feedback: %v", err)
	}
	if len(list.Items) != 2 {
		t.Fatalf("feedback count = %d, want 2", len(list.Items))
	}

	if err := fixture.app.DeleteHomepageRecommendationFeedback(ctx, actorFeedback.ID); err != nil {
		t.Fatalf("delete feedback: %v", err)
	}
	list, err = fixture.app.ListHomepageRecommendationFeedback(ctx)
	if err != nil || len(list.Items) != 1 || list.Items[0].ID != snooze.ID {
		t.Fatalf("feedback after delete = %#v, err=%v", list, err)
	}
}

func TestHomepageRecommendationFeedbackConcurrentDuplicateIsIdempotent(t *testing.T) {
	t.Parallel()
	fixture := newHomepageRecommendationFixture(t, 1)
	ctx := context.Background()

	const workers = 8
	ids := make(chan string, workers)
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			item, err := fixture.app.CreateHomepageRecommendationFeedback(ctx, contracts.CreateRecommendationFeedbackBody{
				Action:        "not_interested",
				TargetType:    "movie",
				TargetValue:   "m01",
				SourceMovieID: "m01",
			})
			if err != nil {
				errs <- err
				return
			}
			ids <- item.ID
		}()
	}
	wg.Wait()
	close(ids)
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent create: %v", err)
	}
	var first string
	for id := range ids {
		if first == "" {
			first = id
		}
		if id != first {
			t.Fatalf("duplicate returned ids %q and %q", first, id)
		}
	}
	list, err := fixture.app.ListHomepageRecommendationFeedback(ctx)
	if err != nil || len(list.Items) != 1 {
		t.Fatalf("list = %#v, err=%v", list, err)
	}
}

func TestHomepageRecommendationFeedbackRejectsInvalidShapes(t *testing.T) {
	t.Parallel()
	fixture := newHomepageRecommendationFixture(t, 1)
	ctx := context.Background()

	for _, body := range []contracts.CreateRecommendationFeedbackBody{
		{Action: "snooze", TargetType: "movie", TargetValue: "m01", SourceMovieID: "m01"},
		{Action: "snooze", TargetType: "movie", TargetValue: "m01", SourceMovieID: "m01", DurationDays: 366},
		{Action: "less", TargetType: "movie", TargetValue: "m01", SourceMovieID: "m01"},
		{Action: "not_interested", TargetType: "movie", TargetValue: "m01", SourceMovieID: "m01", DurationDays: 7},
	} {
		if _, err := fixture.app.CreateHomepageRecommendationFeedback(ctx, body); !errors.Is(err, contracts.ErrRecommendationFeedbackInvalid) {
			t.Fatalf("body %#v error = %v", body, err)
		}
	}
}

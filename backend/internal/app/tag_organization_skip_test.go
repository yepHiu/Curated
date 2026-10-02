package app

import (
	"context"
	"strings"
	"testing"

	"curated-backend/internal/scraper"
	"curated-backend/internal/storage"
)

func TestTagOrganizationVocabularySkipsOneMovieAndRetriesOnlyOnRequest(t *testing.T) {
	a := governanceTestApp(t)
	ids, _ := seedTopicMovies(t, a, 7)
	ctx := context.Background()
	bad, _ := a.store.TopicMovieInput(ctx, ids[2])
	repaired := false
	retrying := false
	scaleTopicProvider(t, a, func(vocabulary bool, source topicScaleRequest) (any, int) {
		if vocabulary {
			for _, movie := range source.Movies {
				if retrying && movie.Title != bad.Title {
					t.Error("retry reanalyzed a completed movie")
				}
				if movie.Title == bad.Title && !repaired {
					return map[string]any{"topics": []storage.TopicDefinition{{Name: ""}}}, 200
				}
			}
			return map[string]any{"topics": []storage.TopicDefinition{{Name: "Theme"}}}, 200
		}
		for _, movie := range source.Movies {
			if movie.MovieID == bad.MovieID && !repaired {
				t.Error("vocabulary failure reached classification")
			}
			if retrying && movie.MovieID != bad.MovieID {
				t.Error("retry classified a completed movie")
			}
		}
		return scaleTopicClassification(source), 200
	})
	id := createScaleTopicJob(t, a, ids)
	a.runTagOrganization(ctx, id)
	job, err := a.store.GetTagOrganization(ctx, id)
	if err != nil || job.Status != "partial_failed" || job.Succeeded != 6 || job.Failed != 1 || job.Processed != 7 || job.VocabularyProcessed != 7 {
		t.Fatalf("job=%+v err=%v", job, err)
	}
	issues, err := a.store.TagOrganizationIssues(ctx, 25, 0)
	if err != nil || len(issues) != 1 || issues[0].MovieID != bad.MovieID || !strings.HasPrefix(issues[0].Reason, "VOCABULARY_") {
		t.Fatalf("issues=%+v err=%v", issues, err)
	}
	stats, _ := a.store.TagOrganizationStats(ctx)
	if stats.NeedsAttention != 1 || stats.Organized != 6 || stats.Unorganized != 0 {
		t.Fatalf("stats=%+v", stats)
	}
	unchanged, _ := a.store.TopicMovieInput(ctx, ids[0])
	// The next normal run must not consume this failure again.
	if _, err := a.store.CreateRemainingTagOrganization(ctx, "default", "default", "manual", "en", "unorganized"); err != storage.ErrNoOrganizationMovies {
		t.Fatalf("default retry err=%v", err)
	}
	repaired = true
	retrying = true
	if err := a.store.RetryTagOrganization(ctx, id); err != nil {
		t.Fatal(err)
	}
	// A fresh worker has no in-memory knowledge of the previous skip.
	resumed := &App{store: a.store, cfg: a.cfg}
	resumed.runTagOrganization(ctx, id)
	job, _ = a.store.GetTagOrganization(ctx, id)
	if job.Status != "completed" || job.Succeeded != 7 {
		t.Fatalf("retry=%+v", job)
	}
	issues, _ = a.store.TagOrganizationIssues(ctx, 25, 0)
	if len(issues) != 0 {
		t.Fatalf("recovered issue remains: %+v", issues)
	}
	after, _ := a.store.TopicMovieInput(ctx, ids[0])
	if after.Revision != unchanged.Revision {
		t.Fatal("successful movie rewritten")
	}
}

func TestTagOrganizationOversizedMovieIsProblemNotNoMatch(t *testing.T) {
	a := governanceTestApp(t)
	ids, _ := seedTopicMovies(t, a, 3)
	ctx := context.Background()
	if err := a.store.SaveMovieMetadata(ctx, scraper.Metadata{MovieID: ids[1], Title: "Long", Summary: strings.Repeat("字", 9000), Tags: []string{"Theme"}}); err != nil {
		t.Fatal(err)
	}
	topicFixtureProvider(t, a, nil)
	id := createScaleTopicJob(t, a, ids)
	a.runTagOrganization(ctx, id)
	job, _ := a.store.GetTagOrganization(ctx, id)
	if job.Status != "partial_failed" || job.Failed != 1 || job.Succeeded != 2 || job.Unresolved != 0 {
		t.Fatalf("job=%+v", job)
	}
	issues, _ := a.store.TagOrganizationIssues(ctx, 25, 0)
	if len(issues) != 1 || issues[0].Reason != "SOURCE_TOO_LONG" {
		t.Fatalf("issues=%+v", issues)
	}
}

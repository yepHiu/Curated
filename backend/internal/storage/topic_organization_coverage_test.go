package storage

import (
	"context"
	"errors"
	"testing"

	"curated-backend/internal/contracts"
)

func requireTopicStats(t *testing.T, s *SQLiteStore, want contracts.TagOrganizationStatsDTO) {
	t.Helper()
	got, err := s.TagOrganizationStats(context.Background())
	if err != nil || got != want {
		t.Fatalf("stats=%+v want=%+v err=%v", got, want, err)
	}
}

func TestTopicCoverageCompletionFreshnessAndScope(t *testing.T) {
	s := newTopicTestStore(t)
	ctx := context.Background()
	requireTopicStats(t, s, contracts.TagOrganizationStatsDTO{Total: 2, Unorganized: 2})
	startTopicTestJob(t, s, "first")
	input, _ := s.TopicMovieInput(ctx, "a")
	if err := s.ApplyMovieTopics(ctx, "first", input, []string{"Theme"}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateTagOrganization(ctx, "first", "completed", "finished", ""); err != nil {
		t.Fatal(err)
	}
	// Ratings, user tags and repeated successful calls do not invalidate source analysis.
	if err := s.PatchMovieUserPrefs(ctx, "a", contracts.PatchMovieInput{UserTagsSet: true, UserTags: []string{"Personal"}}); err != nil {
		t.Fatal(err)
	}
	requireTopicStats(t, s, contracts.TagOrganizationStatsDTO{Total: 2, Organized: 1, Unorganized: 1})
	id, err := s.CreateRemainingTagOrganization(ctx, "remaining", "remaining", "manual", "en", "unorganized")
	if err != nil {
		t.Fatal(err)
	}
	items, err := s.TagOrganizationItems(ctx, id, false, 100, 0)
	if err != nil || len(items) != 1 || items[0].MovieID != "b" {
		t.Fatalf("items=%+v err=%v", items, err)
	}
	if _, err = s.db.Exec(`UPDATE movies SET title='Changed' WHERE id='a'`); err != nil {
		t.Fatal(err)
	}
	// The newly stale movie cannot expand an already captured task.
	job, _ := s.GetTagOrganization(ctx, id)
	if job.Total != 1 {
		t.Fatalf("scope expanded: %+v", job)
	}
	if err = s.UpdateTagOrganization(ctx, id, "running", "applying", ""); err != nil {
		t.Fatal(err)
	}
	input, _ = s.TopicMovieInput(ctx, "b")
	if err = s.CompleteUnresolvedTopic(ctx, id, input); err != nil {
		t.Fatal(err)
	}
	if err = s.UpdateTagOrganization(ctx, id, "completed", "finished", ""); err != nil {
		t.Fatal(err)
	}
	requireTopicStats(t, s, contracts.TagOrganizationStatsDTO{Total: 2, Organized: 1, Outdated: 1, Unresolved: 1})
	if _, err = s.CreateRemainingTagOrganization(ctx, "empty", "empty", "manual", "en", "unorganized"); !errors.Is(err, ErrNoOrganizationMovies) {
		t.Fatalf("err=%v", err)
	}
	stale, err := s.CreateRemainingTagOrganization(ctx, "stale", "stale", "manual", "en", "outdated")
	if err != nil {
		t.Fatal(err)
	}
	items, _ = s.TagOrganizationItems(ctx, stale, false, 100, 0)
	if len(items) != 1 || items[0].MovieID != "a" {
		t.Fatalf("stale=%+v", items)
	}
	if _, err = s.db.Exec(`UPDATE movies SET trashed_at='2026-10-03' WHERE id='b'`); err != nil {
		t.Fatal(err)
	}
	requireTopicStats(t, s, contracts.TagOrganizationStatsDTO{Total: 1, Outdated: 1})
}

func TestTopicCoverageUnresolvedRechecksInputAndRollsBack(t *testing.T) {
	s := newTopicTestStore(t)
	ctx := context.Background()
	startTopicTestJob(t, s, "job")
	input, _ := s.TopicMovieInput(ctx, "a")
	if _, err := s.db.Exec(`UPDATE movies SET summary='New source' WHERE id='a'`); err != nil {
		t.Fatal(err)
	}
	if err := s.CompleteUnresolvedTopic(ctx, "job", input); !errors.Is(err, ErrAIWriteConflict) {
		t.Fatalf("err=%v", err)
	}
	requireTopicStats(t, s, contracts.TagOrganizationStatsDTO{Total: 2, Unorganized: 2})
	input, _ = s.TopicMovieInput(ctx, "a")
	if _, err := s.db.Exec(`CREATE TRIGGER reject_analysis BEFORE INSERT ON movie_topic_analysis BEGIN SELECT RAISE(ABORT,'fixture'); END;`); err != nil {
		t.Fatal(err)
	}
	if err := s.CompleteUnresolvedTopic(ctx, "job", input); err == nil {
		t.Fatal("checkpoint failure accepted")
	}
	job, _ := s.GetTagOrganization(ctx, "job")
	if job.Processed != 0 {
		t.Fatalf("checkpoint not atomic: %+v", job)
	}
	if _, err := s.db.Exec(`DROP TRIGGER reject_analysis`); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateTagOrganization(ctx, "job", "cancelled", "stopped", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.CompleteUnresolvedTopic(ctx, "job", input); !errors.Is(err, ErrAIWriteConflict) {
		t.Fatalf("cancel err=%v", err)
	}
	requireTopicStats(t, s, contracts.TagOrganizationStatsDTO{Total: 2, Unorganized: 2})
}

func TestTopicCoverageUndoDoesNotEraseLaterAnalysis(t *testing.T) {
	for _, later := range []bool{false, true} {
		t.Run(map[bool]string{false: "original", true: "later"}[later], func(t *testing.T) {
			s := newTopicTestStore(t)
			ctx := context.Background()
			startTopicTestJob(t, s, "first")
			input, _ := s.TopicMovieInput(ctx, "a")
			if err := s.ApplyMovieTopics(ctx, "first", input, []string{"Theme"}); err != nil {
				t.Fatal(err)
			}
			if err := s.UpdateTagOrganization(ctx, "first", "completed", "finished", ""); err != nil {
				t.Fatal(err)
			}
			if later {
				startTopicTestJob(t, s, "second")
				input, _ = s.TopicMovieInput(ctx, "a")
				if err := s.CompleteUnresolvedTopic(ctx, "second", input); err != nil {
					t.Fatal(err)
				}
				if err := s.UpdateTagOrganization(ctx, "second", "completed", "finished", ""); err != nil {
					t.Fatal(err)
				}
			}
			undo, err := s.UndoTopicOrganization(ctx, "first")
			if err != nil || undo.Restored != 1 {
				t.Fatalf("undo=%+v err=%v", undo, err)
			}
			want := contracts.TagOrganizationStatsDTO{Total: 2, Unorganized: 2}
			if later {
				want = contracts.TagOrganizationStatsDTO{Total: 2, Organized: 1, Unorganized: 1, Unresolved: 1}
			}
			requireTopicStats(t, s, want)
		})
	}
}

func TestTopicCoverageMigrationRecognizesHistoryAndUndo(t *testing.T) {
	s := newTopicTestStore(t)
	ctx := context.Background()
	startTopicTestJob(t, s, "legacy")
	a, _ := s.TopicMovieInput(ctx, "a")
	b, _ := s.TopicMovieInput(ctx, "b")
	if err := s.ApplyMovieTopics(ctx, "legacy", a, []string{"Theme"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`INSERT INTO ai_tag_organization_items(job_id,movie_id,status,input_fingerprint) VALUES('legacy','b','unresolved',?)`, b.Fingerprint); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateTagOrganization(ctx, "legacy", "completed", "finished", ""); err != nil {
		t.Fatal(err)
	}
	// Simulate the pre-migration schema with historical checkpoints only.
	if _, err := s.db.Exec(`DROP TABLE movie_topic_analysis`); err != nil {
		t.Fatal(err)
	}
	migration, err := migrationFiles.ReadFile("migrations/0062_movie_topic_analysis.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(string(migration)); err != nil {
		t.Fatal(err)
	}
	requireTopicStats(t, s, contracts.TagOrganizationStatsDTO{Total: 2, Organized: 2, Unresolved: 1})
	if _, err = s.UndoTopicOrganization(ctx, "legacy"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(`DROP TABLE movie_topic_analysis`); err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(string(migration)); err != nil {
		t.Fatal(err)
	}
	requireTopicStats(t, s, contracts.TagOrganizationStatsDTO{Total: 2, Organized: 1, Unorganized: 1, Unresolved: 1})
	if err = s.PatchMovieUserPrefs(ctx, "b", contracts.PatchMovieInput{MetadataTagsSet: true, MetadataTags: []string{"Changed"}}); err != nil {
		t.Fatal(err)
	}
	requireTopicStats(t, s, contracts.TagOrganizationStatsDTO{Total: 2, Unorganized: 1, Outdated: 1})
}

func TestTopicCoverageFailedCompletionRollsBackTags(t *testing.T) {
	s := newTopicTestStore(t)
	ctx := context.Background()
	startTopicTestJob(t, s, "job")
	input, _ := s.TopicMovieInput(ctx, "a")
	if _, err := s.db.Exec(`CREATE TRIGGER reject_analysis BEFORE INSERT ON movie_topic_analysis BEGIN SELECT RAISE(ABORT,'fixture'); END;`); err != nil {
		t.Fatal(err)
	}
	if err := s.ApplyMovieTopics(ctx, "job", input, []string{"Theme"}); err == nil {
		t.Fatal("completion failure accepted")
	}
	after, _ := s.TopicMovieInput(ctx, "a")
	if after.Revision != input.Revision || !sameUserTagSet(after.UserTags, input.UserTags) {
		t.Fatalf("partial commit: %+v", after)
	}
	job, _ := s.GetTagOrganization(ctx, "job")
	if job.Processed != 0 {
		t.Fatalf("checkpoint advanced: %+v", job)
	}
	requireTopicStats(t, s, contracts.TagOrganizationStatsDTO{Total: 2, Unorganized: 2})
}

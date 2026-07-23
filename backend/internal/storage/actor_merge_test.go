package storage

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"curated-backend/internal/contracts"
	"curated-backend/internal/scraper"
)

func newActorMergeTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "actor-merge.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Migrate(context.Background()); err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func seedActorMergeActor(t *testing.T, store *SQLiteStore, name string, fields map[string]any) int64 {
	t.Helper()
	ctx := context.Background()
	result, err := store.db.ExecContext(ctx, `
		INSERT INTO actors (
			name, normalized_name, avatar, avatar_local_path, summary, homepage,
			provider, provider_actor_id, height, birthday, profile_updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		name,
		NormalizeActorIdentity(name),
		fields["avatar"],
		fields["avatar_local_path"],
		fields["summary"],
		fields["homepage"],
		fields["provider"],
		fields["provider_actor_id"],
		fields["height"],
		fields["birthday"],
		fields["profile_updated_at"],
	)
	if err != nil {
		t.Fatal(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func actorMergeFields(overrides map[string]any) map[string]any {
	fields := map[string]any{
		"avatar":             "",
		"avatar_local_path":  "",
		"summary":            "",
		"homepage":           "",
		"provider":           "",
		"provider_actor_id":  "",
		"height":             0,
		"birthday":           "",
		"profile_updated_at": "",
	}
	for key, value := range overrides {
		fields[key] = value
	}
	return fields
}

func seedActorMergeMovie(t *testing.T, store *SQLiteStore, id string, actorIDs ...int64) {
	t.Helper()
	ctx := context.Background()
	if _, err := store.db.ExecContext(ctx, `
		INSERT INTO movies (
			id, code, title, studio, summary, location, resolution, year, added_at
		) VALUES (?, ?, ?, '', '', ?, '', 0, '2026-07-21')`,
		id, id, id, filepath.Join(t.TempDir(), id+".mp4")); err != nil {
		t.Fatal(err)
	}
	for _, actorID := range actorIDs {
		if _, err := store.db.ExecContext(ctx,
			`INSERT INTO movie_actors (movie_id, actor_id) VALUES (?, ?)`, id, actorID); err != nil {
			t.Fatal(err)
		}
	}
}

func seedActorMergeTag(t *testing.T, store *SQLiteStore, actorID int64, tag string) {
	t.Helper()
	if _, err := store.db.Exec(`INSERT INTO actor_user_tags (actor_id, tag) VALUES (?, ?)`, actorID, tag); err != nil {
		t.Fatal(err)
	}
}

func seedActorMergeLink(t *testing.T, store *SQLiteStore, actorID int64, link string, order int) {
	t.Helper()
	if _, err := store.db.Exec(`
		INSERT INTO actor_external_links (actor_id, url, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, '2026-07-21T00:00:00Z', '2026-07-21T00:00:00Z')`, actorID, link, order); err != nil {
		t.Fatal(err)
	}
}

func TestActorMergePreviewAndApplyPreserveAssociationsAndResolveAliases(t *testing.T) {
	store := newActorMergeTestStore(t)
	ctx := context.Background()
	sourceID := seedActorMergeActor(t, store, "Ａlice   Smith", actorMergeFields(map[string]any{
		"summary":            "source summary",
		"provider":           "source-provider",
		"provider_actor_id":  "source-42",
		"height":             168,
		"profile_updated_at": "2026-07-20T00:00:00Z",
	}))
	targetID := seedActorMergeActor(t, store, "Alice Smith", actorMergeFields(map[string]any{
		"avatar":             "https://img.example/target.jpg",
		"summary":            "target summary",
		"birthday":           "1999-01-02",
		"profile_updated_at": "2026-07-21T00:00:00Z",
	}))
	seedActorMergeMovie(t, store, "movie-source", sourceID)
	seedActorMergeMovie(t, store, "movie-shared", sourceID, targetID)
	seedActorMergeMovie(t, store, "movie-target", targetID)
	seedActorMergeTag(t, store, sourceID, "source-tag")
	seedActorMergeTag(t, store, targetID, "target-tag")
	seedActorMergeLink(t, store, sourceID, "https://source.example", 0)
	seedActorMergeLink(t, store, targetID, "https://target.example", 0)
	if _, err := store.db.Exec(`
		INSERT INTO homepage_recommendation_feedback (
			id, action, target_type, target_value, normalized_target, source_movie_id,
			expires_at, created_at, updated_at
		) VALUES
			('feedback-source', 'less', 'actor', 'Ａlice   Smith', ?, 'movie-source', '', '2026-07-21T00:00:00Z', '2026-07-21T00:00:00Z'),
			('feedback-target', 'less', 'actor', 'Alice Smith', ?, 'movie-target', '', '2026-07-21T01:00:00Z', '2026-07-21T01:00:00Z')`,
		"ａlice   smith", NormalizeActorIdentity("Alice Smith")); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`
		INSERT INTO curated_frames (
			id, movie_id, title, code, actors_json, position_sec, captured_at, tags_json, image_blob
		) VALUES ('frame-alias', 'movie-shared', 'frame', 'frame', ?, 1, '2026-07-21T00:00:00Z', '[]', X'01')`,
		`["Ａlice   Smith","Alice Smith"]`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`
		INSERT INTO actor_aliases (canonical_actor_id, alias, normalized_alias, created_at, updated_at)
		VALUES (?, 'A. Smith', ?, '2026-07-21T00:00:00Z', '2026-07-21T00:00:00Z')`,
		sourceID, NormalizeActorIdentity("A. Smith")); err != nil {
		t.Fatal(err)
	}

	preview, err := store.PreviewActorMerge(ctx, contracts.ActorMergePreviewRequest{
		SourceName: "Ａlice   Smith",
		TargetName: "Alice Smith",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !preview.CanApply || preview.Movies.SourceCount != 2 || preview.Movies.TargetCount != 2 ||
		preview.Movies.DuplicateCount != 1 || preview.Movies.ResultCount != 3 {
		t.Fatalf("unexpected preview: %+v", preview)
	}
	if !reflect.DeepEqual(preview.UserTags.Result, []string{"target-tag", "source-tag"}) {
		t.Fatalf("unexpected merged tags: %#v", preview.UserTags.Result)
	}
	if !reflect.DeepEqual(preview.ExternalLinks.Result, []string{"https://target.example", "https://source.example"}) {
		t.Fatalf("unexpected merged links: %#v", preview.ExternalLinks.Result)
	}
	if !reflect.DeepEqual(preview.RequiredDecisions, []string{"summary"}) {
		t.Fatalf("unexpected required decisions: %#v", preview.RequiredDecisions)
	}
	if preview.RecommendationFeedback.SourceCount != 1 || preview.RecommendationFeedback.TargetCount != 1 ||
		preview.RecommendationFeedback.DuplicateCount != 1 || preview.RecommendationFeedback.ResultCount != 1 ||
		preview.CuratedFramesAffected != 1 {
		t.Fatalf("unexpected secondary association preview: feedback=%+v frames=%d", preview.RecommendationFeedback, preview.CuratedFramesAffected)
	}

	audit, err := store.ApplyActorMerge(ctx, contracts.ApplyActorMergeRequest{
		SourceName:   preview.Source.Name,
		TargetName:   preview.Target.Name,
		PreviewToken: preview.PreviewToken,
		Confirm:      true,
		ProfileDecisions: map[string]string{
			"summary": "source",
		},
	}, time.Date(2026, 7, 21, 2, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if audit.SourceActorID != sourceID || audit.TargetActorID != targetID || audit.ID == "" {
		t.Fatalf("unexpected audit: %+v", audit)
	}
	profile, err := store.GetActorProfile(ctx, "Ａlice Smith")
	if err != nil {
		t.Fatal(err)
	}
	if profile.Name != "Alice Smith" || profile.Summary != "source summary" || profile.Provider != "source-provider" || profile.AvatarRemoteURL != "https://img.example/target.jpg" {
		t.Fatalf("unexpected canonical profile: %+v", profile)
	}
	aliasProfile, err := store.GetActorProfile(ctx, "A. Smith")
	if err != nil || aliasProfile.Name != "Alice Smith" {
		t.Fatalf("alias did not resolve to canonical actor: profile=%+v err=%v", aliasProfile, err)
	}
	var sourceRows int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM actors WHERE id = ?`, sourceID).Scan(&sourceRows); err != nil || sourceRows != 0 {
		t.Fatalf("source actor still exists: count=%d err=%v", sourceRows, err)
	}
	var movieCount int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM movie_actors WHERE actor_id = ?`, targetID).Scan(&movieCount); err != nil || movieCount != 3 {
		t.Fatalf("unexpected target movie count=%d err=%v", movieCount, err)
	}
	var feedbackCount int
	var feedbackTarget string
	if err := store.db.QueryRow(`
		SELECT COUNT(*), MAX(target_value)
		FROM homepage_recommendation_feedback
		WHERE action = 'less' AND target_type = 'actor'`).Scan(&feedbackCount, &feedbackTarget); err != nil || feedbackCount != 1 || feedbackTarget != "Alice Smith" {
		t.Fatalf("recommendation feedback not canonicalized: count=%d target=%q err=%v", feedbackCount, feedbackTarget, err)
	}
	var frameActorsJSON string
	if err := store.db.QueryRow(`SELECT actors_json FROM curated_frames WHERE id = 'frame-alias'`).Scan(&frameActorsJSON); err != nil || frameActorsJSON != `["Alice Smith"]` {
		t.Fatalf("curated frame actors not canonicalized: json=%q err=%v", frameActorsJSON, err)
	}
	audits, err := store.ListActorMergeAudits(ctx, 10, 0)
	if err != nil || audits.Total != 1 || len(audits.Items) != 1 || audits.Items[0].ID != audit.ID {
		t.Fatalf("unexpected audits: %+v err=%v", audits, err)
	}

	seedActorMergeMovie(t, store, "movie-new-alias")
	if err := store.SaveMovieMetadata(ctx, scraper.Metadata{
		MovieID: "movie-new-alias",
		Number:  "movie-new-alias",
		Title:   "alias ingestion",
		Actors:  []string{"A. Smith"},
	}); err != nil {
		t.Fatal(err)
	}
	var ingestedActorID int64
	if err := store.db.QueryRow(`SELECT actor_id FROM movie_actors WHERE movie_id = 'movie-new-alias'`).Scan(&ingestedActorID); err != nil || ingestedActorID != targetID {
		t.Fatalf("metadata alias did not reuse canonical actor: actorID=%d err=%v", ingestedActorID, err)
	}
	if err := store.UpdateActorProfile(ctx, scraper.ActorProfile{DisplayName: "A. Smith", Summary: "updated through alias"}); err != nil {
		t.Fatal(err)
	}
	profile, err = store.GetActorProfile(ctx, "Alice Smith")
	if err != nil || profile.Summary != "updated through alias" {
		t.Fatalf("scraper alias update did not target canonical actor: %+v err=%v", profile, err)
	}
}

func TestActorMergeRejectsStalePreviewAndMissingConflictDecision(t *testing.T) {
	store := newActorMergeTestStore(t)
	ctx := context.Background()
	sourceID := seedActorMergeActor(t, store, "Source", actorMergeFields(map[string]any{"summary": "source"}))
	seedActorMergeActor(t, store, "Target", actorMergeFields(map[string]any{"summary": "target"}))
	preview, err := store.PreviewActorMerge(ctx, contracts.ActorMergePreviewRequest{SourceName: "Source", TargetName: "Target"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.ApplyActorMerge(ctx, contracts.ApplyActorMergeRequest{
		SourceName: "Source", TargetName: "Target", PreviewToken: preview.PreviewToken, Confirm: true,
	}, time.Now())
	if !errors.Is(err, contracts.ErrActorMergeConflict) {
		t.Fatalf("missing profile decision error=%v", err)
	}
	seedActorMergeTag(t, store, sourceID, "changed-after-preview")
	_, err = store.ApplyActorMerge(ctx, contracts.ApplyActorMergeRequest{
		SourceName: "Source", TargetName: "Target", PreviewToken: preview.PreviewToken, Confirm: true,
		ProfileDecisions: map[string]string{"summary": "target"},
	}, time.Now())
	if !errors.Is(err, contracts.ErrActorMergeStalePreview) {
		t.Fatalf("stale preview error=%v", err)
	}
}

func TestActorMergeRollsBackEveryWriteWhenTransactionFails(t *testing.T) {
	store := newActorMergeTestStore(t)
	ctx := context.Background()
	sourceID := seedActorMergeActor(t, store, "Rollback Source", actorMergeFields(nil))
	targetID := seedActorMergeActor(t, store, "Rollback Target", actorMergeFields(nil))
	seedActorMergeMovie(t, store, "rollback-movie", sourceID)
	seedActorMergeTag(t, store, sourceID, "rollback-tag")
	preview, err := store.PreviewActorMerge(ctx, contracts.ActorMergePreviewRequest{SourceName: "Rollback Source", TargetName: "Rollback Target"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(fmt.Sprintf(`
		CREATE TRIGGER fail_actor_merge_delete
		BEFORE DELETE ON actors
		WHEN OLD.id = %d
		BEGIN
			SELECT RAISE(ABORT, 'forced actor merge rollback');
		END`, sourceID)); err != nil {
		t.Fatal(err)
	}
	_, err = store.ApplyActorMerge(ctx, contracts.ApplyActorMergeRequest{
		SourceName: "Rollback Source", TargetName: "Rollback Target", PreviewToken: preview.PreviewToken, Confirm: true,
	}, time.Now())
	if err == nil {
		t.Fatal("expected forced transaction failure")
	}
	var sourceMovieCount, targetMovieCount, auditCount int
	_ = store.db.QueryRow(`SELECT COUNT(*) FROM movie_actors WHERE actor_id = ?`, sourceID).Scan(&sourceMovieCount)
	_ = store.db.QueryRow(`SELECT COUNT(*) FROM movie_actors WHERE actor_id = ?`, targetID).Scan(&targetMovieCount)
	_ = store.db.QueryRow(`SELECT COUNT(*) FROM actor_merge_audits`).Scan(&auditCount)
	if sourceMovieCount != 1 || targetMovieCount != 0 || auditCount != 0 {
		t.Fatalf("partial writes survived rollback: source=%d target=%d audits=%d", sourceMovieCount, targetMovieCount, auditCount)
	}
}

func TestActorMergeBlocksLinkLimitAndAliasCollision(t *testing.T) {
	store := newActorMergeTestStore(t)
	ctx := context.Background()
	sourceID := seedActorMergeActor(t, store, "Collision", actorMergeFields(nil))
	targetID := seedActorMergeActor(t, store, "Destination", actorMergeFields(nil))
	seedActorMergeActor(t, store, "Ｃollision", actorMergeFields(nil))
	for i := 0; i < 9; i++ {
		seedActorMergeLink(t, store, sourceID, fmt.Sprintf("https://source.example/%d", i), i)
		seedActorMergeLink(t, store, targetID, fmt.Sprintf("https://target.example/%d", i), i)
	}
	preview, err := store.PreviewActorMerge(ctx, contracts.ActorMergePreviewRequest{SourceName: "Collision", TargetName: "Destination"})
	if err != nil {
		t.Fatal(err)
	}
	if preview.CanApply || len(preview.BlockingReasons) < 2 {
		t.Fatalf("expected link and alias blockers: %+v", preview.BlockingReasons)
	}
	_, err = store.ApplyActorMerge(ctx, contracts.ApplyActorMergeRequest{
		SourceName: "Collision", TargetName: "Destination", PreviewToken: preview.PreviewToken, Confirm: true,
	}, time.Now())
	if !errors.Is(err, contracts.ErrActorMergeLinkLimit) {
		t.Fatalf("expected link limit error, got %v", err)
	}
}

func TestActorMergeRejectsSelfAndMergedAliasAsSource(t *testing.T) {
	store := newActorMergeTestStore(t)
	ctx := context.Background()
	seedActorMergeActor(t, store, "One", actorMergeFields(nil))
	seedActorMergeActor(t, store, "Two", actorMergeFields(nil))
	_, err := store.PreviewActorMerge(ctx, contracts.ActorMergePreviewRequest{SourceName: "One", TargetName: "One"})
	if !errors.Is(err, contracts.ErrActorMergeSelf) {
		t.Fatalf("self merge error=%v", err)
	}
	preview, err := store.PreviewActorMerge(ctx, contracts.ActorMergePreviewRequest{SourceName: "One", TargetName: "Two"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ApplyActorMerge(ctx, contracts.ApplyActorMergeRequest{
		SourceName: "One", TargetName: "Two", PreviewToken: preview.PreviewToken, Confirm: true,
	}, time.Now()); err != nil {
		t.Fatal(err)
	}
	_, err = store.PreviewActorMerge(ctx, contracts.ActorMergePreviewRequest{SourceName: "One", TargetName: "Two"})
	if !errors.Is(err, contracts.ErrActorMergeSourceAlias) {
		t.Fatalf("merged alias source error=%v", err)
	}
}

func TestActorMergeAllowsCanonicalTargetToMergeAgainWithoutLosingAudits(t *testing.T) {
	store := newActorMergeTestStore(t)
	ctx := context.Background()
	seedActorMergeActor(t, store, "First", actorMergeFields(nil))
	seedActorMergeActor(t, store, "Second", actorMergeFields(nil))
	seedActorMergeActor(t, store, "Final", actorMergeFields(nil))

	firstPreview, err := store.PreviewActorMerge(ctx, contracts.ActorMergePreviewRequest{
		SourceName: "First",
		TargetName: "Second",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ApplyActorMerge(ctx, contracts.ApplyActorMergeRequest{
		SourceName:   firstPreview.Source.Name,
		TargetName:   firstPreview.Target.Name,
		PreviewToken: firstPreview.PreviewToken,
		Confirm:      true,
	}, time.Date(2026, 7, 21, 3, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}

	secondPreview, err := store.PreviewActorMerge(ctx, contracts.ActorMergePreviewRequest{
		SourceName: "Second",
		TargetName: "Final",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ApplyActorMerge(ctx, contracts.ApplyActorMergeRequest{
		SourceName:   secondPreview.Source.Name,
		TargetName:   secondPreview.Target.Name,
		PreviewToken: secondPreview.PreviewToken,
		Confirm:      true,
	}, time.Date(2026, 7, 21, 4, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}

	for _, alias := range []string{"First", "Second"} {
		canonical, err := store.ResolveActorCanonicalName(ctx, alias)
		if err != nil || canonical != "Final" {
			t.Fatalf("alias %q did not resolve to final canonical actor: canonical=%q err=%v", alias, canonical, err)
		}
	}
	audits, err := store.ListActorMergeAudits(ctx, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if audits.Total != 2 || len(audits.Items) != 2 ||
		audits.Items[0].SourceName != "Second" || audits.Items[0].TargetName != "Final" ||
		audits.Items[1].SourceName != "First" || audits.Items[1].TargetName != "Second" {
		t.Fatalf("chained merge audits were not preserved: %+v", audits)
	}
}

func TestActorIdentityBackfillDeduplicatesLegacyRecommendationFeedback(t *testing.T) {
	store := newActorMergeTestStore(t)
	ctx := context.Background()
	actorID := seedActorMergeActor(t, store, "Alice Smith", actorMergeFields(nil))
	seedActorMergeMovie(t, store, "feedback-backfill-movie", actorID)
	if _, err := store.db.Exec(`
		INSERT INTO homepage_recommendation_feedback (
			id, action, target_type, target_value, normalized_target, source_movie_id,
			expires_at, created_at, updated_at
		) VALUES
			('legacy-fullwidth', 'less', 'actor', 'Ａlice Smith', 'ａlice smith', 'feedback-backfill-movie', '', '2026-07-20T00:00:00Z', '2026-07-20T00:00:00Z'),
			('legacy-ascii', 'less', 'actor', 'Alice Smith', 'alice smith', 'feedback-backfill-movie', '', '2026-07-21T00:00:00Z', '2026-07-21T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	var count int
	var id, normalized string
	if err := store.db.QueryRow(`
		SELECT COUNT(*), MAX(id), MAX(normalized_target)
		FROM homepage_recommendation_feedback
		WHERE target_type = 'actor'`).Scan(&count, &id, &normalized); err != nil {
		t.Fatal(err)
	}
	if count != 1 || id != "legacy-ascii" || normalized != "alice smith" {
		t.Fatalf("legacy feedback backfill mismatch: count=%d id=%q normalized=%q", count, id, normalized)
	}
}

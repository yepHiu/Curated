package storage

import (
	"context"
	"curated-backend/internal/contracts"
	"reflect"
	"testing"
)

// TestRenameTopicPreservesIdentityEvidenceAndUndo covers user-only naming without changing movie decisions.
func TestRenameTopicPreservesIdentityEvidenceAndUndo(t *testing.T) {
	s := newTopicTestStore(t)
	ctx := context.Background()
	id := topicIdentifier("Theme")
	startTopicTestJob(t, s, "rename")
	input, _ := s.TopicMovieInput(ctx, "a")
	if err := s.SetTagOrganizationVocabulary(ctx, "rename", []TopicDefinition{{Name: "Theme", Aliases: []string{"Original"}}}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetTopicEvidence(ctx, "rename", "a", input.Fingerprint, []contracts.TopicEvidenceDTO{{Topic: "Theme", Field: "metadataTags", Quote: "Theme"}}); err != nil {
		t.Fatal(err)
	}
	if err := s.ApplyMovieTopics(ctx, "rename", input, []string{"Theme"}); err != nil {
		t.Fatal(err)
	}
	if err := s.RenameLibraryTopic(ctx, id, "Theme", "题材"); err == nil {
		t.Fatal("renamed during active job")
	}
	_ = s.UpdateTagOrganization(ctx, "rename", "completed", "finished", "")
	if err := s.RenameLibraryTopic(ctx, id, "Theme", "题材"); err != nil {
		t.Fatal(err)
	}
	after, _ := s.TopicMovieInput(ctx, "a")
	if !reflect.DeepEqual(after.UserTags, []string{"题材"}) || !reflect.DeepEqual(after.MetadataTags, []string{"Theme"}) {
		t.Fatalf("tag isolation: %+v", after)
	}
	topic, err := s.GetLibraryTopic(ctx, id)
	if err != nil || topic.Name != "题材" {
		t.Fatalf("stable ID: %+v %v", topic, err)
	}
	items, _ := s.TagOrganizationItems(ctx, "rename", false, 10, 0)
	if items[0].Evidence[0].Topic != "题材" || items[0].Evidence[0].Quote != "Theme" {
		t.Fatal("evidence quote changed")
	}
	defs, _ := s.GetTagOrganizationVocabulary(ctx, "rename")
	if defs[0].Name != "题材" {
		t.Fatal("retry vocabulary stale")
	}
	if err := s.SaveTopicVocabulary(ctx, []TopicDefinition{{Name: "题材"}}); err != nil {
		t.Fatal(err)
	}
	var count int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM library_topics`).Scan(&count)
	if count != 1 {
		t.Fatal("duplicate topic after rename")
	}
	undo, err := s.UndoTopicOrganization(ctx, "rename")
	if err != nil || undo.Restored != 1 {
		t.Fatalf("undo broken: %+v %v", undo, err)
	}
	restored, _ := s.TopicMovieInput(ctx, "a")
	if !reflect.DeepEqual(restored.UserTags, []string{"Original"}) {
		t.Fatal("undo lost original tags")
	}
	startTopicTestJob(t, s, "again")
	input, _ = s.TopicMovieInput(ctx, "a")
	if err := s.ApplyMovieTopics(ctx, "again", input, []string{"题材"}); err != nil {
		t.Fatal("renamed topic cannot classify", err)
	}
}

// TestRenameTopicPreservesManualDecisionsAndRejectsCollision protects manual exclusions and name ownership.
func TestRenameTopicPreservesManualDecisionsAndRejectsCollision(t *testing.T) {
	s := newTopicTestStore(t)
	ctx := context.Background()
	id := topicIdentifier("Theme")
	_ = s.PatchMovieUserPrefs(ctx, "a", contracts.PatchMovieInput{UserTagsSet: true, UserTags: []string{"Theme"}})
	_ = s.PatchMovieUserPrefs(ctx, "a", contracts.PatchMovieInput{UserTagsSet: true, UserTags: []string{"Personal"}})
	if err := s.RenameLibraryTopic(ctx, id, "stale", "题材"); err == nil {
		t.Fatal("stale rename accepted")
	}
	if err := s.RenameLibraryTopic(ctx, id, "Theme", "Personal"); err == nil {
		t.Fatal("manual label merged")
	}
	if err := s.RenameLibraryTopic(ctx, id, "Theme", "题材"); err != nil {
		t.Fatal(err)
	}
	startTopicTestJob(t, s, "manual")
	input, _ := s.TopicMovieInput(ctx, "a")
	if err := s.ApplyMovieTopics(ctx, "manual", input, []string{"题材"}); err != nil {
		t.Fatal(err)
	}
	after, _ := s.TopicMovieInput(ctx, "a")
	if !reflect.DeepEqual(after.UserTags, []string{"Personal"}) {
		t.Fatal("manual exclusion lost")
	}
	var count int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM movie_user_tag_decisions WHERE movie_id='a' AND name='题材' AND decision='exclude'`).Scan(&count)
	if count != 1 {
		t.Fatal("manual naming decision lost")
	}
}

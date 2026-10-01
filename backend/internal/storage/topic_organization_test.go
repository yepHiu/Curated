package storage

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"curated-backend/internal/contracts"
)

// newTopicTestStore 创建隔离测试库，并用相同名称的源标签测试类型隔离。
func newTopicTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	s, err := NewSQLiteStore(filepath.Join(t.TempDir(), "topics.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	ctx := context.Background()
	if err = s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"a", "b"} {
		_, err = s.db.Exec(`INSERT INTO movies(id,title,code,studio,summary,added_at,location,resolution,year) VALUES(?,?,?,'','A clear theme','2026-10-01',?,'',2026)`, id, id, id, id)
		if err != nil {
			t.Fatal(err)
		}
	}
	if err = s.PatchMovieUserPrefs(ctx, "a", contracts.PatchMovieInput{MetadataTagsSet: true, MetadataTags: []string{"Theme"}, UserTagsSet: true, UserTags: []string{"Original"}}); err != nil {
		t.Fatal(err)
	}
	if err = s.PatchMovieUserPrefs(ctx, "b", contracts.PatchMovieInput{MetadataTagsSet: true, MetadataTags: []string{"Theme"}}); err != nil {
		t.Fatal(err)
	}
	if err = s.SaveTopicVocabulary(ctx, []TopicDefinition{{Name: "Theme", Aliases: []string{"Original"}}}); err != nil {
		t.Fatal(err)
	}
	return s
}

// startTopicTestJob 将给定范围置为运行态以测试专用存储写入。
func startTopicTestJob(t *testing.T, s *SQLiteStore, id string) {
	t.Helper()
	ctx := context.Background()
	if _, err := s.CreateTagOrganization(ctx, id, id, "manual", "zh-CN", []string{"a"}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateTagOrganization(ctx, id, "running", "applying", ""); err != nil {
		t.Fatal(err)
	}
}

// TestTopicOrganizationNFOIsolationAndUndo 验证转换和撤销仅作用于用户标签。
func TestTopicOrganizationNFOIsolationAndUndo(t *testing.T) {
	s := newTopicTestStore(t)
	ctx := context.Background()
	startTopicTestJob(t, s, "job")
	input, err := s.TopicMovieInput(ctx, "a")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.ApplyMovieTopics(ctx, "job", input, []string{"Theme"}); err != nil {
		t.Fatal(err)
	}
	if err = s.ApplyMovieTopics(ctx, "job", input, []string{"Theme"}); err != nil {
		t.Fatal(err)
	}
	after, err := s.TopicMovieInput(ctx, "a")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(after.MetadataTags, input.MetadataTags) || !reflect.DeepEqual(after.UserTags, []string{"Theme"}) {
		t.Fatalf("unexpected tags: %+v", after)
	}
	page, err := s.ListMovies(ctx, contracts.ListMoviesRequest{TopicID: topicIdentifier("Theme")})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || page.Items[0].ID != "a" {
		t.Fatalf("NFO-only movie leaked: %+v", page)
	}
	if err = s.UpdateTagOrganization(ctx, "job", "completed", "applying", ""); err != nil {
		t.Fatal(err)
	}
	u, err := s.UndoTopicOrganization(ctx, "job")
	if err != nil || u.Restored != 1 {
		t.Fatalf("undo=%+v %v", u, err)
	}
	restored, err := s.TopicMovieInput(ctx, "a")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(restored.UserTags, input.UserTags) || !reflect.DeepEqual(restored.MetadataTags, input.MetadataTags) {
		t.Fatalf("restored=%+v", restored)
	}
	u, err = s.UndoTopicOrganization(ctx, "job")
	if err != nil || u.Restored != 0 {
		t.Fatalf("repeat undo=%+v %v", u, err)
	}
}

// TestTopicOrganizationRejectsNFOAndStaleWrites 检验数据库隔离、范围和人工并发保护。
func TestTopicOrganizationRejectsNFOAndStaleWrites(t *testing.T) {
	s := newTopicTestStore(t)
	ctx := context.Background()
	if _, err := s.db.Exec(`INSERT INTO library_topics(id,tag_id) SELECT 'bad',id FROM tags WHERE name='Theme' AND type='nfo'`); err == nil {
		t.Fatal("NFO topic accepted")
	}
	startTopicTestJob(t, s, "job")
	input, err := s.TopicMovieInput(ctx, "a")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.PatchMovieUserPrefs(ctx, "a", contracts.PatchMovieInput{UserTagsSet: true, UserTags: []string{"manual"}}); err != nil {
		t.Fatal(err)
	}
	if err = s.ApplyMovieTopics(ctx, "job", input, []string{"Theme"}); !errors.Is(err, ErrAIWriteConflict) {
		t.Fatalf("want conflict: %v", err)
	}
	b, err := s.TopicMovieInput(ctx, "b")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.ApplyMovieTopics(ctx, "job", b, []string{"Theme"}); err == nil {
		t.Fatal("out-of-scope write accepted")
	}
}

// TestTopicOrganizationHonorsManualExclusionAndUndoConflict 确保手工纠错后 AI 不加回且撤销不覆盖。
func TestTopicOrganizationHonorsManualExclusionAndUndoConflict(t *testing.T) {
	s := newTopicTestStore(t)
	ctx := context.Background()
	startTopicTestJob(t, s, "first")
	input, _ := s.TopicMovieInput(ctx, "a")
	if err := s.ApplyMovieTopics(ctx, "first", input, []string{"Theme"}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateTagOrganization(ctx, "first", "completed", "applying", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.PatchMovieUserPrefs(ctx, "a", contracts.PatchMovieInput{UserTagsSet: true, UserTags: []string{"mine"}}); err != nil {
		t.Fatal(err)
	}
	u, err := s.UndoTopicOrganization(ctx, "first")
	if err != nil || u.Conflicts != 1 || u.Restored != 0 {
		t.Fatalf("undo=%+v %v", u, err)
	}
	startTopicTestJob(t, s, "second")
	input, _ = s.TopicMovieInput(ctx, "a")
	if err = s.ApplyMovieTopics(ctx, "second", input, []string{"Theme"}); err != nil {
		t.Fatal(err)
	}
	after, _ := s.TopicMovieInput(ctx, "a")
	if !reflect.DeepEqual(after.UserTags, []string{"mine"}) {
		t.Fatalf("manual exclusion lost: %+v", after)
	}
}

// TestTopicManualNoopPreservesUndo verifies an unchanged tag form does not create a false conflict.
func TestTopicManualNoopPreservesUndo(t *testing.T) {
	s := newTopicTestStore(t)
	ctx := context.Background()
	startTopicTestJob(t, s, "noop")
	input, _ := s.TopicMovieInput(ctx, "a")
	if err := s.ApplyMovieTopics(ctx, "noop", input, []string{"Theme"}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateTagOrganization(ctx, "noop", "completed", "finished", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.PatchMovieUserPrefs(ctx, "a", contracts.PatchMovieInput{UserTagsSet: true, UserTags: []string{"Theme"}}); err != nil {
		t.Fatal(err)
	}
	undo, err := s.UndoTopicOrganization(ctx, "noop")
	if err != nil || undo.Restored != 1 || undo.Conflicts != 0 {
		t.Fatalf("undo=%+v %v", undo, err)
	}
	if err := s.RetryTagOrganization(ctx, "noop"); err == nil {
		t.Fatal("undone task was replayed")
	}
}

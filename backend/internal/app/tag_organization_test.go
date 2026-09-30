package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/llm"
	"curated-backend/internal/storage"
)

// topicFixtureProvider only understands synthetic local fixtures; it never calls an external provider.
func topicFixtureProvider(t *testing.T, a *App, before func(bool)) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mirror the real completion envelope while preserving strict source evidence.
		var req struct {
			Messages []llm.ChatMessage `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Messages) < 2 {
			http.Error(w, "invalid fixture", 400)
			return
		}
		vocabulary := strings.Contains(req.Messages[0].Content, "vocabulary v1")
		if before != nil {
			before(vocabulary)
		}
		var result any
		if vocabulary {
			result = map[string]any{"topics": []storage.TopicDefinition{{Name: "Theme", Description: "Synthetic subject", Aliases: []string{"Alias"}}}}
		} else {
			var source struct {
				Movies []storage.TopicMovieInput `json:"movies"`
			}
			raw := strings.TrimSuffix(strings.TrimPrefix(req.Messages[1].Content, "<source>"), "</source>")
			if err := json.Unmarshal([]byte(raw), &source); err != nil {
				http.Error(w, "invalid source", 400)
				return
			}
			movies := []topicClassification{}
			for _, input := range source.Movies {
				movies = append(movies, topicClassification{MovieID: input.MovieID, Matches: []topicMatch{{Topic: "Theme", Field: "metadataTags", Quote: "Theme"}}})
			}
			result = map[string]any{"movies": movies}
		}
		content, _ := json.Marshal(result)
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": string(content)}}}})
	}))
	t.Cleanup(server.Close)
	a.cfg.AIProvider = config.AIProviderConfig{BaseURL: server.URL, Model: "synthetic-topic-fixture"}
	// The production setting is limited to 60; this fixture avoids waiting for real-time quotas.
	a.cfg.AIGovernance.WritePerMinute = 10000
}

// seedTopicMovies creates a temporary library and literal NFO files to detect accidental file writes.
func seedTopicMovies(t *testing.T, a *App, count int) ([]string, string) {
	t.Helper()
	ctx := context.Background()
	root := t.TempDir()
	nfo := filepath.Join(root, "movie.nfo")
	if err := os.WriteFile(nfo, []byte("<movie><tag>Theme</tag></movie>"), 0600); err != nil {
		t.Fatal(err)
	}
	ids := []string{}
	for i := 0; i < count; i++ {
		name := fmt.Sprintf("SYN-%04d", i)
		row, err := a.store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{TaskID: "synthetic", Path: filepath.Join(root, name+".mp4"), FileName: name + ".mp4", Number: name})
		if err != nil {
			t.Fatal(err)
		}
		if err = a.store.PatchMovieUserPrefs(ctx, row.MovieID, contracts.PatchMovieInput{MetadataTagsSet: true, MetadataTags: []string{"Theme"}, UserTagsSet: true, UserTags: []string{"Alias", "Personal note"}}); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, row.MovieID)
	}
	return ids, nfo
}

// TestTagOrganization600AndUndo covers the full fake-provider pipeline and NFO/user-name isolation.
func TestTagOrganization600AndUndo(t *testing.T) {
	a := governanceTestApp(t)
	ids, nfo := seedTopicMovies(t, a, 600)
	var calls atomic.Int64
	topicFixtureProvider(t, a, func(bool) { calls.Add(1) })
	ctx := context.Background()
	id, err := a.store.CreateTagOrganization(ctx, "bulk", "bulk", "manual", ids)
	if err != nil {
		t.Fatal(err)
	}
	a.runTagOrganization(ctx, id)
	job, err := a.GetTagOrganization(ctx, id)
	if err != nil || job.Status != "completed" || job.Succeeded != 600 {
		t.Fatalf("job=%+v err=%v", job, err)
	}
	if calls.Load() != 150 {
		t.Fatalf("unbounded or missing batch requests: %d", calls.Load())
	}
	for _, mid := range ids {
		input, err := a.store.TopicMovieInput(ctx, mid)
		if err != nil || !reflect.DeepEqual(input.MetadataTags, []string{"Theme"}) || !reflect.DeepEqual(input.UserTags, []string{"Personal note", "Theme"}) {
			t.Fatalf("tags=%+v err=%v", input, err)
		}
	}
	groups, err := a.HomepageTopics(ctx)
	if err != nil || len(groups) != 1 || len(groups[0].Movies) != 6 || groups[0].Topic.MovieCount != 600 {
		t.Fatalf("groups=%+v err=%v", groups, err)
	}
	// Re-entering a completed job must not call the provider again.
	a.runTagOrganization(ctx, id)
	if calls.Load() != 150 {
		t.Fatal("completed job reprocessed")
	}
	undo, err := a.UndoTagOrganization(ctx, id)
	if err != nil || undo.Restored != 600 || undo.Conflicts != 0 {
		t.Fatalf("undo=%+v err=%v", undo, err)
	}
	for _, mid := range ids {
		input, err := a.store.TopicMovieInput(ctx, mid)
		if err != nil || !reflect.DeepEqual(input.MetadataTags, []string{"Theme"}) || !reflect.DeepEqual(input.UserTags, []string{"Alias", "Personal note"}) {
			t.Fatalf("undo tags=%+v err=%v", input, err)
		}
	}
	bytes, err := os.ReadFile(nfo)
	if err != nil || string(bytes) != "<movie><tag>Theme</tag></movie>" {
		t.Fatal("NFO file changed")
	}
}

// TestTagOrganizationCancelAndResume proves cancellation stops late results and retry resumes checkpoints.
func TestTagOrganizationCancelAndResume(t *testing.T) {
	a := governanceTestApp(t)
	ids, _ := seedTopicMovies(t, a, 6)
	entered, release := make(chan struct{}), make(chan struct{})
	var blocked atomic.Bool
	topicFixtureProvider(t, a, func(vocabulary bool) {
		// Hold one provider reply until after the persisted cancellation.
		if !vocabulary && blocked.CompareAndSwap(false, true) {
			close(entered)
			<-release
		}
	})
	ctx := context.Background()
	job, err := a.StartTagOrganization(ctx, contracts.TagOrganizationRequest{Scope: "selected", MovieIDs: ids, RequestID: "cancel-test"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.tagRT.cancel(); <-a.tagRT.done })
	select {
	case <-entered:
	case <-time.After(10 * time.Second):
		close(release)
		t.Fatal("worker did not classify")
	}
	if err = a.CancelTagOrganization(ctx, job.ID); err != nil {
		close(release)
		t.Fatal(err)
	}
	close(release)
	// Cancelling a running model request cannot commit any of its six movie results.
	a.tagRT.cancel()
	<-a.tagRT.done
	cancelled, _ := a.GetTagOrganization(ctx, job.ID)
	if cancelled.Status != "cancelled" || cancelled.Processed != 0 {
		t.Fatalf("late result: %+v", cancelled)
	}
	if err = a.store.RetryTagOrganization(ctx, job.ID); err != nil {
		t.Fatal(err)
	}
	a.runTagOrganization(ctx, job.ID)
	resumed, _ := a.GetTagOrganization(ctx, job.ID)
	if resumed.Status != "completed" || resumed.Succeeded != 6 {
		t.Fatalf("resume=%+v", resumed)
	}
	if err = a.CancelTagOrganization(ctx, job.ID); err != nil {
		t.Fatal(err)
	}
	complete, _ := a.GetTagOrganization(ctx, job.ID)
	if complete.Status != "completed" {
		t.Fatal("cancel overwrote terminal status")
	}
}

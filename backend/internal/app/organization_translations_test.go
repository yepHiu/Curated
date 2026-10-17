package app

import (
	"context"
	"curated-backend/internal/config"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestOrganizationTranslationOnlyWorker(t *testing.T) {
	a := governanceTestApp(t)
	ids, _ := seedTopicMovies(t, a, 2)
	ctx := context.Background()
	var calls atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": "Translated title"}}}})
	}))
	defer server.Close()
	a.cfg.AIProvider = config.AIProviderConfig{BaseURL: server.URL, Model: "translation-fixture"}
	a.cfg.AIGovernance.WritePerMinute = 10000
	id, err := a.store.CreateTagOrganization(ctx, "translate-only", "translate-only", "manual", "en", ids, "title")
	if err != nil {
		t.Fatal(err)
	}
	a.runTagOrganization(ctx, id)
	job, err := a.GetTagOrganization(ctx, id)
	if err != nil || job.Status != "completed" || job.Succeeded != 2 || calls.Load() != 2 {
		t.Fatalf("job %+v calls %d error %v", job, calls.Load(), err)
	}
	for _, mid := range ids {
		detail, err := a.GetMovieDetail(ctx, mid)
		if err != nil || detail.Title != "Translated title" {
			t.Fatalf("detail %+v %v", detail, err)
		}
	}
	a.runTagOrganization(ctx, id)
	if calls.Load() != 2 {
		t.Fatal("completed translations repeated")
	}
}

package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/llm"
	"curated-backend/internal/storage"
)

func TestTopicReuseExactNameAndAlias(t *testing.T) {
	a := governanceTestApp(t)
	resolved, err := a.reuseTopicProposals(context.Background(), []storage.TopicDefinition{{Name: "trip"}, {Name: "TRAVEL"}}, []storage.TopicDefinition{{Name: "Travel", Description: "Travel subject", Aliases: []string{"Trip"}}})
	if err != nil || len(resolved) != 2 || resolved[0].Name != "Travel" || resolved[1].Name != "Travel" {
		t.Fatalf("reuse=%+v err=%v", resolved, err)
	}
}

func TestTopicReuseSemanticMappingAcrossPages(t *testing.T) {
	for _, invalid := range []bool{false, true} {
		t.Run(fmt.Sprintf("invalid=%v", invalid), func(t *testing.T) {
			a := governanceTestApp(t)
			catalog := []storage.TopicDefinition{}
			for i := 0; i < 150; i++ {
				catalog = append(catalog, storage.TopicDefinition{Name: fmt.Sprintf("Topic-%03d", i)})
			}
			catalog = append(catalog, storage.TopicDefinition{Name: "旅行"})
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				var req struct {
					Messages []llm.ChatMessage `json:"messages"`
				}
				_ = json.NewDecoder(r.Body).Decode(&req)
				content := `{"matches":[]}`
				if strings.Contains(req.Messages[1].Content, `"name":"旅行"`) {
					target := "旅行"
					if invalid {
						target = "invented target"
					}
					content = fmt.Sprintf(`{"matches":[{"proposedName":"旅游","canonicalName":%q}]}`, target)
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": content}}}})
			}))
			defer server.Close()
			a.cfg.AIProvider = config.AIProviderConfig{BaseURL: server.URL, Model: "reuse-fixture"}
			resolved, err := a.reuseTopicProposals(context.Background(), []storage.TopicDefinition{{Name: "旅游"}, {Name: "登山"}}, catalog)
			if invalid {
				if err == nil {
					t.Fatal("out of scope target accepted")
				}
				return
			}
			if err != nil || calls != 2 || resolved[0].Name != "旅行" || resolved[1].Name != "登山" {
				t.Fatalf("reuse=%+v calls=%d err=%v", resolved, calls, err)
			}
		})
	}
}

func TestTagOrganizationReusesOrdinaryUserLabelInSelectedScope(t *testing.T) {
	a := governanceTestApp(t)
	ids, _ := seedTopicMovies(t, a, 2)
	ctx := context.Background()
	if err := a.store.PatchMovieUserPrefs(ctx, ids[1], contracts.PatchMovieInput{UserTagsSet: true, UserTags: []string{"旅行"}}); err != nil {
		t.Fatal(err)
	}
	if err := a.store.PatchMovieUserPrefs(ctx, ids[0], contracts.PatchMovieInput{MetadataTagsSet: true, MetadataTags: []string{"旅游"}}); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Messages []llm.ChatMessage `json:"messages"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		content := `{"topics":[{"name":"旅游","description":"Travel subject","aliases":[]}]}`
		switch {
		case strings.Contains(req.Messages[0].Content, "topic reuse v"):
			if !strings.Contains(req.Messages[1].Content, "旅行") {
				t.Error("existing user label missing")
			}
			content = `{"matches":[{"proposedName":"旅游","canonicalName":"旅行"}]}`
		case strings.Contains(req.Messages[0].Content, "classification v"):
			content = fmt.Sprintf(`{"movies":[{"movieId":%q,"matches":[{"topic":"旅行","field":"metadataTags","quote":"旅游"}]}]}`, ids[0])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": content}}}})
	}))
	defer server.Close()
	a.cfg.AIProvider = config.AIProviderConfig{BaseURL: server.URL, Model: "reuse-scope-fixture"}
	id := createScaleTopicJob(t, a, ids[:1])
	a.runTagOrganization(ctx, id)
	job, err := a.store.GetTagOrganization(ctx, id)
	if err != nil || job.Status != "completed" || job.Succeeded != 1 {
		t.Fatalf("job=%+v err=%v", job, err)
	}
	defs, err := a.store.TopicVocabulary(ctx)
	if err != nil || len(defs) != 1 || defs[0].Name != "旅行" {
		t.Fatalf("duplicate created: %+v err=%v", defs, err)
	}
	first, _ := a.store.TopicMovieInput(ctx, ids[0])
	other, _ := a.store.TopicMovieInput(ctx, ids[1])
	if strings.Join(first.MetadataTags, ",") != "旅游" || strings.Join(other.UserTags, ",") != "旅行" {
		t.Fatal("source or outside scope movie modified")
	}
	if _, err := a.UndoTagOrganization(ctx, id); err != nil {
		t.Fatal(err)
	}
	first, _ = a.store.TopicMovieInput(ctx, ids[0])
	if strings.Join(first.UserTags, ",") != "Alias,Personal note" {
		t.Fatalf("undo=%+v", first)
	}
}

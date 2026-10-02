package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"curated-backend/internal/config"
	"curated-backend/internal/storage"
)

func TestTopicProposalNormalizationPreservesDistinctSubjects(t *testing.T) {
	proposed := []storage.TopicDefinition{
		{Name: " Travel ", Aliases: []string{" trip ", "TRIP", "shared", "Climbing", "Existing"}},
		{Name: "travel", Aliases: []string{"Journey"}},
		{Name: "Climbing", Aliases: []string{"shared", "Hiking"}},
	}
	got, err := normalizeTopicProposals(proposed, []storage.TopicDefinition{{Name: "Existing"}})
	want := []storage.TopicDefinition{{Name: "Travel", Aliases: []string{"trip", "Journey"}}, {Name: "Climbing", Aliases: []string{"Hiking"}}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	if proposed[0].Name != " Travel " || proposed[0].Aliases[0] != " trip " {
		t.Fatal("input mutated")
	}
}

func TestTopicVocabularyCorrectionAndBoundedFailure(t *testing.T) {
	for _, corrected := range []bool{false, true} {
		t.Run(fmt.Sprint(corrected), func(t *testing.T) {
			a := governanceTestApp(t)
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				var request map[string]any
				_ = json.NewDecoder(r.Body).Decode(&request)
				if calls == 2 {
					data, _ := json.Marshal(request)
					if !strings.Contains(string(data), "AI_ORGANIZATION_VOCABULARY_INVALID") {
						t.Error("missing correction feedback")
					}
				}
				content := `{"topics":[{"name":" ","description":"","aliases":[]}]}`
				if corrected && calls == 2 {
					content = `{"topics":[{"name":" Travel ","description":"Travel","aliases":["Trip"," trip "]}]}`
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": content}}}})
			}))
			defer server.Close()
			a.cfg.AIProvider = config.AIProviderConfig{BaseURL: server.URL, Model: "correction-fixture"}
			got, err := a.proposeTopicVocabulary(context.Background(), nil, nil, "zh-CN")
			if calls != 2 {
				t.Fatalf("calls=%d", calls)
			}
			if corrected {
				if err != nil || len(got) != 1 || got[0].Name != "Travel" || len(got[0].Aliases) != 1 {
					t.Fatalf("got=%+v err=%v", got, err)
				}
			} else if topicOrganizationErrorCode(err) != "AI_ORGANIZATION_VOCABULARY_INVALID" {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestTopicReuseCorrectionIsAtomicAndDuplicateSafe(t *testing.T) {
	a := governanceTestApp(t)
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		content := `{"matches":[{"proposedName":"Trip","canonicalName":"Travel"},{"proposedName":"Trip","canonicalName":"Invented"}]}`
		if calls == 2 {
			content = `{"matches":[{"proposedName":" trip ","canonicalName":" TRAVEL "},{"proposedName":"Trip","canonicalName":"Travel"}]}`
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": content}}}})
	}))
	defer server.Close()
	a.cfg.AIProvider = config.AIProviderConfig{BaseURL: server.URL, Model: "reuse-correction"}
	got, err := a.reuseTopicProposals(context.Background(), []storage.TopicDefinition{{Name: "Trip"}}, []storage.TopicDefinition{{Name: "Travel"}})
	if err != nil || calls != 2 || len(got) != 1 || got[0].Name != "Travel" {
		t.Fatalf("got=%+v calls=%d err=%v", got, calls, err)
	}
}

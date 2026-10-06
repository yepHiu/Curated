package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"curated-backend/internal/config"
	"curated-backend/internal/llm"
	"curated-backend/internal/storage"
)

// Exercise the actual outbound payload: movie batches must share the entire
// vocabulary prefix while retaining their distinct source evidence.
func TestTopicClassificationSharesVocabularyPrefixAcrossBatches(t *testing.T) {
	a := governanceTestApp(t)
	var payloads []string
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Messages []llm.ChatMessage `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil || len(request.Messages) != 2 {
			t.Error("invalid request")
			http.Error(w, "invalid request", 400)
			return
		}
		payloads = append(payloads, request.Messages[1].Content)
		var source topicScaleRequest
		raw := strings.TrimSuffix(strings.TrimPrefix(request.Messages[1].Content, "<source>"), "</source>")
		if err := json.Unmarshal([]byte(raw), &source); err != nil {
			t.Error(err)
		}
		content, _ := json.Marshal(scaleTopicClassification(source))
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": string(content)}}}})
	}))
	defer provider.Close()
	a.cfg.AIProvider = config.AIProviderConfig{BaseURL: provider.URL, Model: "cache-fixture", ContextWindow: 32768}
	defs := []storage.TopicDefinition{{Name: "Theme", Description: strings.Repeat("definition ", 500)}}
	for _, id := range []string{"first", "second"} {
		input := storage.TopicMovieInput{MovieID: id, Title: id, MetadataTags: []string{"Theme"}, UserTags: []string{}}
		_, matches, err := a.classifyTopicBatch(context.Background(), defs, []storage.TopicMovieInput{input})
		if err != nil || len(matches[id]) != 1 || matches[id][0] != "Theme" {
			t.Fatalf("classification lost source evidence: %v, %v", matches, err)
		}
	}
	if len(payloads) != 2 {
		t.Fatalf("requests=%d", len(payloads))
	}
	vocabulary, _ := json.Marshal(defs)
	wantPrefix := `<source>{"vocabulary":` + string(vocabulary) + `,"movies":`
	for _, payload := range payloads {
		if !strings.HasPrefix(payload, wantPrefix) {
			t.Fatal("changing source data precedes the shared vocabulary")
		}
	}
	if payloads[0] == payloads[1] || !strings.Contains(payloads[0], `"movieId":"first"`) || !strings.Contains(payloads[1], `"movieId":"second"`) {
		t.Fatal("distinct batches did not retain their source data")
	}
}

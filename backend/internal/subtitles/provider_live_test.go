package subtitles

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"curated-backend/internal/agent/prompts"
	"curated-backend/internal/llm"
)

// Explicitly opt-in; ordinary regression tests never contact or charge a provider.
func TestLiveBailianShortAudioAndTranslationModels(t *testing.T) {
	key, source := os.Getenv("CURATED_SUBTITLE_LIVE_KEY"), os.Getenv("CURATED_SUBTITLE_LIVE_AUDIO")
	if key == "" || source == "" {
		t.Skip("short-audio live provider validation is opt-in")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	client := &http.Client{Timeout: 90 * time.Second}
	provider := Bailian{Client: client}
	if e := provider.TestPolicy(ctx, key); e != nil {
		t.Fatal(e)
	}
	uploaded, e := provider.Upload(ctx, key, source)
	if e != nil {
		t.Fatal(e)
	}
	id, e := provider.Submit(ctx, key, uploaded)
	if e != nil {
		t.Fatal(e)
	}
	var cues []Cue
	for {
		var done bool
		cues, done, e = provider.Result(ctx, key, id)
		if e != nil {
			t.Fatal(e)
		}
		if done {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("live recognition timeout")
		case <-time.After(3 * time.Second):
		}
	}
	if len(cues) == 0 {
		t.Fatal("sample has no recognized speech; choose a speech sample")
	}
	if len(cues) > 3 {
		cues = cues[:3]
	}
	input := []map[string]any{}
	for i := range cues {
		cues[i].ID = i + 1
		input = append(input, map[string]any{"id": cues[i].ID, "text": cues[i].Text})
	}
	raw, _ := json.Marshal(input)
	for _, model := range []string{"qwen-flash", "qwen-plus"} {
		disabled := false
		c := llm.NewClient(llm.ClientConfig{BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1", APIKey: key, Model: model, EnableThinking: &disabled}, client)
		body, e := c.Complete(ctx, []llm.ChatMessage{{Role: "system", Content: prompts.SubtitleTranslationPrompt().Text}, {Role: "user", Content: string(raw)}}, 2048)
		if e != nil {
			t.Fatalf("%s request failed (%s)", model, llm.ErrorCategory(e))
		}
		var translated []Cue
		if e = json.Unmarshal([]byte(body), &translated); e != nil {
			t.Fatalf("%s invalid JSON", model)
		}
		if _, e = validateTranslations(cues, translated); e != nil {
			var shapes []map[string]any
			_ = json.Unmarshal([]byte(body), &shapes)
			for _, shape := range shapes {
				for key, value := range shape {
					if str, ok := value.(string); ok {
						t.Logf("field %s chars=%d", key, len([]rune(str)))
					}
				}
			}
			for _, c := range translated {
				t.Logf("%s cue id=%d translationChars=%d", model, c.ID, len([]rune(c.Translation)))
			}
			t.Fatalf("%s: %v", model, e)
		}
		t.Logf("%s: %d cue IDs preserved with non-empty translations", model, len(translated))
	}
	t.Logf("Qwen 3.1: short-audio upload, async submission, polling and signed result download passed")
}

package prompts

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"testing"

	"curated-backend/internal/contracts"
)

// TestTXTTemplatesPreserveMessages 对照迁移前捕获的消息哈希，保护空白、默认值及源资料不被二次渲染。
func TestTXTTemplatesPreserveMessages(t *testing.T) {
	page := &contracts.AIChatContext{Route: "movie-detail", MovieID: "m1", ActorName: "A", SelectedMovieIDs: []string{"m2"}, ActiveFilters: &contracts.AIChatActiveFilters{Query: "query", PlayState: "unwatched"}, Mentions: []contracts.AIChatMention{{Kind: "movie", ID: "m3", Label: "{{.Locale}}"}}}
	cases := map[string]string{
		"system-empty": SystemPrompt("", nil), "system-context": SystemPrompt("zh-CN", page),
		"polish":            CommentActionPrompt("  笔记 {{.Locale}} <source>\n第二行  "),
		"translate-ja":      TranslateDisplayPrompt("synopsis", "原文 {{.Source}}\n次の行", "ja"),
		"translate-default": TranslateDisplayPrompt("", "  source  ", ""),
		"insights":          InsightsNarrativePrompt("en", `{"count":3,"name":"{{.Locale}}"}`),
	}
	expected := map[string]string{
		"system-empty":      "0cc34a4671602e6f1e983b379b34c9ea74dca9c8dae99f01e350bff1f1d50e7a",
		"system-context":    "095a5554c7b43bfab7ebf2fd0517c1241cc30b0fe577402ac9f442d0c3213eab",
		"polish":            "394fd84289a9ce2f8fd0326508ccda4248d2a487e16c1ab269abb77c9b88548a",
		"translate-ja":      "8f9f733d76c3893f3ce89817d97fb567564344fd5a59758ca65186f4bc9e2d54",
		"translate-default": "94e4bc78f430e2534c841942681519f098139bba9aaba917a0f40950fbd63fec",
		"insights":          "a608c30903d5785a76e7c4403f9fb983d2ce46e9aa5a6f3f1d630b45d24d1124",
	}
	for name, value := range cases {
		if got := fmt.Sprintf("%x", sha256.Sum256([]byte(value))); got != expected[name] {
			t.Errorf("%s: message changed: %s", name, got)
		}
	}
}

// TestTXTTemplatesRejectMissingVariables 确认缺失变量无法生成并发送不完整的内置提示词。
func TestTXTTemplatesRejectMissingVariables(t *testing.T) {
	for _, name := range []string{"polish-comment.txt", "translate-display.txt", "insights-narrative.txt", "response-language.txt", "page-context.txt", "active-filters.txt", "mentions.txt"} {
		var out bytes.Buffer
		if err := promptTemplates.ExecuteTemplate(&out, name, map[string]string{}); err == nil {
			t.Errorf("%s accepted missing variables", name)
		}
	}
}

// TestTXTStaticPromptsPreserveRules 对照原内联文本，确保迁移未改变整理规则、摘要约束和纠正策略。
func TestTXTStaticPromptsPreserveRules(t *testing.T) {
	vocabulary, classification := TopicVocabularyPrompt(), TopicClassificationPrompt()
	if vocabulary.Version != "topic-vocabulary-v5" || classification.Version != "topic-classification-v1" {
		t.Fatal("topic audit versions changed")
	}
	cases := []struct{ name, text, hash string }{
		{"topic-vocabulary.txt", vocabulary.Text, "260b36f261e69bb4e9a26a2feecd3062ab7a3e0cd1e31d168b9507e8b16e5d1f"},
		{"topic-classification.txt", classification.Text, "978671cc99f7ba26d825b8df914450db76e43dc10ccb3521ee9210fcbbd75acf"},
		{"memory-checkpoint.txt", MemoryCheckpointPrompt(), "2a0a98f491fab02f3c6f2ab968ac6907381bddb67ecb18b38aa54252b74b3640"},
		{"answer-correction.txt", AnswerCorrectionPrompt(), "173ff2d1aa3bf3eb4685b39f77fd40ae4eaa9c4e3e7a626761a6ccc8cf8ec314"},
		{"history-omitted.txt", HistoryOmittedPrompt(), "1c7d972257f75742e7b4e80bbc66b158cd9981782062821d3b5d53f78b41ba25"},
		{"provider-probe.txt", ProviderProbePrompt(), "34f22dd3f9172cb9d328266e996b8d46727e41171d05a4137bd59bb7fee0ee7c"},
	}
	for _, tc := range cases {
		if got := fmt.Sprintf("%x", sha256.Sum256([]byte(tc.text))); got != tc.hash {
			t.Errorf("%s: rules changed: %s", tc.name, got)
		}
	}
}

package prompts

import (
	"strings"
	"testing"
)

func TestCommentActionPromptGroundsInSource(t *testing.T) {
	t.Parallel()
	got := CommentActionPrompt("节奏慢但结尾反转不错")
	for _, want := range []string{
		"<source>",
		"节奏慢但结尾反转不错",
		"Do not invent facts",
		"Detect the language",
		"Do not translate",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q:\n%s", want, got)
		}
	}
}

func TestTranslateSummaryPromptUsesLocale(t *testing.T) {
	t.Parallel()
	got := TranslateSummaryPrompt("Visit xxxx.com for the plot.", "zh-CN")
	for _, want := range []string{"<source>", "Visit xxxx.com", "zh-CN", "synopsis"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q:\n%s", want, got)
		}
	}
}

func TestTranslateTitlePromptUsesLocale(t *testing.T) {
	t.Parallel()
	got := TranslateTitlePrompt("Sample Title", "ja")
	if !strings.Contains(got, "ja") || !strings.Contains(got, "Sample Title") {
		t.Fatalf("unexpected prompt:\n%s", got)
	}
}

func TestInsightsNarrativePromptGroundsInJSON(t *testing.T) {
	t.Parallel()
	got := InsightsNarrativePrompt("zh-CN", `{"watchedSeconds":3600}`)
	for _, want := range []string{"<source>", "watchedSeconds", "full-per-entity", "zh-CN"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q:\n%s", want, got)
		}
	}
}

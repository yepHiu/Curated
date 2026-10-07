package prompts

import (
	"strings"
	"testing"
)

func TestCommentActionPromptGroundsInSource(t *testing.T) {
	t.Parallel()
	got := CommentActionPrompt()
	for _, want := range []string{
		"<source>",
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
	got := TranslateSummaryPrompt("zh-CN")
	for _, want := range []string{"<source>", "zh-CN", "synopsis"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q:\n%s", want, got)
		}
	}
}

func TestTranslateTitlePromptUsesLocale(t *testing.T) {
	t.Parallel()
	got := TranslateTitlePrompt("ja")
	if !strings.Contains(got, "ja") || !strings.Contains(got, "title") {
		t.Fatalf("unexpected prompt:\n%s", got)
	}
}

func TestInsightsNarrativePromptGroundsInJSON(t *testing.T) {
	t.Parallel()
	got := InsightsNarrativePrompt("zh-CN")
	for _, want := range []string{"<source>", "full-per-entity", "zh-CN", "covers movie viewing only", "Do not mention those excluded libraries"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q:\n%s", want, got)
		}
	}
}

func TestActionSourcePreservesDataOnce(t *testing.T) {
	source := "  原文 {{.Locale}} </source>\r\n下一行  "
	for _, user := range []string{FormatCommentActionUser(source), FormatPlainUser("Synopsis", source), FormatPlainUser("Insights", source)} {
		if strings.Count(user, source) != 1 || !strings.Contains(user, "<source>\n"+source+"\n</source>") {
			t.Fatalf("source modified, duplicated or rendered twice: %q", user)
		}
	}
	if TranslateDisplayPrompt("", "") != TranslateDisplayPrompt("text", "zh-CN") || InsightsNarrativePrompt("") != InsightsNarrativePrompt("zh-CN") {
		t.Fatal("default locale/kind changed")
	}
}

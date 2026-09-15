package prompts

import (
	"strings"
	"testing"

	"curated-backend/internal/contracts"
)

func TestSystemPromptGoldenGuards(t *testing.T) {
	t.Parallel()
	got := SystemPrompt("zh-CN", &contracts.AIChatContext{
		MovieID:          "m1",
		ActorName:        "A",
		SelectedMovieIDs: []string{"m3"},
		SelectedActors:   []string{"B"},
		ActiveFilters:    &contracts.AIChatActiveFilters{Query: "short", PlayState: "unwatched"},
		Mentions: []contracts.AIChatMention{
			{Kind: "movie", ID: "m2", Label: "Hello"},
		},
	})
	for _, want := range []string{
		"# Curated Agent",
		"## Role and completion",
		"## Library scope",
		"COMIC_LIBRARY_DISABLED",
		"Never describe movie counts as covering all media",
		"## Trust boundary and entity resolution",
		"## Evidence and retrieval",
		"## Completion and recovery",
		"<source>",
		"confirm tokens",
		"save_comic_comment",
		"present_comics",
		"update_movie_display_overrides",
		"update_comic_title",
		"update_photo_title",
		"create_saved_view",
		"present_movies",
		"search_provider_titles",
		"get_source_page",
		"movieId=m1",
		"actor=A",
		"selectedMovieIds=m3",
		"selectedActors=B",
		"Active library filters",
		"playState=unwatched",
		"zh-CN",
		"@-mentions",
		"movie id=m2 label=Hello",
		"ambiguous",
		"needs input",
		"partial",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("prompt missing %q:\n%s", want, got)
		}
	}
}

func TestSystemPromptKeepsBookPageData(t *testing.T) {
	got := SystemPrompt("zh-CN", &contracts.AIChatContext{Route: "photo-detail", MovieID: "private-book-id", PhotoID: "photo-1", Query: "private-book-title"})
	if strings.Contains(got, "private-book-id") {
		t.Fatal("book page movieId reached model prompt")
	}
	if !strings.Contains(got, "photoId=photo-1") || !strings.Contains(got, "private-book-title") {
		t.Fatal("book page context missing from model prompt")
	}
}

func TestSystemPromptUsesExternalTemplate(t *testing.T) {
	t.Parallel()
	if !strings.Contains(systemPromptTemplate, "# Curated Agent") {
		t.Fatal("embedded system.md lost the prompt heading")
	}
	if strings.Contains(systemPromptTemplate, "Visible page context") {
		t.Fatal("request-scoped context must remain outside the static prompt asset")
	}
	if Version != "agent-system-v6" {
		t.Fatalf("version = %q", Version)
	}
}

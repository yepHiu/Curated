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
		"<source>",
		"confirm tokens",
		"save_movie_comment",
		"update_movie_display_overrides",
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
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("prompt missing %q:\n%s", want, got)
		}
	}
}

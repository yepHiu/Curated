package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"curated-backend/internal/agent/core"
	"curated-backend/internal/contracts"
)

var errMovieNotFound = errors.New("movie not found")

type stubQuery struct {
	movies contracts.MoviesPageDTO
}

func (s stubQuery) AgentLibraryOverview(context.Context) (map[string]any, error) {
	return map[string]any{"movieCount": s.movies.Total}, nil
}
func (s stubQuery) ListMovies(context.Context, contracts.ListMoviesRequest) (contracts.MoviesPageDTO, error) {
	return s.movies, nil
}
func (s stubQuery) GetMovieDetail(context.Context, string) (contracts.MovieDetailDTO, error) {
	if len(s.movies.Items) == 0 {
		return contracts.MovieDetailDTO{}, errMovieNotFound
	}
	return contracts.MovieDetailDTO{MovieListItemDTO: s.movies.Items[0], Summary: "hello"}, nil
}
func (stubQuery) GetMovieComment(context.Context, string) (contracts.MovieCommentDTO, error) {
	return contracts.MovieCommentDTO{}, nil
}
func (stubQuery) GetPlaybackProgress(context.Context, string) (float64, float64, string, bool, error) {
	return 0, 0, "", false, nil
}
func (stubQuery) ListActors(context.Context, contracts.ListActorsRequest) (contracts.ListActorsResponse, error) {
	return contracts.ListActorsResponse{}, nil
}
func (stubQuery) GetActorProfile(context.Context, string) (contracts.ActorProfileDTO, error) {
	return contracts.ActorProfileDTO{}, nil
}
func (stubQuery) GetPersonalInsightsOverview(context.Context, string, string) (contracts.PersonalInsightsOverviewDTO, error) {
	return contracts.PersonalInsightsOverviewDTO{}, nil
}
func (stubQuery) GetPersonalInsightsBreakdown(context.Context, string, string, string, int) (contracts.PersonalInsightsBreakdownDTO, error) {
	return contracts.PersonalInsightsBreakdownDTO{}, nil
}
func (stubQuery) ListWatchHistory(context.Context, int) ([]WatchHistoryItem, error) {
	return nil, nil
}
func (stubQuery) QueryCuratedFrames(context.Context, string, string, string, string, int, int) (contracts.CuratedFramesListDTO, error) {
	return contracts.CuratedFramesListDTO{}, nil
}
func (stubQuery) CountCuratedFrames(context.Context) (int, error) { return 0, nil }
func (stubQuery) GetTask(context.Context, string) (contracts.TaskDTO, bool) {
	return contracts.TaskDTO{}, false
}

func TestSearchMoviesProjectsCardsWithoutLocation(t *testing.T) {
	t.Parallel()
	reg := core.NewRegistry()
	query := stubQuery{movies: contracts.MoviesPageDTO{
		Total: 1,
		Items: []contracts.MovieListItemDTO{{
			ID:       "m1",
			Title:    "Hello",
			Code:     "ABC-123",
			Location: `D:\secret\file.mp4`,
			Actors:   []string{"A"},
		}},
	}}
	if err := RegisterQueryTools(reg, query); err != nil {
		t.Fatal(err)
	}
	gateway := core.NewGateway(reg, nil, nil, nil)
	result := gateway.Invoke(context.Background(), core.Call{
		Name:     "search_movies",
		Args:     json.RawMessage(`{"q":"ABC","limit":20}`),
		Sanitize: core.SanitizeSanitized,
	})
	if !result.OK {
		t.Fatalf("search_movies = %+v", result)
	}
	raw, _ := json.Marshal(result.Data)
	if strings.Contains(string(raw), "secret") || strings.Contains(string(raw), "location") {
		t.Fatalf("path leaked: %s", raw)
	}
	if !strings.Contains(string(raw), "ABC-123") {
		t.Fatalf("missing code: %s", raw)
	}
}

func TestRegisterQueryToolsNames(t *testing.T) {
	t.Parallel()
	reg := core.NewRegistry()
	if err := RegisterQueryTools(reg, stubQuery{}); err != nil {
		t.Fatal(err)
	}
	if _, ok := reg.Get("search_movies"); !ok {
		t.Fatal("search_movies missing")
	}
	if _, ok := reg.Get(resolveEntitiesName); !ok {
		t.Fatal("resolve_entities missing")
	}
	if _, ok := reg.Get("get_insights_overview"); !ok {
		t.Fatal("get_insights_overview missing")
	}
	if len(reg.List()) != 12 {
		t.Fatalf("tool count = %d, want 12", len(reg.List()))
	}
}

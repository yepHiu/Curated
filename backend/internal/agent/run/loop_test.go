package run

import (
	"context"
	"encoding/json"
	"strings"
	"sync/atomic"
	"testing"

	"curated-backend/internal/agent/core"
	"curated-backend/internal/agent/tools"
	"curated-backend/internal/contracts"
	"curated-backend/internal/llm"
)

func boolPtr(v bool) *bool { return &v }

func objectSchema(props map[string]core.Schema, required ...string) core.Schema {
	return core.Schema{Type: "object", Properties: props, Required: required, AdditionalProperties: boolPtr(false)}
}

func collectEvents(t *testing.T, loop *Loop, history []llm.ChatMessage) []contracts.AIChatSSEEvent {
	t.Helper()
	var events []contracts.AIChatSSEEvent
	err := loop.Run(context.Background(), "ses_test", "msg_test", history, nil, func(ev contracts.AIChatSSEEvent) {
		events = append(events, ev)
	})
	if err != nil {
		t.Fatal(err)
	}
	return events
}

func TestLoopRejectsForgedConfirmToken(t *testing.T) {
	t.Parallel()
	reg := core.NewRegistry()
	var writes atomic.Int32
	if err := reg.Register(core.ToolDefinition{
		Name:         "set_note",
		Description:  "write",
		ParamsSchema: objectSchema(map[string]core.Schema{"body": {Type: "string"}}, "body"),
		Permission:   core.PermissionWriteApply,
		Domain:       core.DomainUserWrite,
		Handler: func(ctx context.Context, call core.Call) (core.Result, error) {
			writes.Add(1)
			return core.Result{OK: true, Data: map[string]any{"saved": true}}, nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	gateway := core.NewGateway(reg, core.NewConfirmStore(), nil, func() core.Settings {
		return core.Settings{WriteEnabled: true, StepLimit: 8}
	})
	streamer := &llm.ScriptedStreamer{Turns: []llm.AssistantTurn{
		{ToolCalls: []llm.ToolCall{{
			ID: "call_1",
			Function: llm.ToolCallFunction{
				Name:      "set_note",
				Arguments: `{"body":"hi","confirmToken":"cfm_forged"}`,
			},
		}}},
		{Content: "I did not write anything."},
	}}
	loop := NewLoop(gateway, streamer, core.SanitizeFull, "zh-CN")
	events := collectEvents(t, loop, []llm.ChatMessage{{Role: "user", Content: "save a note"}})
	if writes.Load() != 0 {
		t.Fatalf("write handler ran %d times", writes.Load())
	}
	var sawToolResult bool
	for _, ev := range events {
		if ev.Type == "tool_call_result" {
			sawToolResult = true
			if ev.OK != nil && *ev.OK {
				t.Fatalf("forged token result should fail: %+v", ev)
			}
		}
	}
	if !sawToolResult {
		t.Fatalf("missing tool_call_result: %+v", events)
	}
}

func TestLoopStopsAtStepLimit(t *testing.T) {
	t.Parallel()
	reg := core.NewRegistry()
	var hits atomic.Int32
	if err := reg.Register(core.ToolDefinition{
		Name:         "get_ping",
		Description:  "ping",
		ParamsSchema: objectSchema(nil),
		Permission:   core.PermissionRead,
		Domain:       core.DomainQuery,
		Handler: func(ctx context.Context, call core.Call) (core.Result, error) {
			hits.Add(1)
			return core.Result{OK: true, Data: map[string]any{"pong": true}}, nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	gateway := core.NewGateway(reg, core.NewConfirmStore(), nil, func() core.Settings {
		return core.Settings{StepLimit: 3}
	})
	turns := make([]llm.AssistantTurn, 0, 8)
	for i := 0; i < 8; i++ {
		turns = append(turns, llm.AssistantTurn{ToolCalls: []llm.ToolCall{{
			ID:       "call",
			Function: llm.ToolCallFunction{Name: "get_ping", Arguments: `{}`},
		}}})
	}
	loop := NewLoop(gateway, &llm.ScriptedStreamer{Turns: turns}, core.SanitizeFull, "zh-CN")
	events := collectEvents(t, loop, []llm.ChatMessage{{Role: "user", Content: "ping forever"}})
	if hits.Load() != 3 {
		t.Fatalf("handler hits = %d, want 3", hits.Load())
	}
	var done bool
	var outcome *contracts.AIChatOutcomeDTO
	for _, ev := range events {
		if ev.Type == "message_done" {
			done = true
			outcome = ev.Outcome
		}
	}
	if !done {
		t.Fatalf("missing message_done")
	}
	if outcome == nil || outcome.Status != "partial" || outcome.ReasonCode != "tool_step_limit" {
		t.Fatalf("step limit outcome = %+v", outcome)
	}
}

func TestLoopEmitsEvidenceAndPartialOutcomeAfterReadFailure(t *testing.T) {
	t.Parallel()
	reg := core.NewRegistry()
	if err := reg.Register(core.ToolDefinition{
		Name: "get_broken_source", Description: "broken", ParamsSchema: objectSchema(nil), Permission: core.PermissionRead, Domain: core.DomainQuery,
		Handler: func(context.Context, core.Call) (core.Result, error) {
			return core.Result{OK: false, Error: &core.ToolError{Code: "AI_CHAT_FAILED", Message: "source unavailable"}}, nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	loop := NewLoop(core.NewGateway(reg, nil, nil, nil), &llm.ScriptedStreamer{Turns: []llm.AssistantTurn{
		{ToolCalls: []llm.ToolCall{{ID: "broken", Function: llm.ToolCallFunction{Name: "get_broken_source", Arguments: `{}`}}}},
		{Content: "Only confirmed facts are available."},
	}}, core.SanitizeFull, "en")
	events := collectEvents(t, loop, []llm.ChatMessage{{Role: "user", Content: "check source"}})
	var evidence *contracts.AIEvidenceDTO
	var outcome *contracts.AIChatOutcomeDTO
	for _, event := range events {
		if event.Type == "tool_call_result" {
			evidence = event.Evidence
		}
		if event.Type == "message_done" {
			outcome = event.Outcome
		}
	}
	if evidence == nil || !evidence.Failed || evidence.ErrorCode != "AI_CHAT_FAILED" || evidence.Source != "local" {
		t.Fatalf("evidence = %+v", evidence)
	}
	if outcome == nil || outcome.Status != "partial" || !outcome.Retryable {
		t.Fatalf("outcome = %+v", outcome)
	}
}

func TestLoopDoesNotTrustAmbiguousEntityCandidates(t *testing.T) {
	t.Parallel()
	reg := core.NewRegistry()
	if err := reg.Register(core.ToolDefinition{
		Name: "resolve_entities", Description: "resolve", ParamsSchema: objectSchema(nil), Permission: core.PermissionRead, Domain: core.DomainQuery,
		Handler: func(context.Context, core.Call) (core.Result, error) {
			return core.Result{OK: true, Data: map[string]any{"source": map[string]any{
				"query": "Same", "kind": "movie", "status": "ambiguous", "candidates": []map[string]any{
					{"kind": "movie", "movieId": "m1", "title": "Same", "code": "ABC-001"},
					{"kind": "movie", "movieId": "m2", "title": "Same", "code": "ABC-002"},
				},
			}}}, nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	gateway := core.NewGateway(reg, nil, nil, nil)
	if err := tools.RegisterPresentTools(reg, gateway.MovieRefs()); err != nil {
		t.Fatal(err)
	}
	loop := NewLoop(gateway, &llm.ScriptedStreamer{Turns: []llm.AssistantTurn{
		{ToolCalls: []llm.ToolCall{{ID: "resolve", Function: llm.ToolCallFunction{Name: "resolve_entities", Arguments: `{}`}}}},
		{ToolCalls: []llm.ToolCall{{ID: "present", Function: llm.ToolCallFunction{Name: core.PresentMoviesName, Arguments: `{"items":[{"movieId":"m1","reason":"guess"}]}`}}}},
		{Content: "Please choose."},
	}}, core.SanitizeFull, "en")
	events := collectEvents(t, loop, []llm.ChatMessage{{Role: "user", Content: "show Same"}})
	var resolution *contracts.AIEntityResolutionDTO
	var presentOK *bool
	var outcome *contracts.AIChatOutcomeDTO
	for _, event := range events {
		if event.Type == "tool_call_result" && event.Name == "resolve_entities" {
			resolution = event.Resolution
		}
		if event.Type == "tool_call_result" && event.Name == core.PresentMoviesName {
			presentOK = event.OK
		}
		if event.Type == "message_done" {
			outcome = event.Outcome
		}
	}
	if resolution == nil || resolution.Status != "ambiguous" {
		t.Fatalf("resolution = %+v", resolution)
	}
	if presentOK == nil || *presentOK {
		t.Fatalf("ambiguous candidate entered present_movies: %+v", events)
	}
	if outcome == nil || outcome.Status != "needs_input" {
		t.Fatalf("outcome = %+v", outcome)
	}
}

func TestLoopPresentMoviesEmitsCardsAfterSearch(t *testing.T) {
	t.Parallel()
	reg := core.NewRegistry()
	query := stubQueryForLoop{movie: contracts.MovieListItemDTO{
		ID: "m1", Title: "Hello", Code: "ABC-123", CoverURL: "/api/cover", Actors: []string{"A"},
	}}
	if err := tools.RegisterQueryTools(reg, query); err != nil {
		t.Fatal(err)
	}
	gateway := core.NewGateway(reg, nil, nil, nil)
	if err := tools.RegisterPresentTools(reg, gateway.MovieRefs()); err != nil {
		t.Fatal(err)
	}
	streamer := &llm.ScriptedStreamer{Turns: []llm.AssistantTurn{
		{ToolCalls: []llm.ToolCall{{
			ID:       "call_search",
			Function: llm.ToolCallFunction{Name: "search_movies", Arguments: `{"q":"ABC","limit":5}`},
		}}},
		{ToolCalls: []llm.ToolCall{{
			ID: "call_present",
			Function: llm.ToolCallFunction{
				Name:      core.PresentMoviesName,
				Arguments: `{"items":[{"movieId":"m1","reason":"轻松短片"}]}`,
			},
		}}},
		{Content: "今晚看这部。"},
	}}
	loop := NewLoop(gateway, streamer, core.SanitizeFull, "zh-CN")
	events := collectEvents(t, loop, []llm.ChatMessage{{Role: "user", Content: "今晚看什么"}})
	var cards []contracts.AIAgentMovieCardDTO
	for _, ev := range events {
		if ev.Type == "movie_cards" {
			cards = ev.Movies
		}
	}
	if len(cards) != 1 || cards[0].MovieID != "m1" || cards[0].Reason != "轻松短片" {
		t.Fatalf("movie_cards = %+v events=%+v", cards, events)
	}
}

func TestLoopSeedsPageActorForProviderSearch(t *testing.T) {
	t.Parallel()
	reg := core.NewRegistry()
	query := stubQueryForLoop{movie: contracts.MovieListItemDTO{ID: "m1", Title: "Hello", Code: "ABC-123", Actors: []string{"Alice"}}}
	if err := tools.RegisterQueryTools(reg, query); err != nil {
		t.Fatal(err)
	}
	gateway := core.NewGateway(reg, nil, nil, nil)
	if err := tools.RegisterProviderTools(reg, query, stubLoopProvider{
		hits: []tools.ProviderTitleHit{{Code: "ABC-124", Title: "Other", Provider: "javbus"}},
	}, nil, gateway.MovieRefs(), gateway.ActorRefs(), gateway.SourceURLs()); err != nil {
		t.Fatal(err)
	}
	streamer := &llm.ScriptedStreamer{Turns: []llm.AssistantTurn{
		{ToolCalls: []llm.ToolCall{{
			ID:       "call_provider",
			Function: llm.ToolCallFunction{Name: core.SearchProviderTitlesName, Arguments: `{"actorName":"Alice"}`},
		}}},
		{Content: "源站还有其他作品。"},
	}}
	loop := NewLoop(gateway, streamer, core.SanitizeFull, "zh-CN")
	var events []contracts.AIChatSSEEvent
	err := loop.Run(context.Background(), "ses_test", "msg_test", []llm.ChatMessage{{Role: "user", Content: "还拍过什么"}}, &contracts.AIChatContext{ActorName: "Alice"}, func(ev contracts.AIChatSSEEvent) {
		events = append(events, ev)
	})
	if err != nil {
		t.Fatal(err)
	}
	var okResult bool
	var providerRows []contracts.AIAgentProviderTitleDTO
	for _, ev := range events {
		if ev.Type == "tool_call_result" && ev.Name == core.SearchProviderTitlesName && ev.OK != nil && *ev.OK {
			okResult = true
			providerRows = ev.ProviderRows
		}
	}
	if !okResult {
		t.Fatalf("page actor should anchor provider search: %+v", events)
	}
	if len(providerRows) != 1 || providerRows[0].InLibrary || providerRows[0].MovieID != "" || providerRows[0].Code != "ABC-124" {
		t.Fatalf("provider rows = %+v", providerRows)
	}
}

func TestLoopSeedsSelectedContextEntities(t *testing.T) {
	t.Parallel()
	reg := core.NewRegistry()
	query := stubQueryForLoop{movie: contracts.MovieListItemDTO{ID: "m1", Title: "Hello", Code: "ABC-123", Actors: []string{"Alice"}}}
	if err := tools.RegisterQueryTools(reg, query); err != nil {
		t.Fatal(err)
	}
	gateway := core.NewGateway(reg, nil, nil, nil)
	if err := tools.RegisterPresentTools(reg, gateway.MovieRefs()); err != nil {
		t.Fatal(err)
	}
	streamer := &llm.ScriptedStreamer{Turns: []llm.AssistantTurn{
		{ToolCalls: []llm.ToolCall{{
			ID:       "call_present",
			Function: llm.ToolCallFunction{Name: core.PresentMoviesName, Arguments: `{"items":[{"movieId":"m1","reason":"selected"}]}`},
		}}},
		{Content: "已展示选中的影片。"},
	}}
	loop := NewLoop(gateway, streamer, core.SanitizeFull, "zh-CN")
	var events []contracts.AIChatSSEEvent
	err := loop.Run(context.Background(), "ses_selected", "msg_selected", []llm.ChatMessage{{Role: "user", Content: "展示这部"}}, &contracts.AIChatContext{
		ContextVersion:   1,
		SelectedMovieIDs: []string{"m1"},
	}, func(event contracts.AIChatSSEEvent) { events = append(events, event) })
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if event.Type == "movie_cards" && len(event.Movies) == 1 && event.Movies[0].MovieID == "m1" {
			return
		}
	}
	t.Fatalf("selected local movie was not seeded: %+v", events)
}

type stubLoopProvider struct {
	hits []tools.ProviderTitleHit
}

func (s stubLoopProvider) SearchProviderTitles(context.Context, string, int) ([]tools.ProviderTitleHit, error) {
	return s.hits, nil
}

func (stubLoopProvider) FindLibraryMoviesByCodes(context.Context, []string) (map[string]contracts.MovieListItemDTO, error) {
	return map[string]contracts.MovieListItemDTO{}, nil
}

func TestLoopEmitsConfirmRequiredAndStops(t *testing.T) {
	t.Parallel()
	reg := core.NewRegistry()
	write := &commentWriteStub{exists: true, body: "old"}
	if err := tools.RegisterWriteTools(reg, write); err != nil {
		t.Fatal(err)
	}
	gateway := core.NewGateway(reg, core.NewConfirmStore(), nil, func() core.Settings { return core.Settings{} })
	streamer := &llm.ScriptedStreamer{Turns: []llm.AssistantTurn{
		{ToolCalls: []llm.ToolCall{{
			ID: "call_save",
			Function: llm.ToolCallFunction{
				Name:      core.SaveMovieCommentName,
				Arguments: `{"movieId":"m1","body":"polished"}`,
			},
		}}},
		{Content: "should not run"},
	}}
	loop := NewLoop(gateway, streamer, core.SanitizeFull, "zh-CN")
	events := collectEvents(t, loop, []llm.ChatMessage{{Role: "user", Content: "润色笔记"}})
	var sawConfirm bool
	for _, ev := range events {
		if ev.Type == "confirm_required" {
			sawConfirm = true
			if ev.ConfirmToken == "" || ev.Name != core.SaveMovieCommentName {
				t.Fatalf("confirm event = %+v", ev)
			}
		}
		if ev.Type == "text_delta" && strings.Contains(ev.Delta, "should not run") {
			t.Fatalf("loop continued after confirm: %+v", events)
		}
	}
	if !sawConfirm {
		t.Fatalf("missing confirm_required: %+v", events)
	}
	if write.writes != 0 {
		t.Fatal("preview wrote")
	}
	if outcome := events[len(events)-1].Outcome; outcome == nil || outcome.Status != "needs_confirmation" || outcome.ReasonCode != "confirmation_required" {
		t.Fatalf("preview outcome = %+v", outcome)
	}
}

type commentWriteStub struct {
	exists bool
	body   string
	writes int
}

func (s *commentWriteStub) MovieExists(context.Context, string) (bool, error) { return s.exists, nil }
func (s *commentWriteStub) GetMovieComment(context.Context, string) (contracts.MovieCommentDTO, error) {
	return contracts.MovieCommentDTO{Body: s.body}, nil
}
func (s *commentWriteStub) UpsertMovieComment(_ context.Context, _, body string, expected ...string) (contracts.MovieCommentDTO, error) {
	s.writes++
	s.body = body
	return contracts.MovieCommentDTO{Body: body, UpdatedAt: "t"}, nil
}
func (s *commentWriteStub) GetMovieDetail(context.Context, string) (contracts.MovieDetailDTO, error) {
	return contracts.MovieDetailDTO{}, nil
}
func (s *commentWriteStub) PatchMovieDisplayOverrides(context.Context, string, contracts.PatchMovieInput) (contracts.MovieDetailDTO, error) {
	return contracts.MovieDetailDTO{}, nil
}
func (s *commentWriteStub) CreateSavedView(context.Context, string, contracts.SavedViewFiltersV1) (contracts.SavedViewDTO, error) {
	return contracts.SavedViewDTO{}, nil
}

type stubQueryForLoop struct {
	movie contracts.MovieListItemDTO
}

func (s stubQueryForLoop) AgentLibraryOverview(context.Context) (map[string]any, error) {
	return map[string]any{}, nil
}
func (s stubQueryForLoop) ListMovies(context.Context, contracts.ListMoviesRequest) (contracts.MoviesPageDTO, error) {
	return contracts.MoviesPageDTO{Total: 1, Items: []contracts.MovieListItemDTO{s.movie}}, nil
}
func (s stubQueryForLoop) GetMovieDetail(context.Context, string) (contracts.MovieDetailDTO, error) {
	return contracts.MovieDetailDTO{MovieListItemDTO: s.movie}, nil
}
func (stubQueryForLoop) GetMovieComment(context.Context, string) (contracts.MovieCommentDTO, error) {
	return contracts.MovieCommentDTO{}, nil
}
func (stubQueryForLoop) GetPlaybackProgress(context.Context, string) (float64, float64, string, bool, error) {
	return 0, 0, "", false, nil
}
func (stubQueryForLoop) ListActors(context.Context, contracts.ListActorsRequest) (contracts.ListActorsResponse, error) {
	return contracts.ListActorsResponse{}, nil
}
func (stubQueryForLoop) GetActorProfile(context.Context, string) (contracts.ActorProfileDTO, error) {
	return contracts.ActorProfileDTO{}, nil
}
func (stubQueryForLoop) GetPersonalInsightsOverview(context.Context, string, string) (contracts.PersonalInsightsOverviewDTO, error) {
	return contracts.PersonalInsightsOverviewDTO{}, nil
}
func (stubQueryForLoop) GetPersonalInsightsBreakdown(context.Context, string, string, string, int) (contracts.PersonalInsightsBreakdownDTO, error) {
	return contracts.PersonalInsightsBreakdownDTO{}, nil
}
func (stubQueryForLoop) ListWatchHistory(context.Context, int) ([]tools.WatchHistoryItem, error) {
	return nil, nil
}
func (stubQueryForLoop) QueryCuratedFrames(context.Context, string, string, string, string, int, int) (contracts.CuratedFramesListDTO, error) {
	return contracts.CuratedFramesListDTO{}, nil
}
func (stubQueryForLoop) CountCuratedFrames(context.Context) (int, error) { return 0, nil }
func (stubQueryForLoop) GetTask(context.Context, string) (contracts.TaskDTO, bool) {
	return contracts.TaskDTO{}, false
}

func TestExtractConfirmTokenStripsField(t *testing.T) {
	t.Parallel()
	token, cleaned := extractConfirmToken(`{"body":"hi","confirmToken":"cfm_abc"}`)
	if token != "cfm_abc" {
		t.Fatalf("token = %q", token)
	}
	var obj map[string]any
	if err := json.Unmarshal(cleaned, &obj); err != nil {
		t.Fatal(err)
	}
	if _, ok := obj["confirmToken"]; ok {
		t.Fatalf("confirmToken leaked: %+v", obj)
	}
	if obj["body"] != "hi" {
		t.Fatalf("body = %v", obj["body"])
	}
}

type thinkingStreamer struct {
	content string
}

func (s thinkingStreamer) StreamTurn(_ context.Context, req llm.TurnRequest, onDelta func(string)) (llm.AssistantTurn, error) {
	if req.OnThinking != nil {
		req.OnThinking("先查未看")
	}
	if onDelta != nil && s.content != "" {
		onDelta(s.content)
	}
	return llm.AssistantTurn{Content: s.content}, nil
}

func TestLoopEmitsThinkingDelta(t *testing.T) {
	t.Parallel()
	reg := core.NewRegistry()
	gateway := core.NewGateway(reg, nil, nil, nil)
	loop := NewLoop(gateway, thinkingStreamer{content: "今晚看这部"}, core.SanitizeFull, "zh-CN")
	events := collectEvents(t, loop, []llm.ChatMessage{{Role: "user", Content: "今晚看什么"}})
	var thinking string
	for _, ev := range events {
		if ev.Type == "thinking_delta" {
			thinking += ev.Delta
		}
	}
	if thinking != "先查未看" {
		t.Fatalf("thinking_delta = %q events=%+v", thinking, events)
	}
}

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"

	"curated-backend/internal/contracts"
	"curated-backend/internal/llm"
)

type stubAIChatProvider struct {
	request  contracts.AIChatRequest
	events   []contracts.AIChatSSEEvent
	failWith error
	sessions []contracts.AIChatSessionDTO
}

func (s *stubAIChatProvider) StreamAIChat(_ context.Context, req contracts.AIChatRequest, emit func(contracts.AIChatSSEEvent)) error {
	s.request = req
	if s.failWith != nil {
		return s.failWith
	}
	events := s.events
	if len(events) == 0 {
		events = []contracts.AIChatSSEEvent{
			{Type: "message_start", SessionID: "ses_test", MessageID: "msg_test", Seq: 1},
			{Type: "text_delta", SessionID: "ses_test", MessageID: "msg_test", Seq: 2, Delta: "你好"},
			{Type: "text_delta", SessionID: "ses_test", MessageID: "msg_test", Seq: 3, Delta: "，世界"},
			{Type: "message_done", SessionID: "ses_test", MessageID: "msg_test", Seq: 4},
		}
	}
	for _, ev := range events {
		if emit != nil {
			emit(ev)
		}
	}
	return nil
}

func (s *stubAIChatProvider) TestAIProvider(_ context.Context, override *contracts.AIProviderSettingsDTO) contracts.AIProviderTestResponse {
	if override != nil && override.BaseURL == "" {
		return contracts.AIProviderTestResponse{OK: false, Message: "missing baseUrl"}
	}
	return contracts.AIProviderTestResponse{OK: true, LatencyMs: 12}
}

func (s *stubAIChatProvider) ListAIChatSessions(_ context.Context) (contracts.AIChatSessionListDTO, error) {
	items := s.sessions
	if items == nil {
		items = []contracts.AIChatSessionDTO{}
	}
	return contracts.AIChatSessionListDTO{Items: items}, nil
}

func (s *stubAIChatProvider) CreateAIChatSession(_ context.Context, title string) (contracts.AIChatSessionDTO, error) {
	dto := contracts.AIChatSessionDTO{ID: "ses_new", Title: title, CreatedAt: "t", UpdatedAt: "t"}
	s.sessions = append(s.sessions, dto)
	return dto, nil
}

func (s *stubAIChatProvider) GetAIChatSession(_ context.Context, id string, cursor ...string) (contracts.AIChatSessionDetailDTO, error) {
	return contracts.AIChatSessionDetailDTO{
		AIChatSessionDTO: contracts.AIChatSessionDTO{ID: id, Title: "hi"},
		Messages:         []contracts.AIChatStoredMessageDTO{},
	}, nil
}

func (s *stubAIChatProvider) DeleteAIChatSession(_ context.Context, id string) error {
	if id == "missing" {
		return fmt.Errorf("ai chat session not found")
	}
	return nil
}

func (s *stubAIChatProvider) RunAIAction(_ context.Context, name string, req contracts.AIActionRequest) (contracts.AIActionPreviewDTO, error) {
	switch name {
	case "polish_comment", "translate_summary", "translate_title", "insights_narrative":
	default:
		return contracts.AIActionPreviewDTO{}, fmt.Errorf("unknown action")
	}
	return contracts.AIActionPreviewDTO{
		Action:       name,
		Name:         "save_movie_comment",
		SessionID:    "act_stub",
		OriginalText: req.Body,
		ProposedText: "polished",
		ConfirmToken: "cfm_stub",
		Arguments:    json.RawMessage(`{"movieId":"m1","body":"polished"}`),
		Changes:      []contracts.AIConfirmChangeDTO{{Path: "comment.body", Before: req.Body, After: "polished"}},
	}, nil
}

func (s *stubAIChatProvider) ApplyAITool(_ context.Context, req contracts.AIToolApplyRequest) (contracts.AIToolApplyDTO, error) {
	if req.ConfirmToken == "" {
		return contracts.AIToolApplyDTO{}, fmt.Errorf("confirm token is required")
	}
	return contracts.AIToolApplyDTO{OK: true, Name: req.Name, Data: map[string]any{"body": "saved"}}, nil
}

type stubAISettingsController struct {
	current contracts.AIProviderSettingsDTO
	patches []contracts.PatchAIProviderSettings
}

func (s *stubAISettingsController) AIProviderSettings() contracts.AIProviderSettingsDTO {
	return s.current
}

func (s *stubAISettingsController) SetAIProviderSettingsPatch(p contracts.PatchAIProviderSettings) error {
	s.patches = append(s.patches, p)
	if p.BaseURL != nil {
		s.current.BaseURL = *p.BaseURL
	}
	if p.Model != nil {
		s.current.Model = *p.Model
	}
	if p.APIKey != nil {
		s.current.APIKey = *p.APIKey
	}
	return nil
}

func newAIHandler(t *testing.T, chat *stubAIChatProvider, settings *stubAISettingsController) *httptest.Server {
	t.Helper()
	deps := Deps{Logger: zap.NewNop()}
	if chat != nil {
		deps.AIChatProvider = chat
	}
	if settings != nil {
		deps.AISettingsCtl = settings
	}
	server := httptest.NewServer(NewHandler(deps).Routes())
	t.Cleanup(server.Close)
	return server
}

func TestAIChatStreamsSSEEvents(t *testing.T) {
	t.Parallel()
	chat := &stubAIChatProvider{}
	server := newAIHandler(t, chat, nil)

	resp, err := http.Post(server.URL+"/api/ai/chat", "application/json",
		strings.NewReader(`{"messages":[{"role":"system","content":"You are Curated."},{"role":"user","content":"打个招呼"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Type"); !strings.Contains(got, "text/event-stream") {
		t.Fatalf("Content-Type = %q", got)
	}
	body, _ := io.ReadAll(resp.Body)
	text := string(body)
	for _, want := range []string{
		"event: message_start",
		"event: text_delta",
		`"delta":"你好"`,
		`"delta":"，世界"`,
		"event: message_done",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("SSE body missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "event: error") {
		t.Fatalf("unexpected error event:\n%s", text)
	}
	if len(chat.request.Messages) != 2 || chat.request.Messages[0].Role != "system" || chat.request.Messages[1].Content != "打个招呼" {
		t.Fatalf("provider messages = %+v", chat.request.Messages)
	}
}

func TestAIChatProviderUnconfiguredEmitsUnavailableError(t *testing.T) {
	t.Parallel()
	chat := &stubAIChatProvider{failWith: fmt.Errorf("%w: provider baseUrl and model are required", llm.ErrInvalidConfig)}
	server := newAIHandler(t, chat, nil)

	resp, err := http.Post(server.URL+"/api/ai/chat", "application/json",
		strings.NewReader(`{"messages":[{"role":"user","content":"hi"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	text := string(body)
	if !strings.Contains(text, "event: error") || !strings.Contains(text, contracts.ErrorCodeAIProviderUnavailable) {
		t.Fatalf("missing %s error event:\n%s", contracts.ErrorCodeAIProviderUnavailable, text)
	}
	if strings.Contains(text, "event: message_done") {
		t.Fatalf("unexpected message_done:\n%s", text)
	}
}

func TestAIChatRejectsInvalidBodies(t *testing.T) {
	t.Parallel()
	server := newAIHandler(t, &stubAIChatProvider{}, nil)

	cases := []struct {
		name string
		body string
	}{
		{"empty messages", `{"messages":[]}`},
		{"no user turn", `{"messages":[{"role":"system","content":"s"}]}`},
		{"bad role", `{"messages":[{"role":"tool","content":"x"}]}`},
		{"empty content", `{"messages":[{"role":"user","content":" "}]}`},
		{"broken json", `{"messages":`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resp, err := http.Post(server.URL+"/api/ai/chat", "application/json", strings.NewReader(tc.body))
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", resp.StatusCode)
			}
		})
	}
}

func TestAIChatNormalizesBoundedContextV1(t *testing.T) {
	t.Parallel()
	chat := &stubAIChatProvider{}
	server := newAIHandler(t, chat, nil)

	resp, err := http.Post(server.URL+"/api/ai/chat", "application/json", strings.NewReader(`{
  "messages":[{"role":"user","content":"find one"}],
  "context":{
    "contextVersion":1,
    "route":" library ",
    "selectedMovieIds":["m1","m1","m2"],
    "selectedActors":[" Ada ","Ada"],
    "activeFilters":{"query":" hello ","playState":"unwatched","runtime":"short"}
  }
}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if chat.request.Context == nil {
		t.Fatal("context was not passed to provider")
	}
	got := chat.request.Context
	if got.Route != "library" || len(got.SelectedMovieIDs) != 2 || got.SelectedMovieIDs[0] != "m1" || len(got.SelectedActors) != 1 || got.SelectedActors[0] != "Ada" {
		t.Fatalf("normalized context = %+v", got)
	}
	if got.ActiveFilters == nil || got.ActiveFilters.Query != "hello" || got.ActiveFilters.PlayState != "unwatched" {
		t.Fatalf("filters = %+v", got.ActiveFilters)
	}
}

func TestAIChatRejectsUnsupportedOrUnversionedContextV1(t *testing.T) {
	t.Parallel()
	server := newAIHandler(t, &stubAIChatProvider{}, nil)
	cases := []string{
		`{"messages":[{"role":"user","content":"hi"}],"context":{"contextVersion":2}}`,
		`{"messages":[{"role":"user","content":"hi"}],"context":{"selectedMovieIds":["m1"]}}`,
		`{"messages":[{"role":"user","content":"hi"}],"context":{"contextVersion":1,"activeFilters":{"playState":"invented"}}}`,
	}
	for _, body := range cases {
		resp, err := http.Post(server.URL+"/api/ai/chat", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("status = %d for %s", resp.StatusCode, body)
		}
	}
}

func TestAIProviderTestEndpoint(t *testing.T) {
	t.Parallel()
	server := newAIHandler(t, &stubAIChatProvider{}, nil)

	resp, err := http.Post(server.URL+"/api/ai/provider/test", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var got contracts.AIProviderTestResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if !got.OK {
		t.Fatalf("persisted-config test = %+v, want ok", got)
	}

	resp2, err := http.Post(server.URL+"/api/ai/provider/test", "application/json",
		strings.NewReader(`{"provider":{"baseUrl":"","model":"m"}}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	var got2 contracts.AIProviderTestResponse
	if err := json.NewDecoder(resp2.Body).Decode(&got2); err != nil {
		t.Fatal(err)
	}
	if got2.OK || got2.Message == "" {
		t.Fatalf("draft test = %+v, want ok=false with message", got2)
	}
}

func TestAIChatSessionCRUD(t *testing.T) {
	t.Parallel()
	chat := &stubAIChatProvider{}
	server := newAIHandler(t, chat, nil)

	create, err := http.Post(server.URL+"/api/ai/sessions", "application/json", strings.NewReader(`{"title":"今晚看什么"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer create.Body.Close()
	if create.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d", create.StatusCode)
	}

	list, err := http.Get(server.URL + "/api/ai/sessions")
	if err != nil {
		t.Fatal(err)
	}
	defer list.Body.Close()
	var listed contracts.AIChatSessionListDTO
	if err := json.NewDecoder(list.Body).Decode(&listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Items) != 1 || listed.Items[0].Title != "今晚看什么" {
		t.Fatalf("list = %+v", listed)
	}

	del, err := http.NewRequest(http.MethodDelete, server.URL+"/api/ai/sessions/ses_new", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := server.Client().Do(del)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d", resp.StatusCode)
	}
}

func TestSettingsExposeAndPatchAIProvider(t *testing.T) {
	t.Parallel()
	settings := &stubAISettingsController{current: contracts.AIProviderSettingsDTO{Kind: "openai-compatible"}}
	server := newSettingsCuratedExportFormatTestServer(t, Deps{
		OrganizeLibraryCtl:  stubOrganizeCtl{},
		AutoLibraryWatchCtl: stubAutoWatchCtl{},
		MetadataScrapeCtl:   stubMetadataCtl{},
		AISettingsCtl:       settings,
	})

	resp, err := http.Get(server.URL + "/api/settings")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var dto contracts.SettingsDTO
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		t.Fatal(err)
	}
	if dto.AIProvider.Kind != "openai-compatible" {
		t.Fatalf("default AIProvider.Kind = %q", dto.AIProvider.Kind)
	}

	patch, err := http.NewRequest(http.MethodPatch, server.URL+"/api/settings",
		strings.NewReader(`{"aiProvider":{"baseUrl":"http://127.0.0.1:11434/v1","model":"qwen3"}}`))
	if err != nil {
		t.Fatal(err)
	}
	patchResp, err := server.Client().Do(patch)
	if err != nil {
		t.Fatal(err)
	}
	defer patchResp.Body.Close()
	if patchResp.StatusCode != http.StatusOK {
		t.Fatalf("patch status = %d", patchResp.StatusCode)
	}
	if len(settings.patches) != 1 || settings.patches[0].BaseURL == nil || *settings.patches[0].BaseURL != "http://127.0.0.1:11434/v1" {
		t.Fatalf("patches = %+v", settings.patches)
	}
	if settings.current.Model != "qwen3" {
		t.Fatalf("updated model = %q", settings.current.Model)
	}
}

func TestAIActionAndConfirm(t *testing.T) {
	t.Parallel()
	server := newAIHandler(t, &stubAIChatProvider{}, nil)
	defer server.Close()

	resp, err := server.Client().Post(server.URL+"/api/ai/actions/polish_comment", "application/json",
		strings.NewReader(`{"movieId":"m1","body":"slow"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("action status = %d", resp.StatusCode)
	}
	var preview contracts.AIActionPreviewDTO
	if err := json.NewDecoder(resp.Body).Decode(&preview); err != nil {
		t.Fatal(err)
	}
	if preview.ConfirmToken != "cfm_stub" || preview.Name != "save_movie_comment" {
		t.Fatalf("preview = %+v", preview)
	}

	apply, err := server.Client().Post(server.URL+"/api/ai/confirm", "application/json",
		strings.NewReader(`{"sessionId":"act_stub","name":"save_movie_comment","confirmToken":"cfm_stub","arguments":{"movieId":"m1","body":"polished"}}`))
	if err != nil {
		t.Fatal(err)
	}
	defer apply.Body.Close()
	if apply.StatusCode != http.StatusOK {
		t.Fatalf("confirm status = %d", apply.StatusCode)
	}
}

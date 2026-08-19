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
	messages []contracts.AIChatMessage
	deltas   []string
	failWith error
}

func (s *stubAIChatProvider) StreamAIChat(_ context.Context, messages []contracts.AIChatMessage, onDelta func(string)) error {
	s.messages = messages
	if s.failWith != nil {
		return s.failWith
	}
	for _, d := range s.deltas {
		if onDelta != nil {
			onDelta(d)
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
	chat := &stubAIChatProvider{deltas: []string{"你好", "，世界"}}
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
	if len(chat.messages) != 2 || chat.messages[0].Role != "system" || chat.messages[1].Content != "打个招呼" {
		t.Fatalf("provider messages = %+v", chat.messages)
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
	if !strings.Contains(text, "event: message_start") {
		t.Fatalf("missing message_start:\n%s", text)
	}
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

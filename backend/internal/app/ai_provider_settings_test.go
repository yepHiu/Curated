package app

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/llm"
)

func TestAIProviderSettingsPatchPersistsAndReloads(t *testing.T) {
	t.Parallel()
	settingsPath := filepath.Join(t.TempDir(), "library-config.cfg")
	a := &App{
		cfg:                 config.Default(),
		logger:              zap.NewNop(),
		librarySettingsPath: settingsPath,
	}
	window := 204800

	if err := a.SetAIProviderSettingsPatch(contracts.PatchAIProviderSettings{
		ContextWindow: &window,
		BaseURL:       stringPtrForAI("  http://127.0.0.1:11434/v1  "),
		APIKey:        stringPtrForAI("secret"),
		Model:         stringPtrForAI("  qwen3  "),
	}); err != nil {
		t.Fatal(err)
	}

	got := a.AIProviderSettings()
	if got.ContextWindow != window {
		t.Fatalf("context window = %d", got.ContextWindow)
	}
	if got.Kind != config.AIProviderKindOpenAICompatible {
		t.Fatalf("Kind = %q, want %q", got.Kind, config.AIProviderKindOpenAICompatible)
	}
	if got.BaseURL != "http://127.0.0.1:11434/v1" {
		t.Fatalf("BaseURL = %q", got.BaseURL)
	}
	if got.Model != "qwen3" {
		t.Fatalf("Model = %q", got.Model)
	}
	if got.APIKey != "secret" {
		t.Fatalf("APIKey = %q", got.APIKey)
	}

	reloaded := config.Default()
	if err := config.MergeLibrarySettingsFile(&reloaded, settingsPath); err != nil {
		t.Fatal(err)
	}
	if reloaded.AIProvider.BaseURL != "http://127.0.0.1:11434/v1" || reloaded.AIProvider.Model != "qwen3" {
		t.Fatalf("reloaded AIProvider = %+v", reloaded.AIProvider)
	}
	if reloaded.AIProvider.ContextWindow != window {
		t.Fatalf("persisted context = %d", reloaded.AIProvider.ContextWindow)
	}
	if err := a.SetAIProviderSettingsPatch(contracts.PatchAIProviderSettings{Model: stringPtrForAI("other-model")}); err != nil {
		t.Fatal(err)
	}
	if a.AIProviderSettings().ContextWindow != window {
		t.Fatal("partial update reset context window")
	}

	// Clearing: explicit empty baseUrl/model clears the config.
	if err := a.SetAIProviderSettingsPatch(contracts.PatchAIProviderSettings{
		BaseURL: stringPtrForAI(""),
		APIKey:  stringPtrForAI(""),
		Model:   stringPtrForAI(""),
	}); err != nil {
		t.Fatal(err)
	}
	if got := a.AIProviderSettings(); got.BaseURL != "" || got.Model != "" || got.APIKey != "" {
		t.Fatalf("cleared AIProvider = %+v", got)
	}
}

func TestAIProviderSettingsPatchRejectsInvalidValues(t *testing.T) {
	t.Parallel()
	a := &App{
		cfg:                 config.Default(),
		librarySettingsPath: filepath.Join(t.TempDir(), "library-config.cfg"),
	}
	if a.AIProviderSettings().ContextWindow != config.DefaultAIContextWindow {
		t.Fatal("missing legacy default")
	}
	for _, window := range []int{-1, 0, 32767, 2097153} {
		if err := a.SetAIProviderSettingsPatch(contracts.PatchAIProviderSettings{ContextWindow: &window}); err == nil {
			t.Fatalf("accepted context %d", window)
		}
		if a.AIProviderSettings().ContextWindow != config.DefaultAIContextWindow {
			t.Fatal("invalid patch changed configuration")
		}
	}
	if err := a.SetAIProviderSettingsPatch(contracts.PatchAIProviderSettings{
		BaseURL: stringPtrForAI("not-a-url"),
		Model:   stringPtrForAI("m"),
	}); err == nil || !strings.Contains(err.Error(), "http(s)") {
		t.Fatalf("bad baseUrl error = %v", err)
	}
	if err := a.SetAIProviderSettingsPatch(contracts.PatchAIProviderSettings{
		Kind: stringPtrForAI("mcp-magic"),
	}); err == nil || !strings.Contains(err.Error(), "kind") {
		t.Fatalf("bad kind error = %v", err)
	}
	if err := a.SetAIProviderSettingsPatch(contracts.PatchAIProviderSettings{
		Model: stringPtrForAI("m"),
	}); err == nil || !strings.Contains(err.Error(), "baseUrl is required") {
		t.Fatalf("half-configured error = %v", err)
	}
}

func TestStreamAIChatNotConfiguredWrapsInvalidConfig(t *testing.T) {
	t.Parallel()
	a := &App{
		cfg:                 config.Default(),
		logger:              zap.NewNop(),
		librarySettingsPath: filepath.Join(t.TempDir(), "library-config.cfg"),
	}
	err := a.StreamAIChat(context.Background(), contracts.AIChatRequest{
		Messages: []contracts.AIChatMessage{{Role: "user", Content: "hi"}},
	}, nil)
	if err == nil {
		t.Fatal("StreamAIChat() = nil, want not-configured error")
	}
	if !errors.Is(err, llm.ErrInvalidConfig) {
		t.Fatalf("StreamAIChat() error = %v, want wrapped llm.ErrInvalidConfig", err)
	}
}

func TestTestAIProviderAgainstFakeEndpoint(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/chat/completions" {
			var request struct {
				MaxTokens int `json:"max_tokens"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Error(err)
				w.WriteHeader(400)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			if request.MaxTokens < 128 {
				_, _ = w.Write([]byte(`{"choices":[{"finish_reason":"length","message":{"reasoning_content":"Checking connectivity","content":""}}]}`))
				return
			}
			if request.MaxTokens > 1024 {
				t.Error("connectivity probe is not bounded")
			}
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"pong"}}]}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	a := &App{
		cfg:                 config.Default(),
		logger:              zap.NewNop(),
		librarySettingsPath: filepath.Join(t.TempDir(), "library-config.cfg"),
	}
	resp := a.TestAIProvider(context.Background(), &contracts.AIProviderSettingsDTO{
		BaseURL: server.URL,
		Model:   "test-model",
	})
	if !resp.OK {
		t.Fatalf("TestAIProvider() = %+v, want ok", resp)
	}

	bad := a.TestAIProvider(context.Background(), &contracts.AIProviderSettingsDTO{
		BaseURL: server.URL + "/missing",
		Model:   "test-model",
	})
	if bad.OK || bad.Message == "" {
		t.Fatalf("TestAIProvider(missing) = %+v, want ok=false with message", bad)
	}
}

func stringPtrForAI(v string) *string { return &v }

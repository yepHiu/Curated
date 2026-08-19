package app

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/llm"
	"curated-backend/internal/proxyenv"
)

// ErrAIProviderNotConfigured reports that the experimental agent provider is
// missing baseUrl/model; it wraps llm.ErrInvalidConfig so the server layer can
// map it to AI_PROVIDER_UNAVAILABLE without importing this package.
var ErrAIProviderNotConfigured = fmt.Errorf("%w: provider baseUrl and model are required", llm.ErrInvalidConfig)

// aiProviderTestTimeout bounds the provider connectivity test request.
const aiProviderTestTimeout = 15 * time.Second

// AIProviderSettings returns the current experimental agent provider configuration.
func (a *App) AIProviderSettings() contracts.AIProviderSettingsDTO {
	a.aiProviderMu.RLock()
	defer a.aiProviderMu.RUnlock()
	return aiProviderSettingsDTOFromConfig(a.cfg.AIProvider)
}

// SetAIProviderSettingsPatch partially updates and persists the agent provider
// configuration to library-config.cfg, then updates in-memory config.
func (a *App) SetAIProviderSettingsPatch(p contracts.PatchAIProviderSettings) error {
	path := a.librarySettingsPath
	if path == "" {
		return fmt.Errorf("library settings path not configured")
	}

	a.aiProviderMu.Lock()
	target := a.cfg.AIProvider
	if p.Kind != nil {
		target.Kind = *p.Kind
	}
	if p.BaseURL != nil {
		target.BaseURL = *p.BaseURL
	}
	if p.APIKey != nil {
		target.APIKey = *p.APIKey
	}
	if p.Model != nil {
		target.Model = *p.Model
	}
	normalized, err := normalizeAIProviderConfig(target)
	if err != nil {
		a.aiProviderMu.Unlock()
		return err
	}
	a.aiProviderMu.Unlock()

	if err := config.WriteLibrarySettingsMerge(path, func(m map[string]any) error {
		provider := map[string]any{
			"kind":    normalized.Kind,
			"baseUrl": normalized.BaseURL,
			"model":   normalized.Model,
		}
		if normalized.APIKey != "" {
			provider["apiKey"] = normalized.APIKey
		}
		m["aiProvider"] = provider
		return nil
	}); err != nil {
		return err
	}
	a.aiProviderMu.Lock()
	a.cfg.AIProvider = normalized
	a.aiProviderMu.Unlock()
	return nil
}

// TestAIProvider performs a minimal chat completion against the draft provider
// config (when override is non-nil) or the persisted config, reporting latency.
// The response always carries OK=false on failure instead of an HTTP error so the
// settings UI can render the message, matching the proxy ping contract.
func (a *App) TestAIProvider(ctx context.Context, override *contracts.AIProviderSettingsDTO) contracts.AIProviderTestResponse {
	cfg := a.currentAIProviderConfig()
	if override != nil {
		cfg = config.AIProviderConfig{
			Kind:    override.Kind,
			BaseURL: override.BaseURL,
			APIKey:  override.APIKey,
			Model:   override.Model,
		}
	}
	normalized, err := normalizeAIProviderConfig(cfg)
	if err != nil {
		return contracts.AIProviderTestResponse{OK: false, Message: err.Error()}
	}

	client, err := newAIHTTPClient(a.currentProxyConfig(), aiProviderTestTimeout)
	if err != nil {
		return contracts.AIProviderTestResponse{OK: false, Message: err.Error()}
	}
	testCtx, cancel := context.WithTimeout(ctx, aiProviderTestTimeout)
	defer cancel()

	start := time.Now()
	_, err = llm.NewClient(llm.ClientConfig{
		BaseURL: normalized.BaseURL,
		APIKey:  normalized.APIKey,
		Model:   normalized.Model,
	}, client).Complete(testCtx, []llm.ChatMessage{
		{Role: "user", Content: "ping"},
	}, 16)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return contracts.AIProviderTestResponse{OK: false, LatencyMs: latency, Message: err.Error()}
	}
	return contracts.AIProviderTestResponse{OK: true, LatencyMs: latency}
}

// StreamAIChat streams an OpenAI-compatible chat completion for the experimental
// agent window (E1: plain chat, no tool calls). onDelta receives content deltas.
func (a *App) StreamAIChat(ctx context.Context, messages []contracts.AIChatMessage, onDelta func(string)) error {
	cfg, err := normalizeAIProviderConfig(a.currentAIProviderConfig())
	if err != nil {
		return fmt.Errorf("%w: %v", llm.ErrInvalidConfig, err)
	}
	if strings.TrimSpace(cfg.BaseURL) == "" || strings.TrimSpace(cfg.Model) == "" {
		return ErrAIProviderNotConfigured
	}
	// Streaming must not carry a whole-request client timeout; ctx (client
	// disconnect) governs the lifetime instead.
	client, err := newAIHTTPClient(a.currentProxyConfig(), 0)
	if err != nil {
		return err
	}
	chat := make([]llm.ChatMessage, 0, len(messages))
	for _, m := range messages {
		chat = append(chat, llm.ChatMessage{Role: m.Role, Content: m.Content})
	}
	_, err = llm.NewClient(llm.ClientConfig{
		BaseURL: cfg.BaseURL,
		APIKey:  cfg.APIKey,
		Model:   cfg.Model,
	}, client).StreamChat(ctx, chat, onDelta)
	return err
}

func (a *App) currentAIProviderConfig() config.AIProviderConfig {
	a.aiProviderMu.RLock()
	defer a.aiProviderMu.RUnlock()
	return a.cfg.AIProvider
}

func (a *App) currentProxyConfig() config.ProxyConfig {
	return a.cfg.Proxy
}

func aiProviderSettingsDTOFromConfig(cfg config.AIProviderConfig) contracts.AIProviderSettingsDTO {
	return contracts.AIProviderSettingsDTO{
		Kind:    config.NormalizeAIProviderKind(cfg.Kind),
		BaseURL: strings.TrimSpace(cfg.BaseURL),
		APIKey:  cfg.APIKey,
		Model:   strings.TrimSpace(cfg.Model),
	}
}

// normalizeAIProviderConfig trims and validates a provider config; empty baseUrl
// plus empty model stays valid ("not configured"), anything half-configured fails.
func normalizeAIProviderConfig(cfg config.AIProviderConfig) (config.AIProviderConfig, error) {
	kind := config.NormalizeAIProviderKind(cfg.Kind)
	if !config.ValidAIProviderKind(kind) {
		return config.AIProviderConfig{}, fmt.Errorf("unsupported aiProvider kind %q", cfg.Kind)
	}
	baseURL := strings.TrimSpace(cfg.BaseURL)
	model := strings.TrimSpace(cfg.Model)
	if baseURL != "" {
		u, err := url.Parse(baseURL)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
			return config.AIProviderConfig{}, fmt.Errorf("aiProvider.baseUrl must be an http(s) URL")
		}
	}
	if baseURL == "" && model != "" {
		return config.AIProviderConfig{}, fmt.Errorf("aiProvider.baseUrl is required when model is set")
	}
	return config.AIProviderConfig{
		Kind:    kind,
		BaseURL: baseURL,
		APIKey:  cfg.APIKey,
		Model:   model,
	}, nil
}

// newAIHTTPClient builds a proxy-aware HTTP client. timeout <= 0 means no client
// timeout (streaming); callers rely on request context cancellation instead.
func newAIHTTPClient(proxyCfg config.ProxyConfig, timeout time.Duration) (*http.Client, error) {
	client, err := proxyenv.NewHTTPClientForProxy(proxyCfg, timeout)
	if err != nil {
		return nil, err
	}
	if timeout <= 0 {
		client.Timeout = 0
	}
	return client, nil
}

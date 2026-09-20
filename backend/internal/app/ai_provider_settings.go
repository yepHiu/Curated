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
const aiProviderTestTimeout = 30 * time.Second

// Reasoning providers may spend the first output tokens before producing text.
// Keep the probe bounded, but do not exhaust its budget before the final "pong".
const aiProviderTestOutputLimit = 1024

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
	defer a.aiProviderMu.Unlock()
	target := a.cfg.AIProvider
	if p.ContextWindow != nil {
		if *p.ContextWindow < config.MinAIContextWindow || *p.ContextWindow > config.MaxAIContextWindow {
			return fmt.Errorf("aiProvider.contextWindow must be an integer between %d and %d tokens", config.MinAIContextWindow, config.MaxAIContextWindow)
		}
		target.ContextWindow = *p.ContextWindow
	}
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
		return err
	}

	if err := config.WriteLibrarySettingsMerge(path, func(m map[string]any) error {
		provider := map[string]any{
			"contextWindow": normalized.ContextWindow,
			"kind":          normalized.Kind,
			"baseUrl":       normalized.BaseURL,
			"model":         normalized.Model,
		}
		if normalized.APIKey != "" {
			provider["apiKey"] = normalized.APIKey
		}
		m["aiProvider"] = provider
		return nil
	}); err != nil {
		return err
	}
	a.cfg.AIProvider = normalized
	return nil
}

// TestAIProvider performs a minimal chat completion against the draft provider
// config (when override is non-nil) or the persisted config, reporting latency.
// The response always carries OK=false on failure instead of an HTTP error so the
// settings UI can render the message, matching the proxy ping contract.
func (a *App) TestAIProvider(ctx context.Context, override *contracts.AIProviderSettingsDTO) (result contracts.AIProviderTestResponse) {
	ctx, observation, finish := a.beginAIRun(ctx, "test", "")
	defer func() {
		var failure error
		if !result.OK {
			failure = llm.ErrInvalidConfig
		}
		finish(failure)
	}()
	cfg := a.currentAIProviderConfig()
	if override != nil {
		cfg = config.AIProviderConfig{
			ContextWindow: override.ContextWindow,
			Kind:          override.Kind,
			BaseURL:       override.BaseURL,
			APIKey:        override.APIKey,
			Model:         override.Model,
		}
	}
	observation.row.Model = cfg.Model
	observation.row.Provider = config.NormalizeAIProviderKind(cfg.Kind)
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
	completer := llm.NewClient(llm.ClientConfig{
		BaseURL: normalized.BaseURL,
		APIKey:  normalized.APIKey,
		Model:   normalized.Model,
	}, client)
	completer.Observe = observation.observe
	_, err = completer.Complete(testCtx, []llm.ChatMessage{
		{Role: "user", Content: "Reply with only the word pong."},
	}, aiProviderTestOutputLimit)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return contracts.AIProviderTestResponse{OK: false, LatencyMs: latency, Message: err.Error()}
	}
	return contracts.AIProviderTestResponse{OK: true, LatencyMs: latency}
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
		ContextWindow: config.EffectiveAIContextWindow(cfg.ContextWindow),
		Kind:          config.NormalizeAIProviderKind(cfg.Kind),
		BaseURL:       strings.TrimSpace(cfg.BaseURL),
		APIKey:        cfg.APIKey,
		Model:         strings.TrimSpace(cfg.Model),
	}
}

// normalizeAIProviderConfig trims and validates a provider config; empty baseUrl
// plus empty model stays valid ("not configured"), anything half-configured fails.
func normalizeAIProviderConfig(cfg config.AIProviderConfig) (config.AIProviderConfig, error) {
	if !config.ValidAIContextWindow(cfg.ContextWindow) {
		return config.AIProviderConfig{}, fmt.Errorf("aiProvider.contextWindow must be between %d and %d tokens", config.MinAIContextWindow, config.MaxAIContextWindow)
	}
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
		ContextWindow: config.EffectiveAIContextWindow(cfg.ContextWindow),
		Kind:          kind,
		BaseURL:       baseURL,
		APIKey:        cfg.APIKey,
		Model:         model,
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

package app

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"curated-backend/internal/agent/prompts"
	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/llm"
	"curated-backend/internal/storage"
	"curated-backend/internal/subtitles"
)

func (a *App) SubtitleConfig() config.SubtitleConfig {
	a.aiProviderMu.RLock()
	defer a.aiProviderMu.RUnlock()
	return a.cfg.SubtitleProvider
}
func (a *App) SubtitleSettings() contracts.SubtitleSettings {
	c := a.SubtitleConfig()
	_ = c.Validate()
	return contracts.SubtitleSettings{RecognitionModel: c.RecognitionModel, Enabled: c.Enabled, APIKeyConfigured: c.APIKey != "", TranslationModel: c.TranslationModel, ConfiguredModel: a.currentAIProviderConfig().Model}
}
func (a *App) SetSubtitleSettings(p contracts.SubtitleSettingsPatch) error {
	a.aiProviderMu.Lock()
	defer a.aiProviderMu.Unlock()
	c := a.cfg.SubtitleProvider
	if p.RecognitionModel != nil {
		c.RecognitionModel = *p.RecognitionModel
	}
	if p.Enabled != nil {
		c.Enabled = *p.Enabled
	}
	if p.APIKey != nil {
		c.APIKey = strings.TrimSpace(*p.APIKey)
	}
	if p.TranslationModel != nil {
		c.TranslationModel = *p.TranslationModel
	}
	if e := c.Validate(); e != nil {
		return e
	}
	if a.librarySettingsPath == "" {
		return fmt.Errorf("library settings path unavailable")
	}
	if e := config.WriteLibrarySettingsMerge(a.librarySettingsPath, func(m map[string]any) error { m["subtitleProvider"] = c; return nil }); e != nil {
		return e
	}
	a.cfg.SubtitleProvider = c
	return nil
}
func (a *App) subtitleRuntime(model string) (subtitles.RuntimeConfig, error) {
	c := a.SubtitleConfig()
	_ = c.Validate()
	if model == "" {
		model = c.TranslationModel
	}
	r := subtitles.RuntimeConfig{RecognitionModel: c.RecognitionModel, Enabled: c.Enabled, APIKey: c.APIKey, DefaultModel: c.TranslationModel}
	switch model {
	case "configured-ai":
		p := a.currentAIProviderConfig()
		r.Text = llm.ClientConfig{BaseURL: p.BaseURL, APIKey: p.APIKey, Model: p.Model}
	case "qwen-flash", "qwen-plus":
		disabled := false
		r.Text = llm.ClientConfig{BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1", APIKey: c.APIKey, Model: model, EnableThinking: &disabled}
	default:
		return r, fmt.Errorf("unsupported translation model")
	}
	return r, r.Text.Validate()
}
func (a *App) startSubtitles() {
	client, e := newAIHTTPClient(a.currentProxyConfig(), 3*time.Minute)
	if e != nil {
		return
	}
	a.subtitles = subtitles.New(a.appCtx, subtitles.Options{Store: a.store, Root: filepath.Join(a.cfg.CacheDir, "subtitle-jobs"), FFmpeg: a.cfg.Player.FFmpegCommand, Provider: &subtitles.Bailian{Client: client}, Config: a.subtitleRuntime, Translate: a.translateSubtitleCues})
}

func (a *App) TestSubtitleProvider(ctx context.Context) error {
	client, e := newAIHTTPClient(a.currentProxyConfig(), 30*time.Second)
	if e != nil {
		return e
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return (&subtitles.Bailian{Client: client}).TestPolicy(ctx, a.SubtitleConfig().APIKey, a.SubtitleConfig().RecognitionModel)
}
func (a *App) translateSubtitleCues(ctx context.Context, r storage.SubtitleRecord, cues []subtitles.Cue) (result []subtitles.Cue, err error) {
	cfg, err := a.subtitleRuntime(r.Job.TranslationModel)
	if err != nil {
		return nil, err
	}
	if cfg.Text.BaseURL != r.TextBaseURL || cfg.Text.Model != r.TextModel {
		return nil, fmt.Errorf("translation model configuration changed")
	}
	ctx, observation, finish := a.beginAIRun(ctx, "subtitle", r.Job.ID)
	defer func() { finish(err) }()
	observation.row.Model = cfg.Text.Model
	observation.row.PromptVersion = r.PromptVersion
	observation.row.Provider = r.Job.TranslationModel
	client, err := newAIHTTPClient(a.currentProxyConfig(), 3*time.Minute)
	if err != nil {
		return nil, err
	}
	completer := llm.NewClient(cfg.Text, client)
	completer.Observe = observation.observe
	input := make([]map[string]any, 0, len(cues))
	for _, c := range cues {
		input = append(input, map[string]any{"id": c.ID, "text": c.Text})
	}
	raw, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	// Omit max_tokens so the provider applies its default output budget.
	text, err := completer.Complete(ctx, []llm.ChatMessage{{Role: "system", Content: prompts.SubtitleTranslationPrompt().Text}, {Role: "user", Content: string(raw)}}, 0)
	if err != nil {
		return nil, &subtitles.TranslationError{Category: llm.ErrorCategory(err), Cause: err}
	}
	if err = json.Unmarshal([]byte(text), &result); err != nil {
		return nil, fmt.Errorf("translation response is not a JSON cue array")
	}
	return result, nil
}

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSubtitleConfigMergeAndModels(t *testing.T) {
	for _, model := range []string{"", "configured-ai", "qwen-flash", "qwen-plus", "arbitrary"} {
		cfg := SubtitleConfig{TranslationModel: model}
		e := cfg.Validate()
		if (model == "arbitrary") != (e != nil) {
			t.Fatalf("model %q err=%v", model, e)
		}
	}
	p := filepath.Join(t.TempDir(), "library-config.cfg")
	if e := os.WriteFile(p, []byte(`{"subtitleProvider":{"enabled":true,"apiKey":"saved-key","translationModel":"qwen-plus"}}`), 0600); e != nil {
		t.Fatal(e)
	}
	cfg := Default()
	if e := MergeLibrarySettingsFile(&cfg, p); e != nil {
		t.Fatal(e)
	}
	if !cfg.SubtitleProvider.Enabled || cfg.SubtitleProvider.APIKey != "saved-key" || cfg.SubtitleProvider.TranslationModel != "qwen-plus" {
		t.Fatalf("config not restored: %+v", cfg.SubtitleProvider)
	}
}

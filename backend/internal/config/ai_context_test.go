package config

import "testing"

func TestParseAIContextCapacity(t *testing.T) {
	for _, value := range []any{-1.0, 32767.0, 65536.5, "128000", 2097153.0, nil} {
		cfg := AIProviderConfig{}
		if err := parseAIProviderConfig(map[string]any{"contextWindow": value}, &cfg); err == nil {
			t.Fatalf("accepted invalid capacity %v", value)
		}
	}
	for _, value := range []int{0, 32768, 204800, 1000000, 2097152} {
		cfg := AIProviderConfig{}
		if err := parseAIProviderConfig(map[string]any{"contextWindow": float64(value)}, &cfg); err != nil || cfg.ContextWindow != value {
			t.Fatalf("capacity=%d config=%+v err=%v", value, cfg, err)
		}
	}
}

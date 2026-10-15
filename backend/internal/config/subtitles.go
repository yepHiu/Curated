package config

import "fmt"

type SubtitleConfig struct {
	Enabled          bool   `json:"enabled"`
	APIKey           string `json:"apiKey"`
	TranslationModel string `json:"translationModel"`
}

func (c *SubtitleConfig) Validate() error {
	if c.TranslationModel == "" {
		c.TranslationModel = "configured-ai"
	}
	switch c.TranslationModel {
	case "configured-ai", "qwen-flash", "qwen-plus":
		return nil
	}
	return fmt.Errorf("unsupported subtitle translation model")
}

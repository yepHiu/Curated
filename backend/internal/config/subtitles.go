package config

import "fmt"

type SubtitleConfig struct {
	RecognitionModel string `json:"recognitionModel"`
	Enabled          bool   `json:"enabled"`
	APIKey           string `json:"apiKey"`
	TranslationModel string `json:"translationModel"`
}

func (c *SubtitleConfig) Validate() error {
	if c.RecognitionModel == "" {
		c.RecognitionModel = "qwen-audio-3.1-asr-flash-filetrans"
	}
	if c.RecognitionModel != "qwen-audio-3.1-asr-flash-filetrans" && c.RecognitionModel != "qwen3-asr-flash-filetrans" {
		return fmt.Errorf("unsupported subtitle recognition model")
	}
	if c.TranslationModel == "" {
		c.TranslationModel = "configured-ai"
	}
	switch c.TranslationModel {
	case "configured-ai", "qwen-flash", "qwen-plus":
		return nil
	}
	return fmt.Errorf("unsupported subtitle translation model")
}

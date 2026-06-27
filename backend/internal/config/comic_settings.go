package config

import (
	"encoding/json"
	"fmt"
	"strings"
)

const DefaultComicCacheMaxBytes int64 = 2 * 1024 * 1024 * 1024

// ComicReaderConfig stores global default comic reader preferences.
type ComicReaderConfig struct {
	Mode      string `json:"mode,omitempty"`
	Fit       string `json:"fit,omitempty"`
	Direction string `json:"direction,omitempty"`
}

// ComicCacheConfig stores comic thumbnail/page cache limits.
type ComicCacheConfig struct {
	MaxBytes int64 `json:"maxBytes,omitempty"`
}

func DefaultComicReaderConfig() ComicReaderConfig {
	return ComicReaderConfig{
		Mode:      "page",
		Fit:       "contain",
		Direction: "ltr",
	}
}

func DefaultComicCacheConfig() ComicCacheConfig {
	return ComicCacheConfig{MaxBytes: DefaultComicCacheMaxBytes}
}

func NormalizeComicReaderConfig(v ComicReaderConfig) ComicReaderConfig {
	return ComicReaderConfig{
		Mode:      NormalizeComicReaderMode(v.Mode),
		Fit:       NormalizeComicFitMode(v.Fit),
		Direction: NormalizeComicReadingDirection(v.Direction),
	}
}

func NormalizeComicReaderMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "scroll":
		return "scroll"
	case "", "page":
		return "page"
	default:
		return "page"
	}
}

func NormalizeComicFitMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "width":
		return "width"
	case "", "contain":
		return "contain"
	default:
		return "contain"
	}
}

func NormalizeComicReadingDirection(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "rtl":
		return "rtl"
	case "", "ltr":
		return "ltr"
	default:
		return "ltr"
	}
}

func NormalizeComicCacheConfig(v ComicCacheConfig) ComicCacheConfig {
	if v.MaxBytes == 0 {
		v.MaxBytes = DefaultComicCacheMaxBytes
	}
	if v.MaxBytes < 0 {
		v.MaxBytes = -1
	}
	return v
}

func parseComicReaderConfig(v any, cfg *ComicReaderConfig) error {
	if v == nil {
		return nil
	}
	m, ok := v.(map[string]any)
	if !ok {
		return fmt.Errorf("comicReader: expected object, got %T", v)
	}
	next := *cfg
	if raw, ok := m["mode"]; ok {
		s, err := parseJSONStringTrim(raw)
		if err != nil {
			return fmt.Errorf("comicReader.mode: %w", err)
		}
		next.Mode = NormalizeComicReaderMode(s)
	}
	if raw, ok := m["fit"]; ok {
		s, err := parseJSONStringTrim(raw)
		if err != nil {
			return fmt.Errorf("comicReader.fit: %w", err)
		}
		next.Fit = NormalizeComicFitMode(s)
	}
	if raw, ok := m["direction"]; ok {
		s, err := parseJSONStringTrim(raw)
		if err != nil {
			return fmt.Errorf("comicReader.direction: %w", err)
		}
		next.Direction = NormalizeComicReadingDirection(s)
	}
	*cfg = NormalizeComicReaderConfig(next)
	return nil
}

func parseComicCacheConfig(v any, cfg *ComicCacheConfig) error {
	if v == nil {
		return nil
	}
	m, ok := v.(map[string]any)
	if !ok {
		return fmt.Errorf("comicCache: expected object, got %T", v)
	}
	next := *cfg
	if raw, ok := m["maxBytes"]; ok {
		n, err := parseJSONInt64(raw, "comicCache.maxBytes")
		if err != nil {
			return err
		}
		next.MaxBytes = n
	}
	*cfg = NormalizeComicCacheConfig(next)
	return nil
}

func parseJSONInt64(v any, key string) (int64, error) {
	switch x := v.(type) {
	case int:
		return int64(x), nil
	case int64:
		return x, nil
	case float64:
		if x != float64(int64(x)) {
			return 0, fmt.Errorf("%s: invalid number %v", key, x)
		}
		return int64(x), nil
	case json.Number:
		i64, err := x.Int64()
		if err != nil {
			return 0, fmt.Errorf("%s: invalid number %v", key, v)
		}
		return i64, nil
	default:
		return 0, fmt.Errorf("%s: unsupported type %T", key, v)
	}
}

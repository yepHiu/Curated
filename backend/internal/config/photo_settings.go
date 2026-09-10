package config

import (
	"fmt"
	"strings"
)

const DefaultPhotoCacheMaxBytes int64 = 5 * 1024 * 1024 * 1024

// PhotoViewerConfig stores global default photo book viewer preferences.
type PhotoViewerConfig struct {
	Mode      string `json:"mode,omitempty"`
	Fit       string `json:"fit,omitempty"`
	Direction string `json:"direction,omitempty"`
}

// PhotoCacheConfig stores photo book thumbnail/page cache limits.
type PhotoCacheConfig struct {
	MaxBytes int64 `json:"maxBytes,omitempty"`
}

func DefaultPhotoViewerConfig() PhotoViewerConfig {
	return PhotoViewerConfig{
		Mode:      "page",
		Fit:       "contain",
		Direction: "ltr",
	}
}

func DefaultPhotoCacheConfig() PhotoCacheConfig {
	return PhotoCacheConfig{MaxBytes: DefaultPhotoCacheMaxBytes}
}

func NormalizePhotoViewerConfig(v PhotoViewerConfig) PhotoViewerConfig {
	return PhotoViewerConfig{
		Mode:      NormalizePhotoViewerMode(v.Mode),
		Fit:       NormalizePhotoFitMode(v.Fit),
		Direction: NormalizePhotoViewingDirection(v.Direction),
	}
}

func NormalizePhotoViewerMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "scroll":
		return "scroll"
	case "", "page":
		return "page"
	default:
		return "page"
	}
}

func NormalizePhotoFitMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "width":
		return "width"
	case "", "contain":
		return "contain"
	default:
		return "contain"
	}
}

func NormalizePhotoViewingDirection(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "rtl":
		return "rtl"
	case "", "ltr":
		return "ltr"
	default:
		return "ltr"
	}
}

func NormalizePhotoCacheConfig(v PhotoCacheConfig) PhotoCacheConfig {
	if v.MaxBytes == 0 {
		v.MaxBytes = DefaultPhotoCacheMaxBytes
	}
	if v.MaxBytes < 0 {
		v.MaxBytes = -1
	}
	return v
}

func parsePhotoViewerConfig(v any, cfg *PhotoViewerConfig) error {
	if v == nil {
		return nil
	}
	m, ok := v.(map[string]any)
	if !ok {
		return fmt.Errorf("photoViewer: expected object, got %T", v)
	}
	next := *cfg
	if raw, ok := m["mode"]; ok {
		s, err := parseJSONStringTrim(raw)
		if err != nil {
			return fmt.Errorf("photoViewer.mode: %w", err)
		}
		next.Mode = NormalizePhotoViewerMode(s)
	}
	if raw, ok := m["fit"]; ok {
		s, err := parseJSONStringTrim(raw)
		if err != nil {
			return fmt.Errorf("photoViewer.fit: %w", err)
		}
		next.Fit = NormalizePhotoFitMode(s)
	}
	if raw, ok := m["direction"]; ok {
		s, err := parseJSONStringTrim(raw)
		if err != nil {
			return fmt.Errorf("photoViewer.direction: %w", err)
		}
		next.Direction = NormalizePhotoViewingDirection(s)
	}
	*cfg = NormalizePhotoViewerConfig(next)
	return nil
}

func parsePhotoCacheConfig(v any, cfg *PhotoCacheConfig) error {
	if v == nil {
		return nil
	}
	m, ok := v.(map[string]any)
	if !ok {
		return fmt.Errorf("photoCache: expected object, got %T", v)
	}
	next := *cfg
	if raw, ok := m["maxBytes"]; ok {
		n, err := parseJSONInt64(raw, "photoCache.maxBytes")
		if err != nil {
			return err
		}
		next.MaxBytes = n
	}
	*cfg = NormalizePhotoCacheConfig(next)
	return nil
}

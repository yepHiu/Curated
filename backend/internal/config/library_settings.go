package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const librarySettingsFileName = "library-config.cfg"

// DefaultLibrarySettingsPath returns the path to the JSON file under repo config/ that stores
// persistent library UI settings (e.g. organizeLibrary). Resolution matches defaultDatabasePath cwd rules.
func DefaultLibrarySettingsPath() string {
	cwd, err := os.Getwd()
	if err == nil && filepath.Base(cwd) == "backend" {
		return filepath.FromSlash("../config/" + librarySettingsFileName)
	}
	return filepath.FromSlash("config/" + librarySettingsFileName)
}

// MergeLibrarySettingsFile reads library-config.cfg (if present) and applies recognized keys
// onto cfg. Unknown JSON keys are ignored for the in-memory struct; they are preserved on disk
// when using WriteLibrarySettingsMerge. If the file is missing or empty, cfg is unchanged.
func MergeLibrarySettingsFile(cfg *Config, path string) error {
	if cfg == nil {
		return errors.New("nil config")
	}
	path = filepath.Clean(path)
	m, err := readLibrarySettingsMap(path)
	if err != nil {
		return err
	}
	if v, ok := m["organizeLibrary"]; ok {
		b, err := parseJSONBool(v, "organizeLibrary")
		if err != nil {
			return fmt.Errorf("library settings %q: %w", path, err)
		}
		cfg.OrganizeLibrary = b
	}
	if v, ok := m["autoLibraryWatch"]; ok {
		b, err := parseJSONBool(v, "autoLibraryWatch")
		if err != nil {
			return fmt.Errorf("library settings %q: %w", path, err)
		}
		cfg.AutoLibraryWatch = b
	}
	if v, ok := m["metadataMovieProvider"]; ok {
		s, err := parseJSONStringTrim(v)
		if err != nil {
			return fmt.Errorf("library settings %q: %w", path, err)
		}
		cfg.MetadataMovieProvider = s
	}
	if v, ok := m["extendedLibraryImport"]; ok {
		b, err := parseJSONBool(v, "extendedLibraryImport")
		if err != nil {
			return fmt.Errorf("library settings %q: %w", path, err)
		}
		cfg.ExtendedLibraryImport = b
	}
	return nil
}

func parseJSONStringTrim(v any) (string, error) {
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x), nil
	case nil:
		return "", nil
	case float64:
		// unlikely in hand-edited JSON; tolerate numeric as empty
		if x == 0 {
			return "", nil
		}
		return "", fmt.Errorf("metadataMovieProvider: invalid number %v", x)
	default:
		return "", fmt.Errorf("metadataMovieProvider: unsupported type %T", x)
	}
}

// WriteLibrarySettingsMerge reads the existing JSON object (or starts empty), lets mutator update it,
// then writes atomically via a temp file + rename. Preserves unrelated keys for forward-compatible settings.
func WriteLibrarySettingsMerge(path string, mutator func(map[string]any) error) error {
	if mutator == nil {
		return errors.New("nil mutator")
	}
	path = filepath.Clean(path)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	m, err := readLibrarySettingsMap(path)
	if err != nil {
		return err
	}
	if err := mutator(m); err != nil {
		return err
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(filepath.Dir(path), "."+librarySettingsFileName+".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func readLibrarySettingsMap(path string) (map[string]any, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return make(map[string]any), nil
	}
	if err != nil {
		return nil, err
	}
	b = bytes.TrimSpace(b)
	if len(b) == 0 {
		return make(map[string]any), nil
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	if m == nil {
		return make(map[string]any), nil
	}
	return m, nil
}

func parseJSONBool(v any, key string) (bool, error) {
	switch x := v.(type) {
	case bool:
		return x, nil
	case string:
		switch x {
		case "true", "1":
			return true, nil
		case "false", "0":
			return false, nil
		default:
			return false, fmt.Errorf("%s: invalid string %q", key, x)
		}
	case float64:
		// JSON numbers unmarshaled into map[string]any
		if x == 0 {
			return false, nil
		}
		if x == 1 {
			return true, nil
		}
		return false, fmt.Errorf("%s: invalid number %v", key, x)
	default:
		return false, fmt.Errorf("%s: unsupported type %T", key, v)
	}
}

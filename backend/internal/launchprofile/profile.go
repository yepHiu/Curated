// Package launchprofile preserves the data location selected during a Full upgrade.
package launchprofile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Profile struct {
	Schema       int    `json:"schema"`
	DataRoot     string `json:"dataRoot"`
	ConfigPath   string `json:"configPath,omitempty"`
	DatabasePath string `json:"databasePath"`
}

func Path(localAppData string) string {
	return filepath.Join(localAppData, "Curated", "server-startup.json")
}

func (p Profile) Validate() error {
	if p.Schema != 1 || !filepath.IsAbs(p.DataRoot) || !filepath.IsAbs(p.DatabasePath) || (p.ConfigPath != "" && !filepath.IsAbs(p.ConfigPath)) {
		return fmt.Errorf("invalid migrated Server startup profile")
	}
	return nil
}

// Resolve leaves explicit command-line/environment selections authoritative.
// A missing migrated database is an error, never permission to create an empty one.
func Resolve(path, explicitConfig, dataRoot string) (*Profile, error) {
	if explicitConfig != "" {
		return nil, nil
	}
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var p Profile
	if err = json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	if dataRoot != "" && !strings.EqualFold(filepath.Clean(dataRoot), filepath.Clean(p.DataRoot)) {
		return nil, nil
	}
	if err = p.Validate(); err != nil {
		return nil, err
	}
	for _, file := range []string{p.DatabasePath, p.ConfigPath} {
		if file == "" {
			continue
		}
		info, err := os.Stat(file)
		if err != nil {
			return nil, fmt.Errorf("migrated Server file unavailable (%s): %w", file, err)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("migrated Server file is not regular: %s", file)
		}
	}
	return &p, nil
}

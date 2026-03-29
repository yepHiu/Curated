// Package curated parses and validates Curated.json beside a movie root directory.
package curated

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"curated-backend/internal/library/moviecode"
)

const ManifestFileName = "Curated.json"

const maxManifestBytes = 64 * 1024

var videoExt = map[string]struct{}{
	"mp4": {}, "mkv": {}, "avi": {}, "mov": {}, "ts": {},
}

// Manifest is the on-disk Curated.json schema (v1).
type Manifest struct {
	SchemaVersion int    `json:"schemaVersion"`
	Layout        string `json:"layout"`
	Code          string `json:"code"`
	PrimaryVideo  string `json:"primaryVideo,omitempty"`
	CreatedAt     string `json:"createdAt,omitempty"`
	App           string `json:"app,omitempty"`
	AppVersion    string `json:"appVersion,omitempty"`
}

var (
	ErrInvalidManifest   = errors.New("curated: invalid manifest")
	ErrAmbiguousVideos   = errors.New("curated: multiple video files without primaryVideo")
	ErrPrimaryVideo      = errors.New("curated: primaryVideo missing or invalid")
	ErrCodeDirMismatch   = errors.New("curated: code does not match directory name")
	ErrUnsupportedSchema = errors.New("curated: unsupported schemaVersion")
)

// LoadAndValidate reads {rootDir}/Curated.json and validates it against rootDir and on-disk videos.
func LoadAndValidate(rootDir string) (*Manifest, error) {
	rootDir = filepath.Clean(rootDir)
	path := filepath.Join(rootDir, ManifestFileName)
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(b) > maxManifestBytes {
		return nil, fmt.Errorf("%w: file too large", ErrInvalidManifest)
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidManifest, err)
	}
	if err := validateManifest(&m, rootDir); err != nil {
		return nil, err
	}
	return &m, nil
}

func validateManifest(m *Manifest, rootDir string) error {
	if m.SchemaVersion != 1 {
		return fmt.Errorf("%w: %d", ErrUnsupportedSchema, m.SchemaVersion)
	}
	if strings.TrimSpace(m.Layout) == "" {
		return fmt.Errorf("%w: layout required", ErrInvalidManifest)
	}
	code := strings.TrimSpace(m.Code)
	if code == "" {
		return fmt.Errorf("%w: code required", ErrInvalidManifest)
	}
	base := filepath.Base(rootDir)
	if moviecode.NormalizeForStorageID(base) != moviecode.NormalizeForStorageID(code) {
		return ErrCodeDirMismatch
	}

	videos, err := listVideoBasenames(rootDir)
	if err != nil {
		return err
	}
	pv := strings.TrimSpace(m.PrimaryVideo)
	if pv != "" {
		pv = filepath.Clean(pv)
		if strings.Contains(pv, "..") || filepath.IsAbs(pv) {
			return fmt.Errorf("%w: primaryVideo must be a plain filename under root", ErrPrimaryVideo)
		}
		if filepath.Dir(pv) != "." {
			return fmt.Errorf("%w: primaryVideo must not contain path separators", ErrPrimaryVideo)
		}
		full := filepath.Join(rootDir, pv)
		if rel, err := filepath.Rel(rootDir, full); err != nil || strings.HasPrefix(rel, "..") {
			return ErrPrimaryVideo
		}
		if !isVideoFile(full) {
			return ErrPrimaryVideo
		}
		return nil
	}
	switch len(videos) {
	case 0:
		return ErrPrimaryVideo
	case 1:
		return nil
	default:
		return ErrAmbiguousVideos
	}
}

func listVideoBasenames(rootDir string) ([]string, error) {
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if isVideoBasename(name) {
			out = append(out, name)
		}
	}
	return out, nil
}

func isVideoBasename(name string) bool {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(name)), ".")
	_, ok := videoExt[ext]
	return ok
}

func isVideoFile(path string) bool {
	fi, err := os.Stat(path)
	if err != nil || fi.IsDir() {
		return false
	}
	return isVideoBasename(filepath.Base(path))
}

// WriteManifest writes Curated.json into rootDir (atomic via temp + rename in same dir).
func WriteManifest(rootDir string, m *Manifest) error {
	rootDir = filepath.Clean(rootDir)
	if err := validateManifest(m, rootDir); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(rootDir, ".Curated.json.*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	defer func() { _ = os.Remove(tmpPath) }()
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return err
	}
	dest := filepath.Join(rootDir, ManifestFileName)
	return os.Rename(tmpPath, dest)
}

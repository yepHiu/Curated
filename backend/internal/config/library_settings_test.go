package config

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestMergeLibrarySettingsFile_MissingFile_NoChangeToDefault(t *testing.T) {
	t.Parallel()
	cfg := Default()
	if !cfg.OrganizeLibrary {
		t.Fatal("Default() should have organizeLibrary true")
	}
	path := filepath.Join(t.TempDir(), "nope.cfg")
	if err := MergeLibrarySettingsFile(&cfg, path); err != nil {
		t.Fatal(err)
	}
	if !cfg.OrganizeLibrary {
		t.Fatal("expected organizeLibrary still true")
	}
}

func TestMergeLibrarySettingsFile_ExplicitFalse(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	path := filepath.Join(root, "library-config.cfg")
	if err := os.WriteFile(path, []byte(`{"organizeLibrary": false, "futureKey": 1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := Default()
	if err := MergeLibrarySettingsFile(&cfg, path); err != nil {
		t.Fatal(err)
	}
	if cfg.OrganizeLibrary {
		t.Fatal("expected false from file")
	}
}

func TestWriteLibrarySettingsMerge_PreservesUnknownKeys(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	path := filepath.Join(root, "library-config.cfg")
	if err := os.WriteFile(path, []byte(`{
  "futureKey": "keep-me",
  "organizeLibrary": false
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := WriteLibrarySettingsMerge(path, func(m map[string]any) error {
		m["organizeLibrary"] = true
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b, []byte("keep-me")) {
		t.Fatalf("expected futureKey preserved, got %s", string(b))
	}
	if !bytes.Contains(b, []byte(`"organizeLibrary": true`)) {
		t.Fatalf("expected organizeLibrary true, got %s", string(b))
	}
}

func TestWriteLibrarySettingsMerge_CreatesFile(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	path := filepath.Join(root, "nested", "library-config.cfg")
	if err := WriteLibrarySettingsMerge(path, func(m map[string]any) error {
		m["organizeLibrary"] = false
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	cfg := Default()
	if err := MergeLibrarySettingsFile(&cfg, path); err != nil {
		t.Fatal(err)
	}
	if cfg.OrganizeLibrary {
		t.Fatal("expected false")
	}
}

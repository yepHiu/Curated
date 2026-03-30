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
	if !cfg.AutoLibraryWatch {
		t.Fatal("expected AutoLibraryWatch still true (default)")
	}
}

func TestMergeLibrarySettingsFile_ExtendedLibraryImportTrue(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	path := filepath.Join(root, "library-config.cfg")
	if err := os.WriteFile(path, []byte(`{"extendedLibraryImport": true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := Default()
	if err := MergeLibrarySettingsFile(&cfg, path); err != nil {
		t.Fatal(err)
	}
	if !cfg.ExtendedLibraryImport {
		t.Fatal("expected extendedLibraryImport true from file")
	}
}

func TestMergeLibrarySettingsFile_AutoLibraryWatchFalse(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	path := filepath.Join(root, "library-config.cfg")
	if err := os.WriteFile(path, []byte(`{"autoLibraryWatch": false}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := Default()
	if err := MergeLibrarySettingsFile(&cfg, path); err != nil {
		t.Fatal(err)
	}
	if cfg.AutoLibraryWatch {
		t.Fatal("expected autoLibraryWatch false from file")
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

func TestMergeLibrarySettingsFile_MetadataMovieProvider(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	path := filepath.Join(root, "library-config.cfg")
	if err := os.WriteFile(path, []byte(`{"metadataMovieProvider": "  Fanza  "}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := Default()
	if err := MergeLibrarySettingsFile(&cfg, path); err != nil {
		t.Fatal(err)
	}
	if got, want := cfg.MetadataMovieProvider, "Fanza"; got != want {
		t.Fatalf("MetadataMovieProvider = %q, want %q", got, want)
	}
}

func TestWriteLibrarySettingsMerge_MetadataMovieProvider(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	path := filepath.Join(root, "library-config.cfg")
	if err := os.WriteFile(path, []byte(`{"organizeLibrary": true, "metadataMovieProvider": "old"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteLibrarySettingsMerge(path, func(m map[string]any) error {
		m["metadataMovieProvider"] = "new"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	cfg := Default()
	if err := MergeLibrarySettingsFile(&cfg, path); err != nil {
		t.Fatal(err)
	}
	if got, want := cfg.MetadataMovieProvider, "new"; got != want {
		t.Fatalf("MetadataMovieProvider = %q, want %q", got, want)
	}
	if !cfg.OrganizeLibrary {
		t.Fatal("expected organizeLibrary preserved true")
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

func TestMergeLibrarySettingsFile_BackendLog(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	path := filepath.Join(root, "library-config.cfg")
	raw := `{"logDir": " D:\\logs ", "logFilePrefix": "app", "logMaxAgeDays": 14, "logLevel": "debug"}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := Default()
	if err := MergeLibrarySettingsFile(&cfg, path); err != nil {
		t.Fatal(err)
	}
	if got, want := cfg.LogDir, `D:\logs`; got != want {
		t.Fatalf("LogDir = %q, want %q", got, want)
	}
	if got, want := cfg.LogFilePrefix, "app"; got != want {
		t.Fatalf("LogFilePrefix = %q, want %q", got, want)
	}
	if got, want := cfg.LogMaxAgeDays, 14; got != want {
		t.Fatalf("LogMaxAgeDays = %d, want %d", got, want)
	}
	if got, want := cfg.LogLevel, "debug"; got != want {
		t.Fatalf("LogLevel = %q, want %q", got, want)
	}
}

func TestMergeLibrarySettingsFile_InvalidLogLevel(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	path := filepath.Join(root, "library-config.cfg")
	if err := os.WriteFile(path, []byte(`{"logLevel": "not-a-level"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := Default()
	if err := MergeLibrarySettingsFile(&cfg, path); err == nil {
		t.Fatal("expected error for invalid logLevel")
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

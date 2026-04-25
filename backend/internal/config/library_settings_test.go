package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"
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
	if cfg.LaunchAtLogin {
		t.Fatal("expected LaunchAtLogin still false (default)")
	}
}

func TestDefault_LaunchAtLoginFalse(t *testing.T) {
	t.Parallel()
	cfg := Default()
	if cfg.LaunchAtLogin {
		t.Fatal("Default() should keep launchAtLogin false")
	}
}

func TestDefaultPlayerSettings_DisableHardwareDecodeAndStreamPush(t *testing.T) {
	t.Parallel()
	cfg := Default()
	if cfg.Player.HardwareDecode {
		t.Fatal("Default() should disable hardwareDecode")
	}
	if cfg.Player.StreamPushEnabled {
		t.Fatal("Default() should disable streamPushEnabled")
	}
	if cfg.Player.ForceStreamPush {
		t.Fatal("Default() should keep forceStreamPush disabled")
	}
}

func TestMergeLibrarySettingsFile_LegacyExtendedLibraryImportIgnored(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	path := filepath.Join(root, "library-config.cfg")
	if err := os.WriteFile(path, []byte(`{"extendedLibraryImport": true, "organizeLibrary": false}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := Default()
	if err := MergeLibrarySettingsFile(&cfg, path); err != nil {
		t.Fatal(err)
	}
	if cfg.OrganizeLibrary {
		t.Fatal("expected organizeLibrary false from file")
	}
	if cfg.AutoLibraryWatch != Default().AutoLibraryWatch {
		t.Fatal("legacy extendedLibraryImport should be ignored")
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

func TestMergeLibrarySettingsFile_AutoActorProfileScrapeTrue(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	path := filepath.Join(root, "library-config.cfg")
	if err := os.WriteFile(path, []byte(`{"autoActorProfileScrape": true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := Default()
	if err := MergeLibrarySettingsFile(&cfg, path); err != nil {
		t.Fatal(err)
	}
	if !cfg.AutoActorProfileScrape {
		t.Fatal("expected autoActorProfileScrape true from file")
	}
}

func TestMergeLibrarySettingsFile_LaunchAtLoginTrue(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	path := filepath.Join(root, "library-config.cfg")
	if err := os.WriteFile(path, []byte(`{"launchAtLogin": true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := Default()
	if err := MergeLibrarySettingsFile(&cfg, path); err != nil {
		t.Fatal(err)
	}
	if !cfg.LaunchAtLogin {
		t.Fatal("expected launchAtLogin true from file")
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

func TestMergeLibrarySettingsFile_PlayerHardwareEncoder(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	path := filepath.Join(root, "library-config.cfg")
	raw := `{"player": {"hardwareDecode": true, "hardwareEncoder": "amf"}}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := Default()
	if err := MergeLibrarySettingsFile(&cfg, path); err != nil {
		t.Fatal(err)
	}
	if got, want := cfg.Player.HardwareEncoder, "amf"; got != want {
		t.Fatalf("Player.HardwareEncoder = %q, want %q", got, want)
	}
}

func TestMergeLibrarySettingsFile_PlayerNativePlayerPreset(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	path := filepath.Join(root, "library-config.cfg")
	raw := `{"player": {"nativePlayerPreset": "potplayer", "nativePlayerCommand": "PotPlayerMini64.exe"}}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := Default()
	if err := MergeLibrarySettingsFile(&cfg, path); err != nil {
		t.Fatal(err)
	}
	if got, want := cfg.Player.NativePlayerPreset, "potplayer"; got != want {
		t.Fatalf("Player.NativePlayerPreset = %q, want %q", got, want)
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

func TestWriteLibrarySettingsMerge_ReplacesExistingFile(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	path := filepath.Join(root, "library-config.cfg")
	if err := os.WriteFile(path, []byte(`{"organizeLibrary":true,"futureKey":"keep-me"}`), 0o644); err != nil {
		t.Fatal(err)
	}

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
		t.Fatal("expected organizeLibrary=false after merge")
	}

	m, err := readLibrarySettingsMap(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprint(m["futureKey"]); got != "keep-me" {
		t.Fatalf("futureKey = %q, want keep-me", got)
	}
}

func TestWriteLibrarySettingsMerge_SerializesConcurrentWriters(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	path := filepath.Join(root, "library-config.cfg")

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := WriteLibrarySettingsMerge(path, func(m map[string]any) error {
				m[fmt.Sprintf("key_%d", i)] = i
				return nil
			})
			if err != nil {
				t.Errorf("writer %d: %v", i, err)
			}
		}()
	}
	wg.Wait()

	m, err := readLibrarySettingsMap(path)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 8; i++ {
		key := fmt.Sprintf("key_%d", i)
		if got := fmt.Sprint(m[key]); got != fmt.Sprint(i) {
			t.Fatalf("%s = %q, want %d", key, got, i)
		}
	}
}

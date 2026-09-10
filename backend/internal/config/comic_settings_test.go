package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultComicSettings(t *testing.T) {
	t.Parallel()

	cfg := Default()
	if cfg.ComicLibraryEnabled {
		t.Fatal("comic library should be disabled by default")
	}
	if !cfg.AutoComicLibraryWatch {
		t.Fatal("comic auto watch should be enabled by default")
	}
	if got, want := cfg.ComicReader.Mode, "page"; got != want {
		t.Fatalf("ComicReader.Mode = %q, want %q", got, want)
	}
	if got, want := cfg.ComicReader.Fit, "contain"; got != want {
		t.Fatalf("ComicReader.Fit = %q, want %q", got, want)
	}
	if got, want := cfg.ComicReader.Direction, "ltr"; got != want {
		t.Fatalf("ComicReader.Direction = %q, want %q", got, want)
	}
	if got, want := cfg.ComicCache.MaxBytes, int64(2*1024*1024*1024); got != want {
		t.Fatalf("ComicCache.MaxBytes = %d, want %d", got, want)
	}
}

func TestMergeLibrarySettingsFile_ComicSettings(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "library-config.cfg")
	raw := `{
  "comicLibraryEnabled": true,
  "autoComicLibraryWatch": false,
  "defaultComicImportLibraryPathId": " comic-lib-main ",
  "comicReader": {
    "mode": "scroll",
    "fit": "width",
    "direction": "rtl"
  },
  "comicCache": {
    "maxBytes": 5368709120
  }
}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := Default()
	if err := MergeLibrarySettingsFile(&cfg, path); err != nil {
		t.Fatal(err)
	}

	if !cfg.ComicLibraryEnabled {
		t.Fatal("expected comicLibraryEnabled true from library-config.cfg")
	}
	if cfg.AutoComicLibraryWatch {
		t.Fatal("expected autoComicLibraryWatch false from library-config.cfg")
	}
	if got, want := cfg.DefaultComicImportLibraryPathID, "comic-lib-main"; got != want {
		t.Fatalf("DefaultComicImportLibraryPathID = %q, want %q", got, want)
	}
	if got, want := cfg.ComicReader.Mode, "scroll"; got != want {
		t.Fatalf("ComicReader.Mode = %q, want %q", got, want)
	}
	if got, want := cfg.ComicReader.Fit, "width"; got != want {
		t.Fatalf("ComicReader.Fit = %q, want %q", got, want)
	}
	if got, want := cfg.ComicReader.Direction, "rtl"; got != want {
		t.Fatalf("ComicReader.Direction = %q, want %q", got, want)
	}
	if got, want := cfg.ComicCache.MaxBytes, int64(5368709120); got != want {
		t.Fatalf("ComicCache.MaxBytes = %d, want %d", got, want)
	}
}

func TestWriteLibrarySettingsMerge_ComicSettingsPreservesUnknownKeys(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "library-config.cfg")
	if err := os.WriteFile(path, []byte(`{"futureKey":"keep-me"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := WriteLibrarySettingsMerge(path, func(m map[string]any) error {
		m["comicLibraryEnabled"] = true
		m["autoComicLibraryWatch"] = false
		m["defaultComicImportLibraryPathId"] = "comic-lib-2"
		m["comicReader"] = map[string]any{
			"mode":      "page",
			"fit":       "width",
			"direction": "rtl",
		}
		m["comicCache"] = map[string]any{
			"maxBytes": int64(1073741824),
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	cfg := Default()
	if err := MergeLibrarySettingsFile(&cfg, path); err != nil {
		t.Fatal(err)
	}
	if !cfg.ComicLibraryEnabled {
		t.Fatal("expected comic library enabled after write merge")
	}
	if cfg.AutoComicLibraryWatch {
		t.Fatal("expected comic auto watch disabled after write merge")
	}
	if got, want := cfg.DefaultComicImportLibraryPathID, "comic-lib-2"; got != want {
		t.Fatalf("DefaultComicImportLibraryPathID = %q, want %q", got, want)
	}
	if got, want := cfg.ComicReader.Direction, "rtl"; got != want {
		t.Fatalf("ComicReader.Direction = %q, want %q", got, want)
	}
	if got, want := cfg.ComicCache.MaxBytes, int64(1073741824); got != want {
		t.Fatalf("ComicCache.MaxBytes = %d, want %d", got, want)
	}

	m, err := readLibrarySettingsMap(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := m["futureKey"], "keep-me"; got != want {
		t.Fatalf("futureKey = %q, want %q", got, want)
	}
}

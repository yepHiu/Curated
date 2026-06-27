package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
)

func TestComicSettingsPersistToLibraryConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "library-config.cfg")
	if err := os.WriteFile(path, []byte(`{"unknownKey":"preserve"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := &App{
		cfg:                 config.Default(),
		librarySettingsPath: path,
	}

	if err := a.SetComicLibraryEnabled(true); err != nil {
		t.Fatal(err)
	}
	if err := a.SetDefaultComicImportLibraryPathID("comic-lib-main"); err != nil {
		t.Fatal(err)
	}
	if err := a.SetComicReaderSettings(contracts.ComicReaderSettingsDTO{
		Mode:      "scroll",
		Fit:       "width",
		Direction: "rtl",
	}); err != nil {
		t.Fatal(err)
	}
	if err := a.SetComicCacheSettings(contracts.ComicCacheSettingsDTO{MaxBytes: 5 * 1024 * 1024 * 1024}); err != nil {
		t.Fatal(err)
	}

	if !a.ComicLibraryEnabled() {
		t.Fatal("expected comic library enabled")
	}
	if got := a.DefaultComicImportLibraryPathID(); got != "comic-lib-main" {
		t.Fatalf("DefaultComicImportLibraryPathID = %q", got)
	}
	if got := a.ComicReaderSettings(); got.Mode != "scroll" || got.Fit != "width" || got.Direction != "rtl" {
		t.Fatalf("ComicReaderSettings = %#v", got)
	}
	if got := a.ComicCacheSettings(); got.MaxBytes != 5*1024*1024*1024 {
		t.Fatalf("ComicCacheSettings.MaxBytes = %d", got.MaxBytes)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var persisted map[string]any
	if err := json.Unmarshal(raw, &persisted); err != nil {
		t.Fatal(err)
	}
	if persisted["unknownKey"] != "preserve" {
		t.Fatalf("unknownKey was not preserved: %#v", persisted)
	}
	if persisted["comicLibraryEnabled"] != true {
		t.Fatalf("comicLibraryEnabled not persisted: %#v", persisted["comicLibraryEnabled"])
	}
	if persisted["defaultComicImportLibraryPathId"] != "comic-lib-main" {
		t.Fatalf("defaultComicImportLibraryPathId not persisted: %#v", persisted["defaultComicImportLibraryPathId"])
	}
	reader, ok := persisted["comicReader"].(map[string]any)
	if !ok || reader["mode"] != "scroll" || reader["fit"] != "width" || reader["direction"] != "rtl" {
		t.Fatalf("comicReader not persisted: %#v", persisted["comicReader"])
	}
	cache, ok := persisted["comicCache"].(map[string]any)
	if !ok || cache["maxBytes"] != float64(5*1024*1024*1024) {
		t.Fatalf("comicCache not persisted: %#v", persisted["comicCache"])
	}
}

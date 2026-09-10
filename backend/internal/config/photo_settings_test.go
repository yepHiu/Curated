package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultPhotoSettings(t *testing.T) {
	t.Parallel()

	cfg := Default()
	if cfg.PhotoLibraryEnabled {
		t.Fatal("photo library should be disabled by default")
	}
	if !cfg.AutoPhotoLibraryWatch {
		t.Fatal("photo auto watch should be enabled by default")
	}
	if got, want := cfg.PhotoViewer.Mode, "page"; got != want {
		t.Fatalf("PhotoViewer.Mode = %q, want %q", got, want)
	}
	if got, want := cfg.PhotoViewer.Fit, "contain"; got != want {
		t.Fatalf("PhotoViewer.Fit = %q, want %q", got, want)
	}
	if got, want := cfg.PhotoViewer.Direction, "ltr"; got != want {
		t.Fatalf("PhotoViewer.Direction = %q, want %q", got, want)
	}
	if got, want := cfg.PhotoCache.MaxBytes, int64(5*1024*1024*1024); got != want {
		t.Fatalf("PhotoCache.MaxBytes = %d, want %d", got, want)
	}
}

func TestMergeLibrarySettingsFile_PhotoSettings(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "library-config.cfg")
	raw := `{
  "photoLibraryEnabled": true,
  "autoPhotoLibraryWatch": false,
  "defaultPhotoImportLibraryPathId": " photo-lib-main ",
  "photoViewer": {
    "mode": "scroll",
    "fit": "width",
    "direction": "rtl"
  },
  "photoCache": {
    "maxBytes": 2147483648
  }
}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := Default()
	if err := MergeLibrarySettingsFile(&cfg, path); err != nil {
		t.Fatal(err)
	}

	if !cfg.PhotoLibraryEnabled {
		t.Fatal("expected photoLibraryEnabled true from library-config.cfg")
	}
	if cfg.AutoPhotoLibraryWatch {
		t.Fatal("expected autoPhotoLibraryWatch false from library-config.cfg")
	}
	if got, want := cfg.DefaultPhotoImportLibraryPathID, "photo-lib-main"; got != want {
		t.Fatalf("DefaultPhotoImportLibraryPathID = %q, want %q", got, want)
	}
	if got, want := cfg.PhotoViewer.Mode, "scroll"; got != want {
		t.Fatalf("PhotoViewer.Mode = %q, want %q", got, want)
	}
	if got, want := cfg.PhotoViewer.Fit, "width"; got != want {
		t.Fatalf("PhotoViewer.Fit = %q, want %q", got, want)
	}
	if got, want := cfg.PhotoViewer.Direction, "rtl"; got != want {
		t.Fatalf("PhotoViewer.Direction = %q, want %q", got, want)
	}
	if got, want := cfg.PhotoCache.MaxBytes, int64(2147483648); got != want {
		t.Fatalf("PhotoCache.MaxBytes = %d, want %d", got, want)
	}
}

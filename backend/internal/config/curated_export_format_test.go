package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault_CuratedFrameExportFormatIsJPG(t *testing.T) {
	t.Parallel()

	cfg := Default()
	if got, want := cfg.CuratedFrameExportFormat, "jpg"; got != want {
		t.Fatalf("CuratedFrameExportFormat = %q, want %q", got, want)
	}
}

func TestMergeLibrarySettingsFile_CuratedFrameExportFormat(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "library-config.cfg")
	if err := os.WriteFile(path, []byte(`{"curatedFrameExportFormat":"png"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := Default()
	if err := MergeLibrarySettingsFile(&cfg, path); err != nil {
		t.Fatal(err)
	}
	if got, want := cfg.CuratedFrameExportFormat, "png"; got != want {
		t.Fatalf("CuratedFrameExportFormat = %q, want %q", got, want)
	}
}

func TestMergeLibrarySettingsFile_InvalidCuratedFrameExportFormatFallsBackToJPG(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "library-config.cfg")
	if err := os.WriteFile(path, []byte(`{"curatedFrameExportFormat":"gif"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := Default()
	if err := MergeLibrarySettingsFile(&cfg, path); err != nil {
		t.Fatal(err)
	}
	if got, want := cfg.CuratedFrameExportFormat, "jpg"; got != want {
		t.Fatalf("CuratedFrameExportFormat = %q, want %q", got, want)
	}
}

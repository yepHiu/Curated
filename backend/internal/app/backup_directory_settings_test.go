package app

import (
	"path/filepath"
	"testing"

	"curated-backend/internal/config"
)

func TestSetBackupDirectoryPersistsAndCanBeReloaded(t *testing.T) {
	t.Parallel()
	settingsPath := filepath.Join(t.TempDir(), "library-config.cfg")
	a := &App{
		cfg:                 config.Default(),
		librarySettingsPath: settingsPath,
	}

	if err := a.SetBackupDirectory(`  D:\Backups  `); err != nil {
		t.Fatal(err)
	}
	if got, want := a.BackupDirectory(), `D:\Backups`; got != want {
		t.Fatalf("BackupDirectory() = %q, want %q", got, want)
	}

	reloaded := config.Default()
	if err := config.MergeLibrarySettingsFile(&reloaded, settingsPath); err != nil {
		t.Fatal(err)
	}
	if got, want := reloaded.BackupDirectory, `D:\Backups`; got != want {
		t.Fatalf("reloaded BackupDirectory = %q, want %q", got, want)
	}

	if err := a.SetBackupDirectory(""); err != nil {
		t.Fatal(err)
	}
	reloaded = config.Default()
	if err := config.MergeLibrarySettingsFile(&reloaded, settingsPath); err != nil {
		t.Fatal(err)
	}
	if reloaded.BackupDirectory != "" {
		t.Fatalf("cleared BackupDirectory = %q, want empty", reloaded.BackupDirectory)
	}
}

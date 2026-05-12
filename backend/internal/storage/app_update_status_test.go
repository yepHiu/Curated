package storage

import (
	"context"
	"path/filepath"
	"testing"
)

func TestAppUpdateStatusSnapshotPersistsArtifactState(t *testing.T) {
	t.Parallel()

	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "app-update-artifact.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	downloadedPath := filepath.Join(t.TempDir(), "Curated-Setup-1.4.5.exe")
	want := AppUpdateStatusSnapshot{
		InstalledVersion:     "1.4.4",
		LatestVersion:        "1.4.5",
		Status:               "update-available",
		CheckedAt:            "2026-05-12T12:00:00Z",
		PublishedAt:          "2026-05-12T10:00:00Z",
		ReleaseName:          "Curated v1.4.5",
		ReleaseURL:           "https://github.com/yepHiu/Curated/releases/tag/v1.4.5",
		InstallerDownloadURL: "https://github.com/yepHiu/Curated/releases/download/v1.4.5/Curated-Setup-1.4.5.exe",
		InstallerSHA256:      "ABCDEF",
		ArtifactStatus:       "verified",
		DownloadedVersion:    "1.4.5",
		DownloadedFileName:   "Curated-Setup-1.4.5.exe",
		DownloadedFilePath:   downloadedPath,
		DownloadedBytes:      123,
		TotalBytes:           123,
		SignatureStatus:      "not_checked",
		InstallReady:         true,
		LastInstallAttemptAt: "2026-05-12T12:05:00Z",
		LastInstallError:     "previous failure",
		Source:               "github-releases",
	}

	if err := store.UpsertAppUpdateStatusSnapshot(context.Background(), want); err != nil {
		t.Fatalf("UpsertAppUpdateStatusSnapshot() error = %v", err)
	}

	got, ok, err := store.GetAppUpdateStatusSnapshot(context.Background())
	if err != nil {
		t.Fatalf("GetAppUpdateStatusSnapshot() error = %v", err)
	}
	if !ok {
		t.Fatal("expected snapshot")
	}
	if got.ArtifactStatus != want.ArtifactStatus || got.DownloadedVersion != want.DownloadedVersion || !got.InstallReady {
		t.Fatalf("artifact state = %+v, want %+v", got, want)
	}
	if got.InstallerSHA256 != want.InstallerSHA256 || got.DownloadedBytes != want.DownloadedBytes || got.TotalBytes != want.TotalBytes {
		t.Fatalf("artifact metadata = %+v, want %+v", got, want)
	}
	if got.DownloadedFilePath != downloadedPath || got.SignatureStatus != "not_checked" || got.LastInstallError != "previous failure" {
		t.Fatalf("artifact details = %+v, want path/signature/error preserved", got)
	}
}

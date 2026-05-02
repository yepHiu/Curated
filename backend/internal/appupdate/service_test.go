package appupdate

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap"

	"curated-backend/internal/storage"
	"curated-backend/internal/version"
)

func TestPackageVersionUsesDevFallbackVersion(t *testing.T) {
	original := version.InstallerVersion
	version.InstallerVersion = ""
	t.Cleanup(func() {
		version.InstallerVersion = original
	})

	got := version.PackageVersion()
	if got != "0.0.0" {
		t.Fatalf("PackageVersion() = %q, want %q", got, "0.0.0")
	}
}

func TestCheckNowIncludesLatestInstallerDownloadURL(t *testing.T) {
	original := version.InstallerVersion
	version.InstallerVersion = "1.2.7"
	t.Cleanup(func() {
		version.InstallerVersion = original
	})

	releaseServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"tag_name": "v1.2.8",
			"name": "Curated v1.2.8",
			"html_url": "https://github.com/yepHiu/Curated/releases/tag/v1.2.8",
			"published_at": "2026-04-20T12:00:00Z",
			"assets": [
				{
					"name": "Curated-Portable-1.2.8.zip",
					"browser_download_url": "https://example.com/Curated-Portable-1.2.8.zip"
				},
				{
					"name": "Curated-Setup-1.2.8.exe",
					"browser_download_url": "https://example.com/Curated-Setup-1.2.8.exe"
				}
			]
		}`))
	}))
	t.Cleanup(releaseServer.Close)

	store, err := storage.NewSQLiteStore(filepath.Join(t.TempDir(), "app-update.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	service := NewService(store, zap.NewNop())
	service.latestReleaseAPIURL = releaseServer.URL
	service.now = func() time.Time {
		return time.Date(2026, 4, 20, 13, 0, 0, 0, time.UTC)
	}

	dto, err := service.CheckNow(context.Background())
	if err != nil {
		t.Fatalf("CheckNow() error = %v", err)
	}
	if dto.InstallerDownloadURL != "https://example.com/Curated-Setup-1.2.8.exe" {
		t.Fatalf("InstallerDownloadURL = %q", dto.InstallerDownloadURL)
	}

	cached, err := service.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}
	if cached.InstallerDownloadURL != "https://example.com/Curated-Setup-1.2.8.exe" {
		t.Fatalf("cached InstallerDownloadURL = %q", cached.InstallerDownloadURL)
	}
}

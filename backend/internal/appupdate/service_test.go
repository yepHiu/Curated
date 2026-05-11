package appupdate

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"curated-backend/internal/storage"
	"curated-backend/internal/version"
)

func TestNormalizeReleaseNotesForCache(t *testing.T) {
	t.Parallel()
	if got := normalizeReleaseNotesForCache(""); got != "" {
		t.Fatalf("empty = %q", got)
	}
	if got := normalizeReleaseNotesForCache("  hello\r\nworld  "); got != "hello\nworld" {
		t.Fatalf("trim + crlf = %q", got)
	}
	long := strings.Repeat("a", 200_000)
	got := normalizeReleaseNotesForCache(long)
	if !strings.HasSuffix(got, "\n\n…") {
		t.Fatalf("expected ellipsis suffix")
	}
	body := got[:len(got)-len("\n\n…")]
	if len([]rune(body)) != 100_000 {
		t.Fatalf("expected 100000 runes before ellipsis, got %d", len([]rune(body)))
	}
}

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

func TestGetStatusRefreshesFreshLegacyTitleOnlyReleaseNotesCache(t *testing.T) {
	original := version.InstallerVersion
	version.InstallerVersion = "1.4.2"
	t.Cleanup(func() {
		version.InstallerVersion = original
	})

	var requestCount int
	releaseServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"tag_name": "v1.4.3",
			"name": "Curated v1.4.3",
			"html_url": "https://github.com/yepHiu/Curated/releases/tag/v1.4.3",
			"published_at": "2026-05-04T08:52:02Z",
			"body": "# Curated v1.4.3\n\nThis release includes the full body.\n\n## Highlights\n\n- Poster cards now preserve loaded image state."
		}`))
	}))
	t.Cleanup(releaseServer.Close)

	store, err := storage.NewSQLiteStore(filepath.Join(t.TempDir(), "legacy-notes-cache.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	now := time.Date(2026, 5, 5, 8, 0, 0, 0, time.UTC)
	if err := store.UpsertAppUpdateStatusSnapshot(context.Background(), storage.AppUpdateStatusSnapshot{
		InstalledVersion:    "1.4.2",
		LatestVersion:       "1.4.3",
		Status:              "update-available",
		CheckedAt:           now.Add(-time.Hour).Format(time.RFC3339),
		PublishedAt:         "2026-05-04T08:52:02Z",
		ReleaseName:         "Curated v1.4.3",
		ReleaseURL:          "https://github.com/yepHiu/Curated/releases/tag/v1.4.3",
		ReleaseNotesSnippet: "# Curated v1.4.3",
		Source:              updateSourceGitHubReleases,
	}); err != nil {
		t.Fatalf("UpsertAppUpdateStatusSnapshot() error = %v", err)
	}

	service := NewService(store, zap.NewNop())
	service.latestReleaseAPIURL = releaseServer.URL
	service.now = func() time.Time { return now }

	dto, err := service.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}
	if requestCount != 1 {
		t.Fatalf("release request count = %d, want 1", requestCount)
	}
	if !strings.Contains(dto.ReleaseNotesSnippet, "Poster cards now preserve loaded image state") {
		t.Fatalf("ReleaseNotesSnippet = %q", dto.ReleaseNotesSnippet)
	}
}

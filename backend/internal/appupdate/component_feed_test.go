package appupdate

import (
	"curated-backend/internal/storage"
	"curated-backend/internal/version"
	"encoding/json"
	"strings"
	"testing"
)

func TestServerManifestRejectsWrongComponentAndAmbiguousAssets(t *testing.T) {
	asset := map[string]string{"component": "server", "variant": "standalone", "channel": "stable", "version": "1.5.9", "platform": "windows", "arch": "x64", "format": "exe", "fileName": "Curated-Server-Setup-1.5.9-windows-x64.exe", "sha256": strings.Repeat("a", 64), "url": "https://github.com/yepHiu/Curated/releases/download/server-v1.5.9/Curated-Server-Setup-1.5.9-windows-x64.exe"}
	encode := func(entries ...map[string]string) []byte {
		data, _ := json.Marshal(map[string]any{"schema": 1, "artifacts": entries})
		return data
	}
	release, err := selectServerRelease(encode(asset), "windows", "x64")
	if err != nil || release.TagName != "1.5.9" || len(release.Assets) != 1 {
		t.Fatalf("unexpected release: %+v %v", release, err)
	}
	if _, err := selectServerRelease(encode(asset, asset), "windows", "x64"); err == nil {
		t.Fatal("ambiguous assets accepted")
	}
	for key, bad := range map[string]string{"component": "desktop", "variant": "bundle", "channel": "beta", "version": "1.5.9-beta", "platform": "macos", "arch": "arm64", "format": "zip", "url": "https://evil.test/setup.exe", "fileName": "Curated-Setup-1.5.9.exe", "sha256": "bad"} {
		original := asset[key]
		asset[key] = bad
		if _, err := selectServerRelease(encode(asset), "windows", "x64"); err == nil {
			t.Fatalf("invalid %s accepted", key)
		}
		asset[key] = original
	}
}

func TestStandaloneServerRejectsLegacyCachedInstaller(t *testing.T) {
	previous := version.Distribution
	version.Distribution = "server"
	t.Cleanup(func() { version.Distribution = previous })
	service := &Service{}
	snapshot := storage.AppUpdateStatusSnapshot{Source: updateSourceGitHubReleases, LatestVersion: "1.5.9", InstallerDownloadURL: "https://github.com/yepHiu/Curated/releases/download/v1.5.9/Curated-Setup-1.5.9.exe", InstallReady: true, ArtifactStatus: "verified"}
	if service.acceptsSnapshot(snapshot) {
		t.Fatal("accepted legacy installer cache")
	}
	snapshot.Source = serverUpdateSource
	if service.acceptsSnapshot(snapshot) {
		t.Fatal("accepted wrong component filename")
	}
	snapshot.InstallerDownloadURL = "https://github.com/yepHiu/Curated/releases/download/server-v1.5.9/Curated-Server-Setup-1.5.9-windows-x64.exe"
	if !service.acceptsSnapshot(snapshot) {
		t.Fatal("rejected matching Server cache")
	}
}

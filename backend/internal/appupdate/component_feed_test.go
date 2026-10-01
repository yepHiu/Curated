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

func TestServerBatchTagsKeepComponentSemverAndExactDownload(t *testing.T) {
	for _, tag := range []string{"server-v1.7.7", "release-20261001", "release-20261001-2", "release-20261001-10"} {
		t.Run(tag, func(t *testing.T) {
			name := "Curated-Server-Setup-1.7.8-windows-x64.exe"
			url := "https://github.com/yepHiu/Curated/releases/download/" + tag + "/" + name
			data, _ := json.Marshal(map[string]any{"schema": 1, "artifacts": []map[string]string{{
				"component": "server", "variant": "standalone", "channel": "stable", "version": "1.7.8",
				"platform": "windows", "arch": "x64", "format": "exe", "fileName": name, "url": url, "sha256": strings.Repeat("a", 64),
			}}})
			release, err := selectServerRelease(data, "windows", "x64")
			if err != nil || release.TagName != "1.7.8" || release.Assets[0].BrowserDownloadURL != url {
				t.Fatalf("batch changed version or download selection: %+v %v", release, err)
			}
			if !strings.HasSuffix(release.HTMLURL, "/"+tag) {
				t.Fatalf("wrong release link: %s", release.HTMLURL)
			}
		})
	}
}

func TestServerRejectsMalformedAndForeignBatchTags(t *testing.T) {
	for _, tag := range []string{"release-20260230", "release-20261001-1", "release-20261001-02", "release-20261001-0",
		"release-20261001183000", "release-20261001/other", "desktop-v1.7.8", "server-vanything", "v1.7.8"} {
		if validServerReleaseTag(tag) {
			t.Fatalf("accepted invalid release tag: %s", tag)
		}
	}
	if !strings.HasSuffix(serverFeedURL, "/server-v2.json") {
		t.Fatal("bridge Server must read the new feed, leaving old clients on the bridge feed")
	}
}

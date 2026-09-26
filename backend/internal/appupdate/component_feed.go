package appupdate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"runtime"
	"strings"

	"curated-backend/internal/storage"
	"curated-backend/internal/version"
)

const serverFeedURL = "https://raw.githubusercontent.com/yepHiu/Curated/release-channels/server.json"
const serverUpdateSource = "curated-server-stable"

var componentVersionPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)
var componentDigestPattern = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)

type componentManifest struct {
	Schema    int `json:"schema"`
	Artifacts []struct {
		Component string `json:"component"`
		Variant   string `json:"variant"`
		Channel   string `json:"channel"`
		Version   string `json:"version"`
		Platform  string `json:"platform"`
		Arch      string `json:"arch"`
		Format    string `json:"format"`
		FileName  string `json:"fileName"`
		URL       string `json:"url"`
		SHA256    string `json:"sha256"`
	} `json:"artifacts"`
}

func (s *Service) updateSource() string {
	if version.Distribution == "server" {
		return serverUpdateSource
	}
	return updateSourceGitHubReleases
}

func (s *Service) acceptsSnapshot(snapshot storage.AppUpdateStatusSnapshot) bool {
	if version.Distribution != "server" {
		return true
	}
	return snapshot.Source == serverUpdateSource &&
		snapshot.InstallerDownloadURL != "" &&
		strings.HasSuffix(snapshot.InstallerDownloadURL, "/Curated-Server-Setup-"+snapshot.LatestVersion+"-windows-x64.exe")
}

func selectServerRelease(data []byte, platform, arch string) (latestReleaseResponse, error) {
	var manifest componentManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return latestReleaseResponse{}, err
	}
	if manifest.Schema != 1 {
		return latestReleaseResponse{}, fmt.Errorf("invalid Server manifest schema")
	}
	var result latestReleaseResponse
	for _, entry := range manifest.Artifacts {
		if entry.Component != "server" || entry.Variant != "standalone" || entry.Channel != "stable" || entry.Platform != platform || entry.Arch != arch || entry.Format != "exe" {
			continue
		}
		name := "Curated-Server-Setup-" + entry.Version + "-windows-x64.exe"
		prefix := "https://github.com/yepHiu/Curated/releases/download/"
		tail := strings.TrimPrefix(entry.URL, prefix)
		parts := strings.Split(tail, "/")
		if !componentVersionPattern.MatchString(entry.Version) || !componentDigestPattern.MatchString(entry.SHA256) || entry.FileName != name ||
			!strings.HasPrefix(entry.URL, prefix) || len(parts) != 2 || parts[1] != name ||
			!(strings.HasPrefix(parts[0], "full-v") || strings.HasPrefix(parts[0], "server-v")) {
			return latestReleaseResponse{}, fmt.Errorf("invalid Server update asset")
		}
		if result.TagName != "" {
			return latestReleaseResponse{}, fmt.Errorf("ambiguous Server update assets")
		}
		result = latestReleaseResponse{TagName: entry.Version, Name: "Curated Server " + entry.Version,
			HTMLURL: "https://github.com/yepHiu/Curated/releases/tag/" + parts[0],
			Assets:  []latestReleaseAsset{{Name: name, BrowserDownloadURL: entry.URL, Digest: "sha256:" + entry.SHA256}}}
	}
	if result.TagName == "" {
		return result, fmt.Errorf("no compatible Server installer in the component feed")
	}
	return result, nil
}

func (s *Service) fetchServerRelease(ctx context.Context) (latestReleaseResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, serverFeedURL, nil)
	if err != nil {
		return latestReleaseResponse{}, err
	}
	client := *s.httpClient
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return fmt.Errorf("component feed redirects are forbidden")
	}
	resp, err := client.Do(req)
	if err != nil {
		return latestReleaseResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return latestReleaseResponse{}, fmt.Errorf("Server feed returned HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024+1))
	if err != nil {
		return latestReleaseResponse{}, err
	}
	if len(data) > 1024*1024 {
		return latestReleaseResponse{}, fmt.Errorf("Server manifest too large")
	}
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x64"
	}
	return selectServerRelease(data, runtime.GOOS, arch)
}

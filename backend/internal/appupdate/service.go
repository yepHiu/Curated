package appupdate

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
	"curated-backend/internal/version"
)

const (
	DefaultLatestReleaseAPIURL = "https://api.github.com/repos/yepHiu/Curated/releases/latest"
	DefaultReleasePageURL      = "https://github.com/yepHiu/Curated/releases"
	defaultCacheTTL            = 24 * time.Hour
	defaultRequestTimeout      = 8 * time.Second
	updateSourceGitHubReleases = "github-releases"
)

type Service struct {
	store               *storage.SQLiteStore
	logger              *zap.Logger
	httpClient          *http.Client
	now                 func() time.Time
	cacheTTL            time.Duration
	latestReleaseAPIURL string
	releasePageURL      string
}

type latestReleaseResponse struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	HTMLURL     string `json:"html_url"`
	Body        string `json:"body"`
	PublishedAt string `json:"published_at"`
	Draft       bool   `json:"draft"`
	Prerelease  bool   `json:"prerelease"`
}

type semanticVersion struct {
	Major int
	Minor int
	Patch int
}

func NewService(store *storage.SQLiteStore, logger *zap.Logger) *Service {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = http.ProxyFromEnvironment

	return &Service{
		store:               store,
		logger:              logger,
		httpClient:          &http.Client{Timeout: defaultRequestTimeout, Transport: transport},
		now:                 time.Now,
		cacheTTL:            defaultCacheTTL,
		releasePageURL:      DefaultReleasePageURL,
		latestReleaseAPIURL: DefaultLatestReleaseAPIURL,
	}
}

func (s *Service) GetStatus(ctx context.Context) (contracts.AppUpdateStatusDTO, error) {
	return s.getStatus(ctx, false)
}

func (s *Service) CheckNow(ctx context.Context) (contracts.AppUpdateStatusDTO, error) {
	return s.getStatus(ctx, true)
}

func (s *Service) getStatus(ctx context.Context, force bool) (contracts.AppUpdateStatusDTO, error) {
	installedVersion, installedSemver, err := normalizedSemver(version.PackageVersion())
	if err != nil || installedVersion == "" {
		return s.unsupportedStatus(), nil
	}

	if !force {
		snapshot, ok, err := s.store.GetAppUpdateStatusSnapshot(ctx)
		if err != nil {
			return contracts.AppUpdateStatusDTO{}, err
		}
		if ok && snapshot.InstalledVersion == installedVersion && s.isSnapshotFresh(snapshot.CheckedAt) {
			return snapshotToDTO(snapshot), nil
		}
	}

	existing, _, err := s.store.GetAppUpdateStatusSnapshot(ctx)
	if err != nil {
		return contracts.AppUpdateStatusDTO{}, err
	}

	release, err := s.fetchLatestRelease(ctx)
	if err != nil {
		snapshot := storage.AppUpdateStatusSnapshot{
			InstalledVersion:    installedVersion,
			LatestVersion:       existing.LatestVersion,
			Status:              "error",
			CheckedAt:           s.now().UTC().Format(time.RFC3339),
			PublishedAt:         existing.PublishedAt,
			ReleaseName:         existing.ReleaseName,
			ReleaseURL:          firstNonEmpty(existing.ReleaseURL, s.releasePageURL),
			ReleaseNotesSnippet: existing.ReleaseNotesSnippet,
			Source:              updateSourceGitHubReleases,
			ErrorMessage:        err.Error(),
		}
		if saveErr := s.store.UpsertAppUpdateStatusSnapshot(ctx, snapshot); saveErr != nil {
			return contracts.AppUpdateStatusDTO{}, saveErr
		}
		return snapshotToDTO(snapshot), nil
	}

	latestVersion, latestSemver, err := normalizedSemver(release.TagName)
	if err != nil {
		snapshot := storage.AppUpdateStatusSnapshot{
			InstalledVersion: installedVersion,
			Status:           "error",
			CheckedAt:        s.now().UTC().Format(time.RFC3339),
			PublishedAt:      release.PublishedAt,
			ReleaseName:      firstNonEmpty(strings.TrimSpace(release.Name), strings.TrimSpace(release.TagName)),
			ReleaseURL:       firstNonEmpty(strings.TrimSpace(release.HTMLURL), s.releasePageURL),
			Source:           updateSourceGitHubReleases,
			ErrorMessage:     err.Error(),
		}
		if saveErr := s.store.UpsertAppUpdateStatusSnapshot(ctx, snapshot); saveErr != nil {
			return contracts.AppUpdateStatusDTO{}, saveErr
		}
		return snapshotToDTO(snapshot), nil
	}

	status := "up-to-date"
	if compareSemanticVersion(latestSemver, installedSemver) > 0 {
		status = "update-available"
	}

	snapshot := storage.AppUpdateStatusSnapshot{
		InstalledVersion:    installedVersion,
		LatestVersion:       latestVersion,
		Status:              status,
		CheckedAt:           s.now().UTC().Format(time.RFC3339),
		PublishedAt:         strings.TrimSpace(release.PublishedAt),
		ReleaseName:         firstNonEmpty(strings.TrimSpace(release.Name), strings.TrimSpace(release.TagName)),
		ReleaseURL:          firstNonEmpty(strings.TrimSpace(release.HTMLURL), s.releasePageURL),
		ReleaseNotesSnippet: summarizeReleaseNotes(release.Body),
		Source:              updateSourceGitHubReleases,
	}
	if err := s.store.UpsertAppUpdateStatusSnapshot(ctx, snapshot); err != nil {
		return contracts.AppUpdateStatusDTO{}, err
	}

	return snapshotToDTO(snapshot), nil
}

func (s *Service) unsupportedStatus() contracts.AppUpdateStatusDTO {
	return contracts.AppUpdateStatusDTO{
		Supported:    false,
		Status:       "unsupported",
		ReleaseURL:   s.releasePageURL,
		Source:       updateSourceGitHubReleases,
		ErrorMessage: "installer version unavailable in current runtime",
	}
}

func (s *Service) isSnapshotFresh(checkedAt string) bool {
	checkedAt = strings.TrimSpace(checkedAt)
	if checkedAt == "" {
		return false
	}
	parsed, err := time.Parse(time.RFC3339, checkedAt)
	if err != nil {
		return false
	}
	return s.now().UTC().Sub(parsed.UTC()) < s.cacheTTL
}

func (s *Service) fetchLatestRelease(ctx context.Context) (latestReleaseResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.latestReleaseAPIURL, nil)
	if err != nil {
		return latestReleaseResponse{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "Curated/"+firstNonEmpty(strings.TrimSpace(version.PackageVersion()), "dev"))

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return latestReleaseResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return latestReleaseResponse{}, fmt.Errorf("github release request failed with HTTP %d", resp.StatusCode)
	}

	var release latestReleaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return latestReleaseResponse{}, err
	}
	if release.Draft || release.Prerelease {
		return latestReleaseResponse{}, fmt.Errorf("latest release response is not a stable release")
	}
	return release, nil
}

func snapshotToDTO(snapshot storage.AppUpdateStatusSnapshot) contracts.AppUpdateStatusDTO {
	dto := contracts.AppUpdateStatusDTO{
		Supported:           snapshot.Status != "unsupported",
		Status:              snapshot.Status,
		InstalledVersion:    snapshot.InstalledVersion,
		LatestVersion:       snapshot.LatestVersion,
		CheckedAt:           snapshot.CheckedAt,
		PublishedAt:         snapshot.PublishedAt,
		ReleaseName:         snapshot.ReleaseName,
		ReleaseURL:          snapshot.ReleaseURL,
		ReleaseNotesSnippet: snapshot.ReleaseNotesSnippet,
		Source:              snapshot.Source,
		ErrorMessage:        snapshot.ErrorMessage,
	}
	dto.HasUpdate = snapshot.Status == "update-available"
	return dto
}

func normalizedSemver(raw string) (display string, parsed semanticVersion, err error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", semanticVersion{}, nil
	}
	trimmed = strings.TrimPrefix(trimmed, "v")
	parts := strings.Split(trimmed, ".")
	if len(parts) != 3 {
		return "", semanticVersion{}, fmt.Errorf("invalid semantic version: %q", raw)
	}
	values := [3]int{}
	for index, part := range parts {
		if part == "" {
			return "", semanticVersion{}, fmt.Errorf("invalid semantic version: %q", raw)
		}
		value, convErr := strconv.Atoi(part)
		if convErr != nil || value < 0 {
			return "", semanticVersion{}, fmt.Errorf("invalid semantic version: %q", raw)
		}
		values[index] = value
	}
	return fmt.Sprintf("%d.%d.%d", values[0], values[1], values[2]), semanticVersion{
		Major: values[0],
		Minor: values[1],
		Patch: values[2],
	}, nil
}

func summarizeReleaseNotes(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}
	parts := strings.Split(body, "\n\n")
	for _, part := range parts {
		candidate := strings.TrimSpace(part)
		if candidate == "" {
			continue
		}
		candidate = strings.ReplaceAll(candidate, "\n", " ")
		candidate = strings.Join(strings.Fields(candidate), " ")
		if len(candidate) <= 220 {
			return candidate
		}
		return strings.TrimSpace(candidate[:217]) + "..."
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func compareSemanticVersion(left semanticVersion, right semanticVersion) int {
	switch {
	case left.Major != right.Major:
		if left.Major > right.Major {
			return 1
		}
		return -1
	case left.Minor != right.Minor:
		if left.Minor > right.Minor {
			return 1
		}
		return -1
	case left.Patch != right.Patch:
		if left.Patch > right.Patch {
			return 1
		}
		return -1
	default:
		return 0
	}
}

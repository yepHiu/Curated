// Package appupdate checks for new packaged-app releases from GitHub Releases and caches the status in SQLite.
package appupdate

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
	"curated-backend/internal/tasks"
	"curated-backend/internal/version"
)

const (
	// DefaultLatestReleaseAPIURL is the GitHub API endpoint for the latest stable release.
	DefaultLatestReleaseAPIURL = "https://api.github.com/repos/yepHiu/Curated/releases/latest"
	// DefaultReleasePageURL is the public GitHub Releases page for the repository.
	DefaultReleasePageURL      = "https://github.com/yepHiu/Curated/releases"
	defaultCacheTTL            = 24 * time.Hour
	defaultRequestTimeout      = 8 * time.Second
	defaultDownloadTimeout     = 10 * time.Minute
	updateSourceGitHubReleases = "github-releases"
	appUpdateTaskDownload      = "app-update.download"
)

var launchInstallerProcess = func(ctx context.Context, installerPath string, args []string) error {
	cmd := exec.CommandContext(ctx, installerPath, args...)
	return cmd.Start()
}

// Service checks and caches packaged-app update availability from GitHub Releases.
type Service struct {
	store               *storage.SQLiteStore
	logger              *zap.Logger
	httpClient          *http.Client
	now                 func() time.Time
	cacheTTL            time.Duration
	latestReleaseAPIURL string
	releasePageURL      string
	cacheDir            string
	tasks               *tasks.Manager
	downloadMu          sync.Mutex
}

type latestReleaseResponse struct {
	TagName     string               `json:"tag_name"`
	Name        string               `json:"name"`
	HTMLURL     string               `json:"html_url"`
	Body        string               `json:"body"`
	PublishedAt string               `json:"published_at"`
	Draft       bool                 `json:"draft"`
	Prerelease  bool                 `json:"prerelease"`
	Assets      []latestReleaseAsset `json:"assets"`
}

type latestReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Digest             string `json:"digest"`
	Size               int64  `json:"size"`
}

type semanticVersion struct {
	Major int
	Minor int
	Patch int
}

// NewService creates an app update Service that uses ProxyFromEnvironment for outbound requests.
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
		cacheDir:            defaultUpdateCacheDir(),
	}
}

// SetCacheDir points update downloads at a controlled application cache directory.
func (s *Service) SetCacheDir(cacheDir string) {
	cacheDir = strings.TrimSpace(cacheDir)
	if cacheDir == "" {
		return
	}
	s.cacheDir = filepath.Join(cacheDir, "updates")
}

// SetTaskManager wires optional task progress reporting for installer downloads.
func (s *Service) SetTaskManager(manager *tasks.Manager) {
	s.tasks = manager
}

// GetStatus returns the cached app update status, refreshing only when the cache is stale.
func (s *Service) GetStatus(ctx context.Context) (contracts.AppUpdateStatusDTO, error) {
	return s.getStatus(ctx, false)
}

// CheckNow forces a fresh GitHub Releases check regardless of cache age.
func (s *Service) CheckNow(ctx context.Context) (contracts.AppUpdateStatusDTO, error) {
	return s.getStatus(ctx, true)
}

// DownloadInstaller downloads and verifies the latest installer for the cached update-available release.
func (s *Service) DownloadInstaller(ctx context.Context) (contracts.AppUpdateStatusDTO, error) {
	s.downloadMu.Lock()
	defer s.downloadMu.Unlock()

	snapshot, ok, err := s.store.GetAppUpdateStatusSnapshot(ctx)
	if err != nil {
		return contracts.AppUpdateStatusDTO{}, err
	}
	if !ok || snapshot.Status != "update-available" {
		return contracts.AppUpdateStatusDTO{}, errors.New("no update is available for installer download")
	}
	if strings.TrimSpace(snapshot.InstallerDownloadURL) == "" {
		return contracts.AppUpdateStatusDTO{}, errors.New("installer download url is unavailable")
	}
	if strings.TrimSpace(snapshot.InstallerSHA256) == "" {
		return contracts.AppUpdateStatusDTO{}, errors.New("installer checksum is unavailable")
	}
	if err := validateInstallerDownloadURL(snapshot.InstallerDownloadURL); err != nil {
		return contracts.AppUpdateStatusDTO{}, err
	}

	fileName := installerFileName(snapshot.InstallerDownloadURL, snapshot.LatestVersion)
	targetDir := filepath.Join(s.cacheDir, snapshot.LatestVersion)
	targetPath := filepath.Join(targetDir, fileName)
	partPath := targetPath + ".part"
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return contracts.AppUpdateStatusDTO{}, err
	}

	task := contracts.TaskDTO{}
	if s.tasks != nil {
		task = s.tasks.Create(appUpdateTaskDownload, map[string]any{
			"version":  snapshot.LatestVersion,
			"fileName": fileName,
			"stage":    "downloading",
		})
		s.tasks.Start(task.TaskID, "Downloading app update installer")
	}

	snapshot.ArtifactStatus = "downloading"
	snapshot.DownloadedVersion = snapshot.LatestVersion
	snapshot.DownloadedFileName = fileName
	snapshot.DownloadedFilePath = targetPath
	snapshot.DownloadedBytes = 0
	snapshot.TotalBytes = 0
	snapshot.SignatureStatus = "not_checked"
	snapshot.InstallReady = false
	snapshot.LastInstallError = ""
	if err := s.store.UpsertAppUpdateStatusSnapshot(ctx, snapshot); err != nil {
		return contracts.AppUpdateStatusDTO{}, err
	}

	downloadCtx, cancel := context.WithTimeout(ctx, defaultDownloadTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(downloadCtx, http.MethodGet, snapshot.InstallerDownloadURL, nil)
	if err != nil {
		return s.markDownloadFailed(ctx, snapshot, task.TaskID, err)
	}
	req.Header.Set("User-Agent", "Curated/"+firstNonEmpty(strings.TrimSpace(version.PackageVersion()), "dev"))

	downloadClient := *s.httpClient
	downloadClient.Timeout = defaultDownloadTimeout
	resp, err := downloadClient.Do(req)
	if err != nil {
		return s.markDownloadFailed(ctx, snapshot, task.TaskID, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return s.markDownloadFailed(ctx, snapshot, task.TaskID, fmt.Errorf("installer download failed with HTTP %d", resp.StatusCode))
	}

	partFile, err := os.OpenFile(partPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return s.markDownloadFailed(ctx, snapshot, task.TaskID, err)
	}

	hasher := sha256.New()
	buffer := make([]byte, 1024*1024)
	var downloaded int64
	totalBytes := resp.ContentLength
	var copyErr error
	for {
		nr, er := resp.Body.Read(buffer)
		if nr > 0 {
			chunk := buffer[:nr]
			if _, err := partFile.Write(chunk); err != nil {
				copyErr = err
				break
			}
			if _, err := hasher.Write(chunk); err != nil {
				copyErr = err
				break
			}
			downloaded += int64(nr)
			if s.tasks != nil && task.TaskID != "" {
				progress := downloadProgress(downloaded, totalBytes)
				s.tasks.ProgressWithMetadata(task.TaskID, progress, "Downloading app update installer", map[string]any{
					"downloadedBytes": downloaded,
					"totalBytes":      totalBytes,
					"stage":           "downloading",
				})
			}
		}
		if er != nil {
			if errors.Is(er, io.EOF) {
				break
			}
			copyErr = er
			break
		}
	}
	if closeErr := partFile.Close(); copyErr == nil && closeErr != nil {
		copyErr = closeErr
	}
	if copyErr != nil {
		return s.markDownloadFailed(ctx, snapshot, task.TaskID, copyErr)
	}

	actualSHA256 := fmt.Sprintf("%X", hasher.Sum(nil))
	expectedSHA256 := normalizeSHA256Digest(snapshot.InstallerSHA256)
	if actualSHA256 != expectedSHA256 {
		_ = os.Remove(partPath)
		return s.markDownloadFailed(ctx, snapshot, task.TaskID, fmt.Errorf("installer checksum mismatch"))
	}
	if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
		return s.markDownloadFailed(ctx, snapshot, task.TaskID, err)
	}
	if err := os.Rename(partPath, targetPath); err != nil {
		return s.markDownloadFailed(ctx, snapshot, task.TaskID, err)
	}

	snapshot.ArtifactStatus = "verified"
	snapshot.DownloadedVersion = snapshot.LatestVersion
	snapshot.DownloadedFileName = fileName
	snapshot.DownloadedFilePath = targetPath
	snapshot.DownloadedBytes = downloaded
	if totalBytes > 0 {
		snapshot.TotalBytes = totalBytes
	} else {
		snapshot.TotalBytes = downloaded
	}
	snapshot.SignatureStatus = "not_checked"
	snapshot.InstallReady = true
	snapshot.LastInstallError = ""
	if err := s.store.UpsertAppUpdateStatusSnapshot(ctx, snapshot); err != nil {
		return contracts.AppUpdateStatusDTO{}, err
	}
	if s.tasks != nil && task.TaskID != "" {
		s.tasks.Complete(task.TaskID, "App update installer downloaded")
	}
	return snapshotToDTO(snapshot), nil
}

// Install launches a verified downloaded installer using the requested mode.
func (s *Service) Install(ctx context.Context, req contracts.AppUpdateInstallRequest) (contracts.AppUpdateStatusDTO, error) {
	snapshot, ok, err := s.store.GetAppUpdateStatusSnapshot(ctx)
	if err != nil {
		return contracts.AppUpdateStatusDTO{}, err
	}
	if !ok || !snapshot.InstallReady || snapshot.ArtifactStatus != "verified" {
		return contracts.AppUpdateStatusDTO{}, errors.New("verified installer is not ready")
	}
	if strings.TrimSpace(snapshot.DownloadedFilePath) == "" {
		return contracts.AppUpdateStatusDTO{}, errors.New("downloaded installer path is unavailable")
	}
	if _, err := os.Stat(snapshot.DownloadedFilePath); err != nil {
		snapshot.LastInstallError = err.Error()
		_ = s.store.UpsertAppUpdateStatusSnapshot(ctx, snapshot)
		return snapshotToDTO(snapshot), err
	}

	mode := normalizeInstallMode(req.Mode)
	args, err := installerArgs(mode)
	if err != nil {
		return contracts.AppUpdateStatusDTO{}, err
	}
	snapshot.LastInstallAttemptAt = s.now().UTC().Format(time.RFC3339)
	snapshot.LastInstallError = ""
	if err := launchInstallerProcess(ctx, snapshot.DownloadedFilePath, args); err != nil {
		snapshot.LastInstallError = err.Error()
		_ = s.store.UpsertAppUpdateStatusSnapshot(ctx, snapshot)
		return snapshotToDTO(snapshot), err
	}

	snapshot.ArtifactStatus = "install-launched"
	snapshot.InstallReady = false
	if err := s.store.UpsertAppUpdateStatusSnapshot(ctx, snapshot); err != nil {
		return contracts.AppUpdateStatusDTO{}, err
	}
	return snapshotToDTO(snapshot), nil
}

// ClearDownloadedInstaller removes cached installer metadata and the downloaded file when present.
func (s *Service) ClearDownloadedInstaller(ctx context.Context) (contracts.AppUpdateStatusDTO, error) {
	snapshot, ok, err := s.store.GetAppUpdateStatusSnapshot(ctx)
	if err != nil {
		return contracts.AppUpdateStatusDTO{}, err
	}
	if !ok {
		return s.unsupportedStatus(), nil
	}
	if strings.TrimSpace(snapshot.DownloadedFilePath) != "" {
		_ = os.Remove(snapshot.DownloadedFilePath)
		_ = os.Remove(snapshot.DownloadedFilePath + ".part")
	}
	clearArtifactState(&snapshot)
	if err := s.store.UpsertAppUpdateStatusSnapshot(ctx, snapshot); err != nil {
		return contracts.AppUpdateStatusDTO{}, err
	}
	return snapshotToDTO(snapshot), nil
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
		if ok && snapshot.InstalledVersion == installedVersion && s.isSnapshotFresh(snapshot.CheckedAt) && !hasLegacyReleaseNotesSnippet(snapshot) {
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
			InstalledVersion:     installedVersion,
			LatestVersion:        existing.LatestVersion,
			Status:               "error",
			CheckedAt:            s.now().UTC().Format(time.RFC3339),
			PublishedAt:          existing.PublishedAt,
			ReleaseName:          existing.ReleaseName,
			ReleaseURL:           firstNonEmpty(existing.ReleaseURL, s.releasePageURL),
			InstallerDownloadURL: existing.InstallerDownloadURL,
			ReleaseNotesSnippet:  existing.ReleaseNotesSnippet,
			Source:               updateSourceGitHubReleases,
			ErrorMessage:         err.Error(),
		}
		if saveErr := s.store.UpsertAppUpdateStatusSnapshot(ctx, snapshot); saveErr != nil {
			return contracts.AppUpdateStatusDTO{}, saveErr
		}
		return snapshotToDTO(snapshot), nil
	}

	latestVersion, latestSemver, err := normalizedSemver(release.TagName)
	if err != nil {
		installer := resolveInstallerAsset(release)
		snapshot := storage.AppUpdateStatusSnapshot{
			InstalledVersion:     installedVersion,
			Status:               "error",
			CheckedAt:            s.now().UTC().Format(time.RFC3339),
			PublishedAt:          release.PublishedAt,
			ReleaseName:          firstNonEmpty(strings.TrimSpace(release.Name), strings.TrimSpace(release.TagName)),
			ReleaseURL:           firstNonEmpty(strings.TrimSpace(release.HTMLURL), s.releasePageURL),
			InstallerDownloadURL: installer.DownloadURL,
			InstallerSHA256:      installer.SHA256,
			Source:               updateSourceGitHubReleases,
			ErrorMessage:         err.Error(),
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

	installer := resolveInstallerAsset(release)
	snapshot := storage.AppUpdateStatusSnapshot{
		InstalledVersion:     installedVersion,
		LatestVersion:        latestVersion,
		Status:               status,
		CheckedAt:            s.now().UTC().Format(time.RFC3339),
		PublishedAt:          strings.TrimSpace(release.PublishedAt),
		ReleaseName:          firstNonEmpty(strings.TrimSpace(release.Name), strings.TrimSpace(release.TagName)),
		ReleaseURL:           firstNonEmpty(strings.TrimSpace(release.HTMLURL), s.releasePageURL),
		InstallerDownloadURL: installer.DownloadURL,
		InstallerSHA256:      installer.SHA256,
		ReleaseNotesSnippet:  normalizeReleaseNotesForCache(release.Body),
		Source:               updateSourceGitHubReleases,
	}
	if existing.DownloadedVersion == latestVersion && existing.ArtifactStatus != "" {
		copyArtifactState(&snapshot, existing)
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

func hasLegacyReleaseNotesSnippet(snapshot storage.AppUpdateStatusSnapshot) bool {
	notes := strings.TrimSpace(snapshot.ReleaseNotesSnippet)
	if notes == "" || strings.Contains(notes, "\n") {
		return false
	}
	if strings.HasSuffix(notes, "...") && len([]rune(notes)) <= 280 {
		return true
	}
	if !strings.HasPrefix(notes, "#") {
		return false
	}

	title := normalizeReleaseNotesTitle(notes)
	if title == "" {
		return false
	}
	if normalizeReleaseNotesTitle(snapshot.ReleaseName) == title {
		return true
	}
	latest := normalizeReleaseNotesTitle(snapshot.LatestVersion)
	return latest != "" && strings.Contains(title, latest)
}

func normalizeReleaseNotesTitle(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimLeft(value, "#")
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "v")
	return strings.ToLower(value)
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
		Supported:            snapshot.Status != "unsupported",
		Status:               snapshot.Status,
		InstalledVersion:     snapshot.InstalledVersion,
		LatestVersion:        snapshot.LatestVersion,
		CheckedAt:            snapshot.CheckedAt,
		PublishedAt:          snapshot.PublishedAt,
		ReleaseName:          snapshot.ReleaseName,
		ReleaseURL:           snapshot.ReleaseURL,
		InstallerDownloadURL: snapshot.InstallerDownloadURL,
		InstallerSHA256:      snapshot.InstallerSHA256,
		ArtifactStatus:       snapshot.ArtifactStatus,
		DownloadedVersion:    snapshot.DownloadedVersion,
		DownloadedFileName:   snapshot.DownloadedFileName,
		DownloadedBytes:      snapshot.DownloadedBytes,
		TotalBytes:           snapshot.TotalBytes,
		SignatureStatus:      snapshot.SignatureStatus,
		InstallReady:         snapshot.InstallReady,
		LastInstallAttemptAt: snapshot.LastInstallAttemptAt,
		LastInstallError:     snapshot.LastInstallError,
		ReleaseNotesSnippet:  snapshot.ReleaseNotesSnippet,
		Source:               snapshot.Source,
		ErrorMessage:         snapshot.ErrorMessage,
	}
	dto.HasUpdate = snapshot.Status == "update-available"
	dto.DownloadProgress = downloadProgress(snapshot.DownloadedBytes, snapshot.TotalBytes)
	return dto
}

func resolveInstallerDownloadURL(release latestReleaseResponse) string {
	return resolveInstallerAsset(release).DownloadURL
}

type resolvedInstallerAsset struct {
	Name        string
	DownloadURL string
	SHA256      string
}

func resolveInstallerAsset(release latestReleaseResponse) resolvedInstallerAsset {
	var fallback string
	var fallbackName string
	var fallbackSHA256 string
	for _, asset := range release.Assets {
		name := strings.TrimSpace(asset.Name)
		downloadURL := strings.TrimSpace(asset.BrowserDownloadURL)
		if downloadURL == "" {
			continue
		}

		normalizedName := strings.ToLower(name)
		normalizedURL := strings.ToLower(strings.Split(downloadURL, "?")[0])
		if !strings.HasSuffix(normalizedName, ".exe") && !strings.HasSuffix(normalizedURL, ".exe") {
			continue
		}
		if fallback == "" {
			fallback = downloadURL
			fallbackName = name
			fallbackSHA256 = normalizeSHA256Digest(asset.Digest)
		}
		searchText := normalizedName + " " + normalizedURL
		if strings.Contains(searchText, "setup") || strings.Contains(searchText, "installer") {
			return resolvedInstallerAsset{Name: name, DownloadURL: downloadURL, SHA256: normalizeSHA256Digest(asset.Digest)}
		}
	}
	return resolvedInstallerAsset{Name: fallbackName, DownloadURL: fallback, SHA256: fallbackSHA256}
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

func normalizeReleaseNotesForCache(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}
	body = strings.ReplaceAll(body, "\r\n", "\n")
	const maxRunes = 100_000
	runes := []rune(body)
	if len(runes) <= maxRunes {
		return body
	}
	return strings.TrimSpace(string(runes[:maxRunes])) + "\n\n…"
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

func (s *Service) markDownloadFailed(ctx context.Context, snapshot storage.AppUpdateStatusSnapshot, taskID string, err error) (contracts.AppUpdateStatusDTO, error) {
	snapshot.ArtifactStatus = "failed"
	snapshot.InstallReady = false
	if err != nil {
		snapshot.LastInstallError = err.Error()
	}
	_ = s.store.UpsertAppUpdateStatusSnapshot(ctx, snapshot)
	if s.tasks != nil && taskID != "" {
		s.tasks.Fail(taskID, contracts.ErrorCodeAppUpdateDownloadFailed, snapshot.LastInstallError)
	}
	return snapshotToDTO(snapshot), err
}

func copyArtifactState(dst *storage.AppUpdateStatusSnapshot, src storage.AppUpdateStatusSnapshot) {
	dst.ArtifactStatus = src.ArtifactStatus
	dst.DownloadedVersion = src.DownloadedVersion
	dst.DownloadedFileName = src.DownloadedFileName
	dst.DownloadedFilePath = src.DownloadedFilePath
	dst.DownloadedBytes = src.DownloadedBytes
	dst.TotalBytes = src.TotalBytes
	dst.SignatureStatus = src.SignatureStatus
	dst.InstallReady = src.InstallReady
	dst.LastInstallAttemptAt = src.LastInstallAttemptAt
	dst.LastInstallError = src.LastInstallError
}

func clearArtifactState(snapshot *storage.AppUpdateStatusSnapshot) {
	snapshot.ArtifactStatus = ""
	snapshot.DownloadedVersion = ""
	snapshot.DownloadedFileName = ""
	snapshot.DownloadedFilePath = ""
	snapshot.DownloadedBytes = 0
	snapshot.TotalBytes = 0
	snapshot.SignatureStatus = ""
	snapshot.InstallReady = false
	snapshot.LastInstallAttemptAt = ""
	snapshot.LastInstallError = ""
}

func normalizeInstallMode(mode string) string {
	mode = strings.ToLower(strings.TrimSpace(mode))
	switch mode {
	case "silent", "verysilent":
		return mode
	default:
		return "interactive"
	}
}

func installerArgs(mode string) ([]string, error) {
	switch mode {
	case "interactive":
		return []string{"/NORESTART", "/SP-", "/CLOSEAPPLICATIONS"}, nil
	case "silent":
		return []string{"/SILENT", "/NORESTART", "/SP-", "/CLOSEAPPLICATIONS"}, nil
	case "verysilent":
		return []string{"/VERYSILENT", "/SUPPRESSMSGBOXES", "/NORESTART", "/SP-", "/CLOSEAPPLICATIONS"}, nil
	default:
		return nil, fmt.Errorf("unsupported install mode: %s", mode)
	}
}

func defaultUpdateCacheDir() string {
	if dir, err := os.UserCacheDir(); err == nil && strings.TrimSpace(dir) != "" {
		return filepath.Join(dir, "Curated", "updates")
	}
	return filepath.Join(os.TempDir(), "Curated", "updates")
}

func installerFileName(downloadURL string, versionValue string) string {
	if parsed, err := url.Parse(downloadURL); err == nil {
		name := sanitizeInstallerFileName(filepath.Base(parsed.Path))
		if strings.HasSuffix(strings.ToLower(name), ".exe") {
			return name
		}
	}
	versionValue = strings.TrimSpace(versionValue)
	if versionValue == "" {
		versionValue = "update"
	}
	return "Curated-Setup-" + sanitizeInstallerFileName(versionValue) + ".exe"
}

func sanitizeInstallerFileName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == string(filepath.Separator) {
		return ""
	}
	replacer := strings.NewReplacer("\\", "_", "/", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_")
	name = replacer.Replace(name)
	name = strings.Trim(name, " .")
	return name
}

func validateInstallerDownloadURL(raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return err
	}
	if parsed.Scheme == "https" {
		return nil
	}
	if parsed.Scheme == "http" && isLoopbackHost(parsed.Hostname()) {
		return nil
	}
	return fmt.Errorf("installer download url must use https")
}

func isLoopbackHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func normalizeSHA256Digest(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(strings.ToLower(value), "sha256:")
	value = strings.TrimSpace(value)
	if len(value) != 64 {
		return ""
	}
	for _, r := range value {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return ""
		}
	}
	return strings.ToUpper(value)
}

func downloadProgress(downloaded int64, total int64) int {
	if total <= 0 || downloaded <= 0 {
		return 0
	}
	progress := int((downloaded * 100) / total)
	if progress < 0 {
		return 0
	}
	if progress > 100 {
		return 100
	}
	return progress
}

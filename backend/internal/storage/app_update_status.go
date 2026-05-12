package storage

import (
	"context"
	"database/sql"
)

const appUpdateStatusKey = "app-update"

// AppUpdateStatusSnapshot holds the cached packaged-app update check result.
type AppUpdateStatusSnapshot struct {
	InstalledVersion     string
	LatestVersion        string
	Status               string
	CheckedAt            string
	PublishedAt          string
	ReleaseName          string
	ReleaseURL           string
	InstallerDownloadURL string
	InstallerSHA256      string
	ArtifactStatus       string
	DownloadedVersion    string
	DownloadedFileName   string
	DownloadedFilePath   string
	DownloadedBytes      int64
	TotalBytes           int64
	SignatureStatus      string
	InstallReady         bool
	LastInstallAttemptAt string
	LastInstallError     string
	ReleaseNotesSnippet  string
	Source               string
	ErrorMessage         string
}

// GetAppUpdateStatusSnapshot returns the cached app update status. The bool is false when no snapshot exists.
func (s *SQLiteStore) GetAppUpdateStatusSnapshot(ctx context.Context) (AppUpdateStatusSnapshot, bool, error) {
	var snapshot AppUpdateStatusSnapshot
	var installReadyInt int

	err := s.db.QueryRowContext(ctx, `
		SELECT
			installed_version,
			latest_version,
			status,
			checked_at,
			published_at,
			release_name,
			release_url,
			installer_download_url,
			installer_sha256,
			artifact_status,
			downloaded_version,
			downloaded_file_name,
			downloaded_file_path,
			downloaded_bytes,
			total_bytes,
			signature_status,
			install_ready,
			last_install_attempt_at,
			last_install_error,
			release_notes_snippet,
			source,
			error_message
		FROM app_update_status
		WHERE status_key = ?
	`, appUpdateStatusKey).Scan(
		&snapshot.InstalledVersion,
		&snapshot.LatestVersion,
		&snapshot.Status,
		&snapshot.CheckedAt,
		&snapshot.PublishedAt,
		&snapshot.ReleaseName,
		&snapshot.ReleaseURL,
		&snapshot.InstallerDownloadURL,
		&snapshot.InstallerSHA256,
		&snapshot.ArtifactStatus,
		&snapshot.DownloadedVersion,
		&snapshot.DownloadedFileName,
		&snapshot.DownloadedFilePath,
		&snapshot.DownloadedBytes,
		&snapshot.TotalBytes,
		&snapshot.SignatureStatus,
		&installReadyInt,
		&snapshot.LastInstallAttemptAt,
		&snapshot.LastInstallError,
		&snapshot.ReleaseNotesSnippet,
		&snapshot.Source,
		&snapshot.ErrorMessage,
	)
	if err == sql.ErrNoRows {
		return AppUpdateStatusSnapshot{}, false, nil
	}
	if err != nil {
		return AppUpdateStatusSnapshot{}, false, err
	}

	snapshot.InstallReady = installReadyInt != 0
	return snapshot, true, nil
}

// UpsertAppUpdateStatusSnapshot inserts or replaces the cached app update status snapshot.
func (s *SQLiteStore) UpsertAppUpdateStatusSnapshot(ctx context.Context, snapshot AppUpdateStatusSnapshot) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO app_update_status (
			status_key,
			installed_version,
			latest_version,
			status,
			checked_at,
			published_at,
			release_name,
			release_url,
			installer_download_url,
			installer_sha256,
			artifact_status,
			downloaded_version,
			downloaded_file_name,
			downloaded_file_path,
			downloaded_bytes,
			total_bytes,
			signature_status,
			install_ready,
			last_install_attempt_at,
			last_install_error,
			release_notes_snippet,
			source,
			error_message
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(status_key) DO UPDATE SET
			installed_version = excluded.installed_version,
			latest_version = excluded.latest_version,
			status = excluded.status,
			checked_at = excluded.checked_at,
			published_at = excluded.published_at,
			release_name = excluded.release_name,
			release_url = excluded.release_url,
			installer_download_url = excluded.installer_download_url,
			installer_sha256 = excluded.installer_sha256,
			artifact_status = excluded.artifact_status,
			downloaded_version = excluded.downloaded_version,
			downloaded_file_name = excluded.downloaded_file_name,
			downloaded_file_path = excluded.downloaded_file_path,
			downloaded_bytes = excluded.downloaded_bytes,
			total_bytes = excluded.total_bytes,
			signature_status = excluded.signature_status,
			install_ready = excluded.install_ready,
			last_install_attempt_at = excluded.last_install_attempt_at,
			last_install_error = excluded.last_install_error,
			release_notes_snippet = excluded.release_notes_snippet,
			source = excluded.source,
			error_message = excluded.error_message
	`,
		appUpdateStatusKey,
		snapshot.InstalledVersion,
		snapshot.LatestVersion,
		snapshot.Status,
		snapshot.CheckedAt,
		snapshot.PublishedAt,
		snapshot.ReleaseName,
		snapshot.ReleaseURL,
		snapshot.InstallerDownloadURL,
		snapshot.InstallerSHA256,
		snapshot.ArtifactStatus,
		snapshot.DownloadedVersion,
		snapshot.DownloadedFileName,
		snapshot.DownloadedFilePath,
		snapshot.DownloadedBytes,
		snapshot.TotalBytes,
		snapshot.SignatureStatus,
		boolToInt(snapshot.InstallReady),
		snapshot.LastInstallAttemptAt,
		snapshot.LastInstallError,
		snapshot.ReleaseNotesSnippet,
		snapshot.Source,
		snapshot.ErrorMessage,
	)
	return err
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

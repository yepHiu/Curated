package storage

import (
	"context"
	"database/sql"
)

const appUpdateStatusKey = "app-update"

type AppUpdateStatusSnapshot struct {
	InstalledVersion    string
	LatestVersion       string
	Status              string
	CheckedAt           string
	PublishedAt         string
	ReleaseName         string
	ReleaseURL          string
	ReleaseNotesSnippet string
	Source              string
	ErrorMessage        string
}

func (s *SQLiteStore) GetAppUpdateStatusSnapshot(ctx context.Context) (AppUpdateStatusSnapshot, bool, error) {
	var snapshot AppUpdateStatusSnapshot

	err := s.db.QueryRowContext(ctx, `
		SELECT
			installed_version,
			latest_version,
			status,
			checked_at,
			published_at,
			release_name,
			release_url,
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

	return snapshot, true, nil
}

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
			release_notes_snippet,
			source,
			error_message
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(status_key) DO UPDATE SET
			installed_version = excluded.installed_version,
			latest_version = excluded.latest_version,
			status = excluded.status,
			checked_at = excluded.checked_at,
			published_at = excluded.published_at,
			release_name = excluded.release_name,
			release_url = excluded.release_url,
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
		snapshot.ReleaseNotesSnippet,
		snapshot.Source,
		snapshot.ErrorMessage,
	)
	return err
}

package storage

import (
	"context"
	"database/sql"
	"strings"

	"curated-backend/internal/storagehealth"
)

// UpsertLibraryPathStorageBinding stores the expected backing-volume identity for a library path.
func (s *SQLiteStore) UpsertLibraryPathStorageBinding(ctx context.Context, b storagehealth.Binding) error {
	id := strings.TrimSpace(b.LibraryPathID)
	root := strings.TrimSpace(b.RootPath)
	if id == "" || root == "" {
		return nil
	}
	ts := nowUTC()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO library_path_storage_bindings (
			library_path_id,
			root_path,
			volume_id,
			volume_label,
			file_system,
			drive_type,
			identity_confidence,
			bound_at,
			last_seen_at,
			last_checked_at,
			last_status,
			last_error,
			updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, COALESCE(NULLIF(?, ''), ?), ?, ?, ?, ?, ?)
		ON CONFLICT(library_path_id) DO UPDATE SET
			root_path = excluded.root_path,
			volume_id = excluded.volume_id,
			volume_label = excluded.volume_label,
			file_system = excluded.file_system,
			drive_type = excluded.drive_type,
			identity_confidence = excluded.identity_confidence,
			last_seen_at = excluded.last_seen_at,
			last_checked_at = excluded.last_checked_at,
			last_status = excluded.last_status,
			last_error = excluded.last_error,
			updated_at = excluded.updated_at
	`, id, root, nullableText(b.VolumeID), nullableText(b.VolumeLabel), nullableText(b.FileSystem),
		nullableText(b.DriveType), defaultText(b.IdentityConfidence, "unknown"), "", ts, ts, ts, "", "", ts)
	return err
}

// GetLibraryPathStorageBinding returns the persisted expected storage identity for a library path.
func (s *SQLiteStore) GetLibraryPathStorageBinding(ctx context.Context, libraryPathID string) (storagehealth.Binding, bool, error) {
	id := strings.TrimSpace(libraryPathID)
	if id == "" {
		return storagehealth.Binding{}, false, nil
	}

	var b storagehealth.Binding
	var volumeID, volumeLabel, fileSystem, driveType sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT library_path_id, root_path, volume_id, volume_label, file_system, drive_type, identity_confidence
		FROM library_path_storage_bindings
		WHERE library_path_id = ?
	`, id).Scan(&b.LibraryPathID, &b.RootPath, &volumeID, &volumeLabel, &fileSystem, &driveType, &b.IdentityConfidence)
	if err != nil {
		if err == sql.ErrNoRows {
			return storagehealth.Binding{}, false, nil
		}
		return storagehealth.Binding{}, false, err
	}
	b.VolumeID = nullStringValue(volumeID)
	b.VolumeLabel = nullStringValue(volumeLabel)
	b.FileSystem = nullStringValue(fileSystem)
	b.DriveType = nullStringValue(driveType)
	return b, true, nil
}

func nullableText(v string) any {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return v
}

func defaultText(v string, fallback string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return fallback
	}
	return v
}

func nullStringValue(v sql.NullString) string {
	if !v.Valid {
		return ""
	}
	return v.String
}

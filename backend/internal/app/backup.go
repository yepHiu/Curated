package app

import (
	"context"
	"errors"
	"time"

	"curated-backend/internal/backup"
	"curated-backend/internal/contracts"
	"curated-backend/internal/version"
)

func (a *App) CreateBackup(ctx context.Context, destinationPath string) (contracts.BackupManifestDTO, error) {
	manifest, err := backup.Create(ctx, backup.CreateOptions{
		Store:             a.store,
		DestinationPath:   destinationPath,
		LibraryConfigPath: a.librarySettingsPath,
		AppVersion:        version.PackageVersion(),
		AppChannel:        version.Channel,
	})
	if errors.Is(err, backup.ErrDestinationExists) {
		return contracts.BackupManifestDTO{}, contracts.ErrBackupDestinationExists
	}
	if err != nil {
		return contracts.BackupManifestDTO{}, err
	}
	return backupManifestDTO(manifest), nil
}

func (a *App) VerifyBackup(ctx context.Context, backupPath string) (contracts.BackupVerificationDTO, error) {
	verification, err := backup.Verify(ctx, backupPath)
	if err != nil {
		return contracts.BackupVerificationDTO{}, err
	}
	return backupVerificationDTO(verification), nil
}

func (a *App) PreflightBackupRestore(ctx context.Context, backupPath string) (contracts.BackupRestorePreflightDTO, error) {
	preflight, err := backup.PreflightRestore(ctx, backup.PreflightOptions{
		BackupPath:     backupPath,
		TargetDatabase: a.cfg.DatabasePath,
		TargetConfig:   a.librarySettingsPath,
	})
	if err != nil {
		return contracts.BackupRestorePreflightDTO{}, err
	}
	return backupPreflightDTO(preflight), nil
}

func backupManifestDTO(manifest backup.Manifest) contracts.BackupManifestDTO {
	files := make([]contracts.BackupFileDTO, 0, len(manifest.Files))
	for _, file := range manifest.Files {
		files = append(files, contracts.BackupFileDTO{
			Kind:      file.Kind,
			Path:      file.Path,
			SizeBytes: file.SizeBytes,
			SHA256:    file.SHA256,
		})
	}
	return contracts.BackupManifestDTO{
		Format:        manifest.Format,
		FormatVersion: manifest.FormatVersion,
		CreatedAt:     manifest.CreatedAt.UTC().Format(time.RFC3339Nano),
		AppVersion:    manifest.AppVersion,
		AppChannel:    manifest.AppChannel,
		Scope: contracts.BackupScopeDTO{
			DatabaseIncluded:      manifest.Scope.DatabaseIncluded,
			LibraryConfigIncluded: manifest.Scope.LibraryConfigIncluded,
			UserAssetsIncluded:    manifest.Scope.UserAssetsIncluded,
			MediaFilesIncluded:    manifest.Scope.MediaFilesIncluded,
		},
		SchemaMigrations: append([]string{}, manifest.SchemaMigrations...),
		Files:            files,
	}
}

func backupVerificationDTO(verification backup.Verification) contracts.BackupVerificationDTO {
	var manifest *contracts.BackupManifestDTO
	if verification.Manifest != nil {
		mapped := backupManifestDTO(*verification.Manifest)
		manifest = &mapped
	}
	return contracts.BackupVerificationDTO{
		Valid:     verification.Valid,
		CheckedAt: verification.CheckedAt.UTC().Format(time.RFC3339Nano),
		Manifest:  manifest,
		DatabaseIntegrity: contracts.BackupIntegrityDTO{
			QuickCheck:           verification.DatabaseIntegrity.QuickCheck,
			ForeignKeyViolations: verification.DatabaseIntegrity.ForeignKeyViolations,
		},
		Errors:   append([]string{}, verification.Errors...),
		Warnings: append([]string{}, verification.Warnings...),
	}
}

func backupPreflightDTO(preflight backup.RestorePreflight) contracts.BackupRestorePreflightDTO {
	return contracts.BackupRestorePreflightDTO{
		CanRestore:            preflight.CanRestore,
		CheckedAt:             preflight.CheckedAt.UTC().Format(time.RFC3339Nano),
		Verification:          backupVerificationDTO(preflight.Verification),
		TargetDatabase:        preflight.TargetDatabase,
		TargetDatabaseExists:  preflight.TargetDatabaseExists,
		TargetConfig:          preflight.TargetConfig,
		TargetConfigExists:    preflight.TargetConfigExists,
		RequiredBytes:         preflight.RequiredBytes,
		AvailableBytes:        preflight.AvailableBytes,
		AvailableBytesKnown:   preflight.AvailableBytesKnown,
		UnsupportedMigrations: append([]string{}, preflight.UnsupportedMigrations...),
		Errors:                append([]string{}, preflight.Errors...),
		Warnings:              append([]string{}, preflight.Warnings...),
	}
}

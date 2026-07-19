package maintenance

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"curated-backend/internal/backup"
	"curated-backend/internal/pathmigration"
	"curated-backend/internal/processlock"
	"curated-backend/internal/storage"
)

func runPathMigration(ctx context.Context, options Options, output io.Writer, apply bool) error {
	if apply && !options.ConfirmPathMigration {
		return errors.New("path-migrate-apply requires -confirm-path-migration")
	}
	mapping, err := pathmigration.NewMapping(options.PathFrom, options.PathTo)
	if err != nil {
		return err
	}
	if _, err := os.Stat(options.DatabasePath); err != nil {
		return fmt.Errorf("inspect path migration database: %w", err)
	}
	lock, err := processlock.Acquire(options.DatabasePath + ".runtime.lock")
	if err != nil {
		if errors.Is(err, processlock.ErrAlreadyLocked) {
			return fmt.Errorf("Curated is still using the database; fully exit the app before path migration: %w", err)
		}
		return fmt.Errorf("acquire offline path migration lock: %w", err)
	}
	defer func() { _ = lock.Release() }()

	store, err := storage.NewSQLiteStore(options.DatabasePath)
	if err != nil {
		return fmt.Errorf("open path migration database: %w", err)
	}
	defer func() { _ = store.Close() }()
	if _, err := store.CheckIntegrity(ctx); err != nil {
		return fmt.Errorf("path migration source integrity: %w", err)
	}
	pathOptions := storage.PathMigrationOptions{
		Mapping:      mapping,
		AllowMissing: options.AllowMissingPaths,
		Now:          options.Now,
	}
	plan, err := store.PlanPathMigration(ctx, pathOptions)
	if err != nil {
		return err
	}
	if !apply {
		if err := writeJSON(output, map[string]any{
			"action":       ActionPathMigratePlan,
			"databasePath": options.DatabasePath,
			"plan":         plan,
		}); err != nil {
			return err
		}
		if !plan.CanApply {
			return ErrPathMigrationBlocked
		}
		return nil
	}
	if !plan.CanApply {
		if err := writeJSON(output, map[string]any{
			"action":       ActionPathMigrateApply,
			"databasePath": options.DatabasePath,
			"plan":         plan,
		}); err != nil {
			return err
		}
		return ErrPathMigrationBlocked
	}
	absBackupPath, err := requiredAbsolutePath("-backup-path", options.BackupPath)
	if err != nil {
		return err
	}
	manifest, err := backup.Create(ctx, backup.CreateOptions{
		Store:             store,
		DestinationPath:   absBackupPath,
		LibraryConfigPath: options.LibraryConfigPath,
		AppVersion:        options.AppVersion,
		AppChannel:        options.AppChannel,
		Now:               options.Now,
	})
	if err != nil {
		return fmt.Errorf("create pre-migration backup: %w", err)
	}
	verification, err := backup.Verify(ctx, absBackupPath)
	if err != nil {
		return fmt.Errorf("verify pre-migration backup: %w", err)
	}
	if !verification.Valid {
		return fmt.Errorf("verify pre-migration backup: %w", ErrVerificationFailed)
	}
	pathOptions.BackupPath = absBackupPath
	result, err := store.ApplyPathMigration(ctx, pathOptions)
	if err != nil {
		if errors.Is(err, storage.ErrPathMigrationBlocked) {
			if writeErr := writeJSON(output, map[string]any{
				"action":       ActionPathMigrateApply,
				"databasePath": options.DatabasePath,
				"backupPath":   absBackupPath,
				"migration":    result,
			}); writeErr != nil {
				return writeErr
			}
			return ErrPathMigrationBlocked
		}
		return err
	}
	if _, err := store.CheckIntegrity(ctx); err != nil {
		return fmt.Errorf("post-migration integrity check failed; restore from %s: %w", absBackupPath, err)
	}
	return writeJSON(output, map[string]any{
		"action":       ActionPathMigrateApply,
		"databasePath": options.DatabasePath,
		"backupPath":   absBackupPath,
		"manifest":     manifest,
		"verification": verification,
		"migration":    result,
		"message":      "path migration applied and audited",
	})
}

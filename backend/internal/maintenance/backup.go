// Package maintenance exposes explicit CLI-safe maintenance workflows.
package maintenance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"curated-backend/internal/backup"
	"curated-backend/internal/processlock"
	"curated-backend/internal/storage"
)

const (
	ActionBackupCreate     = "backup-create"
	ActionBackupVerify     = "backup-verify"
	ActionBackupPreflight  = "backup-preflight"
	ActionBackupRestore    = "backup-restore"
	ActionPathMigratePlan  = "path-migrate-plan"
	ActionPathMigrateApply = "path-migrate-apply"
)

var (
	ErrVerificationFailed   = errors.New("backup verification failed")
	ErrPreflightFailed      = errors.New("backup restore preflight failed")
	ErrPathMigrationBlocked = errors.New("path migration plan is blocked")
)

// Options configures one non-interactive maintenance action.
type Options struct {
	Action               string
	BackupPath           string
	DatabasePath         string
	LibraryConfigPath    string
	AppVersion           string
	AppChannel           string
	ConfirmRestore       bool
	PathFrom             string
	PathTo               string
	AllowMissingPaths    bool
	ConfirmPathMigration bool
	Now                  func() time.Time
}

// Run performs one action and writes a machine-readable JSON result.
func Run(ctx context.Context, options Options, output io.Writer) error {
	action := strings.TrimSpace(options.Action)

	switch action {
	case ActionBackupCreate:
		absBackupPath, err := requiredAbsolutePath("-backup-path", options.BackupPath)
		if err != nil {
			return err
		}
		if _, err := os.Stat(options.DatabasePath); err != nil {
			return fmt.Errorf("inspect source database: %w", err)
		}
		store, err := storage.NewSQLiteStore(options.DatabasePath)
		if err != nil {
			return fmt.Errorf("open source database: %w", err)
		}
		manifest, createErr := backup.Create(ctx, backup.CreateOptions{
			Store:             store,
			DestinationPath:   absBackupPath,
			LibraryConfigPath: options.LibraryConfigPath,
			AppVersion:        options.AppVersion,
			AppChannel:        options.AppChannel,
			Now:               options.Now,
		})
		closeErr := store.Close()
		if createErr != nil {
			return createErr
		}
		if closeErr != nil {
			return fmt.Errorf("close source database: %w", closeErr)
		}
		return writeJSON(output, map[string]any{
			"action":     action,
			"backupPath": absBackupPath,
			"manifest":   manifest,
		})
	case ActionBackupVerify:
		absBackupPath, err := requiredAbsolutePath("-backup-path", options.BackupPath)
		if err != nil {
			return err
		}
		verification, err := backup.Verify(ctx, absBackupPath)
		if err != nil {
			return err
		}
		if err := writeJSON(output, map[string]any{
			"action":       action,
			"backupPath":   absBackupPath,
			"verification": verification,
		}); err != nil {
			return err
		}
		if !verification.Valid {
			return ErrVerificationFailed
		}
		return nil
	case ActionBackupPreflight:
		absBackupPath, err := requiredAbsolutePath("-backup-path", options.BackupPath)
		if err != nil {
			return err
		}
		preflight, err := backup.PreflightRestore(ctx, backup.PreflightOptions{
			BackupPath:     absBackupPath,
			TargetDatabase: options.DatabasePath,
			TargetConfig:   options.LibraryConfigPath,
			Now:            options.Now,
		})
		if err != nil {
			return err
		}
		if err := writeJSON(output, map[string]any{
			"action":     action,
			"backupPath": absBackupPath,
			"preflight":  preflight,
		}); err != nil {
			return err
		}
		if !preflight.CanRestore {
			return ErrPreflightFailed
		}
		return nil
	case ActionBackupRestore:
		absBackupPath, err := requiredAbsolutePath("-backup-path", options.BackupPath)
		if err != nil {
			return err
		}
		lock, err := processlock.Acquire(options.DatabasePath + ".runtime.lock")
		if err != nil {
			if errors.Is(err, processlock.ErrAlreadyLocked) {
				return fmt.Errorf("Curated is still using the target database; fully exit the app before restore: %w", err)
			}
			return fmt.Errorf("acquire offline restore lock: %w", err)
		}
		defer func() { _ = lock.Release() }()
		restored, err := backup.Restore(ctx, backup.RestoreOptions{
			PreflightOptions: backup.PreflightOptions{
				BackupPath:     absBackupPath,
				TargetDatabase: options.DatabasePath,
				TargetConfig:   options.LibraryConfigPath,
				Now:            options.Now,
			},
			Confirm: options.ConfirmRestore,
		})
		if err != nil {
			return err
		}
		return writeJSON(output, map[string]any{
			"action":     action,
			"backupPath": absBackupPath,
			"restore":    restored,
		})
	case ActionPathMigratePlan:
		return runPathMigration(ctx, options, output, false)
	case ActionPathMigrateApply:
		return runPathMigration(ctx, options, output, true)
	default:
		return fmt.Errorf("unknown maintenance action %q", action)
	}
}

func requiredAbsolutePath(flagName, value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("%s is required for this maintenance action", flagName)
	}
	absPath, err := filepath.Abs(trimmed)
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", flagName, err)
	}
	return absPath, nil
}

func writeJSON(output io.Writer, value any) error {
	if output == nil {
		return errors.New("maintenance output writer is required")
	}
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return fmt.Errorf("write maintenance result: %w", err)
	}
	return nil
}

package backup

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"curated-backend/internal/storage"
)

// PreflightRestore verifies a package and assesses compatibility and capacity
// without replacing the target database or configuration.
func PreflightRestore(ctx context.Context, options PreflightOptions) (RestorePreflight, error) {
	now := time.Now
	if options.Now != nil {
		now = options.Now
	}
	result := RestorePreflight{CheckedAt: now().UTC()}
	verification, err := Verify(ctx, options.BackupPath)
	if err != nil {
		return result, err
	}
	result.Verification = verification
	if !verification.Valid || verification.Manifest == nil {
		result.Errors = append(result.Errors, "backup package verification failed")
		return finalizePreflight(result), nil
	}

	targetDatabase := strings.TrimSpace(options.TargetDatabase)
	if targetDatabase == "" {
		result.Errors = append(result.Errors, "target database path is required")
		return finalizePreflight(result), nil
	}
	result.TargetDatabase, err = filepath.Abs(targetDatabase)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("resolve target database: %v", err))
		return finalizePreflight(result), nil
	}
	backupPath, absErr := filepath.Abs(strings.TrimSpace(options.BackupPath))
	if absErr == nil && samePath(backupPath, result.TargetDatabase) {
		result.Errors = append(result.Errors, "backup package and target database paths must differ")
	}

	databaseSize, databaseExists, statErr := regularFileSize(result.TargetDatabase)
	if statErr != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("inspect target database: %v", statErr))
	} else {
		result.TargetDatabaseExists = databaseExists
		result.RequiredBytes += databaseSize
	}

	if configPath := strings.TrimSpace(options.TargetConfig); configPath != "" {
		result.TargetConfig, err = filepath.Abs(configPath)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("resolve target config: %v", err))
		} else {
			configSize, configExists, configErr := regularFileSize(result.TargetConfig)
			if configErr != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("inspect target config: %v", configErr))
			} else {
				result.TargetConfigExists = configExists
				result.RequiredBytes += configSize
			}
		}
	} else if verification.Manifest.Scope.LibraryConfigIncluded {
		result.Errors = append(result.Errors, "target config path is required because the backup includes library-config.cfg")
	}

	for _, entry := range verification.Manifest.Files {
		result.RequiredBytes += entry.SizeBytes
	}
	availableMigrations, migrationErr := storage.AvailableMigrations()
	if migrationErr != nil {
		result.Errors = append(result.Errors, migrationErr.Error())
	} else {
		available := make(map[string]struct{}, len(availableMigrations))
		for _, name := range availableMigrations {
			available[name] = struct{}{}
		}
		for _, name := range verification.Manifest.SchemaMigrations {
			if _, ok := available[name]; !ok {
				result.UnsupportedMigrations = append(result.UnsupportedMigrations, name)
			}
		}
		if len(result.UnsupportedMigrations) > 0 {
			result.Errors = append(result.Errors, "backup database contains migrations unknown to this Curated build")
		}
	}

	spacePath, parentErr := nearestExistingDirectory(filepath.Dir(result.TargetDatabase))
	if parentErr != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("locate target volume: %v", parentErr))
	} else {
		available, spaceErr := availableDiskBytes(spacePath)
		if spaceErr != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("check target disk space: %v", spaceErr))
		} else {
			result.AvailableBytesKnown = true
			result.AvailableBytes = available
			if uint64(result.RequiredBytes) > available {
				result.Errors = append(result.Errors, fmt.Sprintf("insufficient disk space: need %d bytes and have %d", result.RequiredBytes, available))
			}
		}
	}

	if result.TargetDatabaseExists {
		currentStore, openErr := storage.NewSQLiteStore(result.TargetDatabase)
		if openErr != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("current database could not be opened for integrity checking: %v", openErr))
		} else {
			if _, integrityErr := currentStore.CheckIntegrity(ctx); integrityErr != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("current database integrity check failed; rollback copy will preserve it: %v", integrityErr))
			}
			_ = currentStore.Close()
		}
	}
	if !verification.Manifest.Scope.LibraryConfigIncluded {
		result.Warnings = append(result.Warnings, "backup does not include library-config.cfg; the current config will be preserved")
	}
	if !verification.Manifest.Scope.UserAssetsIncluded {
		result.Warnings = append(result.Warnings, "backup does not include user asset files")
	}
	if !verification.Manifest.Scope.MediaFilesIncluded {
		result.Warnings = append(result.Warnings, "backup does not include media source files")
	}
	return finalizePreflight(result), nil
}

func regularFileSize(filePath string) (size int64, exists bool, err error) {
	info, err := os.Stat(filePath)
	if errors.Is(err, os.ErrNotExist) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	if !info.Mode().IsRegular() {
		return 0, true, fmt.Errorf("path is not a regular file")
	}
	return info.Size(), true, nil
}

func nearestExistingDirectory(directory string) (string, error) {
	current, err := filepath.Abs(directory)
	if err != nil {
		return "", err
	}
	for {
		info, statErr := os.Stat(current)
		if statErr == nil {
			if !info.IsDir() {
				return "", fmt.Errorf("%s is not a directory", current)
			}
			return current, nil
		}
		if !errors.Is(statErr, os.ErrNotExist) {
			return "", statErr
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("no existing parent for %s", directory)
		}
		current = parent
	}
}

func samePath(left, right string) bool {
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	if filepath.Separator == '\\' {
		return strings.EqualFold(left, right)
	}
	return left == right
}

func finalizePreflight(result RestorePreflight) RestorePreflight {
	result.CanRestore = result.Verification.Valid && len(result.Errors) == 0
	if result.Errors == nil {
		result.Errors = []string{}
	}
	if result.Warnings == nil {
		result.Warnings = []string{}
	}
	if result.UnsupportedMigrations == nil {
		result.UnsupportedMigrations = []string{}
	}
	return result
}

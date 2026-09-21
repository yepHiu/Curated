package backup

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"curated-backend/internal/storage"
)

// Restore replaces the target database and optional library config after a
// successful preflight. Callers must run it before normal application startup.
func Restore(ctx context.Context, options RestoreOptions) (RestoreResult, error) {
	var result RestoreResult
	if !options.Confirm {
		return result, errors.New("offline restore requires explicit confirmation")
	}
	preflight, err := PreflightRestore(ctx, options.PreflightOptions)
	if err != nil {
		return result, err
	}
	if !preflight.CanRestore || preflight.Verification.Manifest == nil {
		return result, fmt.Errorf("restore preflight failed: %s", strings.Join(preflight.Errors, "; "))
	}
	manifest := *preflight.Verification.Manifest

	archive, err := zip.OpenReader(strings.TrimSpace(options.BackupPath))
	if err != nil {
		return result, fmt.Errorf("reopen backup package: %w", err)
	}
	defer func() { _ = archive.Close() }()
	archiveFiles := make(map[string]*zip.File, len(archive.File))
	for _, file := range archive.File {
		archiveFiles[file.Name] = file
	}

	entryByKind := make(map[string]FileEntry, len(manifest.Files))
	for _, entry := range manifest.Files {
		entryByKind[entry.Kind] = entry
	}
	databaseEntry := entryByKind["database"]
	if err := os.MkdirAll(filepath.Dir(preflight.TargetDatabase), 0o700); err != nil {
		return result, fmt.Errorf("create target database directory: %w", err)
	}
	databaseTemp, err := extractEntryToTargetTemp(archiveFiles[databaseEntry.Path], databaseEntry, filepath.Dir(preflight.TargetDatabase))
	if err != nil {
		return result, err
	}
	defer func() { _ = os.Remove(databaseTemp) }()

	configTemp := ""
	if manifest.Scope.LibraryConfigIncluded {
		configEntry := entryByKind["library-config"]
		if err := os.MkdirAll(filepath.Dir(preflight.TargetConfig), 0o700); err != nil {
			return result, fmt.Errorf("create target config directory: %w", err)
		}
		configTemp, err = extractEntryToTargetTemp(archiveFiles[configEntry.Path], configEntry, filepath.Dir(preflight.TargetConfig))
		if err != nil {
			return result, err
		}
		defer func() { _ = os.Remove(configTemp) }()
	}

	// 不可变资产先发布；数据库替换失败时旧数据库和它的资产仍保持不变。
	if err := restoreWishlistAssets(archiveFiles, manifest, preflight.TargetDatabase); err != nil {
		return result, err
	}
	now := time.Now
	if options.Now != nil {
		now = options.Now
	}
	restoredAt := now().UTC()
	databaseRollback, err := moveExistingToRollback(preflight.TargetDatabase, restoredAt)
	if err != nil {
		return result, fmt.Errorf("prepare database rollback copy: %w", err)
	}
	if err := os.Rename(databaseTemp, preflight.TargetDatabase); err != nil {
		rollbackErr := restoreRollback(preflight.TargetDatabase, databaseRollback)
		return result, errors.Join(fmt.Errorf("install restored database: %w", err), rollbackErr)
	}
	databaseTemp = ""

	configRollback := ""
	if manifest.Scope.LibraryConfigIncluded {
		configRollback, err = moveExistingToRollback(preflight.TargetConfig, restoredAt)
		if err != nil {
			rollbackErr := restoreRollback(preflight.TargetDatabase, databaseRollback)
			return result, errors.Join(fmt.Errorf("prepare config rollback copy: %w", err), rollbackErr)
		}
		if err := os.Rename(configTemp, preflight.TargetConfig); err != nil {
			configRollbackErr := restoreRollback(preflight.TargetConfig, configRollback)
			databaseRollbackErr := restoreRollback(preflight.TargetDatabase, databaseRollback)
			return result, errors.Join(fmt.Errorf("install restored config: %w", err), configRollbackErr, databaseRollbackErr)
		}
		configTemp = ""
	}

	if err := verifyRestoredDatabase(ctx, preflight.TargetDatabase, manifest.SchemaMigrations); err != nil {
		configRollbackErr := error(nil)
		if manifest.Scope.LibraryConfigIncluded {
			configRollbackErr = restoreRollback(preflight.TargetConfig, configRollback)
		}
		databaseRollbackErr := restoreRollback(preflight.TargetDatabase, databaseRollback)
		return result, errors.Join(fmt.Errorf("verify restored database: %w", err), configRollbackErr, databaseRollbackErr)
	}

	result = RestoreResult{
		RestoredAt:           restoredAt,
		TargetDatabase:       preflight.TargetDatabase,
		DatabaseRollbackPath: databaseRollback,
		TargetConfig:         preflight.TargetConfig,
		ConfigRollbackPath:   configRollback,
		Manifest:             manifest,
	}
	return result, nil
}

func extractEntryToTargetTemp(file *zip.File, entry FileEntry, targetDir string) (string, error) {
	if file == nil {
		return "", fmt.Errorf("archive entry %s disappeared after verification", entry.Path)
	}
	temp, err := os.CreateTemp(targetDir, ".curated-restore-*.tmp")
	if err != nil {
		return "", fmt.Errorf("create restore temporary file: %w", err)
	}
	tempPath := temp.Name()
	keep := false
	defer func() {
		_ = temp.Close()
		if !keep {
			_ = os.Remove(tempPath)
		}
	}()
	if err := temp.Chmod(0o600); err != nil {
		return "", fmt.Errorf("restrict restore temporary file: %w", err)
	}
	hash := sha256.New()
	actualSize, err := copyZipFileLimited(file, io.MultiWriter(temp, hash), entry.SizeBytes)
	if err != nil {
		return "", fmt.Errorf("extract %s: %w", entry.Path, err)
	}
	if actualSize != entry.SizeBytes {
		return "", fmt.Errorf("extract %s size mismatch", entry.Path)
	}
	if !strings.EqualFold(hex.EncodeToString(hash.Sum(nil)), entry.SHA256) {
		return "", fmt.Errorf("extract %s checksum mismatch", entry.Path)
	}
	if err := temp.Sync(); err != nil {
		return "", fmt.Errorf("sync extracted %s: %w", entry.Path, err)
	}
	if err := temp.Close(); err != nil {
		return "", fmt.Errorf("close extracted %s: %w", entry.Path, err)
	}
	keep = true
	return tempPath, nil
}

func moveExistingToRollback(target string, restoredAt time.Time) (string, error) {
	if _, err := os.Stat(target); errors.Is(err, os.ErrNotExist) {
		return "", nil
	} else if err != nil {
		return "", err
	}
	stamp := restoredAt.UTC().Format("20060102T150405Z")
	for index := 0; index < 1000; index++ {
		suffix := fmt.Sprintf(".pre-restore-%s", stamp)
		if index > 0 {
			suffix += fmt.Sprintf("-%d", index)
		}
		candidate := target + suffix
		if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
			if err := os.Rename(target, candidate); err != nil {
				return "", err
			}
			return candidate, nil
		} else if err != nil {
			return "", err
		}
	}
	return "", errors.New("could not allocate a unique rollback filename")
}

func restoreRollback(target, rollback string) error {
	if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove failed restored file %s: %w", target, err)
	}
	if rollback == "" {
		return nil
	}
	if err := os.Rename(rollback, target); err != nil {
		return fmt.Errorf("restore rollback file %s: %w", rollback, err)
	}
	return nil
}

func verifyRestoredDatabase(ctx context.Context, databasePath string, expectedMigrations []string) error {
	store, err := storage.NewSQLiteStore(databasePath)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()
	if _, err := store.CheckIntegrity(ctx); err != nil {
		return err
	}
	migrations, err := store.AppliedMigrations(ctx)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(migrations, expectedMigrations) {
		return errors.New("restored database migrations differ from manifest")
	}
	return nil
}

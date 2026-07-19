package backup

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"curated-backend/internal/storage"
)

// Create writes a verified, self-describing .curated-backup package.
func Create(ctx context.Context, options CreateOptions) (Manifest, error) {
	var manifest Manifest
	if options.Store == nil {
		return manifest, errors.New("backup store is required")
	}
	destination := strings.TrimSpace(options.DestinationPath)
	if destination == "" {
		return manifest, errors.New("backup destination path is required")
	}
	absDestination, err := filepath.Abs(destination)
	if err != nil {
		return manifest, fmt.Errorf("resolve backup destination: %w", err)
	}
	if _, err := os.Stat(absDestination); err == nil {
		return manifest, fmt.Errorf("backup destination already exists: %s", absDestination)
	} else if !errors.Is(err, os.ErrNotExist) {
		return manifest, fmt.Errorf("inspect backup destination: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(absDestination), 0o700); err != nil {
		return manifest, fmt.Errorf("create backup directory: %w", err)
	}

	if _, err := options.Store.CheckIntegrity(ctx); err != nil {
		return manifest, fmt.Errorf("source database integrity check: %w", err)
	}

	workDir, err := os.MkdirTemp("", "curated-backup-create-*")
	if err != nil {
		return manifest, fmt.Errorf("create backup work directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(workDir) }()

	databasePath := filepath.Join(workDir, "curated.db")
	if err := options.Store.CreateConsistentBackup(ctx, databasePath); err != nil {
		return manifest, err
	}
	backupStore, err := storage.NewSQLiteStore(databasePath)
	if err != nil {
		return manifest, fmt.Errorf("open backup snapshot: %w", err)
	}
	if _, err := backupStore.CheckIntegrity(ctx); err != nil {
		_ = backupStore.Close()
		return manifest, fmt.Errorf("backup snapshot integrity check: %w", err)
	}
	migrations, err := backupStore.AppliedMigrations(ctx)
	closeErr := backupStore.Close()
	if err != nil {
		return manifest, err
	}
	if closeErr != nil {
		return manifest, fmt.Errorf("close backup snapshot: %w", closeErr)
	}

	databaseEntry, err := describeFile("database", DatabaseArchivePath, databasePath)
	if err != nil {
		return manifest, err
	}
	files := []FileEntry{databaseEntry}
	archiveSources := map[string]string{DatabaseArchivePath: databasePath}

	configIncluded := false
	configPath := strings.TrimSpace(options.LibraryConfigPath)
	if configPath != "" {
		info, statErr := os.Stat(configPath)
		switch {
		case statErr == nil:
			if !info.Mode().IsRegular() {
				return manifest, fmt.Errorf("library config is not a regular file: %s", configPath)
			}
			if info.Size() > maxConfigBytes {
				return manifest, fmt.Errorf("library config exceeds %d bytes", maxConfigBytes)
			}
			configContents, readErr := os.ReadFile(configPath)
			if readErr != nil {
				return manifest, fmt.Errorf("read library config: %w", readErr)
			}
			if validateErr := validateLibraryConfig(configContents); validateErr != nil {
				return manifest, fmt.Errorf("validate library config: %w", validateErr)
			}
			configSnapshot := filepath.Join(workDir, "library-config.cfg")
			if writeErr := writePrivateFile(configSnapshot, configContents); writeErr != nil {
				return manifest, fmt.Errorf("snapshot library config: %w", writeErr)
			}
			configEntry, describeErr := describeFile("library-config", LibraryConfigArchivePath, configSnapshot)
			if describeErr != nil {
				return manifest, describeErr
			}
			files = append(files, configEntry)
			archiveSources[LibraryConfigArchivePath] = configSnapshot
			configIncluded = true
		case errors.Is(statErr, os.ErrNotExist):
			// A missing optional library-config.cfg is represented in Scope.
		default:
			return manifest, fmt.Errorf("inspect library config: %w", statErr)
		}
	}

	now := time.Now
	if options.Now != nil {
		now = options.Now
	}
	manifest = Manifest{
		Format:        FormatName,
		FormatVersion: FormatVersion,
		CreatedAt:     now().UTC(),
		AppVersion:    strings.TrimSpace(options.AppVersion),
		AppChannel:    strings.TrimSpace(options.AppChannel),
		Scope: Scope{
			DatabaseIncluded:      true,
			LibraryConfigIncluded: configIncluded,
			UserAssetsIncluded:    false,
			MediaFilesIncluded:    false,
		},
		SchemaMigrations: migrations,
		Files:            files,
	}

	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return Manifest{}, fmt.Errorf("encode backup manifest: %w", err)
	}
	manifestBytes = append(manifestBytes, '\n')

	tempFile, err := os.CreateTemp(filepath.Dir(absDestination), ".curated-backup-*.tmp")
	if err != nil {
		return Manifest{}, fmt.Errorf("create temporary backup package: %w", err)
	}
	tempPath := tempFile.Name()
	committed := false
	defer func() {
		_ = tempFile.Close()
		if !committed {
			_ = os.Remove(tempPath)
		}
	}()
	if err := tempFile.Chmod(0o600); err != nil {
		return Manifest{}, fmt.Errorf("restrict backup package permissions: %w", err)
	}

	zipWriter := zip.NewWriter(tempFile)
	if err := writeArchiveBytes(zipWriter, ManifestPath, manifestBytes); err != nil {
		_ = zipWriter.Close()
		return Manifest{}, err
	}
	for _, entry := range manifest.Files {
		if err := writeArchiveFile(zipWriter, entry.Path, archiveSources[entry.Path]); err != nil {
			_ = zipWriter.Close()
			return Manifest{}, err
		}
	}
	if err := zipWriter.Close(); err != nil {
		return Manifest{}, fmt.Errorf("finalize backup package: %w", err)
	}
	if err := tempFile.Sync(); err != nil {
		return Manifest{}, fmt.Errorf("sync backup package: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return Manifest{}, fmt.Errorf("close backup package: %w", err)
	}
	if err := os.Rename(tempPath, absDestination); err != nil {
		return Manifest{}, fmt.Errorf("commit backup package: %w", err)
	}
	committed = true
	return manifest, nil
}

func writePrivateFile(filePath string, contents []byte) error {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.Write(contents); err != nil {
		_ = file.Close()
		_ = os.Remove(filePath)
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(filePath)
		return err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(filePath)
		return err
	}
	return nil
}

func describeFile(kind, archivePath, sourcePath string) (FileEntry, error) {
	file, err := os.Open(sourcePath)
	if err != nil {
		return FileEntry{}, fmt.Errorf("open backup source %s: %w", kind, err)
	}
	defer func() { _ = file.Close() }()
	hash := sha256.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return FileEntry{}, fmt.Errorf("hash backup source %s: %w", kind, err)
	}
	return FileEntry{
		Kind:      kind,
		Path:      archivePath,
		SizeBytes: size,
		SHA256:    hex.EncodeToString(hash.Sum(nil)),
	}, nil
}

func writeArchiveBytes(writer *zip.Writer, archivePath string, contents []byte) error {
	header := &zip.FileHeader{Name: archivePath, Method: zip.Deflate}
	header.SetMode(0o600)
	entry, err := writer.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("create backup archive entry %s: %w", archivePath, err)
	}
	if _, err := entry.Write(contents); err != nil {
		return fmt.Errorf("write backup archive entry %s: %w", archivePath, err)
	}
	return nil
}

func writeArchiveFile(writer *zip.Writer, archivePath, sourcePath string) error {
	header := &zip.FileHeader{Name: archivePath, Method: zip.Deflate}
	header.SetMode(0o600)
	entry, err := writer.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("create backup archive entry %s: %w", archivePath, err)
	}
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open backup source for %s: %w", archivePath, err)
	}
	defer func() { _ = source.Close() }()
	if _, err := io.Copy(entry, source); err != nil {
		return fmt.Errorf("write backup archive entry %s: %w", archivePath, err)
	}
	return nil
}

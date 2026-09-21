package backup

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"curated-backend/internal/storage"
)

const (
	maxManifestBytes   = 1 << 20
	maxConfigBytes     = 16 << 20
	maxBackupFileBytes = int64(1) << 40
)

// Verify authenticates every declared file and runs SQLite integrity checks.
func Verify(ctx context.Context, backupPath string) (Verification, error) {
	result := Verification{CheckedAt: time.Now().UTC()}
	reader, err := zip.OpenReader(strings.TrimSpace(backupPath))
	if err != nil {
		return result, fmt.Errorf("open backup package: %w", err)
	}
	defer func() { _ = reader.Close() }()

	archiveFiles := make(map[string]*zip.File, len(reader.File))
	for _, file := range reader.File {
		if err := validateArchivePath(file.Name); err != nil {
			result.Errors = append(result.Errors, err.Error())
			continue
		}
		if _, exists := archiveFiles[file.Name]; exists {
			result.Errors = append(result.Errors, fmt.Sprintf("duplicate archive entry %q", file.Name))
			continue
		}
		archiveFiles[file.Name] = file
	}
	manifestFile := archiveFiles[ManifestPath]
	if manifestFile == nil {
		result.Errors = append(result.Errors, "manifest.json is missing")
		return finalizeVerification(result), nil
	}
	manifestBytes, err := readZipFileLimited(manifestFile, maxManifestBytes)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("read manifest.json: %v", err))
		return finalizeVerification(result), nil
	}
	manifest, err := decodeManifest(manifestBytes)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("decode manifest.json: %v", err))
		return finalizeVerification(result), nil
	}
	result.Manifest = &manifest
	validateManifest(&result, manifest, archiveFiles)
	if len(result.Errors) > 0 {
		return finalizeVerification(result), nil
	}

	workDir, err := os.MkdirTemp("", "curated-backup-verify-*")
	if err != nil {
		return result, fmt.Errorf("create backup verification directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(workDir) }()
	databasePath := filepath.Join(workDir, "database", "curated.db")
	if err := os.MkdirAll(filepath.Dir(databasePath), 0o700); err != nil {
		return result, fmt.Errorf("create database verification directory: %w", err)
	}

	for _, entry := range manifest.Files {
		file := archiveFiles[entry.Path]
		var destination *os.File
		var configBuffer bytes.Buffer
		if entry.Kind == "database" {
			destination, err = os.OpenFile(databasePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
			if err != nil {
				return result, fmt.Errorf("create extracted backup database: %w", err)
			}
		}
		hash := sha256.New()
		writers := []io.Writer{hash}
		if destination != nil {
			writers = append(writers, destination)
		}
		if entry.Kind == "library-config" && entry.SizeBytes <= maxConfigBytes {
			writers = append(writers, &configBuffer)
		}
		actualSize, copyErr := copyZipFileLimited(file, io.MultiWriter(writers...), entry.SizeBytes)
		if destination != nil {
			if syncErr := destination.Sync(); copyErr == nil && syncErr != nil {
				copyErr = syncErr
			}
			if closeErr := destination.Close(); copyErr == nil && closeErr != nil {
				copyErr = closeErr
			}
		}
		if copyErr != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("read %s: %v", entry.Path, copyErr))
			continue
		}
		if actualSize != entry.SizeBytes {
			result.Errors = append(result.Errors, fmt.Sprintf("size mismatch for %s: got %d want %d", entry.Path, actualSize, entry.SizeBytes))
		}
		actualHash := hex.EncodeToString(hash.Sum(nil))
		if !strings.EqualFold(actualHash, entry.SHA256) {
			result.Errors = append(result.Errors, fmt.Sprintf("sha256 mismatch for %s", entry.Path))
		}
		if entry.Kind == "library-config" {
			if entry.SizeBytes > maxConfigBytes {
				result.Errors = append(result.Errors, fmt.Sprintf("library config exceeds %d bytes", maxConfigBytes))
			} else if err := validateLibraryConfig(configBuffer.Bytes()); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("invalid library config: %v", err))
			}
		}
	}
	if len(result.Errors) > 0 {
		return finalizeVerification(result), nil
	}

	backupStore, err := storage.NewSQLiteStore(databasePath)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("open backup database: %v", err))
		return finalizeVerification(result), nil
	}
	integrity, integrityErr := backupStore.CheckIntegrity(ctx)
	result.DatabaseIntegrity = integrity
	migrations, migrationsErr := backupStore.AppliedMigrations(ctx)
	if err := verifyWishlistReferences(ctx, backupStore, manifest); err != nil {
		result.Errors = append(result.Errors, err.Error())
	}
	closeErr := backupStore.Close()
	if integrityErr != nil {
		result.Errors = append(result.Errors, integrityErr.Error())
	}
	if migrationsErr != nil {
		result.Errors = append(result.Errors, migrationsErr.Error())
	} else if !reflect.DeepEqual(migrations, manifest.SchemaMigrations) {
		result.Errors = append(result.Errors, "database schema migrations do not match manifest")
	}
	if closeErr != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("close backup database: %v", closeErr))
	}
	return finalizeVerification(result), nil
}

func validateArchivePath(name string) error {
	if name == "" || strings.Contains(name, "\\") || strings.HasPrefix(name, "/") || path.IsAbs(name) {
		return fmt.Errorf("unsafe archive entry path %q", name)
	}
	clean := path.Clean(name)
	if clean != name || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return fmt.Errorf("unsafe archive entry path %q", name)
	}
	return nil
}

func decodeManifest(data []byte) (Manifest, error) {
	var manifest Manifest
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return manifest, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return manifest, fmt.Errorf("manifest contains trailing JSON")
		}
		return manifest, fmt.Errorf("manifest trailing data: %w", err)
	}
	return manifest, nil
}

func validateManifest(result *Verification, manifest Manifest, archiveFiles map[string]*zip.File) {
	if manifest.Format != FormatName {
		result.Errors = append(result.Errors, fmt.Sprintf("unsupported backup format %q", manifest.Format))
	}
	if manifest.FormatVersion != 1 && manifest.FormatVersion != FormatVersion {
		result.Errors = append(result.Errors, fmt.Sprintf("unsupported backup format version %d", manifest.FormatVersion))
	}
	if manifest.CreatedAt.IsZero() {
		result.Errors = append(result.Errors, "manifest createdAt is required")
	}
	if !manifest.Scope.DatabaseIncluded {
		result.Errors = append(result.Errors, "manifest does not include the database")
	}
	declared := map[string]struct{}{ManifestPath: {}}
	databaseCount := 0
	configCount := 0
	for _, entry := range manifest.Files {
		if err := validateArchivePath(entry.Path); err != nil {
			result.Errors = append(result.Errors, err.Error())
		}
		if _, exists := declared[entry.Path]; exists {
			result.Errors = append(result.Errors, fmt.Sprintf("duplicate manifest file %q", entry.Path))
		}
		declared[entry.Path] = struct{}{}
		if entry.SizeBytes < 0 || entry.SizeBytes > maxBackupFileBytes {
			result.Errors = append(result.Errors, fmt.Sprintf("invalid size for %s", entry.Path))
		}
		decodedHash, err := hex.DecodeString(entry.SHA256)
		if err != nil || len(decodedHash) != sha256.Size {
			result.Errors = append(result.Errors, fmt.Sprintf("invalid sha256 for %s", entry.Path))
		}
		switch entry.Kind {
		case "database":
			databaseCount++
			if entry.Path != DatabaseArchivePath {
				result.Errors = append(result.Errors, "database entry path is not canonical")
			}
		case "wishlist-asset":
			if manifest.FormatVersion < 2 || !manifest.Scope.WishlistAssetsIncluded || !strings.HasPrefix(entry.Path, wishlistArchivePrefix) {
				result.Errors = append(result.Errors, "invalid wishlist asset scope or path")
			}
		case "library-config":
			configCount++
			if entry.Path != LibraryConfigArchivePath {
				result.Errors = append(result.Errors, "library config entry path is not canonical")
			}
		default:
			result.Errors = append(result.Errors, fmt.Sprintf("unsupported backup file kind %q", entry.Kind))
		}
		file := archiveFiles[entry.Path]
		if file == nil {
			result.Errors = append(result.Errors, fmt.Sprintf("declared file %s is missing", entry.Path))
		} else if file.UncompressedSize64 != uint64(entry.SizeBytes) {
			result.Errors = append(result.Errors, fmt.Sprintf("archive header size mismatch for %s", entry.Path))
		}
	}
	if databaseCount != 1 {
		result.Errors = append(result.Errors, fmt.Sprintf("manifest contains %d database entries", databaseCount))
	}
	if manifest.Scope.LibraryConfigIncluded && configCount != 1 {
		result.Errors = append(result.Errors, "manifest scope requires one library config entry")
	}
	if !manifest.Scope.LibraryConfigIncluded && configCount != 0 {
		result.Errors = append(result.Errors, "manifest contains library config outside declared scope")
	}
	for name := range archiveFiles {
		if _, ok := declared[name]; !ok {
			result.Errors = append(result.Errors, fmt.Sprintf("undeclared archive entry %s", name))
		}
	}
}

func readZipFileLimited(file *zip.File, maxBytes int64) ([]byte, error) {
	if file.UncompressedSize64 > uint64(maxBytes) {
		return nil, fmt.Errorf("entry exceeds %d bytes", maxBytes)
	}
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()
	return io.ReadAll(io.LimitReader(reader, maxBytes+1))
}

func copyZipFileLimited(file *zip.File, destination io.Writer, expectedSize int64) (int64, error) {
	reader, err := file.Open()
	if err != nil {
		return 0, err
	}
	defer func() { _ = reader.Close() }()
	bytesCopied, err := io.Copy(destination, io.LimitReader(reader, expectedSize+1))
	if err != nil {
		return bytesCopied, err
	}
	if bytesCopied > expectedSize {
		return bytesCopied, fmt.Errorf("entry expands beyond declared size")
	}
	return bytesCopied, nil
}

func validateLibraryConfig(data []byte) error {
	var value map[string]json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	if value == nil {
		return fmt.Errorf("expected a JSON object")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("unexpected trailing JSON")
	}
	return nil
}

func finalizeVerification(result Verification) Verification {
	result.Valid = len(result.Errors) == 0
	if result.Errors == nil {
		result.Errors = []string{}
	}
	if result.Warnings == nil {
		result.Warnings = []string{}
	}
	return result
}

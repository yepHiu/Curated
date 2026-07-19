package backup

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"curated-backend/internal/storage"

	_ "github.com/glebarez/go-sqlite"
)

func TestCreateVerifyPreflightAndRestore(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	sourceDatabase := filepath.Join(root, "source.db")
	sourceStore := openMigratedStore(t, sourceDatabase)
	if _, err := sourceStore.AddLibraryPath(ctx, `D:\Media`, "Source library"); err != nil {
		t.Fatalf("AddLibraryPath(source): %v", err)
	}
	t.Cleanup(func() { _ = sourceStore.Close() })

	sourceConfig := filepath.Join(root, "source-library-config.cfg")
	if err := os.WriteFile(sourceConfig, []byte("{\n  \"organizeLibrary\": true\n}\n"), 0o600); err != nil {
		t.Fatalf("write source config: %v", err)
	}
	backupPath := filepath.Join(root, "curated-20260720.curated-backup")
	fixedTime := time.Date(2026, 7, 20, 2, 30, 0, 0, time.UTC)
	manifest, err := Create(ctx, CreateOptions{
		Store:             sourceStore,
		DestinationPath:   backupPath,
		LibraryConfigPath: sourceConfig,
		AppVersion:        "1.4.11",
		AppChannel:        "dev",
		Now:               func() time.Time { return fixedTime },
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if manifest.Format != FormatName || manifest.FormatVersion != FormatVersion {
		t.Fatalf("unexpected manifest identity: %+v", manifest)
	}
	if !manifest.Scope.DatabaseIncluded || !manifest.Scope.LibraryConfigIncluded || manifest.Scope.MediaFilesIncluded {
		t.Fatalf("unexpected manifest scope: %+v", manifest.Scope)
	}

	verification, err := Verify(ctx, backupPath)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !verification.Valid || len(verification.Errors) != 0 {
		t.Fatalf("verification = %+v", verification)
	}
	if verification.DatabaseIntegrity.QuickCheck != "ok" || verification.DatabaseIntegrity.ForeignKeyViolations != 0 {
		t.Fatalf("unexpected database integrity: %+v", verification.DatabaseIntegrity)
	}

	targetDatabase := filepath.Join(root, "restore", "curated.db")
	targetStore := openMigratedStore(t, targetDatabase)
	if _, err := targetStore.AddLibraryPath(ctx, `E:\Old`, "Old library"); err != nil {
		t.Fatalf("AddLibraryPath(target): %v", err)
	}
	if err := targetStore.Close(); err != nil {
		t.Fatalf("close target store: %v", err)
	}
	targetConfig := filepath.Join(root, "restore", "library-config.cfg")
	if err := os.WriteFile(targetConfig, []byte("{\"organizeLibrary\":false}\n"), 0o600); err != nil {
		t.Fatalf("write target config: %v", err)
	}

	preflight, err := PreflightRestore(ctx, PreflightOptions{
		BackupPath:     backupPath,
		TargetDatabase: targetDatabase,
		TargetConfig:   targetConfig,
		Now:            func() time.Time { return fixedTime },
	})
	if err != nil {
		t.Fatalf("PreflightRestore: %v", err)
	}
	if !preflight.CanRestore || !preflight.TargetDatabaseExists || !preflight.TargetConfigExists {
		t.Fatalf("preflight = %+v", preflight)
	}
	if !preflight.AvailableBytesKnown || preflight.AvailableBytes < uint64(preflight.RequiredBytes) {
		t.Fatalf("unexpected capacity result: %+v", preflight)
	}

	if _, err := Restore(ctx, RestoreOptions{PreflightOptions: PreflightOptions{
		BackupPath:     backupPath,
		TargetDatabase: targetDatabase,
		TargetConfig:   targetConfig,
	}}); err == nil || !strings.Contains(err.Error(), "explicit confirmation") {
		t.Fatalf("Restore without confirmation error = %v", err)
	}

	restored, err := Restore(ctx, RestoreOptions{
		PreflightOptions: PreflightOptions{
			BackupPath:     backupPath,
			TargetDatabase: targetDatabase,
			TargetConfig:   targetConfig,
			Now:            func() time.Time { return fixedTime },
		},
		Confirm: true,
	})
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if restored.DatabaseRollbackPath == "" || restored.ConfigRollbackPath == "" {
		t.Fatalf("restore did not retain rollback paths: %+v", restored)
	}
	assertLibraryPaths(t, targetDatabase, []string{`D:\Media`})
	assertLibraryPaths(t, restored.DatabaseRollbackPath, []string{`E:\Old`})
	configContents, err := os.ReadFile(targetConfig)
	if err != nil {
		t.Fatalf("read restored config: %v", err)
	}
	if !bytes.Contains(configContents, []byte(`"organizeLibrary": true`)) {
		t.Fatalf("unexpected restored config: %s", configContents)
	}
	rollbackConfig, err := os.ReadFile(restored.ConfigRollbackPath)
	if err != nil {
		t.Fatalf("read config rollback: %v", err)
	}
	if !bytes.Contains(rollbackConfig, []byte(`"organizeLibrary":false`)) {
		t.Fatalf("unexpected config rollback: %s", rollbackConfig)
	}
}

func TestVerifyRejectsTamperedDatabaseAndUnsafeEntry(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	store := openMigratedStore(t, filepath.Join(root, "source.db"))
	defer func() { _ = store.Close() }()
	backupPath := filepath.Join(root, "valid.curated-backup")
	if _, err := Create(ctx, CreateOptions{Store: store, DestinationPath: backupPath}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	tamperedPath := filepath.Join(root, "tampered.curated-backup")
	rewriteZip(t, backupPath, tamperedPath, func(name string, data []byte) ([]byte, string) {
		if name == DatabaseArchivePath && len(data) > 0 {
			data[0] ^= 0xff
		}
		return data, name
	})
	verification, err := Verify(ctx, tamperedPath)
	if err != nil {
		t.Fatalf("Verify(tampered): %v", err)
	}
	if verification.Valid || !containsText(verification.Errors, "sha256 mismatch") {
		t.Fatalf("tampered verification = %+v", verification)
	}

	unsafePath := filepath.Join(root, "unsafe.curated-backup")
	file, err := os.Create(unsafePath)
	if err != nil {
		t.Fatalf("create unsafe package: %v", err)
	}
	writer := zip.NewWriter(file)
	entry, err := writer.Create("../manifest.json")
	if err != nil {
		t.Fatalf("create unsafe entry: %v", err)
	}
	_, _ = entry.Write([]byte(`{}`))
	if err := writer.Close(); err != nil {
		t.Fatalf("close unsafe zip: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close unsafe package: %v", err)
	}
	unsafeVerification, err := Verify(ctx, unsafePath)
	if err != nil {
		t.Fatalf("Verify(unsafe): %v", err)
	}
	if unsafeVerification.Valid || !containsText(unsafeVerification.Errors, "unsafe archive entry") {
		t.Fatalf("unsafe verification = %+v", unsafeVerification)
	}
}

func TestPreflightRejectsFutureSchemaMigration(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	databasePath := filepath.Join(root, "future.db")
	store := openMigratedStore(t, databasePath)
	if err := store.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}
	insertSchemaMigration(t, databasePath, "9999_future.sql")
	store, err := storage.NewSQLiteStore(databasePath)
	if err != nil {
		t.Fatalf("reopen future store: %v", err)
	}
	defer func() { _ = store.Close() }()
	backupPath := filepath.Join(root, "future.curated-backup")
	if _, err := Create(ctx, CreateOptions{Store: store, DestinationPath: backupPath}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	preflight, err := PreflightRestore(ctx, PreflightOptions{
		BackupPath:     backupPath,
		TargetDatabase: filepath.Join(root, "target.db"),
	})
	if err != nil {
		t.Fatalf("PreflightRestore: %v", err)
	}
	if preflight.CanRestore || !reflect.DeepEqual(preflight.UnsupportedMigrations, []string{"9999_future.sql"}) {
		t.Fatalf("future preflight = %+v", preflight)
	}
}

func TestCreateRejectsInvalidLibraryConfigWithoutCreatingPackage(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	store := openMigratedStore(t, filepath.Join(root, "source.db"))
	defer func() { _ = store.Close() }()
	configPath := filepath.Join(root, "library-config.cfg")
	if err := os.WriteFile(configPath, []byte(`{"broken":`), 0o600); err != nil {
		t.Fatalf("write invalid config: %v", err)
	}
	backupPath := filepath.Join(root, "invalid.curated-backup")
	if _, err := Create(ctx, CreateOptions{
		Store:             store,
		DestinationPath:   backupPath,
		LibraryConfigPath: configPath,
	}); err == nil || !strings.Contains(err.Error(), "validate library config") {
		t.Fatalf("Create error = %v", err)
	}
	if _, err := os.Stat(backupPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("backup package exists after rejected create: %v", err)
	}
}

func openMigratedStore(t *testing.T, databasePath string) *storage.SQLiteStore {
	t.Helper()
	store, err := storage.NewSQLiteStore(databasePath)
	if err != nil {
		t.Fatalf("NewSQLiteStore(%s): %v", databasePath, err)
	}
	if err := store.Migrate(context.Background()); err != nil {
		_ = store.Close()
		t.Fatalf("Migrate(%s): %v", databasePath, err)
	}
	return store
}

func assertLibraryPaths(t *testing.T, databasePath string, want []string) {
	t.Helper()
	store, err := storage.NewSQLiteStore(databasePath)
	if err != nil {
		t.Fatalf("open %s: %v", databasePath, err)
	}
	defer func() { _ = store.Close() }()
	got, err := store.ListLibraryPathStrings(context.Background())
	if err != nil {
		t.Fatalf("ListLibraryPathStrings(%s): %v", databasePath, err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("library paths in %s = %v, want %v", databasePath, got, want)
	}
}

func insertSchemaMigration(t *testing.T, databasePath, name string) {
	t.Helper()
	database, err := sql.Open("sqlite", databasePath)
	if err != nil {
		t.Fatalf("open database for migration insert: %v", err)
	}
	defer func() { _ = database.Close() }()
	if _, err := database.ExecContext(context.Background(), `INSERT INTO schema_migrations (name) VALUES (?)`, name); err != nil {
		t.Fatalf("insert future migration: %v", err)
	}
}

func rewriteZip(t *testing.T, sourcePath, targetPath string, mutate func(string, []byte) ([]byte, string)) {
	t.Helper()
	source, err := zip.OpenReader(sourcePath)
	if err != nil {
		t.Fatalf("open source zip: %v", err)
	}
	defer func() { _ = source.Close() }()
	target, err := os.Create(targetPath)
	if err != nil {
		t.Fatalf("create target zip: %v", err)
	}
	writer := zip.NewWriter(target)
	for _, sourceFile := range source.File {
		reader, err := sourceFile.Open()
		if err != nil {
			t.Fatalf("open %s: %v", sourceFile.Name, err)
		}
		data, err := io.ReadAll(reader)
		_ = reader.Close()
		if err != nil {
			t.Fatalf("read %s: %v", sourceFile.Name, err)
		}
		data, name := mutate(sourceFile.Name, data)
		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		header.SetMode(0o600)
		entry, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		if _, err := entry.Write(data); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close target zip: %v", err)
	}
	if err := target.Close(); err != nil {
		t.Fatalf("close target file: %v", err)
	}
}

func containsText(values []string, needle string) bool {
	for _, value := range values {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

func TestManifestJSONRoundTrip(t *testing.T) {
	manifest := Manifest{Format: FormatName, FormatVersion: FormatVersion, CreatedAt: time.Now().UTC()}
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	decoded, err := decodeManifest(data)
	if err != nil {
		t.Fatalf("decodeManifest: %v", err)
	}
	if decoded.Format != FormatName || decoded.FormatVersion != FormatVersion {
		t.Fatalf("decoded = %+v", decoded)
	}
}

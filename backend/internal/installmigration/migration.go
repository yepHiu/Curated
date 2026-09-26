// Package installmigration implements the resumable, offline legacy-to-split transition.
// It never migrates the live database schema or removes user data.
package installmigration

import (
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

	"curated-backend/internal/backup"
	"curated-backend/internal/config"
	"curated-backend/internal/launchprofile"
	"curated-backend/internal/processlock"
	"curated-backend/internal/storage"
)

type Legacy struct {
	Directory   string `json:"directory"`
	Version     string `json:"version"`
	Uninstaller string `json:"uninstaller"`
	Machine     bool   `json:"machine"`
}

type Platform interface {
	FindLegacy() (*Legacy, error)
	CheckStopped(*Legacy) error
	CheckNoSplitServer() error
	StartupCommand() (string, error)
	DisableOldStartup(*Legacy, string) error
	Uninstall(context.Context, *Legacy) error
	Complete(string, bool) error
}

type Options struct {
	StateDir    string
	ProfilePath string
	DataRoot    string
	ConfigPath  string
}

type Journal struct {
	Schema        int                   `json:"schema"`
	Stage         string                `json:"stage"`
	Legacy        Legacy                `json:"legacy"`
	Profile       launchprofile.Profile `json:"profile"`
	Backup        string                `json:"backup"`
	Hashes        map[string]string     `json:"hashes"`
	Startup       string                `json:"startup"`
	LaunchAtLogin bool                  `json:"launchAtLogin"`
}

func journalPath(o Options) string { return filepath.Join(o.StateDir, "legacy.json") }

func writeJSON(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".migration-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(append(raw, '\n'))
	}
	if err == nil {
		err = f.Sync()
	}
	err = errors.Join(err, f.Close())
	if err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

func readJournal(o Options) (*Journal, error) {
	raw, err := os.ReadFile(journalPath(o))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var j Journal
	if err = json.Unmarshal(raw, &j); err != nil {
		return nil, err
	}
	if j.Schema != 1 || (j.Stage != "prepared" && j.Stage != "removed" && j.Stage != "complete") {
		return nil, fmt.Errorf("invalid migration journal; keep backups and inspect %s", journalPath(o))
	}
	if err = j.Profile.Validate(); err != nil {
		return nil, err
	}
	if len(j.Hashes) == 0 || j.Hashes[j.Backup] == "" {
		return nil, fmt.Errorf("migration journal has no verified backup")
	}
	return &j, nil
}

func digest(path string) (string, error) {
	// Backups may exceed memory: hash as a stream.
	return fileDigest(path)
}

func verifyJournal(ctx context.Context, j *Journal) error {
	for path, expected := range j.Hashes {
		actual, err := digest(path)
		if err != nil {
			return err
		}
		if actual != expected {
			return fmt.Errorf("migration backup changed: %s", path)
		}
	}
	result, err := backup.Verify(ctx, j.Backup)
	if err != nil {
		return err
	}
	if !result.Valid {
		return fmt.Errorf("migration backup verification failed: %v", result.Errors)
	}
	return nil
}

// Prepare finishes before the component installers run. The old uninstaller is
// invoked only after a verified backup and durable recovery journal exist.
func Prepare(ctx context.Context, o Options, platform Platform) error {
	// A clean/split-only Full installation must not create legacy migration state.
	existing, err := platform.FindLegacy()
	if err != nil {
		return err
	}
	previous, err := readJournal(o)
	if err != nil {
		return err
	}
	if existing == nil && previous == nil {
		return nil
	}
	lock, err := processlock.Acquire(filepath.Join(o.StateDir, "migration.lock"))
	if err != nil {
		return err
	}
	defer lock.Release()
	old, err := platform.FindLegacy()
	if err != nil {
		return err
	}
	j, err := readJournal(o)
	if err != nil {
		return err
	}
	if old == nil {
		if j == nil || j.Stage == "complete" {
			return nil
		}
		if err = verifyJournal(ctx, j); err != nil {
			return err
		}
		if err = checkSelections(o, j); err != nil {
			return err
		}
		if err = checkDatabase(j.Profile.DatabasePath); err != nil {
			return err
		}
		if err = platform.DisableOldStartup(&j.Legacy, j.Startup); err != nil {
			return err
		}
		if err = writeProfile(o, j.Profile); err != nil {
			return err
		}
		j.Stage = "removed"
		return writeJSON(journalPath(o), j)
	}
	if j != nil && j.Stage == "complete" {
		return fmt.Errorf("legacy Curated was reinstalled after migration; resolve the two installations before retrying")
	}
	if j != nil && !strings.EqualFold(j.Legacy.Directory, old.Directory) {
		return fmt.Errorf("legacy installation changed since the interrupted upgrade")
	}
	if err = platform.CheckNoSplitServer(); err != nil {
		return err
	}
	if err = platform.CheckStopped(old); err != nil {
		return err
	}
	if j != nil {
		if err = verifyJournal(ctx, j); err != nil {
			return err
		}
		if err = checkSelections(o, j); err != nil {
			return err
		}
		// Use the saved selection, but take a fresh backup: the old app may have
		// been used after a cancelled UAC prompt or an unsuccessful uninstall.
		o.DataRoot, o.ConfigPath = j.Profile.DataRoot, j.Profile.ConfigPath
	}
	prepared, dataLock, err := snapshot(ctx, o, old, platform)
	if err != nil {
		return err
	}
	defer dataLock.Release()
	if j != nil {
		prepared.Startup = j.Startup
		prepared.LaunchAtLogin = prepared.LaunchAtLogin || j.LaunchAtLogin
	}
	j = prepared
	if err = writeJSON(journalPath(o), j); err != nil {
		return err
	}
	if err = writeProfile(o, j.Profile); err != nil {
		return err
	}
	if err = platform.CheckStopped(old); err != nil {
		return err
	}
	if err = platform.Uninstall(ctx, old); err != nil {
		return fmt.Errorf("old program removal did not finish; backup retained at %s; run Full again: %w", j.Backup, err)
	}
	remaining, err := platform.FindLegacy()
	if err != nil {
		return err
	}
	if remaining != nil {
		return fmt.Errorf("old uninstall registration remains; run Full again after resolving its uninstall error")
	}
	if err = platform.DisableOldStartup(old, j.Startup); err != nil {
		return err
	}
	j.Stage = "removed"
	return writeJSON(journalPath(o), j)
}

func checkSelections(o Options, j *Journal) error {
	if o.DataRoot != "" && !strings.EqualFold(filepath.Clean(o.DataRoot), j.Profile.DataRoot) {
		return fmt.Errorf("data directory differs from the pending migration")
	}
	// A retry may use the original source config; the journal owns a verified copy.
	if o.ConfigPath != "" && o.ConfigPath != j.Profile.ConfigPath {
		a, err := digest(o.ConfigPath)
		if err != nil {
			return err
		}
		b, err := digest(j.Profile.ConfigPath)
		if err != nil {
			return err
		}
		if a != b {
			return fmt.Errorf("configuration differs from the pending migration")
		}
	}
	return nil
}

func writeProfile(o Options, profile launchprofile.Profile) error {
	raw, err := os.ReadFile(o.ProfilePath)
	if err == nil {
		var previous launchprofile.Profile
		if json.Unmarshal(raw, &previous) != nil || previous != profile {
			return fmt.Errorf("an existing Server startup profile conflicts with this migration")
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}
	return writeJSON(o.ProfilePath, profile)
}

func Complete(ctx context.Context, o Options, platform Platform) error {
	previous, err := readJournal(o)
	if err != nil {
		return err
	}
	if previous == nil {
		return nil
	}
	lock, err := processlock.Acquire(filepath.Join(o.StateDir, "migration.lock"))
	if err != nil {
		return err
	}
	defer lock.Release()
	j, err := readJournal(o)
	if err != nil || j == nil || j.Stage == "complete" {
		return err
	}
	old, err := platform.FindLegacy()
	if err != nil {
		return err
	}
	if old != nil {
		return fmt.Errorf("old installation still exists")
	}
	if err = verifyJournal(ctx, j); err != nil {
		return err
	}
	if err = checkDatabase(j.Profile.DatabasePath); err != nil {
		return err
	}
	if err = writeProfile(o, j.Profile); err != nil {
		return err
	}
	if err = platform.Complete(j.Startup, j.LaunchAtLogin); err != nil {
		return err
	}
	j.Stage = "complete"
	return writeJSON(journalPath(o), j)
}

func checkDatabase(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("original database not found; refusing to create an empty library: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() == 0 {
		return fmt.Errorf("original database is not a nonempty regular file")
	}
	return nil
}

func snapshot(ctx context.Context, o Options, old *Legacy, platform Platform) (*Journal, *processlock.Lock, error) {
	if !filepath.IsAbs(o.DataRoot) {
		return nil, nil, fmt.Errorf("select the original absolute data directory")
	}
	if err := outside(old.Directory, o.DataRoot); err != nil {
		return nil, nil, err
	}
	if err := outside(old.Directory, o.StateDir); err != nil {
		return nil, nil, err
	}
	// Uninstall must never touch a database hidden in a program-local layout.
	if err := checkProgramTree(old.Directory); err != nil {
		return nil, nil, err
	}
	if o.ConfigPath != "" {
		if !filepath.IsAbs(o.ConfigPath) {
			return nil, nil, fmt.Errorf("custom -config must be an absolute path")
		}
		if err := validateConfigPaths(o.ConfigPath, old.Directory); err != nil {
			return nil, nil, err
		}
	}
	libConfig := filepath.Join(o.DataRoot, "config", "library-config.cfg")
	if _, err := os.Stat(libConfig); err == nil {
		if err := validateConfigPaths(libConfig, old.Directory); err != nil {
			return nil, nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, nil, err
	}
	prior, hadEnv := os.LookupEnv("CURATED_DATA_DIR")
	if err := os.Setenv("CURATED_DATA_DIR", o.DataRoot); err != nil {
		return nil, nil, err
	}
	defer func() {
		if hadEnv {
			_ = os.Setenv("CURATED_DATA_DIR", prior)
		} else {
			_ = os.Unsetenv("CURATED_DATA_DIR")
		}
	}()
	cfg, err := config.Load(o.ConfigPath)
	if err != nil {
		return nil, nil, err
	}
	if err = config.MergeLibrarySettingsFile(&cfg, libConfig); err != nil {
		return nil, nil, err
	}
	if !filepath.IsAbs(cfg.DatabasePath) {
		return nil, nil, fmt.Errorf("databasePath must be absolute before upgrading")
	}
	if err = outside(old.Directory, cfg.DatabasePath); err != nil {
		return nil, nil, err
	}
	if err = checkDatabase(cfg.DatabasePath); err != nil {
		return nil, nil, err
	}
	startup, err := platform.StartupCommand()
	if err != nil {
		return nil, nil, err
	}
	if strings.Contains(strings.ToLower(startup), "-config") && o.ConfigPath == "" {
		return nil, nil, fmt.Errorf("old startup uses -config; select that configuration in the Full upgrade page")
	}
	dataLock, err := processlock.Acquire(cfg.DatabasePath + ".runtime.lock")
	if err != nil {
		return nil, nil, fmt.Errorf("fully exit Curated before upgrading: %w", err)
	}
	success := false
	defer func() {
		if !success {
			_ = dataLock.Release()
		}
	}()
	store, err := storage.NewSQLiteStore(cfg.DatabasePath)
	if err != nil {
		return nil, nil, err
	}
	defer store.Close()
	if err = checkStoredPaths(ctx, cfg.DatabasePath, old.Directory); err != nil {
		return nil, nil, err
	}
	dir, err := os.MkdirTemp(o.StateDir, "backup-")
	if err != nil {
		return nil, nil, err
	}
	j := &Journal{Schema: 1, Stage: "prepared", Legacy: *old, Backup: filepath.Join(dir, "library.curated-backup"), Hashes: map[string]string{}, Startup: startup, LaunchAtLogin: cfg.LaunchAtLogin,
		Profile: launchprofile.Profile{Schema: 1, DataRoot: filepath.Clean(o.DataRoot), DatabasePath: cfg.DatabasePath}}
	if o.ConfigPath != "" {
		// Keep the exact custom configuration; validated absolute paths do not change meaning.
		target := o.ConfigPath
		if !within(o.StateDir, target) {
			target = filepath.Join(dir, "server-config.json")
			if err = copySnapshot(o.ConfigPath, target); err != nil {
				return nil, nil, err
			}
		}
		j.Profile.ConfigPath = target
		j.Hashes[target], err = digest(target)
		if err != nil {
			return nil, nil, err
		}
	}
	idPath := cfg.DatabasePath + ".server-id"
	if _, err = os.Stat(idPath); err == nil {
		target := filepath.Join(dir, "server-id")
		if err = copySnapshot(idPath, target); err != nil {
			return nil, nil, err
		}
		j.Hashes[target], err = digest(target)
		if err != nil {
			return nil, nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, nil, err
	}
	if _, err = backup.Create(ctx, backup.CreateOptions{Store: store, DestinationPath: j.Backup, LibraryConfigPath: libConfig, AppVersion: old.Version, AppChannel: "release"}); err != nil {
		return nil, nil, fmt.Errorf("backup failed; old installation kept: %w", err)
	}
	j.Hashes[j.Backup], err = digest(j.Backup)
	if err != nil {
		return nil, nil, err
	}
	if err = verifyJournal(ctx, j); err != nil {
		return nil, nil, err
	}
	success = true
	return j, dataLock, nil
}

func fileDigest(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	hash := sha256.New()
	if _, err = io.Copy(hash, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

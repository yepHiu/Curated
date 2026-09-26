package installmigration

import (
	"archive/zip"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"curated-backend/internal/backup"
	"curated-backend/internal/config"
	"curated-backend/internal/processlock"
)

type fakePlatform struct {
	old                                             *Legacy
	stoppedErr, splitErr, uninstallErr, completeErr error
	startup                                         string
	removed, disabled, completed                    int
	beforeRemove                                    func()
}

func (p *fakePlatform) FindLegacy() (*Legacy, error)    { return p.old, nil }
func (p *fakePlatform) CheckStopped(*Legacy) error      { return p.stoppedErr }
func (p *fakePlatform) CheckNoSplitServer() error       { return p.splitErr }
func (p *fakePlatform) StartupCommand() (string, error) { return p.startup, nil }
func (p *fakePlatform) DisableOldStartup(*Legacy, string) error {
	p.disabled++
	p.startup = ""
	return nil
}
func (p *fakePlatform) Uninstall(context.Context, *Legacy) error {
	if p.beforeRemove != nil {
		p.beforeRemove()
	}
	if p.uninstallErr != nil {
		return p.uninstallErr
	}
	p.removed++
	p.old = nil
	return nil
}
func (p *fakePlatform) Complete(string, bool) error {
	if p.completeErr != nil {
		return p.completeErr
	}
	p.completed++
	return nil
}

func fixture(t *testing.T) (Options, *fakePlatform, string) {
	t.Helper()
	root := t.TempDir()
	data := filepath.Join(root, "custom data")
	old := filepath.Join(root, "old program")
	os.MkdirAll(data, 0700)
	os.MkdirAll(old, 0700)
	dbPath := filepath.Join(data, "original.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE schema_migrations(name TEXT PRIMARY KEY, applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP); INSERT INTO schema_migrations(name) VALUES ('0001_init.sql'); CREATE TABLE favorites(id INTEGER PRIMARY KEY,rating INTEGER); INSERT INTO favorites VALUES (1,5);`)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()
	cfgPath := filepath.Join(root, "custom.json")
	raw, _ := json.Marshal(map[string]any{"databasePath": dbPath, "cacheDir": filepath.Join(data, "cache"), "logDir": filepath.Join(data, "logs"), "httpAddr": "127.0.0.1:18881"})
	os.WriteFile(cfgPath, raw, 0600)
	o := Options{StateDir: filepath.Join(root, "migration"), ProfilePath: filepath.Join(root, "server-startup.json"), DataRoot: data, ConfigPath: cfgPath}
	p := &fakePlatform{old: &Legacy{Directory: old, Uninstaller: filepath.Join(old, "unins000.exe"), Version: "1.5.8"}, startup: `old-startup`}
	return o, p, dbPath
}

func TestPrepareBacksUpBeforeRemovalAndPreservesDatabaseAndConfig(t *testing.T) {
	o, p, dbPath := fixture(t)
	ctx := context.Background()
	p.beforeRemove = func() {
		j, err := readJournal(o)
		if err != nil || j == nil || j.Stage != "prepared" {
			t.Fatalf("no durable journal before removal: %+v %v", j, err)
		}
		if err := verifyJournal(ctx, j); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(o.ProfilePath); err != nil {
			t.Fatal(err)
		}
		if lock, err := processlock.Acquire(dbPath + ".runtime.lock"); err == nil {
			lock.Release()
			t.Fatal("DB must remain locked throughout removal")
		}
	}
	if err := Prepare(ctx, o, p); err != nil {
		t.Fatal(err)
	}
	j, _ := readJournal(o)
	if j.Stage != "removed" || p.removed != 1 {
		t.Fatal(j.Stage, p.removed)
	}
	if j.Profile.ConfigPath == o.ConfigPath {
		t.Fatal("custom config was not snapshotted")
	}
	if err := Complete(ctx, o, p); err != nil {
		t.Fatal(err)
	}
	if err := Prepare(ctx, o, p); err != nil {
		t.Fatal(err)
	}
	if err := Complete(ctx, o, p); err != nil {
		t.Fatal(err)
	}
	if p.completed != 1 {
		t.Fatal("completion not idempotent")
	}
	db, _ := sql.Open("sqlite", dbPath)
	defer db.Close()
	var rating, count int
	db.QueryRow("SELECT rating FROM favorites WHERE id=1").Scan(&rating)
	db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	if rating != 5 || count != 1 {
		t.Fatal("source data/schema changed", rating, count)
	}
}

func TestCancelledRemovalTakesFreshBackupOnRetryAndKeepsStartupIntent(t *testing.T) {
	o, p, dbPath := fixture(t)
	ctx := context.Background()
	p.uninstallErr = errors.New("UAC cancelled")
	if err := Prepare(ctx, o, p); err == nil {
		t.Fatal("expected cancellation")
	}
	first, _ := readJournal(o)
	db, _ := sql.Open("sqlite", dbPath)
	db.Exec("UPDATE favorites SET rating=4")
	db.Close()
	p.uninstallErr = nil
	if err := Prepare(ctx, o, p); err != nil {
		t.Fatal(err)
	}
	second, _ := readJournal(o)
	if second.Backup == first.Backup || second.Startup != "old-startup" {
		t.Fatal("retry lost backup freshness/startup", second)
	}
	if _, err := os.Stat(first.Backup); err != nil {
		t.Fatal("old backup removed")
	}
}

func TestResumeAfterRemovalAndPartialComponentFailure(t *testing.T) {
	o, p, _ := fixture(t)
	ctx := context.Background()
	if err := Prepare(ctx, o, p); err != nil {
		t.Fatal(err)
	}
	p.completeErr = errors.New("Desktop missing")
	if err := Complete(ctx, o, p); err == nil {
		t.Fatal("expected incomplete components")
	}
	o.DataRoot, o.ConfigPath = "", ""
	if err := Prepare(ctx, o, p); err != nil {
		t.Fatal(err)
	}
	p.completeErr = nil
	if err := Complete(ctx, o, p); err != nil {
		t.Fatal(err)
	}
	if p.removed != 1 {
		t.Fatal("uninstall repeated")
	}
}

func TestPreflightFailuresNeverRemoveProgram(t *testing.T) {
	for _, scenario := range []string{"running", "split", "missing-db", "locked-db", "program-data", "corrupt-db", "relative-config", "overlap-media"} {
		t.Run(scenario, func(t *testing.T) {
			o, p, dbPath := fixture(t)
			switch scenario {
			case "running":
				p.stoppedErr = errors.New("running")
			case "split":
				p.splitErr = errors.New("split already installed")
			case "missing-db":
				os.Remove(dbPath)
			case "locked-db":
				l, err := processlock.Acquire(dbPath + ".runtime.lock")
				if err != nil {
					t.Fatal(err)
				}
				defer l.Release()
			case "program-data":
				os.WriteFile(filepath.Join(p.old.Directory, "personal.db"), []byte("data"), 0600)
			case "corrupt-db":
				os.WriteFile(dbPath, []byte("not sqlite"), 0600)
			case "relative-config":
				os.WriteFile(o.ConfigPath, []byte(`{"databasePath":"relative.db"}`), 0600)
			case "overlap-media":
				db, _ := sql.Open("sqlite", dbPath)
				db.Exec("CREATE TABLE library_paths(path TEXT)")
				db.Exec("INSERT INTO library_paths VALUES (?)", filepath.Join(p.old.Directory, "media"))
				db.Close()
			}
			if err := Prepare(context.Background(), o, p); err == nil {
				t.Fatal("unsafe preflight passed")
			}
			if p.removed != 0 || p.disabled != 0 {
				t.Fatal("modified old program/startup on failure")
			}
		})
	}
}

func TestTamperedBackupBlocksResume(t *testing.T) {
	o, p, _ := fixture(t)
	ctx := context.Background()
	if err := Prepare(ctx, o, p); err != nil {
		t.Fatal(err)
	}
	j, _ := readJournal(o)
	os.WriteFile(j.Backup, []byte("broken"), 0600)
	if err := Prepare(ctx, o, p); err == nil {
		t.Fatal("tampered backup accepted")
	}
	if err := Complete(ctx, o, p); err == nil {
		t.Fatal("tampered backup completed")
	}
}

func TestSymlinkDataUnderProgramCannotBypassContainment(t *testing.T) {
	o, p, _ := fixture(t)
	alias := filepath.Join(filepath.Dir(o.DataRoot), "alias")
	if err := os.Symlink(p.old.Directory, alias); err != nil {
		t.Skip(err)
	}
	if err := outside(p.old.Directory, filepath.Join(alias, "future", "db")); err == nil {
		t.Fatal("junction/symlink containment bypass")
	}
}

func TestSplitOnlyInstallDoesNotCreateLegacyState(t *testing.T) {
	root := t.TempDir()
	o := Options{StateDir: filepath.Join(root, "migration"), ProfilePath: filepath.Join(root, "server-startup.json")}
	p := &fakePlatform{}
	if err := Prepare(context.Background(), o, p); err != nil {
		t.Fatal(err)
	}
	if err := Complete(context.Background(), o, p); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(o.StateDir); !os.IsNotExist(err) {
		t.Fatal("created migration state for a split-only install", err)
	}
}

func TestDefaultLayoutWithoutCustomConfiguration(t *testing.T) {
	if config.Default().DatabasePath == filepath.FromSlash("backend/runtime/curated.db") || config.Default().DatabasePath == filepath.FromSlash("runtime/curated.db") {
		t.Skip("default production paths require -tags release")
	}
	o, p, dbPath := fixture(t)
	target := filepath.Join(o.DataRoot, "data", "curated.db")
	if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(dbPath, target); err != nil {
		t.Fatal(err)
	}
	o.ConfigPath = ""
	if err := Prepare(context.Background(), o, p); err != nil {
		t.Fatal(err)
	}
	j, err := readJournal(o)
	if err != nil {
		t.Fatal(err)
	}
	if j.Profile.DatabasePath != target || j.Profile.ConfigPath != "" {
		t.Fatalf("wrong default profile: %+v", j.Profile)
	}
}

func TestBackupIncludesCommittedWAL(t *testing.T) {
	o, p, dbPath := fixture(t)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec("PRAGMA journal_mode=WAL; PRAGMA wal_autocheckpoint=0; UPDATE favorites SET rating=3;"); err != nil {
		t.Fatal(err)
	}
	if err = Prepare(context.Background(), o, p); err != nil {
		t.Fatal(err)
	}
	j, _ := readJournal(o)
	archive, err := zip.OpenReader(j.Backup)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	for _, entry := range archive.File {
		if entry.Name != backup.DatabaseArchivePath {
			continue
		}
		input, err := entry.Open()
		if err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(t.TempDir(), "snapshot.db")
		output, err := os.Create(target)
		if err != nil {
			t.Fatal(err)
		}
		_, err = io.Copy(output, input)
		input.Close()
		output.Close()
		if err != nil {
			t.Fatal(err)
		}
		copyDB, err := sql.Open("sqlite", target)
		if err != nil {
			t.Fatal(err)
		}
		defer copyDB.Close()
		var rating int
		if err = copyDB.QueryRow("SELECT rating FROM favorites WHERE id=1").Scan(&rating); err != nil {
			t.Fatal(err)
		}
		if rating != 3 {
			t.Fatalf("backup lost WAL state: %d", rating)
		}
		return
	}
	t.Fatal("database snapshot not found")
}

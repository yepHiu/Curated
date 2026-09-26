package launchprofile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveRetainsCustomConfigAcrossDefaultAndInheritedLaunches(t *testing.T) {
	root := t.TempDir()
	p := Profile{Schema: 1, DataRoot: root, DatabasePath: filepath.Join(root, "old.db"), ConfigPath: filepath.Join(root, "config.json")}
	for _, path := range []string{p.DatabasePath, p.ConfigPath} {
		if err := os.WriteFile(path, []byte("existing"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	profilePath := filepath.Join(root, "profile.json")
	raw, _ := json.Marshal(p)
	os.WriteFile(profilePath, raw, 0600)
	for _, env := range []string{"", root} {
		actual, err := Resolve(profilePath, "", env)
		if err != nil || actual == nil || *actual != p {
			t.Fatalf("resolve %q: %+v %v", env, actual, err)
		}
	}
	for _, input := range [][2]string{{"explicit.json", ""}, {"", filepath.Join(root, "different")}} {
		actual, err := Resolve(profilePath, input[0], input[1])
		if err != nil || actual != nil {
			t.Fatalf("explicit override: %+v %v", actual, err)
		}
	}
	os.Remove(p.DatabasePath)
	if _, err := Resolve(profilePath, "", ""); err == nil {
		t.Fatal("missing migrated DB must not create an empty library")
	}
}

func TestResolveMissingAndCorruptProfile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "profile.json")
	if actual, err := Resolve(p, "", ""); err != nil || actual != nil {
		t.Fatal(actual, err)
	}
	os.WriteFile(p, []byte("{"), 0600)
	if _, err := Resolve(p, "", ""); err == nil {
		t.Fatal("corruption ignored")
	}
}

func TestMissingProfileWithMigrationJournalCannotOpenDefaultEmptyLibrary(t *testing.T) {
	root := t.TempDir()
	journal := filepath.Join(root, "installer-migrations", "legacy.json")
	if err := os.MkdirAll(filepath.Dir(journal), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(journal, []byte(`{"schema":1,"stage":"complete"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve(filepath.Join(root, "server-startup.json"), "", ""); err == nil {
		t.Fatal("lost migrated startup profile silently fell back to defaults")
	}
	if actual, err := Resolve(filepath.Join(root, "server-startup.json"), "explicit.json", ""); err != nil || actual != nil {
		t.Fatal("explicit recovery config was blocked", err)
	}
}

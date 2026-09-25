package serveridentity

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIdentitySurvivesRestartAndIsIndependentOfLibrary(t *testing.T) {
	db := filepath.Join(t.TempDir(), "data", "curated.db")
	first, err := LoadOrCreate(db)
	if err != nil {
		t.Fatal(err)
	}
	again, err := LoadOrCreate(db)
	if err != nil || again != first {
		t.Fatalf("identity changed: %q %v", again, err)
	}
	other, err := LoadOrCreate(filepath.Join(t.TempDir(), "curated.db"))
	if err != nil || other == first {
		t.Fatalf("new installation reused identity: %v", err)
	}
	if err := os.WriteFile(db+".server-id", []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadOrCreate(db); err == nil {
		t.Fatal("must not silently replace a corrupt identity")
	}
}

func TestConnectionHintPreservesCustomPortWithoutLibraryDetails(t *testing.T) {
	root := t.TempDir()
	t.Setenv("LOCALAPPDATA", root)
	if err := WriteConnectionHint("0.0.0.0:9001"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "Curated", "server-connection.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"url":"http://127.0.0.1:9001"}` {
		t.Fatal(string(data))
	}
}

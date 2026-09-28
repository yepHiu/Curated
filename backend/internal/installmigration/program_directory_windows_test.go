//go:build windows

package installmigration

import (
	"os"
	"path/filepath"
	"testing"
)

// TestValidateProgramDirectory checks empty fresh destinations and protects existing program-local data.
func TestValidateProgramDirectory(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "Curated Server")
	if err := ValidateProgramDirectory(target, ""); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(target, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "curated.exe"), []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateProgramDirectory(target, ""); err == nil {
		t.Fatal("adopted unrelated nonempty folder")
	}
	if err := ValidateProgramDirectory(target, target); err != nil {
		t.Fatal(err)
	}
	for _, directory := range []string{".", filepath.VolumeName(root) + `\`, filepath.Join(root, "other")} {
		if err := ValidateProgramDirectory(directory, target); err == nil {
			t.Fatalf("accepted unsafe destination %s", directory)
		}
	}
	if err := os.WriteFile(filepath.Join(target, "curated.db"), []byte("user data"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateProgramDirectory(target, target); err == nil {
		t.Fatal("accepted program-local database in managed tree")
	}
}

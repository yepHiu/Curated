package library

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestOrganizeVideoFile_looseFile(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	src := filepath.Join(root, "489155.com@EBWH-287-C.mp4")
	if err := os.WriteFile(src, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := OrganizeVideoFile(src, "EBWH-287")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "EBWH-287", "EBWH-287.mp4")
	if filepath.Clean(out) != filepath.Clean(want) {
		t.Fatalf("got %q want %q", out, want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("source should be gone: %v", err)
	}
}

func TestOrganizeVideoFile_sameFolderRename(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "ABC-100")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(dir, "foo.mp4")
	if err := os.WriteFile(src, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := OrganizeVideoFile(src, "ABC-100")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "ABC-100.mp4")
	if filepath.Clean(out) != filepath.Clean(want) {
		t.Fatalf("got %q want %q", out, want)
	}
}

func TestOrganizeVideoFile_idempotent(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "XYZ-200")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	canon := filepath.Join(dir, "XYZ-200.mp4")
	if err := os.WriteFile(canon, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := OrganizeVideoFile(canon, "XYZ-200")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Clean(out) != filepath.Clean(canon) {
		t.Fatalf("got %q want %q", out, canon)
	}
}

func TestOrganizeVideoFile_conflict(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "MIDE-300")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(root, "other.mp4")
	if err := os.WriteFile(src, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	occupied := filepath.Join(dir, "MIDE-300.mp4")
	if err := os.WriteFile(occupied, []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := OrganizeVideoFile(src, "MIDE-300")
	if !errors.Is(err, ErrOrganizeConflict) {
		t.Fatalf("expected ErrOrganizeConflict, got %v", err)
	}
}

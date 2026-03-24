package curated

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAndValidate_ok_singleVideo(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "ABC-100")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ABC-100.mp4"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := `{
  "schemaVersion": 1,
  "layout": "curated-movie-root-v1",
  "code": "ABC-100"
}`
	if err := os.WriteFile(filepath.Join(dir, ManifestFileName), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := LoadAndValidate(dir)
	if err != nil {
		t.Fatal(err)
	}
	if m.Code != "ABC-100" {
		t.Fatalf("code %q", m.Code)
	}
}

func TestLoadAndValidate_primaryVideo(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "XYZ-200")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.mp4"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.mp4"), []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := `{
  "schemaVersion": 1,
  "layout": "curated-movie-root-v1",
  "code": "XYZ-200",
  "primaryVideo": "b.mp4"
}`
	if err := os.WriteFile(filepath.Join(dir, ManifestFileName), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadAndValidate(dir)
	if err != nil {
		t.Fatal(err)
	}
}

func TestLoadAndValidate_ambiguous(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "MIDE-300")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.mp4"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.mp4"), []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := `{
  "schemaVersion": 1,
  "layout": "curated-movie-root-v1",
  "code": "MIDE-300"
}`
	if err := os.WriteFile(filepath.Join(dir, ManifestFileName), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadAndValidate(dir)
	if !errors.Is(err, ErrAmbiguousVideos) {
		t.Fatalf("want ErrAmbiguousVideos, got %v", err)
	}
}

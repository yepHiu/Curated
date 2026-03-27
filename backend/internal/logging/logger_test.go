package logging

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNew_ConsoleOnly(t *testing.T) {
	logger, err := New("info", FileSink{})
	if err != nil {
		t.Fatal(err)
	}
	logger.Info("ok")
	_ = logger.Sync()
}

func TestNew_InvalidLevel(t *testing.T) {
	_, err := New("not-a-level", FileSink{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNew_FileDirCreation(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "logs")
	logger, err := New("info", FileSink{
		LogDir:     dir,
		FilePrefix: "javd",
		MaxAgeDays: 7,
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = logger.Sync()
	if _, err := os.Stat(dir); err != nil {
		t.Fatal(err)
	}
}

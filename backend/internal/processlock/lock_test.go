package processlock

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestAcquireIsExclusiveAndReusableAfterRelease(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "curated.db.runtime.lock")
	first, err := Acquire(lockPath)
	if err != nil {
		t.Fatalf("Acquire(first): %v", err)
	}
	if first.Path() == "" {
		t.Fatal("lock path is empty")
	}
	if _, err := Acquire(lockPath); !errors.Is(err, ErrAlreadyLocked) {
		t.Fatalf("Acquire(second) error = %v, want ErrAlreadyLocked", err)
	}
	if err := first.Release(); err != nil {
		t.Fatalf("Release(first): %v", err)
	}
	if err := first.Release(); err != nil {
		t.Fatalf("Release(first again): %v", err)
	}
	second, err := Acquire(lockPath)
	if err != nil {
		t.Fatalf("Acquire(after release): %v", err)
	}
	if err := second.Release(); err != nil {
		t.Fatalf("Release(second): %v", err)
	}
}

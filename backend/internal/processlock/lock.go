// Package processlock provides a cross-process exclusive file lock.
package processlock

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ErrAlreadyLocked means another process currently owns the lock.
var ErrAlreadyLocked = errors.New("process lock is already held")

// Lock owns an exclusive advisory lock until Release is called.
type Lock struct {
	file    *os.File
	path    string
	once    sync.Once
	release error
}

// Acquire opens lockPath and takes a non-blocking exclusive lock.
func Acquire(lockPath string) (*Lock, error) {
	absPath, err := filepath.Abs(lockPath)
	if err != nil {
		return nil, fmt.Errorf("resolve process lock path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(absPath), 0o700); err != nil {
		return nil, fmt.Errorf("create process lock directory: %w", err)
	}
	file, err := os.OpenFile(absPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open process lock: %w", err)
	}
	if err := lockFile(file); err != nil {
		_ = file.Close()
		if errors.Is(err, ErrAlreadyLocked) {
			return nil, fmt.Errorf("%w: %s", ErrAlreadyLocked, absPath)
		}
		return nil, fmt.Errorf("lock %s: %w", absPath, err)
	}
	if err := file.Truncate(0); err == nil {
		_, _ = file.WriteAt([]byte(fmt.Sprintf("pid=%d\nacquiredAt=%s\n", os.Getpid(), time.Now().UTC().Format(time.RFC3339Nano))), 0)
		_ = file.Sync()
	}
	return &Lock{file: file, path: absPath}, nil
}

// Release unlocks and closes the lock file. It is safe to call more than once.
func (l *Lock) Release() error {
	if l == nil {
		return nil
	}
	l.once.Do(func() {
		unlockErr := unlockFile(l.file)
		closeErr := l.file.Close()
		l.release = errors.Join(unlockErr, closeErr)
	})
	return l.release
}

// Path returns the absolute lock file path.
func (l *Lock) Path() string {
	if l == nil {
		return ""
	}
	return l.path
}

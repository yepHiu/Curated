// Package serveridentity owns the persistent identity of a Server installation.
package serveridentity

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// LoadOrCreate is called while holding the database runtime lock. The identity
// is deliberately outside the database backup: restoring a library into a new
// installation must not clone the original server's discovery identity.
func LoadOrCreate(databasePath string) (string, error) {
	path := databasePath + ".server-id"
	data, err := os.ReadFile(path)
	if err == nil {
		id, err := uuid.Parse(strings.TrimSpace(string(data)))
		if err != nil {
			return "", fmt.Errorf("invalid server identity: %w", err)
		}
		return id.String(), nil
	}
	if !os.IsNotExist(err) {
		return "", fmt.Errorf("read server identity: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return "", err
	}
	id := uuid.NewString()
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", fmt.Errorf("create server identity: %w", err)
	}
	_, writeErr := f.WriteString(id + "\n")
	if writeErr == nil {
		writeErr = f.Sync()
	}
	closeErr := f.Close()
	if writeErr != nil {
		return "", writeErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	return id, nil
}

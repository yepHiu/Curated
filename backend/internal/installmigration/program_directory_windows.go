//go:build windows

package installmigration

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidateProgramDirectory protects the installer-owned tree before recursive
// resource replacement. Moving an existing installation needs its own workflow.
func ValidateProgramDirectory(directory, installed string) error {
	if !filepath.IsAbs(directory) || len(filepath.VolumeName(directory)) != 2 {
		return fmt.Errorf("choose an absolute folder on a local drive")
	}
	directory = filepath.Clean(directory)
	if filepath.Dir(directory) == directory {
		return fmt.Errorf("choose a dedicated program folder, not a drive root")
	}
	resolved, err := canonical(directory)
	if err != nil {
		return err
	}
	if !strings.EqualFold(directory, resolved) {
		return fmt.Errorf("linked installation folders are not supported: %s", directory)
	}
	if installed != "" {
		if !filepath.IsAbs(installed) || !strings.EqualFold(directory, filepath.Clean(installed)) {
			return fmt.Errorf("keep the existing installation folder: %s", installed)
		}
		if _, err := os.Stat(directory); os.IsNotExist(err) {
			return nil // reinstall a removed program, retaining its registered path
		}
		return checkProgramTree(directory)
	}
	entries, err := os.ReadDir(directory)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(entries) != 0 {
		return fmt.Errorf("choose an empty folder dedicated to this component")
	}
	return nil
}

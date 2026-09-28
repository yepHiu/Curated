package config

import "path/filepath"

// DefaultBackupDirectory follows the Server data root, outside the installation
// directory. Merely resolving a destination does not create it on disk.
func DefaultBackupDirectory() string {
	root := curatedDataRoot()
	if root == "" {
		root = filepath.Dir(defaultDatabasePath())
	}
	directory := filepath.Join(root, "backups")
	if absolute, err := filepath.Abs(directory); err == nil {
		return absolute
	}
	return directory
}

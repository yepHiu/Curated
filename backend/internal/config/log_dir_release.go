//go:build release

package config

import "path/filepath"

func defaultLogDir() string {
	root := curatedDataRoot()
	if root == "" {
		return filepath.FromSlash("Curated/logs")
	}
	return filepath.Join(root, "logs")
}

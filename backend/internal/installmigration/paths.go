package installmigration

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func within(root, path string) bool {
	root, path = strings.ToLower(filepath.Clean(root)), strings.ToLower(filepath.Clean(path))
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

// Resolve existing ancestors as well: a new path below a junction can still be
// inside the directory the old uninstaller will remove.
func canonical(path string) (string, error) {
	current := filepath.Clean(path)
	tail := []string{}
	for {
		resolved, err := filepath.EvalSymlinks(current)
		if err == nil {
			for i := len(tail) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, tail[i])
			}
			return resolved, nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", err
		}
		tail = append(tail, filepath.Base(current))
		current = parent
	}
}

func outside(root, path string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("migration requires absolute paths: %s", path)
	}
	realRoot, err := canonical(root)
	if err != nil {
		return err
	}
	realPath, err := canonical(path)
	if err != nil {
		return err
	}
	if within(realRoot, realPath) {
		return fmt.Errorf("user data is inside the old program directory (%s); move it with the migration guide before upgrading; no program was removed", path)
	}
	return nil
}

func validateConfigPaths(path, oldRoot string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var object map[string]any
	if err = json.Unmarshal(raw, &object); err != nil {
		return err
	}
	// Only file-system settings: do not interpret URLs, IDs or credentials as paths.
	pathKeys := map[string]bool{"databasePath": true, "cacheDir": true, "logDir": true, "backupDirectory": true, "streamSessionRoot": true}
	var walk func(map[string]any) error
	walk = func(m map[string]any) error {
		for key, value := range m {
			if child, ok := value.(map[string]any); ok {
				if err := walk(child); err != nil {
					return err
				}
			}
			paths := []string{}
			if key == "ffmpegCommand" || key == "nativePlayerCommand" {
				if p, ok := value.(string); ok && strings.ContainsAny(p, `/\`) {
					paths = append(paths, p)
				}
			}
			if pathKeys[key] {
				if p, ok := value.(string); ok && p != "" {
					paths = append(paths, p)
				}
			}
			if key == "libraryPaths" {
				if values, ok := value.([]any); ok {
					for _, v := range values {
						if p, ok := v.(string); ok {
							paths = append(paths, p)
						}
					}
				}
			}
			for _, p := range paths {
				if err := outside(oldRoot, p); err != nil {
					return fmt.Errorf("%s in %s: %w", key, path, err)
				}
			}
		}
		return nil
	}
	return walk(object)
}

func checkProgramTree(root string) error {
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("old program contains a linked path (%s); resolve it before automatic migration", path)
		}
		if entry.IsDir() {
			return nil
		}
		name := strings.ToLower(entry.Name())
		if strings.HasSuffix(name, ".db") || strings.HasSuffix(name, ".sqlite") || strings.HasSuffix(name, ".sqlite3") || name == "library-config.cfg" {
			return fmt.Errorf("old program contains user data/configuration (%s); automatic removal was cancelled", path)
		}
		return nil
	})
}

func checkStoredPaths(ctx context.Context, database, oldRoot string) error {
	db, err := sql.Open("sqlite", database)
	if err != nil {
		return err
	}
	defer db.Close()
	// Old schemas may predate comics/photos. Query only tables present in this DB.
	for _, table := range []string{"library_paths", "comic_library_paths", "photo_library_paths"} {
		var count int
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count); err != nil {
			return err
		}
		if count == 0 {
			continue
		}
		rows, err := db.QueryContext(ctx, "SELECT path FROM "+table)
		if err != nil {
			return err
		}
		for rows.Next() {
			var path string
			if err = rows.Scan(&path); err == nil {
				err = outside(oldRoot, path)
			}
			if err != nil {
				rows.Close()
				return err
			}
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func copySnapshot(source, target string) error {
	from, err := os.Open(source)
	if err != nil {
		return err
	}
	defer from.Close()
	to, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, err = io.Copy(to, from)
	if err == nil {
		err = to.Sync()
	}
	closeErr := to.Close()
	if err != nil {
		return err
	}
	return closeErr
}

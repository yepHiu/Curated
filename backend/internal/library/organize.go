package library

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var (
	// ErrOrganizeConflict means dest file exists and is not the source file.
	ErrOrganizeConflict = errors.New("library organize: destination file already exists")
	// ErrOrganizeInvalid is returned for empty path or number.
	ErrOrganizeInvalid = errors.New("library organize: invalid path or number")
)

// OrganizeVideoFile moves the video into {parent}/{number}/{number}{ext}.
// If the file already sits at that path, returns absPath unchanged.
// If parent folder is already named {number}, only renames to {number}{ext} inside it.
func OrganizeVideoFile(absPath, number string) (string, error) {
	absPath = filepath.Clean(absPath)
	number = strings.TrimSpace(number)
	if absPath == "" || number == "" {
		return "", ErrOrganizeInvalid
	}

	ext := filepath.Ext(absPath)
	if ext == "" {
		return "", fmt.Errorf("%w: missing extension", ErrOrganizeInvalid)
	}

	parent := filepath.Dir(absPath)

	var destDir string
	if strings.EqualFold(filepath.Base(parent), number) {
		destDir = parent
	} else {
		destDir = filepath.Join(parent, number)
	}

	destFile := filepath.Join(destDir, number+ext)

	if strings.EqualFold(filepath.Clean(absPath), filepath.Clean(destFile)) {
		return destFile, nil
	}

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", err
	}

	if fi, err := os.Stat(destFile); err == nil {
		if sameFilePath(absPath, destFile) {
			return destFile, nil
		}
		if fi.IsDir() {
			return "", fmt.Errorf("%w: %q is a directory", ErrOrganizeConflict, destFile)
		}
		return "", fmt.Errorf("%w: %q", ErrOrganizeConflict, destFile)
	}

	if err := moveFile(absPath, destFile); err != nil {
		return "", err
	}

	return filepath.Clean(destFile), nil
}

func sameFilePath(a, b string) bool {
	sa, err1 := os.Stat(a)
	sb, err2 := os.Stat(b)
	if err1 != nil || err2 != nil {
		return false
	}
	return os.SameFile(sa, sb)
}

func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	// Cross-device or other Rename failure: copy + remove.
	if err := copyFile(dst, src); err != nil {
		return err
	}
	return os.Remove(src)
}

func copyFile(dst, src string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()

	if _, err := io.Copy(out, in); err != nil {
		_ = os.Remove(dst)
		return err
	}
	return out.Sync()
}

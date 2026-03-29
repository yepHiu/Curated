// Package shellopen opens the OS file manager at a path (reveal / select file).
package shellopen

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const execTimeout = 20 * time.Second

// RevealInFileManager opens the default file manager and selects the given absolute file path
// where the platform supports it (Windows explorer /select, macOS open -R).
// On Linux it opens the parent directory with xdg-open (file may not be highlighted).
func RevealInFileManager(ctx context.Context, absFilePath string) error {
	if absFilePath == "" {
		return fmt.Errorf("empty path")
	}
	if !filepath.IsAbs(absFilePath) {
		return fmt.Errorf("path must be absolute: %q", absFilePath)
	}

	switch runtime.GOOS {
	case "windows":
		// Must not use exec.CommandContext with the HTTP request context: once the handler
		// returns 204 the request context is cancelled and the explorer child gets killed.
		return revealWindows(absFilePath)
	}

	execCtx, cancel := context.WithTimeout(ctx, execTimeout)
	defer cancel()

	switch runtime.GOOS {
	case "darwin":
		return revealDarwin(execCtx, absFilePath)
	default:
		return revealLinux(execCtx, absFilePath)
	}
}

func revealWindows(abs string) error {
	windir := os.Getenv("WINDIR")
	if windir == "" {
		windir = `C:\Windows`
	}
	explorer := filepath.Join(windir, "explorer.exe")
	p := filepath.Clean(abs)
	// One argv: /select,<path>. Quote path when it contains spaces (explorer.exe rules).
	arg := "/select," + p
	if strings.ContainsAny(p, " \t") {
		arg = `/select,"` + p + `"`
	}
	cmd := exec.Command(explorer, arg)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("explorer: %w", err)
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

func revealDarwin(ctx context.Context, abs string) error {
	cmd := exec.CommandContext(ctx, "open", "-R", abs)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if len(out) > 0 {
			return fmt.Errorf("open -R: %w: %s", err, string(out))
		}
		return fmt.Errorf("open -R: %w", err)
	}
	return nil
}

func revealLinux(ctx context.Context, abs string) error {
	dir := filepath.Dir(abs)
	cmd := exec.CommandContext(ctx, "xdg-open", dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if len(out) > 0 {
			return fmt.Errorf("xdg-open: %w: %s", err, string(out))
		}
		return fmt.Errorf("xdg-open: %w", err)
	}
	return nil
}

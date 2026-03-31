package shellopen

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// OpenURL opens a URL in the default browser.
func OpenURL(ctx context.Context, target string) error {
	if target == "" {
		return fmt.Errorf("empty url")
	}

	switch runtime.GOOS {
	case "windows":
		return openWindows(target)
	case "darwin":
		return runCommand(ctx, "open", target)
	default:
		return runCommand(ctx, "xdg-open", target)
	}
}

// OpenDirectory opens a directory in the default file manager.
func OpenDirectory(ctx context.Context, absDir string) error {
	if absDir == "" {
		return fmt.Errorf("empty directory")
	}
	if !filepath.IsAbs(absDir) {
		return fmt.Errorf("path must be absolute: %q", absDir)
	}
	if err := os.MkdirAll(absDir, 0o755); err != nil {
		return err
	}

	switch runtime.GOOS {
	case "windows":
		return openWindows(absDir)
	case "darwin":
		return runCommand(ctx, "open", absDir)
	default:
		return runCommand(ctx, "xdg-open", absDir)
	}
}

func openWindows(target string) error {
	windir := os.Getenv("WINDIR")
	if windir == "" {
		windir = `C:\Windows`
	}
	explorer := filepath.Join(windir, "explorer.exe")
	cmd := exec.Command(explorer, target)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("explorer: %w", err)
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

func runCommand(ctx context.Context, name string, arg string) error {
	execCtx, cancel := context.WithTimeout(ctx, execTimeout)
	defer cancel()
	cmd := exec.CommandContext(execCtx, name, arg)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if len(out) > 0 {
			return fmt.Errorf("%s: %w: %s", name, err, string(out))
		}
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

// OpenPathBestEffort is used for desktop affordances where a no-op is better than crashing.
func OpenPathBestEffort(target string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if filepath.IsAbs(target) {
		_ = OpenDirectory(ctx, target)
		return
	}
	_ = OpenURL(ctx, target)
}

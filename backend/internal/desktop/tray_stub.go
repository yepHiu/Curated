//go:build !windows

package desktop

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"curated-backend/internal/config"
)

// TrayOptions configures the system tray (no-op on non-Windows).
type TrayOptions struct {
	Logger      *zap.Logger
	Config      config.Config
	OpenURL     string
	SettingsURL string
	LogDir      string
	Cancel      context.CancelFunc
}

// RunTray returns an error on non-Windows platforms.
func RunTray(_ context.Context, _ TrayOptions) error {
	return fmt.Errorf("tray mode is only supported on Windows")
}

// WaitForServerReady is a no-op on non-Windows platforms.
func WaitForServerReady(ctx context.Context, _ string) error {
	return ctx.Err()
}

// ResolveBaseURL normalizes an address into a localhost HTTP base URL.
func ResolveBaseURL(addr string) string {
	host := strings.TrimSpace(addr)
	if host == "" {
		host = config.DefaultHTTPAddr()
	}
	if strings.HasPrefix(host, ":") {
		host = "127.0.0.1" + host
	}
	host = strings.ReplaceAll(host, "0.0.0.0", "127.0.0.1")
	host = strings.ReplaceAll(host, "[::]", "127.0.0.1")
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "http://" + host
	}
	return strings.TrimRight(host, "/")
}

// ResolveDefaultLogDir returns the configured log directory or a fallback.
func ResolveDefaultLogDir(cfg config.Config) string {
	if dir := config.ResolveLogDir(cfg.LogDir); dir != "" {
		return dir
	}
	return "."
}

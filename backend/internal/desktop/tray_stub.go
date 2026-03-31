//go:build !windows

package desktop

import (
	"context"
	"fmt"

	"curated-backend/internal/config"
)

type TrayOptions struct {
	Config      config.Config
	OpenURL     string
	SettingsURL string
	LogDir      string
	Cancel      context.CancelFunc
}

func RunTray(_ context.Context, _ TrayOptions) error {
	return fmt.Errorf("tray mode is only supported on Windows")
}

func WaitForServerReady(ctx context.Context, _ string) error {
	return ctx.Err()
}

func ResolveBaseURL(addr string) string {
	if addr == "" {
		return "http://127.0.0.1:8080"
	}
	return addr
}

func ResolveDefaultLogDir(cfg config.Config) string {
	if cfg.LogDir != "" {
		return cfg.LogDir
	}
	return "."
}

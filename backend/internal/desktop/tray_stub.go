//go:build !windows

package desktop

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"curated-backend/internal/config"
)

type TrayOptions struct {
	Logger      *zap.Logger
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

func ResolveDefaultLogDir(cfg config.Config) string {
	if cfg.LogDir != "" {
		return cfg.LogDir
	}
	return "."
}

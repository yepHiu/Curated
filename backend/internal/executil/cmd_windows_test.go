//go:build windows

package executil

import (
	"context"
	"testing"
)

func TestCommandContextHidesConsoleWindowOnWindows(t *testing.T) {
	t.Parallel()

	cmd := CommandContext(context.Background(), "cmd.exe", "/c", "echo", "ok")
	if cmd == nil {
		t.Fatal("CommandContext returned nil")
	}
	if cmd.SysProcAttr == nil {
		t.Fatal("expected SysProcAttr to be configured on Windows")
	}
	if !cmd.SysProcAttr.HideWindow {
		t.Fatal("expected HideWindow=true on Windows child processes")
	}
}

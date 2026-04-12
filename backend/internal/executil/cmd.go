package executil

import (
	"context"
	"os/exec"
)

// CommandContext centralizes process startup tweaks so background helper
// processes behave consistently across playback code paths.
func CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	configureBackgroundProcess(cmd)
	return cmd
}

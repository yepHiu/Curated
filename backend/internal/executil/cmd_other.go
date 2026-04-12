//go:build !windows

package executil

import "os/exec"

func configureBackgroundProcess(cmd *exec.Cmd) {
	_ = cmd
}

//go:build linux || darwin || freebsd || openbsd || netbsd || dragonfly

package playback

import (
	"os"
	"syscall"
)

func setProcessPaused(process *os.Process, paused bool) error {
	signal := syscall.SIGCONT
	if paused {
		signal = syscall.SIGSTOP
	}
	return process.Signal(signal)
}

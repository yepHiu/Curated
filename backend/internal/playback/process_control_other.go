//go:build !windows && !linux && !darwin && !freebsd && !openbsd && !netbsd && !dragonfly

package playback

import (
	"fmt"
	"os"
)

func setProcessPaused(_ *os.Process, _ bool) error {
	return fmt.Errorf("process suspension is unavailable on this platform")
}

//go:build !windows

package diskutil

import "golang.org/x/sys/unix"

// AvailableDiskBytes reports free bytes available to unprivileged users via statfs.
func AvailableDiskBytes(path string) (uint64, error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0, err
	}
	return stat.Bavail * uint64(stat.Bsize), nil
}

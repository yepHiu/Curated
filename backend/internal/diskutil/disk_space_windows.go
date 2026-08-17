//go:build windows

package diskutil

import (
	"fmt"

	"golang.org/x/sys/windows"
)

// AvailableDiskBytes reports free bytes available to the current user via GetDiskFreeSpaceEx.
func AvailableDiskBytes(path string) (uint64, error) {
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	var available uint64
	var total uint64
	var totalFree uint64
	if err := windows.GetDiskFreeSpaceEx(pathPtr, &available, &total, &totalFree); err != nil {
		return 0, fmt.Errorf("GetDiskFreeSpaceEx: %w", err)
	}
	return available, nil
}

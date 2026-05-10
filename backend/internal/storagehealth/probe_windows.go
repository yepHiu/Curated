//go:build windows

package storagehealth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
)

const (
	driveUnknown   = 0
	driveNoRootDir = 1
	driveRemovable = 2
	driveFixed     = 3
	driveRemote    = 4
	driveCDROM     = 5
	driveRAMDisk   = 6
)

type defaultProbe struct{}

func (defaultProbe) Probe(ctx context.Context, path string) (ProbeResult, error) {
	select {
	case <-ctx.Done():
		return ProbeResult{}, ctx.Err()
	default:
	}

	cleaned := filepath.Clean(strings.TrimSpace(path))
	root := windowsRootFromPath(cleaned)
	out := ProbeResult{
		RootPath:           root,
		IdentityConfidence: "unknown",
	}
	if root == "" {
		out.ErrorMessage = "storage root could not be resolved"
		return out, nil
	}

	rootPtr, err := windows.UTF16PtrFromString(root)
	if err != nil {
		return out, err
	}
	driveType := windows.GetDriveType(rootPtr)
	out.DriveType = driveTypeName(driveType)
	if driveType == driveNoRootDir {
		out.RootAvailable = false
		out.ErrorMessage = "storage root is not available"
		return out, nil
	}
	out.RootAvailable = true

	label, serial, fsName, volErr := windowsVolumeInformation(root)
	if volErr != nil {
		out.ErrorMessage = volErr.Error()
	} else {
		out.VolumeLabel = label
		out.VolumeID = serial
		out.FileSystem = fsName
		if serial != "" {
			out.IdentityConfidence = "high"
		}
	}

	info, statErr := os.Stat(cleaned)
	switch {
	case statErr == nil:
		out.PathExists = true
		out.PathIsDir = info.IsDir()
		out.PathReadable = pathReadable(cleaned)
		if !out.PathReadable {
			out.PermissionDenied = true
		}
	case errors.Is(statErr, os.ErrNotExist):
		out.PathExists = false
		out.PathReadable = false
	case errors.Is(statErr, os.ErrPermission):
		out.PathExists = true
		out.PathReadable = false
		out.PermissionDenied = true
		out.ErrorMessage = statErr.Error()
	default:
		out.ErrorMessage = statErr.Error()
	}

	return out, nil
}

func windowsRootFromPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	cleaned := filepath.Clean(trimmed)
	volume := filepath.VolumeName(cleaned)
	if len(volume) == 2 && volume[1] == ':' {
		return strings.ToUpper(volume) + `\`
	}
	if volume != "" {
		return strings.TrimRight(volume, `\/`) + `\`
	}
	return ""
}

func windowsVolumeInformation(root string) (label string, serial string, fsName string, err error) {
	rootPtr, err := windows.UTF16PtrFromString(root)
	if err != nil {
		return "", "", "", err
	}
	volumeName := make([]uint16, windows.MAX_PATH+1)
	fileSystemName := make([]uint16, windows.MAX_PATH+1)
	var serialNumber uint32
	var maxComponentLen uint32
	var fileSystemFlags uint32
	if err := windows.GetVolumeInformation(
		rootPtr,
		&volumeName[0],
		uint32(len(volumeName)),
		&serialNumber,
		&maxComponentLen,
		&fileSystemFlags,
		&fileSystemName[0],
		uint32(len(fileSystemName)),
	); err != nil {
		return "", "", "", err
	}
	return windows.UTF16ToString(volumeName), fmt.Sprintf("%08X", serialNumber), windows.UTF16ToString(fileSystemName), nil
}

func driveTypeName(t uint32) string {
	switch t {
	case driveNoRootDir:
		return "not_ready"
	case driveRemovable:
		return "removable"
	case driveFixed:
		return "fixed"
	case driveRemote:
		return "remote"
	case driveCDROM:
		return "cdrom"
	case driveRAMDisk:
		return "ramdisk"
	case driveUnknown:
		fallthrough
	default:
		return "unknown"
	}
}

func pathReadable(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	_ = f.Close()
	return true
}

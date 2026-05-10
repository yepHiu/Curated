//go:build !windows

package storagehealth

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type defaultProbe struct{}

func (defaultProbe) Probe(ctx context.Context, path string) (ProbeResult, error) {
	select {
	case <-ctx.Done():
		return ProbeResult{}, ctx.Err()
	default:
	}

	cleaned := filepath.Clean(strings.TrimSpace(path))
	root := "/"
	if volume := filepath.VolumeName(cleaned); volume != "" {
		root = volume
	}
	out := ProbeResult{
		RootPath:           root,
		RootAvailable:      true,
		DriveType:          "unknown",
		IdentityConfidence: "unknown",
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

func pathReadable(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	_ = f.Close()
	return true
}

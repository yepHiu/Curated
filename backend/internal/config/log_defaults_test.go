//go:build !release

package config

import (
	"path/filepath"
	"testing"
)

func TestDefaultLogDir_DevBuildUsesProjectRuntimeLogs(t *testing.T) {
	t.Parallel()

	if got, want := Default().LogDir, filepath.FromSlash("backend/runtime/logs"); got != want {
		t.Fatalf("Default().LogDir = %q, want %q", got, want)
	}
}

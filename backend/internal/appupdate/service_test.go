package appupdate

import (
	"testing"

	"curated-backend/internal/version"
)

func TestPackageVersionUsesDevFallbackVersion(t *testing.T) {
	t.Parallel()

	original := version.InstallerVersion
	version.InstallerVersion = ""
	t.Cleanup(func() {
		version.InstallerVersion = original
	})

	got := version.PackageVersion()
	if got != "0.0.0" {
		t.Fatalf("PackageVersion() = %q, want %q", got, "0.0.0")
	}
}

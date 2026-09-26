package version

import (
	"regexp"
	"strings"
	"testing"
)

func TestProductVersionIsIndependentOfBuildAndInstaller(t *testing.T) {
	previousStamp, previousInstaller := BuildStamp, InstallerVersion
	t.Cleanup(func() { BuildStamp, InstallerVersion = previousStamp, previousInstaller })
	product := ProductVersion()
	if !regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`).MatchString(product) {
		t.Fatalf("invalid numeric Server version: %q", product)
	}
	BuildStamp, InstallerVersion = "20260926.130000", "9.0.0"
	if ProductVersion() != product || PackageVersion() != "9.0.0" || Stamp() != "20260926.130000" {
		t.Fatal("product, installer and build identities must remain independent")
	}
	if !strings.Contains(Display(), product) || !strings.Contains(Display(), BuildStamp) {
		t.Fatalf("display lost product or build identity: %q", Display())
	}
}

//go:build !release

package version

import "testing"

func TestBackendNameDev(t *testing.T) {
	if got, want := BackendName(), "curated-dev"; got != want {
		t.Fatalf("BackendName() = %q, want %q", got, want)
	}
}

func TestDefaultLogFilePrefixDev(t *testing.T) {
	if got, want := DefaultLogFilePrefix(), "curated-dev"; got != want {
		t.Fatalf("DefaultLogFilePrefix() = %q, want %q", got, want)
	}
}

func TestTrayMutexNameDev(t *testing.T) {
	if got, want := TrayMutexName(), `Local\Curated.Dev.Tray.Singleton`; got != want {
		t.Fatalf("TrayMutexName() = %q, want %q", got, want)
	}
}

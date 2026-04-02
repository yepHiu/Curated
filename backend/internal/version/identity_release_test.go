//go:build release

package version

import "testing"

func TestBackendNameRelease(t *testing.T) {
	if got, want := BackendName(), "curated"; got != want {
		t.Fatalf("BackendName() = %q, want %q", got, want)
	}
}

func TestDefaultLogFilePrefixRelease(t *testing.T) {
	if got, want := DefaultLogFilePrefix(), "curated"; got != want {
		t.Fatalf("DefaultLogFilePrefix() = %q, want %q", got, want)
	}
}

func TestTrayMutexNameRelease(t *testing.T) {
	if got, want := TrayMutexName(), `Local\Curated.Tray.Singleton`; got != want {
		t.Fatalf("TrayMutexName() = %q, want %q", got, want)
	}
}

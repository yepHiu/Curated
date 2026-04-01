package playback

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestResolveFFmpegCommandPrefersBundledBinaryForDefaultCommand(t *testing.T) {
	tmpDir := t.TempDir()
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(prevWD)
	})

	bundled := filepath.Join(tmpDir, "backend", "third_party", "ffmpeg", "bin", defaultFFmpegBinaryName())
	if err := os.MkdirAll(filepath.Dir(bundled), 0o755); err != nil {
		t.Fatalf("mkdir bundled dir: %v", err)
	}
	if err := os.WriteFile(bundled, []byte("stub"), 0o755); err != nil {
		t.Fatalf("write bundled ffmpeg: %v", err)
	}

	got := resolveFFmpegCommand("ffmpeg")
	if got != bundled {
		t.Fatalf("resolveFFmpegCommand(default) = %q, want bundled %q", got, bundled)
	}
}

func TestResolveFFmpegCommandKeepsExplicitCustomCommand(t *testing.T) {
	tmpDir := t.TempDir()
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(prevWD)
	})

	bundled := filepath.Join(tmpDir, "backend", "third_party", "ffmpeg", "bin", defaultFFmpegBinaryName())
	if err := os.MkdirAll(filepath.Dir(bundled), 0o755); err != nil {
		t.Fatalf("mkdir bundled dir: %v", err)
	}
	if err := os.WriteFile(bundled, []byte("stub"), 0o755); err != nil {
		t.Fatalf("write bundled ffmpeg: %v", err)
	}

	custom := filepath.Join("custom-tools", "ffmpeg", func() string {
		if runtime.GOOS == "windows" {
			return "ffmpeg.exe"
		}
		return "ffmpeg"
	}())
	got := resolveFFmpegCommand(custom)
	if got != custom {
		t.Fatalf("resolveFFmpegCommand(custom) = %q, want %q", got, custom)
	}
}

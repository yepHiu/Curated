package playback

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWaitForPlaylistSegmentReferenceOptionalReportsFoundSegment(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	playlistPath := filepath.Join(dir, "index.m3u8")
	if err := os.WriteFile(playlistPath, []byte("#EXTM3U\n#EXTINF:2.000,\nsegment-00001.ts\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	found, err := waitForPlaylistSegmentReferenceOptional(
		context.Background(),
		playlistPath,
		"segment-00001.ts",
		make(chan error),
		10*time.Millisecond,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Fatal("expected optional segment reference to be found")
	}
}

func TestWaitForPlaylistSegmentReferenceOptionalContinuesOnTimeout(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	playlistPath := filepath.Join(dir, "index.m3u8")
	if err := os.WriteFile(playlistPath, []byte("#EXTM3U\n#EXTINF:2.000,\nsegment-00000.ts\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	found, err := waitForPlaylistSegmentReferenceOptional(
		context.Background(),
		playlistPath,
		"segment-00001.ts",
		make(chan error),
		1*time.Millisecond,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Fatal("expected optional segment wait to time out without finding segment")
	}
}

func TestWaitForPlaylistSegmentReferenceOptionalReturnsProcessExitError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	playlistPath := filepath.Join(dir, "index.m3u8")
	if err := os.WriteFile(playlistPath, []byte("#EXTM3U\n#EXTINF:2.000,\nsegment-00000.ts\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	waitCh := make(chan error, 1)
	wantErr := errors.New("ffmpeg exited")
	waitCh <- wantErr

	found, err := waitForPlaylistSegmentReferenceOptional(
		context.Background(),
		playlistPath,
		"segment-00001.ts",
		waitCh,
		time.Second,
	)

	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
	if found {
		t.Fatal("expected optional segment wait to report not found after process exit")
	}
}

func TestWaitForPlaylistSegmentReferenceOptionalContinuesOnCleanProcessExit(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	playlistPath := filepath.Join(dir, "index.m3u8")
	if err := os.WriteFile(playlistPath, []byte("#EXTM3U\n#EXTINF:2.000,\nsegment-00000.ts\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	waitCh := make(chan error, 1)
	waitCh <- nil

	found, err := waitForPlaylistSegmentReferenceOptional(
		context.Background(),
		playlistPath,
		"segment-00001.ts",
		waitCh,
		time.Second,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Fatal("expected optional segment wait to report not found after clean process exit")
	}
}

package playback

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestTranscodeStartupPublishesHardwareFirstSegment 验证硬编首片就绪即可返回，
// 软件预缓冲及不完整、未发布分片仍不能通过启动检查。
func TestTranscodeStartupPublishesHardwareFirstSegment(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, profile, fixture string
		ready                  bool
	}{
		{"amf-first-segment", "h264_amf", "published", true},
		{"software-still-prebuffers", "libx264", "published", false},
		{"amf-empty-segment", "h264_amf", "empty", false},
		{"amf-unpublished-segment", "h264_amf", "unpublished", false},
	} {
		// 各子进程独立写入临时目录，模拟只生成首片的转码器。
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			dir := t.TempDir()
			state, err := startTranscodeSession(ctx, executable, "movie", "session", dir, filepath.Join(dir, "index.m3u8"), transcodeProfile{
				Name: tc.profile, SessionKind: "transcode-hls",
				Args: []string{"-test.run=^TestHLSStartupProcess$", "--", "curated-startup-fixture", tc.fixture},
			})
			if !tc.ready {
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Fatalf("expected incomplete startup to wait until cancellation, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("hardware first segment did not become ready: %v", err)
			}
			// 回收已交付的模拟转码进程，防止测试残留子进程。
			t.Cleanup(func() { stopSessionState(state) })
			if playlistReferencesSegment(state.session.PlaylistPath, hlsFourthSegmentName) {
				t.Fatal("fixture unexpectedly generated the fourth segment")
			}
		})
	}
}

// TestHLSStartupProcess 是重入测试进程，仅在显式参数下模拟 FFmpeg 写首片后继续运行。
func TestHLSStartupProcess(t *testing.T) {
	marker := -1
	for i, arg := range os.Args {
		if arg == "curated-startup-fixture" {
			marker = i
			break
		}
	}
	if marker < 0 {
		return
	}
	fixture := os.Args[marker+1]
	segment := "media"
	if fixture == "empty" {
		segment = ""
	}
	playlist := "#EXTM3U\n#EXTINF:2.000,\n" + hlsFirstSegmentName + "\n"
	if fixture == "unpublished" {
		playlist = "#EXTM3U\n"
	}
	for name, data := range map[string]string{hlsInitFilename: "init", hlsFirstSegmentName: segment, "index.m3u8": playlist} {
		if err := os.WriteFile(name, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// 输出协议可为空；保持进程活着，验证启动不会依赖编码结束或后续分片。
	time.Sleep(20 * time.Second)
}

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

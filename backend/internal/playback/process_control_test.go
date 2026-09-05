//go:build windows || linux || darwin

package playback

import (
	"context"
	"curated-backend/internal/executil"
	"io"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// Uses a tiny generated video, never the user's library. Skip when FFmpeg is
// absent; CI and packaged-runtime checks should provide it explicitly.
func TestFFmpegProcessActuallyPausesAndResumes(t *testing.T) {
	name, err := exec.LookPath(resolveFFmpegCommand("ffmpeg"))
	if err != nil {
		t.Skip("FFmpeg not installed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	cmd := executil.CommandContext(ctx, name, "-hide_banner", "-loglevel", "error", "-stats_period", "0.1", "-progress", "pipe:1", "-re", "-f", "lavfi", "-i", "testsrc2=size=64x64:rate=10", "-t", "10", "-f", "null", "-")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = setProcessPaused(cmd.Process, false); cancel(); _ = cmd.Wait() }()
	var micros atomic.Int64
	go consumeFFmpegProgress(stdout, func(p ffmpegProgress) {
		if p.HasOutTime {
			micros.Store(int64(p.OutTimeSec * 1e6))
		}
	})
	waitUntil := func(check func() bool) {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for !check() {
			if time.Now().After(deadline) {
				t.Fatal("FFmpeg did not make expected progress")
			}
			time.Sleep(25 * time.Millisecond)
		}
	}
	waitUntil(func() bool { return micros.Load() >= 500000 })
	if err := setProcessPaused(cmd.Process, true); err != nil {
		t.Fatal(err)
	}
	// Drain progress records emitted immediately before suspension.
	time.Sleep(200 * time.Millisecond)
	before := micros.Load()
	time.Sleep(600 * time.Millisecond)
	if after := micros.Load(); after != before {
		t.Fatalf("paused process advanced: %d -> %d", before, after)
	}
	if err := setProcessPaused(cmd.Process, false); err != nil {
		t.Fatal(err)
	}
	waitUntil(func() bool { return micros.Load() >= before+300000 })
	t.Logf("FFmpeg output held at %.2fs while paused, resumed to %.2fs", float64(before)/1e6, float64(micros.Load())/1e6)
}

func TestThrottleUsesPublishedRemuxDurations(t *testing.T) {
	value, ok := mediaTimeFromPlaylist("#EXTM3U\n#EXTINF:9.6,\nsegment-00000.m4s\n#EXTINF:8.4,\nsegment-00001.m4s\n", "segment-00001.m4s")
	if !ok || value != 18 {
		t.Fatalf("got %v %v", value, ok)
	}
}

func TestFFmpegSameMovieSessionsRemainIndependentlyPlayable(t *testing.T) {
	name, err := exec.LookPath(resolveFFmpegCommand("ffmpeg"))
	if err != nil {
		t.Skip("FFmpeg not installed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	source := filepath.Join(t.TempDir(), "synthetic.mp4")
	fixture := executil.CommandContext(ctx, name, "-hide_banner", "-loglevel", "error", "-f", "lavfi", "-i", "testsrc2=size=64x64:rate=10", "-t", "12", "-c:v", "libx264", "-preset", "ultrafast", source)
	if output, err := fixture.CombinedOutput(); err != nil {
		t.Fatalf("fixture: %v %s", err, output)
	}
	m := New(Config{Enabled: true, FFmpegCommand: name, SessionRoot: t.TempDir()})
	defer m.Close()
	type result struct {
		session Session
		err     error
	}
	results := make(chan result, 2)
	for range 2 {
		go func() {
			session, err := m.StartHLSSession(ctx, "same-movie", source, StartHLSSessionOptions{SourceVideoCodec: "h264", SourceContainer: "mp4"})
			results <- result{session, err}
		}()
	}
	first, second := <-results, <-results
	if first.err != nil || second.err != nil {
		t.Fatalf("startup: %v / %v", first.err, second.err)
	}
	if first.session.ID == second.session.ID {
		t.Fatal("sessions share an identity")
	}
	for _, session := range []Session{first.session, second.session} {
		if _, err := m.ResolveFile(session.ID, hlsFirstSegmentName); err != nil {
			t.Fatalf("session %s lost media: %v", session.ID, err)
		}
	}
	if err := m.DeleteSession(first.session.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := m.ResolveFile(second.session.ID, hlsFirstSegmentName); err != nil {
		t.Fatal("deleting one client interrupted the other", err)
	}
	t.Log("two real HLS sessions for the same synthetic movie remained independently readable")
}

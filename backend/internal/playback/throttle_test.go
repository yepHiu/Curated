package playback

import "testing"

func TestNextThrottleActionPausesWhenFarAhead(t *testing.T) {
	t.Parallel()
	if got := nextThrottleAction(false, 100, 0, 90, 12); got != throttlePause {
		t.Fatalf("action = %d, want pause", got)
	}
	if got := nextThrottleAction(true, 100, 0, 90, 12); got != throttleNone {
		t.Fatalf("already paused action = %d, want none", got)
	}
}

func TestNextThrottleActionResumesWhenPlaybackCatchesUp(t *testing.T) {
	t.Parallel()
	if got := nextThrottleAction(true, 20, 10, 90, 12); got != throttleResume {
		t.Fatalf("action = %d, want resume", got)
	}
	if got := nextThrottleAction(false, 20, 10, 90, 12); got != throttleNone {
		t.Fatalf("running action = %d, want none", got)
	}
}

func TestMediaTimeFromHLSFileName(t *testing.T) {
	t.Parallel()
	sec, ok := mediaTimeFromHLSFileName("segment-00005.m4s", 2)
	if !ok || sec != 10 {
		t.Fatalf("got %v %v, want 10 true", sec, ok)
	}
	if _, ok := mediaTimeFromHLSFileName("index.m3u8", 2); ok {
		t.Fatal("playlist should not map to media time")
	}
	if _, ok := mediaTimeFromHLSFileName("segment-00000.m4s.tmp", 2); ok {
		t.Fatal("temp files should not map to media time")
	}
}

package playback

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestStartupCapacityWaitIsCancellable(t *testing.T) {
	slots := make(chan struct{}, 1)
	slots <- struct{}{}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := acquireSessionSlot(ctx, slots); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v", err)
	}
	if len(slots) != 1 {
		t.Fatal("cancelled waiter changed occupied capacity")
	}
}

func TestFailedStartPreservesExistingSessionAndReleasesCapacity(t *testing.T) {
	m := New(Config{Enabled: true, FFmpegCommand: "curated-missing-ffmpeg-test", SessionRoot: t.TempDir(), MaxSessions: 1})
	t.Cleanup(m.Close)
	m.sessions["old"] = &sessionState{session: Session{ID: "old", MovieID: "movie"}}
	_, err := m.StartHLSSession(context.Background(), "movie", "missing.mp4", StartHLSSessionOptions{})
	if err == nil {
		t.Fatal("expected missing encoder failure")
	}
	if _, err := m.GetSessionSnapshot("old"); err != nil {
		t.Fatal("old session was removed")
	}
	if len(m.sessionSlots) != 0 || len(m.startSlots) != 0 {
		t.Fatal("failed startup leaked capacity")
	}
}

func TestCloseCancelsQueuedStartup(t *testing.T) {
	m := New(Config{MaxSessions: 1})
	m.sessionSlots <- struct{}{}
	done := make(chan error, 1)
	go func() {
		_, err := m.StartHLSSession(context.Background(), "m", "x", StartHLSSessionOptions{})
		done <- err
	}()
	m.Close()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("startup outlived Close")
	}
}

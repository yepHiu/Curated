package playback

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCleanupExpiredSessionsRemovesOnlyIdleExpiredSessions(t *testing.T) {
	t.Parallel()

	manager := New(Config{SessionIdleTimeout: 2 * time.Minute})
	t.Cleanup(manager.Close)
	now := time.Now().UTC()
	expiredDir := t.TempDir()
	freshDir := t.TempDir()

	manager.sessions["sess-expired"] = &sessionState{
		session: Session{
			ID:        "sess-expired",
			MovieID:   "movie-a",
			Directory: expiredDir,
			StartedAt: now.Add(-10 * time.Minute),
		},
		lastAccessedAt: now.Add(-5 * time.Minute),
	}
	manager.sessions["sess-fresh"] = &sessionState{
		session: Session{
			ID:        "sess-fresh",
			MovieID:   "movie-b",
			Directory: freshDir,
			StartedAt: now.Add(-30 * time.Second),
		},
		lastAccessedAt: now.Add(-20 * time.Second),
	}

	reaped := manager.cleanupExpiredSessions(now)
	if len(reaped) != 1 {
		t.Fatalf("reaped session count = %d, want 1", len(reaped))
	}
	if reaped[0].session.ID != "sess-expired" {
		t.Fatalf("expired session id = %q, want sess-expired", reaped[0].session.ID)
	}
	if _, ok := manager.sessions["sess-expired"]; ok {
		t.Fatal("expected expired session to be removed from registry")
	}
	if _, ok := manager.sessions["sess-fresh"]; !ok {
		t.Fatal("expected fresh session to remain registered")
	}
}

func TestResolveFileTouchesSessionLastAccessTime(t *testing.T) {
	t.Parallel()

	manager := New(Config{})
	t.Cleanup(manager.Close)
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "index.m3u8")
	if err := os.WriteFile(targetPath, []byte("#EXTM3U\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	before := time.Now().UTC().Add(-10 * time.Minute)
	manager.sessions["sess-1"] = &sessionState{
		session: Session{
			ID:        "sess-1",
			MovieID:   "movie-a",
			Directory: dir,
			StartedAt: before,
		},
		lastAccessedAt: before,
	}

	if _, err := manager.ResolveFile("sess-1", "index.m3u8"); err != nil {
		t.Fatalf("ResolveFile() error = %v", err)
	}

	snapshot, err := manager.GetSessionSnapshot("sess-1")
	if err != nil {
		t.Fatalf("GetSessionSnapshot() error = %v", err)
	}
	if !snapshot.LastAccessedAt.After(before) {
		t.Fatalf("LastAccessedAt = %v, want later than %v", snapshot.LastAccessedAt, before)
	}
}

func TestListSessionSnapshotsReturnsNewestFirst(t *testing.T) {
	t.Parallel()

	manager := New(Config{})
	t.Cleanup(manager.Close)
	now := time.Now().UTC()
	manager.sessions["sess-old"] = &sessionState{
		session: Session{
			ID:        "sess-old",
			MovieID:   "movie-a",
			StartedAt: now.Add(-2 * time.Minute),
		},
		lastAccessedAt: now.Add(-90 * time.Second),
	}
	manager.sessions["sess-new"] = &sessionState{
		session: Session{
			ID:        "sess-new",
			MovieID:   "movie-b",
			StartedAt: now.Add(-30 * time.Second),
		},
		lastAccessedAt: now.Add(-10 * time.Second),
	}

	snapshots := manager.ListSessionSnapshots(10)
	if len(snapshots) != 2 {
		t.Fatalf("snapshot count = %d, want 2", len(snapshots))
	}
	if snapshots[0].Session.ID != "sess-new" {
		t.Fatalf("first session = %q, want sess-new", snapshots[0].Session.ID)
	}
	if snapshots[1].Session.ID != "sess-old" {
		t.Fatalf("second session = %q, want sess-old", snapshots[1].Session.ID)
	}
}

func TestDeleteSessionKeepsRecentSnapshotHistory(t *testing.T) {
	t.Parallel()

	manager := New(Config{})
	t.Cleanup(manager.Close)
	now := time.Now().UTC()
	dir := t.TempDir()
	manager.sessions["sess-1"] = &sessionState{
		session: Session{
			ID:        "sess-1",
			MovieID:   "movie-a",
			Directory: dir,
			StartedAt: now.Add(-30 * time.Second),
		},
		lastAccessedAt: now.Add(-5 * time.Second),
		waitCh:         make(chan error, 1),
	}
	manager.sessions["sess-1"].waitCh <- nil

	if err := manager.DeleteSession("sess-1"); err != nil {
		t.Fatalf("DeleteSession() error = %v", err)
	}

	snapshot, err := manager.GetSessionSnapshot("sess-1")
	if err != nil {
		t.Fatalf("GetSessionSnapshot() error = %v", err)
	}
	if snapshot.State != "stopped" {
		t.Fatalf("snapshot state = %q, want stopped", snapshot.State)
	}
	if snapshot.FinishedAt.IsZero() {
		t.Fatal("expected deleted session to keep finished timestamp")
	}

	snapshots := manager.ListSessionSnapshots(5)
	if len(snapshots) != 1 {
		t.Fatalf("snapshot count = %d, want 1", len(snapshots))
	}
	if snapshots[0].Session.ID != "sess-1" {
		t.Fatalf("recent session id = %q, want sess-1", snapshots[0].Session.ID)
	}
}

func TestCleanupExpiredSessionsArchivesExpiredSnapshot(t *testing.T) {
	t.Parallel()

	manager := New(Config{SessionIdleTimeout: 45 * time.Second})
	t.Cleanup(manager.Close)
	now := time.Now().UTC()
	manager.sessions["sess-expired"] = &sessionState{
		session: Session{
			ID:        "sess-expired",
			MovieID:   "movie-a",
			StartedAt: now.Add(-4 * time.Minute),
		},
		lastAccessedAt: now.Add(-90 * time.Second),
		waitCh:         make(chan error, 1),
	}
	manager.sessions["sess-expired"].waitCh <- nil

	reaped := manager.cleanupExpiredSessions(now)
	if len(reaped) != 1 {
		t.Fatalf("reaped session count = %d, want 1", len(reaped))
	}

	snapshot, err := manager.GetSessionSnapshot("sess-expired")
	if err != nil {
		t.Fatalf("GetSessionSnapshot() error = %v", err)
	}
	if snapshot.State != "expired" {
		t.Fatalf("snapshot state = %q, want expired", snapshot.State)
	}
	if snapshot.ExpiresAt.IsZero() {
		t.Fatal("expected expired snapshot to retain expiration time")
	}
}

func TestSessionStateSnapshotTracksStateTransitions(t *testing.T) {
	t.Parallel()

	startedAt := time.Date(2026, 4, 25, 8, 0, 0, 0, time.UTC)
	accessedAt := startedAt.Add(45 * time.Second)
	finishedAt := startedAt.Add(2 * time.Minute)
	state := &sessionState{
		session: Session{
			ID:        "sess-state",
			MovieID:   "movie-state",
			StartedAt: startedAt,
		},
	}

	state.touchAt(accessedAt)
	running := state.snapshot(2 * time.Minute)
	if running.LastAccessedAt != accessedAt {
		t.Fatalf("running LastAccessedAt = %v, want %v", running.LastAccessedAt, accessedAt)
	}
	if running.ExpiresAt != accessedAt.Add(2*time.Minute) {
		t.Fatalf("running ExpiresAt = %v, want %v", running.ExpiresAt, accessedAt.Add(2*time.Minute))
	}
	if running.State != "running" {
		t.Fatalf("running State = %q, want running", running.State)
	}

	state.markFinishedAt(finishedAt, errors.New("transcoder exited"))
	failed := state.snapshot(2 * time.Minute)
	if failed.FinishedAt != finishedAt {
		t.Fatalf("failed FinishedAt = %v, want %v", failed.FinishedAt, finishedAt)
	}
	if failed.State != "failed" {
		t.Fatalf("failed State = %q, want failed", failed.State)
	}
	if failed.LastError != "transcoder exited" {
		t.Fatalf("failed LastError = %q, want transcoder exited", failed.LastError)
	}
}

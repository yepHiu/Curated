package playback

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	ErrSessionNotFound    = errors.New("playback session not found")
	ErrStreamPushDisabled = errors.New("stream push is disabled")
)

type Config struct {
	Enabled       bool
	FFmpegCommand string
	SessionRoot   string
}

type Session struct {
	ID           string
	MovieID      string
	PlaylistPath string
	Directory    string
	StartedAt    time.Time
}

type sessionState struct {
	session Session
	cancel  context.CancelFunc
	cmd     *exec.Cmd
}

type Manager struct {
	cfg      Config
	mu       sync.RWMutex
	sessions map[string]*sessionState
}

func New(cfg Config) *Manager {
	return &Manager{
		cfg:      cfg,
		sessions: make(map[string]*sessionState),
	}
}

func (m *Manager) Enabled() bool {
	if m == nil {
		return false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg.Enabled
}

func (m *Manager) SetConfig(cfg Config) {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.cfg = cfg
	m.mu.Unlock()
}

func (m *Manager) StartHLSSession(ctx context.Context, movieID string, sourcePath string) (Session, error) {
	if m == nil {
		return Session{}, ErrStreamPushDisabled
	}
	m.mu.RLock()
	cfg := m.cfg
	m.mu.RUnlock()
	if !cfg.Enabled {
		return Session{}, ErrStreamPushDisabled
	}
	root := strings.TrimSpace(cfg.SessionRoot)
	if root == "" {
		return Session{}, fmt.Errorf("stream session root is empty")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return Session{}, err
	}

	sessionID, err := newSessionID()
	if err != nil {
		return Session{}, err
	}
	dir := filepath.Join(root, sessionID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Session{}, err
	}

	playlistPath := filepath.Join(dir, "index.m3u8")
	segmentPattern := filepath.Join(dir, "segment-%05d.ts")

	cmdName := strings.TrimSpace(cfg.FFmpegCommand)
	if cmdName == "" {
		cmdName = "ffmpeg"
	}

	args := []string{
		"-y",
		"-i", sourcePath,
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-c:a", "aac",
		"-f", "hls",
		"-hls_time", "4",
		"-hls_list_size", "0",
		"-hls_flags", "independent_segments",
		"-hls_segment_filename", segmentPattern,
		playlistPath,
	}

	runCtx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(runCtx, cmdName, args...)
	if err := cmd.Start(); err != nil {
		cancel()
		return Session{}, err
	}

	state := &sessionState{
		session: Session{
			ID:           sessionID,
			MovieID:      movieID,
			PlaylistPath: playlistPath,
			Directory:    dir,
			StartedAt:    time.Now().UTC(),
		},
		cancel: cancel,
		cmd:    cmd,
	}

	m.mu.Lock()
	m.sessions[sessionID] = state
	m.mu.Unlock()

	go func() {
		_ = cmd.Wait()
	}()

	if err := waitForFile(ctx, playlistPath, 12*time.Second); err != nil {
		_ = m.DeleteSession(sessionID)
		return Session{}, err
	}

	return state.session, nil
}

func (m *Manager) ResolveFile(sessionID string, name string) (string, error) {
	if m == nil {
		return "", ErrSessionNotFound
	}
	m.mu.RLock()
	state, ok := m.sessions[sessionID]
	m.mu.RUnlock()
	if !ok {
		return "", ErrSessionNotFound
	}
	cleanName := filepath.Clean(strings.TrimSpace(name))
	if cleanName == "." || cleanName == "" || strings.Contains(cleanName, "..") {
		return "", ErrSessionNotFound
	}
	abs := filepath.Join(state.session.Directory, cleanName)
	rel, err := filepath.Rel(state.session.Directory, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", ErrSessionNotFound
	}
	if _, err := os.Stat(abs); err != nil {
		if os.IsNotExist(err) {
			return "", ErrSessionNotFound
		}
		return "", err
	}
	return abs, nil
}

func (m *Manager) DeleteSession(sessionID string) error {
	if m == nil {
		return ErrSessionNotFound
	}
	m.mu.Lock()
	state, ok := m.sessions[sessionID]
	if ok {
		delete(m.sessions, sessionID)
	}
	m.mu.Unlock()
	if !ok {
		return ErrSessionNotFound
	}
	state.cancel()
	return os.RemoveAll(state.session.Directory)
}

func newSessionID() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func waitForFile(ctx context.Context, path string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		if _, err := os.Stat(path); err == nil {
			return nil
		}
		if ctx != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for stream output")
		}
		time.Sleep(200 * time.Millisecond)
	}
}

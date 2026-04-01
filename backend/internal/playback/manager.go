package playback

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

var (
	ErrSessionNotFound    = errors.New("playback session not found")
	ErrStreamPushDisabled = errors.New("stream push is disabled")
)

const (
	hlsInitialSegmentSeconds = "2"
	hlsTargetSegmentSeconds  = "4"
)

type Config struct {
	Enabled         bool
	HardwareDecode  bool
	HardwareEncoder string
	FFmpegCommand   string
	SessionRoot     string
}

type Session struct {
	ID               string
	MovieID          string
	PlaylistPath     string
	Directory        string
	StartPositionSec float64
	ProfileName      string
	StartedAt        time.Time
}

type sessionState struct {
	session Session
	cancel  context.CancelFunc
	cmd     *exec.Cmd
	waitCh  chan error
}

type transcodeProfile struct {
	Name string
	Args []string
}

type Manager struct {
	cfg                   Config
	lastSuccessfulProfile string
	sessionStartMu        sync.Mutex
	mu                    sync.RWMutex
	sessions              map[string]*sessionState
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

func (m *Manager) StartHLSSession(ctx context.Context, movieID string, sourcePath string, startPositionSec float64) (Session, error) {
	if m == nil {
		return Session{}, ErrStreamPushDisabled
	}
	m.sessionStartMu.Lock()
	defer m.sessionStartMu.Unlock()

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
	root = filepath.Clean(root)
	if !filepath.IsAbs(root) {
		absRoot, err := filepath.Abs(root)
		if err != nil {
			return Session{}, err
		}
		root = absRoot
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return Session{}, err
	}

	for _, stale := range m.takeSessionsForMovie(movieID) {
		stopSessionState(stale)
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
	segmentPattern := "segment-%05d.ts"

	cmdName := resolveFFmpegCommand(cfg.FFmpegCommand)
	preferredProfile := ""
	m.mu.RLock()
	preferredProfile = m.lastSuccessfulProfile
	m.mu.RUnlock()
	if startPositionSec < 0 {
		startPositionSec = 0
	}
	profiles := buildTranscodeProfiles(cfg, sourcePath, segmentPattern, "index.m3u8", preferredProfile, startPositionSec)

	var lastErr error
	for index, profile := range profiles {
		if index > 0 {
			_ = os.RemoveAll(dir)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return Session{}, err
			}
		}

		state, err := startTranscodeSession(ctx, cmdName, movieID, sessionID, dir, playlistPath, profile, startPositionSec)
		if err == nil {
			staleStates := m.replaceSession(sessionID, state, profile.Name)
			for _, stale := range staleStates {
				stopSessionState(stale)
			}
			return state.session, nil
		}
		lastErr = err
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("unable to start playback session")
	}
	_ = os.RemoveAll(dir)
	return Session{}, lastErr
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
	stopSessionState(state)
	return nil
}

func (m *Manager) Close() {
	if m == nil {
		return
	}
	m.sessionStartMu.Lock()
	staleStates := m.takeAllSessionsLocked()
	m.sessionStartMu.Unlock()
	for _, stale := range staleStates {
		stopSessionState(stale)
	}
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

func waitForFileOrProcessExit(ctx context.Context, path string, waitCh <-chan error, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		if _, err := os.Stat(path); err == nil {
			return nil
		}
		select {
		case err := <-waitCh:
			if err == nil {
				return fmt.Errorf("transcoder exited before playlist was ready")
			}
			return err
		default:
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

func waitForNonEmptyFileOrProcessExit(ctx context.Context, path string, waitCh <-chan error, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		if info, err := os.Stat(path); err == nil && info.Size() > 0 {
			return nil
		}
		select {
		case err := <-waitCh:
			if err == nil {
				return fmt.Errorf("transcoder exited before stream file was populated")
			}
			return err
		default:
		}
		if ctx != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for populated stream output")
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func waitForPlaylistSegmentReference(ctx context.Context, playlistPath string, segmentName string, waitCh <-chan error, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		if raw, err := os.ReadFile(playlistPath); err == nil {
			playlist := string(raw)
			if strings.Contains(playlist, "#EXTINF:") && strings.Contains(playlist, segmentName) {
				return nil
			}
		}
		select {
		case err := <-waitCh:
			if err == nil {
				return fmt.Errorf("transcoder exited before playlist referenced a segment")
			}
			return err
		default:
		}
		if ctx != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for playlist segment reference")
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func startTranscodeSession(
	ctx context.Context,
	cmdName string,
	movieID string,
	sessionID string,
	dir string,
	playlistPath string,
	profile transcodeProfile,
	startPositionSec float64,
) (*sessionState, error) {
	runCtx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(runCtx, cmdName, profile.Args...)
	cmd.Dir = dir
	cmd.Stdout = io.Discard
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("%s start failed: %w", profile.Name, err)
	}

	state := &sessionState{
		session: Session{
			ID:               sessionID,
			MovieID:          movieID,
			PlaylistPath:     playlistPath,
			Directory:        dir,
			StartPositionSec: startPositionSec,
			ProfileName:      profile.Name,
			StartedAt:        time.Now().UTC(),
		},
		cancel: cancel,
		cmd:    cmd,
		waitCh: make(chan error, 1),
	}

	go func() {
		state.waitCh <- cmd.Wait()
	}()

	if err := waitForFileOrProcessExit(ctx, playlistPath, state.waitCh, 12*time.Second); err != nil {
		cancel()
		stderrText := strings.TrimSpace(stderr.String())
		return nil, fmt.Errorf("%s session failed: %w: %s", profile.Name, err, stderrText)
	}
	firstSegmentPath := filepath.Join(dir, "segment-00000.ts")
	if err := waitForNonEmptyFileOrProcessExit(ctx, firstSegmentPath, state.waitCh, 8*time.Second); err != nil {
		cancel()
		stderrText := strings.TrimSpace(stderr.String())
		return nil, fmt.Errorf("%s first segment failed: %w: %s", profile.Name, err, stderrText)
	}
	if err := waitForPlaylistSegmentReference(ctx, playlistPath, filepath.Base(firstSegmentPath), state.waitCh, 8*time.Second); err != nil {
		cancel()
		stderrText := strings.TrimSpace(stderr.String())
		return nil, fmt.Errorf("%s playlist readiness failed: %w: %s", profile.Name, err, stderrText)
	}

	return state, nil
}

func buildTranscodeProfiles(cfg Config, sourcePath string, segmentPattern string, playlistPath string, preferredProfile string, startPositionSec float64) []transcodeProfile {
	inputPrefix := []string{"-y"}
	if cfg.HardwareDecode {
		inputPrefix = append(inputPrefix, "-hwaccel", "auto")
	}
	inputSeekArgs, accurateSeekArgs := buildSeekArgs(startPositionSec)
	configuredPreference := normalizeHardwareEncoderProfileName(cfg.HardwareEncoder)

	hlsArgs := []string{
		"-pix_fmt", "yuv420p",
		"-c:a", "aac",
		"-ac", "2",
		"-force_key_frames", "expr:gte(t,n_forced*4)",
		"-f", "hls",
		"-hls_init_time", hlsInitialSegmentSeconds,
		"-hls_time", hlsTargetSegmentSeconds,
		"-hls_list_size", "0",
		"-hls_allow_cache", "0",
		"-hls_flags", "independent_segments",
		"-hls_playlist_type", "event",
		"-start_number", "0",
		"-hls_segment_filename", segmentPattern,
		playlistPath,
	}

	profiles := make([]transcodeProfile, 0, 4)
	if cfg.HardwareDecode {
		switch runtime.GOOS {
		case "windows":
			profiles = append(profiles,
				transcodeProfile{
					Name: "h264_nvenc",
					Args: append(
						append(
							append(append(append([]string{}, inputPrefix...), inputSeekArgs...), "-i", sourcePath),
							append(accurateSeekArgs, "-c:v", "h264_nvenc", "-preset", "p5", "-cq", "19")...,
						),
						hlsArgs...,
					),
				},
				transcodeProfile{
					Name: "h264_qsv",
					Args: append(
						append(
							append(append(append([]string{}, inputPrefix...), inputSeekArgs...), "-i", sourcePath),
							append(accurateSeekArgs, "-c:v", "h264_qsv", "-preset", "medium", "-global_quality", "20")...,
						),
						hlsArgs...,
					),
				},
				transcodeProfile{
					Name: "h264_amf",
					Args: append(
						append(
							append(append(append([]string{}, inputPrefix...), inputSeekArgs...), "-i", sourcePath),
							append(accurateSeekArgs, "-c:v", "h264_amf", "-quality", "quality")...,
						),
						hlsArgs...,
					),
				},
			)
		}
	}

	if configuredPreference != "" {
		preferredProfile = configuredPreference
	}
	if preferredProfile != "" {
		for idx, profile := range profiles {
			if profile.Name != preferredProfile {
				continue
			}
			if idx > 0 {
				profiles[0], profiles[idx] = profiles[idx], profiles[0]
			}
			break
		}
	}

	profiles = append(profiles, transcodeProfile{
		Name: "libx264",
		Args: append(
			append(
				append(append(append([]string{}, inputPrefix...), inputSeekArgs...), "-i", sourcePath),
				append(accurateSeekArgs, "-c:v", "libx264", "-preset", "veryfast", "-crf", "17")...,
			),
			hlsArgs...,
		),
	})
	if configuredPreference == "libx264" {
		return []transcodeProfile{profiles[len(profiles)-1]}
	}
	return profiles
}

func formatSeekOffset(startPositionSec float64) string {
	if startPositionSec <= 0 {
		return "0"
	}
	return fmt.Sprintf("%.3f", startPositionSec)
}

func buildSeekArgs(startPositionSec float64) ([]string, []string) {
	if startPositionSec <= 0 {
		return nil, nil
	}

	const preciseSeekWindowSec = 2.0
	inputSeekSec := startPositionSec - preciseSeekWindowSec
	if inputSeekSec < 0 {
		inputSeekSec = 0
	}
	accurateSeekSec := startPositionSec - inputSeekSec
	if accurateSeekSec < 0 {
		accurateSeekSec = 0
	}

	inputSeekArgs := []string{"-ss", formatSeekOffset(inputSeekSec)}
	if accurateSeekSec <= 0.001 {
		return inputSeekArgs, nil
	}
	return inputSeekArgs, []string{"-ss", formatSeekOffset(accurateSeekSec)}
}

func (m *Manager) replaceSession(sessionID string, state *sessionState, profileName string) []*sessionState {
	m.mu.Lock()
	defer m.mu.Unlock()

	staleStates := make([]*sessionState, 0, len(m.sessions))
	for existingID, existingState := range m.sessions {
		if existingID == sessionID {
			continue
		}
		if existingState.session.MovieID != state.session.MovieID {
			continue
		}
		delete(m.sessions, existingID)
		staleStates = append(staleStates, existingState)
	}
	m.sessions[sessionID] = state
	m.lastSuccessfulProfile = profileName
	return staleStates
}

func (m *Manager) takeSessionsForMovie(movieID string) []*sessionState {
	m.mu.Lock()
	defer m.mu.Unlock()

	staleStates := make([]*sessionState, 0, len(m.sessions))
	for existingID, existingState := range m.sessions {
		if existingState.session.MovieID != movieID {
			continue
		}
		delete(m.sessions, existingID)
		staleStates = append(staleStates, existingState)
	}
	return staleStates
}

func (m *Manager) takeAllSessionsLocked() []*sessionState {
	m.mu.Lock()
	defer m.mu.Unlock()

	staleStates := make([]*sessionState, 0, len(m.sessions))
	for existingID, existingState := range m.sessions {
		delete(m.sessions, existingID)
		staleStates = append(staleStates, existingState)
	}
	return staleStates
}

func stopSessionState(state *sessionState) {
	if state == nil {
		return
	}
	if state.cancel != nil {
		state.cancel()
	}
	if state.waitCh == nil {
		if state.cmd != nil && state.cmd.Process != nil {
			_ = state.cmd.Process.Kill()
		}
		_ = os.RemoveAll(state.session.Directory)
		return
	}
	select {
	case <-state.waitCh:
	case <-time.After(1500 * time.Millisecond):
		if state.cmd != nil && state.cmd.Process != nil {
			_ = state.cmd.Process.Kill()
		}
		select {
		case <-state.waitCh:
		case <-time.After(1500 * time.Millisecond):
		}
	}
	_ = os.RemoveAll(state.session.Directory)
}

func normalizeHardwareEncoderProfileName(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "auto":
		return ""
	case "amf", "h264_amf":
		return "h264_amf"
	case "qsv", "h264_qsv":
		return "h264_qsv"
	case "nvenc", "h264_nvenc":
		return "h264_nvenc"
	case "software", "libx264":
		return "libx264"
	default:
		return ""
	}
}

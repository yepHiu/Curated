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
	"slices"
	"strings"
	"sync"
	"time"

	"curated-backend/internal/executil"
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
	Enabled                bool
	HardwareDecode         bool
	HardwareEncoder        string
	FFmpegCommand          string
	SessionRoot            string
	SessionIdleTimeout     time.Duration
	SessionJanitorInterval time.Duration
}

type Session struct {
	ID               string
	MovieID          string
	PlaylistPath     string
	Directory        string
	StartPositionSec float64
	ProfileName      string
	Kind             string
	StartedAt        time.Time
}

type sessionState struct {
	session        Session
	cancel         context.CancelFunc
	cmd            *exec.Cmd
	waitCh         chan error
	lastAccessedAt time.Time
	finishedAt     time.Time
	lastError      string
}

type transcodeProfile struct {
	Name              string
	SessionKind       string
	Args              []string
	TimelineOriginSec float64
}

type StartHLSSessionOptions struct {
	StartPositionSec float64
	PreferRemux      bool
	SourceVideoCodec string
	SourceAudioCodec string
}

type buildProfileOptions struct {
	PreferredProfile string
	StartPositionSec float64
	PreferRemux      bool
	SourceVideoCodec string
	SourceAudioCodec string
}

type Manager struct {
	cfg                   Config
	lastSuccessfulProfile string
	sessionStartMu        sync.Mutex
	mu                    sync.RWMutex
	sessions              map[string]*sessionState
	// recentSnapshots keeps a bounded in-memory history after sessions leave the
	// active registry, so status/recent APIs can still explain what just happened.
	recentSnapshots []SessionSnapshot
	janitorCancel   context.CancelFunc
	janitorDone     chan struct{}
}

type SessionSnapshot struct {
	Session        Session
	LastAccessedAt time.Time
	ExpiresAt      time.Time
	FinishedAt     time.Time
	// State is a coarse lifecycle label exposed to diagnostics endpoints.
	State     string
	LastError string
}

const recentSessionHistoryLimit = 32

func New(cfg Config) *Manager {
	manager := &Manager{
		cfg:             cfg,
		sessions:        make(map[string]*sessionState),
		recentSnapshots: make([]SessionSnapshot, 0, recentSessionHistoryLimit),
		janitorDone:     make(chan struct{}),
	}
	manager.startJanitorLoop()
	return manager
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

func (m *Manager) StartHLSSession(ctx context.Context, movieID string, sourcePath string, options StartHLSSessionOptions) (Session, error) {
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
	if options.StartPositionSec < 0 {
		options.StartPositionSec = 0
	}
	profiles := buildTranscodeProfiles(cfg, sourcePath, segmentPattern, "index.m3u8", buildProfileOptions{
		PreferredProfile: preferredProfile,
		StartPositionSec: options.StartPositionSec,
		PreferRemux:      options.PreferRemux,
		SourceVideoCodec: options.SourceVideoCodec,
		SourceAudioCodec: options.SourceAudioCodec,
	})

	var lastErr error
	for index, profile := range profiles {
		if index > 0 {
			_ = os.RemoveAll(dir)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return Session{}, err
			}
		}

		state, err := startTranscodeSession(ctx, cmdName, movieID, sessionID, dir, playlistPath, profile)
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
	touchSession(state)
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
		m.archiveSessionStateLocked(state, "stopped", time.Now().UTC())
	}
	m.mu.Unlock()
	if !ok {
		return ErrSessionNotFound
	}
	stopSessionState(state)
	return nil
}

func (m *Manager) GetSessionSnapshot(sessionID string) (SessionSnapshot, error) {
	if m == nil {
		return SessionSnapshot{}, ErrSessionNotFound
	}
	m.mu.RLock()
	state, ok := m.sessions[sessionID]
	cfg := m.cfg
	recentSnapshots := append([]SessionSnapshot(nil), m.recentSnapshots...)
	m.mu.RUnlock()
	if !ok {
		for _, snapshot := range recentSnapshots {
			if snapshot.Session.ID == sessionID {
				return snapshot, nil
			}
		}
		return SessionSnapshot{}, ErrSessionNotFound
	}
	return buildSessionSnapshot(state, sessionIdleTimeout(cfg)), nil
}

func (m *Manager) ListSessionSnapshots(limit int) []SessionSnapshot {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	cfg := m.cfg
	states := make([]*sessionState, 0, len(m.sessions))
	for _, state := range m.sessions {
		states = append(states, state)
	}
	recentSnapshots := append([]SessionSnapshot(nil), m.recentSnapshots...)
	m.mu.RUnlock()

	snapshots := make([]SessionSnapshot, 0, len(states)+len(recentSnapshots))
	timeout := sessionIdleTimeout(cfg)
	for _, state := range states {
		snapshots = append(snapshots, buildSessionSnapshot(state, timeout))
	}
	snapshots = append(snapshots, recentSnapshots...)
	slices.SortFunc(snapshots, func(a, b SessionSnapshot) int {
		switch {
		case a.Session.StartedAt.After(b.Session.StartedAt):
			return -1
		case a.Session.StartedAt.Before(b.Session.StartedAt):
			return 1
		default:
			return 0
		}
	})
	// Active snapshots win over archived ones with the same session ID.
	snapshots = compactSessionSnapshotsByID(snapshots)
	if limit > 0 && len(snapshots) > limit {
		return snapshots[:limit]
	}
	return snapshots
}

func (m *Manager) Close() {
	if m == nil {
		return
	}
	if m.janitorCancel != nil {
		m.janitorCancel()
		<-m.janitorDone
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
) (*sessionState, error) {
	runCtx, cancel := context.WithCancel(context.Background())
	cmd := executil.CommandContext(runCtx, cmdName, profile.Args...)
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
			StartPositionSec: profile.TimelineOriginSec,
			ProfileName:      profile.Name,
			Kind:             profile.SessionKind,
			StartedAt:        time.Now().UTC(),
		},
		cancel:         cancel,
		cmd:            cmd,
		waitCh:         make(chan error, 1),
		lastAccessedAt: time.Now().UTC(),
	}

	go func() {
		err := cmd.Wait()
		markSessionFinished(state, err)
		state.waitCh <- err
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

func buildTranscodeProfiles(cfg Config, sourcePath string, segmentPattern string, playlistPath string, options buildProfileOptions) []transcodeProfile {
	inputPrefix := []string{"-y"}
	if cfg.HardwareDecode {
		inputPrefix = append(inputPrefix, "-hwaccel", "auto")
	}
	seekPlan := buildSeekPlan(options.StartPositionSec)
	inputSeekArgs, accurateSeekArgs := seekPlan.InputArgs, seekPlan.AccurateArgs
	configuredPreference := normalizeHardwareEncoderProfileName(cfg.HardwareEncoder)

	transcodeHLSArgs := []string{
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
	remuxHLSArgs := []string{
		"-c:v", "copy",
		"-c:a", "copy",
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
	if options.PreferRemux && canStreamCopyCodecsForHLS(options.SourceVideoCodec, options.SourceAudioCodec) {
		profiles = append(profiles, transcodeProfile{
			Name:              "remux_copy",
			SessionKind:       "remux-hls",
			TimelineOriginSec: seekPlan.InputSeekSec,
			Args: append(
				append(
					append(append(append([]string{}, inputPrefix...), inputSeekArgs...), "-i", sourcePath),
					accurateSeekArgs...,
				),
				remuxHLSArgs...,
			),
		})
	}
	if cfg.HardwareDecode {
		switch runtime.GOOS {
		case "windows":
			profiles = append(profiles,
				transcodeProfile{
					Name:              "h264_nvenc",
					SessionKind:       "transcode-hls",
					TimelineOriginSec: seekPlan.RequestedStartSec,
					Args: append(
						append(
							append(append(append([]string{}, inputPrefix...), inputSeekArgs...), "-i", sourcePath),
							append(accurateSeekArgs, "-c:v", "h264_nvenc", "-preset", "p5", "-cq", "19")...,
						),
						transcodeHLSArgs...,
					),
				},
				transcodeProfile{
					Name:              "h264_qsv",
					SessionKind:       "transcode-hls",
					TimelineOriginSec: seekPlan.RequestedStartSec,
					Args: append(
						append(
							append(append(append([]string{}, inputPrefix...), inputSeekArgs...), "-i", sourcePath),
							append(accurateSeekArgs, "-c:v", "h264_qsv", "-preset", "medium", "-global_quality", "20")...,
						),
						transcodeHLSArgs...,
					),
				},
				transcodeProfile{
					Name:              "h264_amf",
					SessionKind:       "transcode-hls",
					TimelineOriginSec: seekPlan.RequestedStartSec,
					Args: append(
						append(
							append(append(append([]string{}, inputPrefix...), inputSeekArgs...), "-i", sourcePath),
							append(accurateSeekArgs, "-c:v", "h264_amf", "-quality", "quality")...,
						),
						transcodeHLSArgs...,
					),
				},
			)
		case "darwin":
			profiles = append(profiles, transcodeProfile{
				Name:              "h264_videotoolbox",
				SessionKind:       "transcode-hls",
				TimelineOriginSec: seekPlan.RequestedStartSec,
				Args: append(
					append(
						append(append(append([]string{}, inputPrefix...), inputSeekArgs...), "-i", sourcePath),
						append(accurateSeekArgs, "-c:v", "h264_videotoolbox", "-b:v", "12M", "-allow_sw", "1")...,
					),
					transcodeHLSArgs...,
				),
			})
		}
	}

	if configuredPreference != "" {
		options.PreferredProfile = configuredPreference
	}
	if options.PreferredProfile != "" {
		for idx, profile := range profiles {
			if profile.Name != options.PreferredProfile {
				continue
			}
			if idx > 0 {
				profiles[0], profiles[idx] = profiles[idx], profiles[0]
			}
			break
		}
	}

	profiles = append(profiles, transcodeProfile{
		Name:              "libx264",
		SessionKind:       "transcode-hls",
		TimelineOriginSec: seekPlan.RequestedStartSec,
		Args: append(
			append(
				append(append(append([]string{}, inputPrefix...), inputSeekArgs...), "-i", sourcePath),
				append(accurateSeekArgs, "-c:v", "libx264", "-preset", "veryfast", "-crf", "17")...,
			),
			transcodeHLSArgs...,
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

type seekPlan struct {
	RequestedStartSec float64
	InputSeekSec      float64
	AccurateSeekSec   float64
	InputArgs         []string
	AccurateArgs      []string
}

func buildSeekPlan(startPositionSec float64) seekPlan {
	plan := seekPlan{}
	if startPositionSec <= 0 {
		return plan
	}

	const preciseSeekWindowSec = 2.0
	plan.RequestedStartSec = startPositionSec
	plan.InputSeekSec = startPositionSec - preciseSeekWindowSec
	if plan.InputSeekSec < 0 {
		plan.InputSeekSec = 0
	}
	plan.AccurateSeekSec = startPositionSec - plan.InputSeekSec
	if plan.AccurateSeekSec < 0 {
		plan.AccurateSeekSec = 0
	}

	plan.InputArgs = []string{"-ss", formatSeekOffset(plan.InputSeekSec)}
	if plan.AccurateSeekSec > 0.001 {
		plan.AccurateArgs = []string{"-ss", formatSeekOffset(plan.AccurateSeekSec)}
	}
	return plan
}

func buildSeekArgs(startPositionSec float64) ([]string, []string) {
	plan := buildSeekPlan(startPositionSec)
	return plan.InputArgs, plan.AccurateArgs
}

func (m *Manager) cleanupExpiredSessions(now time.Time) []*sessionState {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	timeout := sessionIdleTimeout(m.cfg)
	if timeout <= 0 {
		return nil
	}

	staleStates := make([]*sessionState, 0)
	for existingID, state := range m.sessions {
		if state.lastAccessedAt.IsZero() {
			state.lastAccessedAt = state.session.StartedAt
		}
		if now.Sub(state.lastAccessedAt) < timeout {
			continue
		}
		delete(m.sessions, existingID)
		m.archiveSessionStateLocked(state, "expired", now)
		staleStates = append(staleStates, state)
	}
	return staleStates
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
		m.archiveSessionStateLocked(existingState, "replaced", time.Now().UTC())
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
		m.archiveSessionStateLocked(existingState, "replaced", time.Now().UTC())
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
		m.archiveSessionStateLocked(existingState, "closed", time.Now().UTC())
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

func touchSession(state *sessionState) {
	if state == nil {
		return
	}
	state.lastAccessedAt = time.Now().UTC()
}

func markSessionFinished(state *sessionState, err error) {
	if state == nil {
		return
	}
	state.finishedAt = time.Now().UTC()
	if err != nil {
		state.lastError = err.Error()
	}
}

func buildSessionSnapshot(state *sessionState, timeout time.Duration) SessionSnapshot {
	lastAccessedAt := state.lastAccessedAt
	if lastAccessedAt.IsZero() {
		lastAccessedAt = state.session.StartedAt
	}
	snapshot := SessionSnapshot{
		Session:        state.session,
		LastAccessedAt: lastAccessedAt,
		FinishedAt:     state.finishedAt,
		State:          "running",
		LastError:      strings.TrimSpace(state.lastError),
	}
	if timeout > 0 {
		snapshot.ExpiresAt = lastAccessedAt.Add(timeout)
	}
	if !state.finishedAt.IsZero() {
		snapshot.State = "finished"
	}
	if snapshot.LastError != "" {
		snapshot.State = "failed"
	}
	return snapshot
}

func (m *Manager) archiveSessionStateLocked(state *sessionState, terminalState string, finishedAt time.Time) {
	if m == nil || state == nil {
		return
	}
	snapshot := buildSessionSnapshot(state, sessionIdleTimeout(m.cfg))
	snapshot = finalizeArchivedSnapshot(snapshot, terminalState, finishedAt)
	m.recentSnapshots = append(m.recentSnapshots, snapshot)
	slices.SortFunc(m.recentSnapshots, func(a, b SessionSnapshot) int {
		switch {
		case a.Session.StartedAt.After(b.Session.StartedAt):
			return -1
		case a.Session.StartedAt.Before(b.Session.StartedAt):
			return 1
		default:
			return 0
		}
	})
	m.recentSnapshots = compactSessionSnapshotsByID(m.recentSnapshots)
	if len(m.recentSnapshots) > recentSessionHistoryLimit {
		m.recentSnapshots = m.recentSnapshots[:recentSessionHistoryLimit]
	}
}

func finalizeArchivedSnapshot(snapshot SessionSnapshot, terminalState string, finishedAt time.Time) SessionSnapshot {
	if snapshot.FinishedAt.IsZero() {
		snapshot.FinishedAt = finishedAt
	}
	if snapshot.State == "failed" || snapshot.State == "finished" {
		return snapshot
	}
	if strings.TrimSpace(terminalState) != "" {
		snapshot.State = terminalState
	}
	return snapshot
}

func compactSessionSnapshotsByID(snapshots []SessionSnapshot) []SessionSnapshot {
	if len(snapshots) == 0 {
		return snapshots
	}
	compacted := make([]SessionSnapshot, 0, len(snapshots))
	seen := make(map[string]struct{}, len(snapshots))
	for _, snapshot := range snapshots {
		sessionID := strings.TrimSpace(snapshot.Session.ID)
		if sessionID == "" {
			continue
		}
		if _, ok := seen[sessionID]; ok {
			continue
		}
		seen[sessionID] = struct{}{}
		compacted = append(compacted, snapshot)
	}
	return compacted
}

func sessionIdleTimeout(cfg Config) time.Duration {
	if cfg.SessionIdleTimeout > 0 {
		return cfg.SessionIdleTimeout
	}
	return 3 * time.Minute
}

func sessionJanitorInterval(cfg Config) time.Duration {
	if cfg.SessionJanitorInterval > 0 {
		return cfg.SessionJanitorInterval
	}
	return 30 * time.Second
}

func (m *Manager) startJanitorLoop() {
	ctx, cancel := context.WithCancel(context.Background())
	m.janitorCancel = cancel
	go func() {
		defer close(m.janitorDone)
		// The janitor only reaps idle sessions; it does not participate in
		// session startup so playback requests stay on the hot path.
		ticker := time.NewTicker(sessionJanitorInterval(m.cfg))
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				for _, stale := range m.cleanupExpiredSessions(time.Now().UTC()) {
					stopSessionState(stale)
				}
			}
		}
	}()
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
	case "videotoolbox", "vt", "h264_videotoolbox":
		return "h264_videotoolbox"
	case "software", "libx264":
		return "libx264"
	default:
		return ""
	}
}

func canStreamCopyCodecsForHLS(videoCodec string, audioCodec string) bool {
	videoCodec = strings.ToLower(strings.TrimSpace(videoCodec))
	audioCodec = strings.ToLower(strings.TrimSpace(audioCodec))
	return videoCodec == "h264" && (audioCodec == "aac" || audioCodec == "mp3" || audioCodec == "ac3" || audioCodec == "eac3")
}

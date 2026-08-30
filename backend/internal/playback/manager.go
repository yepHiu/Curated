// Package playback manages HLS stream push sessions for video playback,
// including remux and hardware-accelerated transcode profiles via ffmpeg.
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
	"slices"
	"strings"
	"sync"
	"time"

	"curated-backend/internal/executil"
)

var (
	// ErrSessionNotFound is returned when a playback session ID is not found in the active registry or recent snapshots.
	ErrSessionNotFound = errors.New("playback session not found")
	// ErrStreamPushDisabled is returned when stream push is not enabled in the manager configuration.
	ErrStreamPushDisabled = errors.New("stream push is disabled")
)

const (
	hlsInitialSegmentSeconds       = "2"
	hlsTargetSegmentSeconds        = "2"
	hlsStartupTranscodeLeadTimeout = 12 * time.Second
	hlsSegmentPattern              = "segment-%05d.m4s"
	hlsInitFilename                = "init.mp4"
	hlsFirstSegmentName            = "segment-00000.m4s"
	hlsFourthSegmentName           = "segment-00003.m4s"
	// Remux (stream-copy) can run tens of times realtime, so it stays capped at
	// 2.5x. Hardware transcode follows the client with pause/resume. Software
	// libx264 is CPU-bound, so it keeps a modest readrate ceiling.
	inputReadRateRemux    = "2.5"
	inputReadRateSoftware = "1.5"
)

// Config holds playback stream push settings including ffmpeg invocation and session lifecycle.
type Config struct {
	Enabled                bool
	HardwareDecode         bool
	HardwareEncoder        string
	FFmpegCommand          string
	SessionRoot            string
	SessionIdleTimeout     time.Duration
	SessionJanitorInterval time.Duration
}

// Session represents an active HLS playback session with its on-disk playlist and segment files.
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
	mu                 sync.RWMutex
	session            Session
	cancel             context.CancelFunc
	cmd                *exec.Cmd
	waitCh             chan error
	lastAccessedAt     time.Time
	finishedAt         time.Time
	lastError          string
	encoderSpeed       string
	writtenDurationSec float64
	lastSeekKind       string
	stdin              io.WriteCloser
	throttlePaused     bool
	lastRequestedSec   float64
}

type transcodeProfile struct {
	Name              string
	SessionKind       string
	Args              []string
	TimelineOriginSec float64
}

// StartHLSSessionOptions controls HLS startup behavior including seek position and remux preference.
type StartHLSSessionOptions struct {
	StartPositionSec float64
	PreferRemux      bool
	SourceVideoCodec string
	SourceAudioCodec string
	SourceContainer  string
}

type buildProfileOptions struct {
	PreferredProfile string
	StartPositionSec float64
	// RemuxInputSeekSec enables the stream-copy remux profile when non-nil; the
	// pointed value is the keyframe-aligned input seek for that session.
	RemuxInputSeekSec *float64
	// TranscodeKeyframeSec is the last keyframe at or before StartPositionSec.
	// Nil means the probe missed or was not needed (start at zero).
	TranscodeKeyframeSec *float64
	SourceVideoCodec     string
	SourceAudioCodec     string
	SourceContainer      string
	SourcePath           string
	// EncoderAvailability filters hardware encoder profiles; nil keeps every
	// candidate so capability-unknown setups keep the try-and-fail chain.
	EncoderAvailability map[string]bool
}

// Manager governs HLS stream push sessions, including lifecycle, file resolution, and diagnostics.
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
	// encoderProbes caches per-command hardware encoder capability results so
	// session startup can skip encoders that cannot run instead of timing out
	// through the readiness chain once per profile.
	encoderProbeMu sync.Mutex
	encoderProbes  map[string]*encoderProbeState
}

// SessionSnapshot is a point-in-time view of a session for diagnostics endpoints.
type SessionSnapshot struct {
	Session        Session
	LastAccessedAt time.Time
	ExpiresAt      time.Time
	FinishedAt     time.Time
	// State is a coarse lifecycle label exposed to diagnostics endpoints.
	State              string
	LastError          string
	EncoderSpeed       string
	WrittenDurationSec float64
	LastSeekKind       string
}

const recentSessionHistoryLimit = 32

// New creates a playback Manager and starts its idle-session janitor loop.
func New(cfg Config) *Manager {
	manager := &Manager{
		cfg:             cfg,
		sessions:        make(map[string]*sessionState),
		recentSnapshots: make([]SessionSnapshot, 0, recentSessionHistoryLimit),
		janitorDone:     make(chan struct{}),
	}
	manager.startJanitorLoop()
	if cfg.Enabled {
		// Warm encoder capability in the background; browsing the library gives
		// the probe enough time to finish before the first playback request.
		go manager.warmEncoderProbe()
	}
	return manager
}

// Enabled reports whether stream push is enabled, safe to call on a nil receiver.
func (m *Manager) Enabled() bool {
	if m == nil {
		return false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg.Enabled
}

// SetConfig replaces the playback configuration, safe to call on a nil receiver.
func (m *Manager) SetConfig(cfg Config) {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.cfg = cfg
	m.mu.Unlock()
}

// StartHLSSession launches a new HLS stream push session, preferring remux and falling back through transcode profiles.
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
	segmentPattern := hlsSegmentPattern

	cmdName := resolveFFmpegCommand(cfg.FFmpegCommand)
	preferredProfile := ""
	m.mu.RLock()
	preferredProfile = m.lastSuccessfulProfile
	m.mu.RUnlock()
	if options.StartPositionSec < 0 {
		options.StartPositionSec = 0
	}
	keyframe := probeStartKeyframe(ctx, cfg, sourcePath, options.StartPositionSec)
	profiles := buildTranscodeProfiles(cfg, sourcePath, segmentPattern, "index.m3u8", buildProfileOptions{
		PreferredProfile:     preferredProfile,
		StartPositionSec:     options.StartPositionSec,
		RemuxInputSeekSec:    remuxSeekFromKeyframe(options, keyframe),
		TranscodeKeyframeSec: keyframe,
		SourceVideoCodec:     options.SourceVideoCodec,
		SourceAudioCodec:     options.SourceAudioCodec,
		SourceContainer:      options.SourceContainer,
		SourcePath:           sourcePath,
		EncoderAvailability:  m.encoderAvailabilitySnapshot(cmdName, 2*time.Second),
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

// ResolveFile maps a session-relative filename to an absolute path, serving HLS segments and playlists.
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
	if strings.HasSuffix(strings.ToLower(cleanName), ".tmp") {
		return "", ErrSessionNotFound
	}
	noteRequestedHLSFile(state, cleanName)
	state.maybeThrottle()
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

// DeleteSession stops and removes a playback session, archiving its terminal snapshot.
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

// GetSessionSnapshot returns a diagnostic snapshot for the given session, checking both active and recent archives.
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

// ListSessionSnapshots returns combined active and recent session snapshots sorted by start time, with active entries overriding archived ones.
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

// Close stops the janitor loop and all active sessions.
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
		if playlistReferencesSegment(playlistPath, segmentName) {
			return nil
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

func waitForPlaylistSegmentReferenceOptional(ctx context.Context, playlistPath string, segmentName string, waitCh <-chan error, timeout time.Duration) (bool, error) {
	deadline := time.Now().Add(timeout)
	for {
		if playlistReferencesSegment(playlistPath, segmentName) {
			return true, nil
		}
		select {
		case err := <-waitCh:
			if err == nil {
				return false, nil
			}
			return false, err
		default:
		}
		if ctx != nil {
			select {
			case <-ctx.Done():
				return false, ctx.Err()
			default:
			}
		}
		if time.Now().After(deadline) {
			return false, nil
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func playlistReferencesSegment(playlistPath string, segmentName string) bool {
	raw, err := os.ReadFile(playlistPath)
	if err != nil {
		return false
	}
	playlist := string(raw)
	return strings.Contains(playlist, "#EXTINF:") && strings.Contains(playlist, segmentName)
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
	stdout, stdoutErr := cmd.StdoutPipe()
	if stdoutErr != nil {
		cmd.Stdout = io.Discard
	}
	stdin, stdinErr := cmd.StdinPipe()
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
		lastSeekKind:   sessionSeekKindForOrigin(profile.TimelineOriginSec),
	}
	if stdinErr == nil {
		state.stdin = stdin
	}
	if stdoutErr == nil {
		go consumeFFmpegProgress(stdout, func(parsed ffmpegProgress) {
			state.applyProgress(parsed)
			state.maybeThrottle()
		})
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
	initPath := filepath.Join(dir, hlsInitFilename)
	if err := waitForNonEmptyFileOrProcessExit(ctx, initPath, state.waitCh, 8*time.Second); err != nil {
		cancel()
		stderrText := strings.TrimSpace(stderr.String())
		return nil, fmt.Errorf("%s fMP4 init failed: %w: %s", profile.Name, err, stderrText)
	}
	firstSegmentPath := filepath.Join(dir, hlsFirstSegmentName)
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
	if profile.SessionKind == "transcode-hls" {
		if _, err := waitForPlaylistSegmentReferenceOptional(ctx, playlistPath, hlsFourthSegmentName, state.waitCh, hlsStartupTranscodeLeadTimeout); err != nil {
			cancel()
			stderrText := strings.TrimSpace(stderr.String())
			return nil, fmt.Errorf("%s startup buffer failed: %w: %s", profile.Name, err, stderrText)
		}
	}

	return state, nil
}

func buildTranscodeProfiles(cfg Config, sourcePath string, segmentPattern string, playlistPath string, options buildProfileOptions) []transcodeProfile {
	remuxInputPrefix := buildHLSInputPrefix(inputReadRateRemux)
	transcodeInputPrefix := buildHLSInputPrefix("")
	seekPlan := buildTranscodeSeekPlan(options)
	inputSeekArgs, accurateSeekArgs := seekPlan.InputArgs, seekPlan.AccurateArgs
	configuredPreference := normalizeHardwareEncoderProfileName(cfg.HardwareEncoder)

	hlsMuxerArgs := []string{
		"-f", "hls",
		"-hls_segment_type", "fmp4",
		"-hls_fmp4_init_filename", hlsInitFilename,
		"-hls_init_time", hlsInitialSegmentSeconds,
		"-hls_time", hlsTargetSegmentSeconds,
		"-hls_list_size", "0",
		"-hls_allow_cache", "0",
		"-hls_flags", "independent_segments+temp_file",
		"-hls_playlist_type", "event",
		"-start_number", "0",
		"-hls_segment_filename", segmentPattern,
		playlistPath,
	}
	transcodeHLSArgs := append([]string{
		"-fps_mode", "cfr",
		"-pix_fmt", "yuv420p",
		"-c:a", "aac",
		"-ac", "2",
		"-force_key_frames", "expr:gte(t,n_forced*2)",
	}, hlsMuxerArgs...)
	remuxHLSArgs := append([]string{
		"-c:v", "copy",
		"-c:a", "copy",
		// Mid-stream stream-copy starts keep their original timestamps, so force
		// the segment media timeline to begin at zero (= TimelineOriginSec).
		"-avoid_negative_ts", "make_zero",
		"-fflags", "+genpts",
		"-muxdelay", "0",
		"-muxpreload", "0",
	}, hlsMuxerArgs...)
	remuxInputPrefix = append(remuxInputPrefix, "-ignore_editlist", "1")

	remuxTimelineOrigin := 0.0
	remuxInputSeekArgs := []string(nil)
	if options.RemuxInputSeekSec != nil {
		remuxTimelineOrigin = *options.RemuxInputSeekSec
		if remuxTimelineOrigin > 0.001 {
			remuxInputSeekArgs = []string{"-ss", formatSeekOffset(remuxTimelineOrigin)}
		}
	}

	profiles := make([]transcodeProfile, 0, 4)
	if options.RemuxInputSeekSec != nil {
		profiles = append(profiles, transcodeProfile{
			Name:              "remux_copy",
			SessionKind:       "remux-hls",
			TimelineOriginSec: remuxTimelineOrigin,
			Args:              buildProfileArgs(remuxInputPrefix, remuxInputSeekArgs, sourcePath, nil, nil, remuxHLSArgs),
		})
	}
	if cfg.HardwareDecode {
		for _, spec := range hardwareEncoderSpecs() {
			if !encoderAllowed(options.EncoderAvailability, spec.Name) {
				continue
			}
			hwPrefix := transcodeInputPrefix
			if len(spec.InputArgs) > 0 {
				hwPrefix = append(append([]string{}, transcodeInputPrefix...), spec.InputArgs...)
			}
			profiles = append(profiles, transcodeProfile{
				Name:              spec.Name,
				SessionKind:       "transcode-hls",
				TimelineOriginSec: seekPlan.TimelineOriginSec,
				Args:              buildProfileArgs(hwPrefix, inputSeekArgs, sourcePath, accurateSeekArgs, spec.EncoderArgs, transcodeHLSArgs),
			})
		}
	}

	if configuredPreference != "" {
		options.PreferredProfile = configuredPreference
	}
	profiles = preferHardwareTranscodeProfile(profiles, options.PreferredProfile)

	softwarePrefix := buildHLSInputPrefix(inputReadRateSoftware)
	profiles = append(profiles, transcodeProfile{
		Name:              "libx264",
		SessionKind:       "transcode-hls",
		TimelineOriginSec: seekPlan.TimelineOriginSec,
		Args: buildProfileArgs(softwarePrefix, inputSeekArgs, sourcePath, accurateSeekArgs,
			[]string{"-c:v", "libx264", "-preset", "veryfast", "-crf", "22"}, transcodeHLSArgs),
	})
	if configuredPreference == "libx264" {
		return []transcodeProfile{profiles[len(profiles)-1]}
	}
	return profiles
}

func preferHardwareTranscodeProfile(profiles []transcodeProfile, name string) []transcodeProfile {
	name = strings.TrimSpace(name)
	if name == "" || name == "libx264" || name == "remux_copy" {
		return profiles
	}
	start := 0
	if len(profiles) > 0 && profiles[0].SessionKind == "remux-hls" {
		start = 1
	}
	for idx := start; idx < len(profiles); idx++ {
		if profiles[idx].Name != name {
			continue
		}
		if idx > start {
			profiles[start], profiles[idx] = profiles[idx], profiles[start]
		}
		break
	}
	return profiles
}

func buildHLSInputPrefix(readRate string) []string {
	prefix := []string{"-y", "-progress", "pipe:1"}
	if strings.TrimSpace(readRate) != "" {
		prefix = append(prefix, "-readrate", readRate)
	}
	return prefix
}

// buildProfileArgs assembles one ffmpeg command line shared by every profile:
// global+input options, input seek, the source, output-side precise seek,
// encoder options, and the HLS muxer options.
func buildProfileArgs(inputPrefix []string, inputSeekArgs []string, sourcePath string, accurateSeekArgs []string, encoderArgs []string, hlsArgs []string) []string {
	args := make([]string, 0, len(inputPrefix)+len(inputSeekArgs)+2+len(accurateSeekArgs)+len(encoderArgs)+len(hlsArgs))
	args = append(args, inputPrefix...)
	args = append(args, inputSeekArgs...)
	args = append(args, "-i", sourcePath)
	args = append(args, accurateSeekArgs...)
	args = append(args, encoderArgs...)
	args = append(args, hlsArgs...)
	return args
}

func probeStartKeyframe(ctx context.Context, cfg Config, sourcePath string, startPositionSec float64) *float64 {
	if startPositionSec <= 0.001 {
		return nil
	}
	keyframeSec, ok := probeKeyframeAtOrBeforeFunc(ctx, sourcePath, cfg.FFmpegCommand, startPositionSec, keyframeProbeWindowSec)
	if !ok {
		return nil
	}
	return &keyframeSec
}

func remuxSeekFromKeyframe(options StartHLSSessionOptions, keyframe *float64) *float64 {
	if !options.PreferRemux {
		return nil
	}
	if !canStreamCopyCodecsForHLS(options.SourceVideoCodec, options.SourceAudioCodec) {
		return nil
	}
	if options.StartPositionSec <= 0.001 {
		zero := 0.0
		return &zero
	}
	return keyframe
}

// resolveRemuxInputSeek decides whether a stream-copy session can serve this
// start request. Start-at-zero always can; mid-stream starts must align to the
// last keyframe at or before the requested position, otherwise transcode remains
// the fallback.
func resolveRemuxInputSeek(ctx context.Context, cfg Config, sourcePath string, options StartHLSSessionOptions) *float64 {
	return remuxSeekFromKeyframe(options, probeStartKeyframe(ctx, cfg, sourcePath, options.StartPositionSec))
}

func formatSeekOffset(startPositionSec float64) string {
	if startPositionSec <= 0 {
		return "0"
	}
	return fmt.Sprintf("%.3f", startPositionSec)
}

type seekPlan struct {
	RequestedStartSec float64
	TimelineOriginSec float64
	InputSeekSec      float64
	AccurateSeekSec   float64
	InputArgs         []string
	AccurateArgs      []string
}

func needsSlowTranscodeSeek(container string, sourcePath string) bool {
	name := strings.ToLower(strings.TrimSpace(container))
	if name == "" {
		name = strings.TrimPrefix(strings.ToLower(filepath.Ext(strings.TrimSpace(sourcePath))), ".")
	}
	switch name {
	case "avi", "wmv", "asf":
		return true
	default:
		return false
	}
}

func buildTranscodeSeekPlan(options buildProfileOptions) seekPlan {
	startPositionSec := options.StartPositionSec
	if startPositionSec <= 0 {
		return seekPlan{}
	}
	if needsSlowTranscodeSeek(options.SourceContainer, options.SourcePath) {
		return buildHybridSeekPlan(startPositionSec)
	}
	origin := startPositionSec
	if options.TranscodeKeyframeSec != nil {
		origin = *options.TranscodeKeyframeSec
		if origin < 0 {
			origin = 0
		}
	}
	return buildFastInputSeekPlan(origin, startPositionSec)
}

func buildFastInputSeekPlan(originSec float64, requestedSec float64) seekPlan {
	plan := seekPlan{
		RequestedStartSec: requestedSec,
		TimelineOriginSec: originSec,
	}
	if originSec > 0.001 {
		plan.InputSeekSec = originSec
		plan.InputArgs = []string{"-ss", formatSeekOffset(originSec)}
	}
	return plan
}

func buildHybridSeekPlan(startPositionSec float64) seekPlan {
	plan := seekPlan{}
	if startPositionSec <= 0 {
		return plan
	}

	const preciseSeekWindowSec = 2.0
	plan.RequestedStartSec = startPositionSec
	plan.TimelineOriginSec = startPositionSec
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
		if now.Sub(state.lastAccessedOrStartedAt()) < timeout {
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
	if state.session.Kind == "transcode-hls" && profileName != "" && profileName != "libx264" {
		m.lastSuccessfulProfile = profileName
	}
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
	state.mu.Lock()
	stdin := state.stdin
	state.stdin = nil
	state.mu.Unlock()
	if stdin != nil {
		_ = stdin.Close()
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

func (s *sessionState) touchAt(now time.Time) {
	if s == nil {
		return
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	s.mu.Lock()
	s.lastAccessedAt = now
	s.mu.Unlock()
}

func (s *sessionState) applyProgress(parsed ffmpegProgress) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if parsed.HasSpeed {
		s.encoderSpeed = formatEncoderSpeed(parsed.Speed)
	}
	if parsed.HasOutTime {
		s.writtenDurationSec = parsed.OutTimeSec
	}
}

func noteRequestedHLSFile(state *sessionState, name string) {
	if state == nil {
		return
	}
	mediaSec, ok := mediaTimeFromHLSFileName(name, hlsSegmentDurationSec)
	if !ok {
		return
	}
	state.mu.Lock()
	if mediaSec > state.lastRequestedSec {
		state.lastRequestedSec = mediaSec
	}
	state.mu.Unlock()
}

func (s *sessionState) maybeThrottle() {
	if s == nil {
		return
	}
	s.mu.Lock()
	action := nextThrottleAction(s.throttlePaused, s.writtenDurationSec, s.lastRequestedSec, throttlePauseLeadSec, throttleResumeLeadSec)
	stdin := s.stdin
	s.mu.Unlock()
	if action == throttleNone || stdin == nil {
		return
	}
	key := "u"
	if action == throttlePause {
		key = "p"
	}
	if _, err := io.WriteString(stdin, key); err != nil {
		return
	}
	s.mu.Lock()
	s.throttlePaused = action == throttlePause
	s.mu.Unlock()
}

func (s *sessionState) markFinishedAt(now time.Time, err error) {
	if s == nil {
		return
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	s.mu.Lock()
	s.finishedAt = now
	if err != nil {
		s.lastError = err.Error()
	}
	s.mu.Unlock()
}

func (s *sessionState) lastAccessedOrStartedAt() time.Time {
	if s == nil {
		return time.Time{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.lastAccessedAt.IsZero() {
		return s.lastAccessedAt
	}
	return s.session.StartedAt
}

func (s *sessionState) snapshot(timeout time.Duration) SessionSnapshot {
	if s == nil {
		return SessionSnapshot{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	lastAccessedAt := s.lastAccessedAt
	if lastAccessedAt.IsZero() {
		lastAccessedAt = s.session.StartedAt
	}
	snapshot := SessionSnapshot{
		Session:            s.session,
		LastAccessedAt:     lastAccessedAt,
		FinishedAt:         s.finishedAt,
		State:              "running",
		LastError:          strings.TrimSpace(s.lastError),
		EncoderSpeed:       s.encoderSpeed,
		WrittenDurationSec: s.writtenDurationSec,
		LastSeekKind:       s.lastSeekKind,
	}
	if timeout > 0 {
		snapshot.ExpiresAt = lastAccessedAt.Add(timeout)
	}
	if !s.finishedAt.IsZero() {
		snapshot.State = "finished"
	}
	if snapshot.LastError != "" {
		snapshot.State = "failed"
	}
	return snapshot
}

func touchSession(state *sessionState) {
	if state == nil {
		return
	}
	state.touchAt(time.Now().UTC())
}

func markSessionFinished(state *sessionState, err error) {
	if state == nil {
		return
	}
	state.markFinishedAt(time.Now().UTC(), err)
}

func buildSessionSnapshot(state *sessionState, timeout time.Duration) SessionSnapshot {
	return state.snapshot(timeout)
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

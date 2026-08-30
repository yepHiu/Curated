package playback

import (
	"context"
	"runtime"
	"strings"
	"time"

	"curated-backend/internal/executil"
)

// hwEncoderProfileSpec describes one hardware encoder candidate for the
// current platform. hardwareEncoderSpecs is the single source of truth used by
// both profile building and runtime capability probing.
type hwEncoderProfileSpec struct {
	Name        string
	InputArgs   []string
	EncoderArgs []string
}

func hardwareEncoderSpecs() []hwEncoderProfileSpec {
	switch runtime.GOOS {
	case "windows":
		return []hwEncoderProfileSpec{
			{Name: "h264_nvenc", InputArgs: []string{"-hwaccel", "d3d11va"}, EncoderArgs: []string{"-c:v", "h264_nvenc", "-preset", "p4", "-cq", "23"}},
			{Name: "h264_qsv", InputArgs: []string{"-hwaccel", "qsv"}, EncoderArgs: []string{"-c:v", "h264_qsv", "-preset", "fast", "-global_quality", "24"}},
			{Name: "h264_amf", InputArgs: []string{"-hwaccel", "d3d11va"}, EncoderArgs: []string{"-c:v", "h264_amf", "-quality", "speed"}},
		}
	case "darwin":
		return []hwEncoderProfileSpec{
			{Name: "h264_videotoolbox", EncoderArgs: []string{"-c:v", "h264_videotoolbox", "-b:v", "12M", "-allow_sw", "1"}},
		}
	default:
		return nil
	}
}

// encoderAllowed reports whether an encoder may be used. A nil availability
// map means capability is unknown, so every candidate stays in the chain and
// the legacy try-and-fail startup behavior applies.
func encoderAllowed(availability map[string]bool, name string) bool {
	if availability == nil {
		return true
	}
	return availability[name]
}

type encoderProbeState struct {
	done      chan struct{}
	available map[string]bool
}

// probeEncoderRuntimeFunc is overridable so tests can stub the ffmpeg test
// encode without spawning real processes.
var probeEncoderRuntimeFunc = probeEncoderRuntime

// warmEncoderProbe kicks off capability probing in the background so the first
// playback request usually finds a finished result.
func (m *Manager) warmEncoderProbe() {
	if m == nil {
		return
	}
	m.encoderProbeMu.Lock()
	defer m.encoderProbeMu.Unlock()
	m.ensureEncoderProbeLocked(resolveFFmpegCommand(m.cfg.FFmpegCommand))
}

// encoderAvailabilitySnapshot returns the cached per-encoder capability map for
// the given ffmpeg command, waiting up to maxWait for an in-flight probe. It
// returns nil when capability is still unknown so callers keep the fallback
// chain intact.
func (m *Manager) encoderAvailabilitySnapshot(cmdName string, maxWait time.Duration) map[string]bool {
	if m == nil {
		return nil
	}
	m.encoderProbeMu.Lock()
	state := m.ensureEncoderProbeLocked(cmdName)
	m.encoderProbeMu.Unlock()

	timer := time.NewTimer(maxWait)
	defer timer.Stop()
	select {
	case <-state.done:
		return state.available
	case <-timer.C:
		return nil
	}
}

func (m *Manager) ensureEncoderProbeLocked(cmdName string) *encoderProbeState {
	if m.encoderProbes == nil {
		m.encoderProbes = make(map[string]*encoderProbeState)
	}
	if state, ok := m.encoderProbes[cmdName]; ok {
		return state
	}
	state := &encoderProbeState{
		done:      make(chan struct{}),
		available: make(map[string]bool),
	}
	m.encoderProbes[cmdName] = state
	go func() {
		defer close(state.done)
		for _, spec := range hardwareEncoderSpecs() {
			state.available[spec.Name] = probeEncoderRuntimeFunc(cmdName, spec.Name)
		}
	}()
	return state
}

// probeEncoderRuntime checks whether ffmpeg can actually encode with the given
// encoder right now. A tiny lavfi test encode is the only reliable signal:
// encoders listed by `-encoders` can still fail at runtime when drivers or
// display sessions are missing, which is exactly the case that used to burn a
// full readiness timeout per profile at session start.
func probeEncoderRuntime(cmdName string, encoder string) bool {
	if !ffmpegListsEncoder(cmdName, encoder) {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	cmd := executil.CommandContext(
		ctx,
		cmdName,
		"-hide_banner",
		"-v", "error",
		"-f", "lavfi", "-i", "nullsrc=s=256x256:d=0.5:r=30",
		"-pix_fmt", "yuv420p",
		"-frames:v", "6",
		"-c:v", encoder,
		"-f", "null", "-",
	)
	return cmd.Run() == nil
}

func ffmpegListsEncoder(cmdName string, encoder string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := executil.CommandContext(ctx, cmdName, "-hide_banner", "-encoders")
	output, err := cmd.Output()
	if err != nil {
		// If we cannot even list encoders, do not deny capability; the normal
		// try-and-fail session chain still handles it.
		return true
	}
	return encoderListed(output, encoder)
}

// encoderListed matches the second whitespace-separated field of `ffmpeg
// -encoders` output lines (flags then encoder name), so an encoder name can
// never match via a substring of another encoder's description.
func encoderListed(output []byte, encoder string) bool {
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == encoder {
			return true
		}
	}
	return false
}

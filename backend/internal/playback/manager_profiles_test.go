package playback

import (
	"context"
	"runtime"
	"strings"
	"testing"
)

func TestBuildTranscodeProfilesIncludesHardwareCandidatesWhenEnabled(t *testing.T) {
	profiles := buildTranscodeProfiles(
		Config{HardwareDecode: true},
		"movie.mkv",
		"segment-%05d.m4s",
		"index.m3u8",
		buildProfileOptions{},
	)
	switch runtime.GOOS {
	case "windows", "darwin":
		if len(profiles) < 2 {
			t.Fatalf("expected hardware profiles plus software fallback, got %d", len(profiles))
		}
	default:
		if len(profiles) < 1 {
			t.Fatalf("expected at least software fallback, got %d", len(profiles))
		}
	}
	if profiles[len(profiles)-1].Name != "libx264" {
		t.Fatalf("last profile = %q, want libx264 fallback", profiles[len(profiles)-1].Name)
	}
}

func TestBuildTranscodeProfilesSoftwareOnlyWhenHardwareDisabled(t *testing.T) {
	profiles := buildTranscodeProfiles(
		Config{HardwareDecode: false},
		"movie.mkv",
		"segment-%05d.m4s",
		"index.m3u8",
		buildProfileOptions{},
	)
	if len(profiles) != 1 {
		t.Fatalf("expected only software fallback, got %d profiles", len(profiles))
	}
	if profiles[0].Name != "libx264" {
		t.Fatalf("profile name = %q, want libx264", profiles[0].Name)
	}
}

func TestBuildTranscodeProfilesPrefersConfiguredHardwareEncoder(t *testing.T) {
	var cfg Config
	var wantFirst string
	switch runtime.GOOS {
	case "windows":
		cfg = Config{HardwareDecode: true, HardwareEncoder: "amf"}
		wantFirst = "h264_amf"
	case "darwin":
		cfg = Config{HardwareDecode: true, HardwareEncoder: "videotoolbox"}
		wantFirst = "h264_videotoolbox"
	default:
		t.Skip("configured hardware encoder ordering is only asserted on windows and darwin")
	}
	profiles := buildTranscodeProfiles(
		cfg,
		"movie.mkv",
		"segment-%05d.m4s",
		"index.m3u8",
		buildProfileOptions{},
	)
	if len(profiles) < 2 {
		t.Fatalf("expected hardware profiles plus software fallback, got %d", len(profiles))
	}
	if profiles[0].Name != wantFirst {
		t.Fatalf("first profile = %q, want %q", profiles[0].Name, wantFirst)
	}
}

func TestBuildTranscodeProfilesCanForceSoftwarePreference(t *testing.T) {
	profiles := buildTranscodeProfiles(
		Config{HardwareDecode: true, HardwareEncoder: "software"},
		"movie.mkv",
		"segment-%05d.m4s",
		"index.m3u8",
		buildProfileOptions{},
	)
	if len(profiles) != 1 {
		t.Fatalf("expected only software fallback, got %d profiles", len(profiles))
	}
	if profiles[0].Name != "libx264" {
		t.Fatalf("profile name = %q, want libx264", profiles[0].Name)
	}
}

func TestBuildTranscodeProfilesUsesDecodeAccelOnlyOnHardwareEncoders(t *testing.T) {
	zero := 0.0
	profiles := buildTranscodeProfiles(
		Config{HardwareDecode: true},
		"movie.mkv",
		"segment-%05d.m4s",
		"index.m3u8",
		buildProfileOptions{
			RemuxInputSeekSec: &zero,
			SourceVideoCodec:  "h264",
			SourceAudioCodec:  "aac",
		},
	)
	if len(profiles) == 0 {
		t.Fatal("expected at least one profile")
	}
	for _, profile := range profiles {
		joined := strings.Join(profile.Args, " ")
		switch profile.Name {
		case "remux_copy", "libx264", "h264_videotoolbox":
			if strings.Contains(joined, "-hwaccel") {
				t.Fatalf("profile %q must not enable hwaccel, got %q", profile.Name, joined)
			}
		default:
			if !strings.Contains(joined, "-hwaccel") {
				t.Fatalf("hardware profile %q should decode with hwaccel, got %q", profile.Name, joined)
			}
		}
	}
}

func TestBuildTranscodeProfilesKeepsSessionOutputsRelativeToCmdDir(t *testing.T) {
	profiles := buildTranscodeProfiles(
		Config{HardwareDecode: false},
		`D:\movie.mkv`,
		"segment-%05d.m4s",
		"index.m3u8",
		buildProfileOptions{},
	)
	if len(profiles) != 1 {
		t.Fatalf("expected only software fallback, got %d profiles", len(profiles))
	}
	args := strings.Join(profiles[0].Args, " ")
	if strings.Contains(args, `runtime\cache\playback-sessions`) {
		t.Fatalf("expected HLS output args to stay relative to cmd.Dir, got %q", args)
	}
	if !strings.Contains(args, "segment-%05d.m4s") || !strings.Contains(args, "index.m3u8") {
		t.Fatalf("expected relative HLS output names in args, got %q", args)
	}
	if !strings.Contains(args, "-hls_segment_type fmp4") || !strings.Contains(args, "init.mp4") {
		t.Fatalf("expected fMP4 HLS muxer flags in args, got %q", args)
	}
	if !strings.Contains(args, "-hls_init_time 2") || !strings.Contains(args, "-hls_time 2") {
		t.Fatalf("expected short HLS segment timings in args, got %q", args)
	}
	if !strings.Contains(args, "-force_key_frames expr:gte(t,n_forced*2)") {
		t.Fatalf("expected 2-second keyframe cadence in args, got %q", args)
	}
	if !strings.Contains(args, "-fps_mode cfr") {
		t.Fatalf("expected constant frame-rate transcode, got %q", args)
	}
	if !strings.Contains(args, "-pix_fmt yuv420p") {
		t.Fatalf("expected browser-safe pixel format in args, got %q", args)
	}
	if !strings.Contains(args, "-hls_allow_cache 0") || !strings.Contains(args, "-start_number 0") {
		t.Fatalf("expected deterministic HLS playlist flags in args, got %q", args)
	}
	if !strings.Contains(args, "-hls_flags independent_segments+temp_file") {
		t.Fatalf("expected complete-segment temp_file HLS flag, got %q", args)
	}
	if !strings.Contains(args, "-crf 22") {
		t.Fatalf("expected realtime libx264 CRF 22, got %q", args)
	}
	if strings.Contains(args, "-hwaccel") {
		t.Fatalf("software transcode must not enable hwaccel, got %q", args)
	}
}

func TestBuildTranscodeProfilesPacesInputReading(t *testing.T) {
	zero := 0.0
	profiles := buildTranscodeProfiles(
		Config{HardwareDecode: false},
		"movie.mkv",
		"segment-%05d.m4s",
		"index.m3u8",
		buildProfileOptions{
			StartPositionSec:  0,
			RemuxInputSeekSec: &zero,
			SourceVideoCodec:  "h264",
			SourceAudioCodec:  "aac",
		},
	)
	if len(profiles) < 2 {
		t.Fatal("expected remux profile plus software fallback")
	}
	assertReadRateBeforeInput(t, profiles[0].Args, "2.5")
	assertReadRateBeforeInput(t, profiles[len(profiles)-1].Args, "1.5")
	assertProgressPipeBeforeInput(t, profiles[len(profiles)-1].Args)
}

func assertNoReadRate(t *testing.T, args []string) {
	t.Helper()
	for _, arg := range args {
		if arg == "-readrate" {
			t.Fatalf("transcode profile must not cap input with -readrate, got %q", strings.Join(args, " "))
		}
	}
}

func assertProgressPipeBeforeInput(t *testing.T, args []string) {
	t.Helper()
	inputIndex := -1
	progressIndex := -1
	for idx, arg := range args {
		switch arg {
		case "-i":
			inputIndex = idx
		case "-progress":
			if progressIndex < 0 {
				progressIndex = idx
			}
		}
	}
	if inputIndex < 0 || progressIndex < 0 || progressIndex > inputIndex {
		t.Fatalf("expected -progress before input, got %q", strings.Join(args, " "))
	}
	if progressIndex+1 >= len(args) || args[progressIndex+1] != "pipe:1" {
		t.Fatalf("expected -progress pipe:1, got %q", strings.Join(args, " "))
	}
}

func assertReadRateBeforeInput(t *testing.T, args []string, wantRate string) {
	t.Helper()
	inputIndex := -1
	readRateIndex := -1
	for idx, arg := range args {
		switch arg {
		case "-i":
			inputIndex = idx
		case "-readrate":
			readRateIndex = idx
		}
	}
	if inputIndex < 0 || readRateIndex < 0 || readRateIndex > inputIndex {
		t.Fatalf("expected -readrate before input, got %q", strings.Join(args, " "))
	}
	if readRateIndex+1 >= len(args) || args[readRateIndex+1] != wantRate {
		t.Fatalf("expected -readrate %s, got %q", wantRate, strings.Join(args, " "))
	}
	progressIndex := -1
	for idx, arg := range args {
		if arg == "-progress" {
			progressIndex = idx
			break
		}
	}
	if progressIndex < 0 || progressIndex > inputIndex {
		t.Fatalf("expected -progress before input, got %q", strings.Join(args, " "))
	}
	if progressIndex+1 >= len(args) || args[progressIndex+1] != "pipe:1" {
		t.Fatalf("expected -progress pipe:1, got %q", strings.Join(args, " "))
	}
}

func TestBuildTranscodeProfilesUsesFastInputSeekWhenKeyframeProbeMisses(t *testing.T) {
	profiles := buildTranscodeProfiles(
		Config{HardwareDecode: false},
		"movie.mkv",
		"segment-%05d.m4s",
		"index.m3u8",
		buildProfileOptions{StartPositionSec: 3723.5},
	)
	if len(profiles) != 1 {
		t.Fatalf("expected only software fallback, got %d profiles", len(profiles))
	}
	args := profiles[0].Args
	seekIndices := make([]int, 0, 2)
	inputIndex := -1
	for idx, arg := range args {
		switch arg {
		case "-i":
			inputIndex = idx
		case "-ss":
			seekIndices = append(seekIndices, idx)
		}
	}
	if inputIndex < 0 || len(seekIndices) != 1 {
		t.Fatalf("expected a single fast input seek, got %q", strings.Join(args, " "))
	}
	if seekIndices[0] >= inputIndex {
		t.Fatalf("expected fast seek before input, got %q", strings.Join(args, " "))
	}
	if seekIndices[0]+1 >= len(args) || args[seekIndices[0]+1] != "3723.500" {
		t.Fatalf("expected input seek offset 3723.500 in ffmpeg args, got %q", strings.Join(args, " "))
	}
}

func TestBuildTranscodeProfilesUsesKeyframeAlignedFastSeekWhenProbed(t *testing.T) {
	keyframe := 3719.512
	profiles := buildTranscodeProfiles(
		Config{HardwareDecode: false},
		"movie.mkv",
		"segment-%05d.m4s",
		"index.m3u8",
		buildProfileOptions{
			StartPositionSec:     3723.5,
			TranscodeKeyframeSec: &keyframe,
		},
	)
	if len(profiles) != 1 {
		t.Fatalf("expected only software fallback, got %d profiles", len(profiles))
	}
	if profiles[0].TimelineOriginSec != keyframe {
		t.Fatalf("timeline origin = %v, want %v", profiles[0].TimelineOriginSec, keyframe)
	}
	args := profiles[0].Args
	inputIndex := -1
	seekIndex := -1
	for idx, arg := range args {
		switch arg {
		case "-i":
			inputIndex = idx
		case "-ss":
			seekIndex = idx
		}
	}
	if inputIndex < 0 || seekIndex < 0 || seekIndex > inputIndex {
		t.Fatalf("expected keyframe seek before input, got %q", strings.Join(args, " "))
	}
	if seekIndex+1 >= len(args) || args[seekIndex+1] != "3719.512" {
		t.Fatalf("expected keyframe seek offset 3719.512, got %q", strings.Join(args, " "))
	}
	if strings.Count(strings.Join(args, " "), "-ss") != 1 {
		t.Fatalf("transcode fast seek must not add output-side precise seek, got %q", strings.Join(args, " "))
	}
}

func TestBuildTranscodeProfilesKeepsHybridSeekForAvi(t *testing.T) {
	profiles := buildTranscodeProfiles(
		Config{HardwareDecode: false},
		"movie.avi",
		"segment-%05d.m4s",
		"index.m3u8",
		buildProfileOptions{
			StartPositionSec: 3723.5,
			SourceContainer:  "avi",
			SourcePath:       "movie.avi",
		},
	)
	if len(profiles) != 1 {
		t.Fatalf("expected only software fallback, got %d profiles", len(profiles))
	}
	args := profiles[0].Args
	seekIndices := make([]int, 0, 2)
	inputIndex := -1
	for idx, arg := range args {
		switch arg {
		case "-i":
			inputIndex = idx
		case "-ss":
			seekIndices = append(seekIndices, idx)
		}
	}
	if inputIndex < 0 || len(seekIndices) != 2 {
		t.Fatalf("expected hybrid input/output seek args for avi, got %q", strings.Join(args, " "))
	}
	if seekIndices[0] >= inputIndex || seekIndices[1] <= inputIndex {
		t.Fatalf("expected fast seek before input and precise seek after input, got %q", strings.Join(args, " "))
	}
	if seekIndices[0]+1 >= len(args) || args[seekIndices[0]+1] != "3721.500" {
		t.Fatalf("expected input seek offset 3721.500 in ffmpeg args, got %q", strings.Join(args, " "))
	}
	if seekIndices[1]+1 >= len(args) || args[seekIndices[1]+1] != "2.000" {
		t.Fatalf("expected accurate seek offset 2.000 in ffmpeg args, got %q", strings.Join(args, " "))
	}
	if profiles[0].TimelineOriginSec != 3723.5 {
		t.Fatalf("avi timeline origin = %v, want 3723.5", profiles[0].TimelineOriginSec)
	}
}

func TestBuildTranscodeProfilesPrefersRemuxBeforeFullTranscodeWhenEligible(t *testing.T) {
	zero := 0.0
	profiles := buildTranscodeProfiles(
		Config{HardwareDecode: false},
		"movie.mkv",
		"segment-%05d.m4s",
		"index.m3u8",
		buildProfileOptions{
			RemuxInputSeekSec: &zero,
			SourceVideoCodec:  "h264",
			SourceAudioCodec:  "aac",
		},
	)
	if len(profiles) < 2 {
		t.Fatalf("expected remux profile plus software fallback, got %d profiles", len(profiles))
	}
	if profiles[0].Name != "remux_copy" {
		t.Fatalf("first profile = %q, want remux_copy", profiles[0].Name)
	}
	args := strings.Join(profiles[0].Args, " ")
	if !strings.Contains(args, "-c:v copy") || !strings.Contains(args, "-c:a copy") {
		t.Fatalf("expected remux profile to copy both codecs, got %q", args)
	}
	if !strings.Contains(args, "-avoid_negative_ts make_zero") {
		t.Fatalf("expected remux profile to normalize copied timestamps, got %q", args)
	}
	if !strings.Contains(args, "-ignore_editlist 1") {
		t.Fatalf("expected remux profile to keep discarded priming IDR frames, got %q", args)
	}
	if !strings.Contains(args, "-hls_segment_type fmp4") || !strings.Contains(args, "init.mp4") {
		t.Fatalf("expected remux profile to emit fMP4 HLS, got %q", args)
	}
	if !strings.Contains(args, "-fflags +genpts") || !strings.Contains(args, "-muxdelay 0") {
		t.Fatalf("expected remux profile to rewrite timestamps for MSE, got %q", args)
	}
}

func TestBuildTranscodeProfilesUsesActualTimelineOriginForRemuxSessions(t *testing.T) {
	zero := 0.0
	profiles := buildTranscodeProfiles(
		Config{HardwareDecode: false},
		"movie.mkv",
		"segment-%05d.m4s",
		"index.m3u8",
		buildProfileOptions{
			StartPositionSec:  0,
			RemuxInputSeekSec: &zero,
			SourceVideoCodec:  "h264",
			SourceAudioCodec:  "aac",
		},
	)
	if len(profiles) < 1 {
		t.Fatal("expected at least one profile")
	}
	if profiles[0].Name != "remux_copy" {
		t.Fatalf("first profile = %q, want remux_copy", profiles[0].Name)
	}
	if profiles[0].TimelineOriginSec != 0 {
		t.Fatalf("timeline origin = %v, want 0", profiles[0].TimelineOriginSec)
	}
	args := strings.Join(profiles[0].Args, " ")
	if strings.Contains(args, "-ss") {
		t.Fatalf("start-at-zero remux must not add input seek args, got %q", args)
	}
}

func TestBuildTranscodeProfilesUsesKeyframeAlignedRemuxForMidStreamStarts(t *testing.T) {
	keyframe := 3719.512
	profiles := buildTranscodeProfiles(
		Config{HardwareDecode: false},
		"movie.mkv",
		"segment-%05d.m4s",
		"index.m3u8",
		buildProfileOptions{
			StartPositionSec:  3723.5,
			RemuxInputSeekSec: &keyframe,
			SourceVideoCodec:  "h264",
			SourceAudioCodec:  "aac",
		},
	)
	if len(profiles) < 2 {
		t.Fatalf("expected remux profile plus software fallback, got %d profiles", len(profiles))
	}
	if profiles[0].Name != "remux_copy" {
		t.Fatalf("first profile = %q, want remux_copy", profiles[0].Name)
	}
	if profiles[0].TimelineOriginSec != keyframe {
		t.Fatalf("timeline origin = %v, want %v", profiles[0].TimelineOriginSec, keyframe)
	}
	args := profiles[0].Args
	inputIndex := -1
	seekIndex := -1
	for idx, arg := range args {
		switch arg {
		case "-i":
			inputIndex = idx
		case "-ss":
			seekIndex = idx
		}
	}
	if inputIndex < 0 || seekIndex < 0 || seekIndex > inputIndex {
		t.Fatalf("expected keyframe seek before input, got %q", strings.Join(args, " "))
	}
	if seekIndex+1 >= len(args) || args[seekIndex+1] != "3719.512" {
		t.Fatalf("expected keyframe seek offset 3719.512, got %q", strings.Join(args, " "))
	}
	if strings.Count(strings.Join(args, " "), "-ss") != 1 {
		t.Fatalf("stream-copy remux must not add output-side precise seek, got %q", strings.Join(args, " "))
	}
}

func TestResolveRemuxInputSeekFallsBackWhenKeyframeProbeMisses(t *testing.T) {
	restore := probeKeyframeAtOrBeforeFunc
	probeKeyframeAtOrBeforeFunc = func(ctx context.Context, sourcePath string, ffmpegCommand string, targetSec float64, windowSec float64) (float64, bool) {
		return 0, false
	}
	defer func() { probeKeyframeAtOrBeforeFunc = restore }()

	if seek := resolveRemuxInputSeek(context.Background(), Config{}, "movie.mkv", StartHLSSessionOptions{
		StartPositionSec: 3723.5,
		PreferRemux:      true,
		SourceVideoCodec: "h264",
		SourceAudioCodec: "aac",
	}); seek != nil {
		t.Fatalf("expected no remux when keyframe probe misses, got %v", *seek)
	}

	zero := 0.0
	if seek := resolveRemuxInputSeek(context.Background(), Config{}, "movie.mkv", StartHLSSessionOptions{
		StartPositionSec: 0,
		PreferRemux:      true,
		SourceVideoCodec: "h264",
		SourceAudioCodec: "aac",
	}); seek == nil || *seek != zero {
		t.Fatal("expected start-at-zero remux without probing the source")
	}

	if seek := resolveRemuxInputSeek(context.Background(), Config{}, "movie.mkv", StartHLSSessionOptions{
		StartPositionSec: 3723.5,
		PreferRemux:      true,
		SourceVideoCodec: "h264",
		SourceAudioCodec: "flac",
	}); seek != nil {
		t.Fatal("expected no remux when codecs are not HLS-friendly")
	}
}

func TestResolveRemuxInputSeekUsesProbedKeyframe(t *testing.T) {
	restore := probeKeyframeAtOrBeforeFunc
	probeKeyframeAtOrBeforeFunc = func(ctx context.Context, sourcePath string, ffmpegCommand string, targetSec float64, windowSec float64) (float64, bool) {
		if targetSec != 3723.5 {
			t.Fatalf("probe target = %v, want 3723.5", targetSec)
		}
		return 3719.512, true
	}
	defer func() { probeKeyframeAtOrBeforeFunc = restore }()

	seek := resolveRemuxInputSeek(context.Background(), Config{}, "movie.mkv", StartHLSSessionOptions{
		StartPositionSec: 3723.5,
		PreferRemux:      true,
		SourceVideoCodec: "h264",
		SourceAudioCodec: "aac",
	})
	if seek == nil || *seek != 3719.512 {
		t.Fatalf("expected keyframe-aligned remux seek, got %v", seek)
	}
}

func TestBuildTranscodeProfilesKeepsRequestedTimelineOriginForTranscodeSessions(t *testing.T) {
	profiles := buildTranscodeProfiles(
		Config{HardwareDecode: false},
		"movie.mkv",
		"segment-%05d.m4s",
		"index.m3u8",
		buildProfileOptions{StartPositionSec: 3723.5},
	)
	if len(profiles) != 1 {
		t.Fatalf("expected only software fallback, got %d profiles", len(profiles))
	}
	if profiles[0].Name != "libx264" {
		t.Fatalf("profile name = %q, want libx264", profiles[0].Name)
	}
	if profiles[0].TimelineOriginSec != 3723.5 {
		t.Fatalf("timeline origin = %v, want 3723.5", profiles[0].TimelineOriginSec)
	}
}

func TestBuildTranscodeProfilesSkipsRemuxWhenSourceAudioIsNotHlsFriendly(t *testing.T) {
	// Codec eligibility now resolves in resolveRemuxInputSeek; an HLS-unfriendly
	// audio track yields a nil seek, which must suppress the remux profile.
	if seek := resolveRemuxInputSeek(context.Background(), Config{}, "movie.mkv", StartHLSSessionOptions{
		PreferRemux:      true,
		SourceVideoCodec: "h264",
		SourceAudioCodec: "dts",
	}); seek != nil {
		t.Fatalf("expected no remux seek for dts audio, got %v", *seek)
	}

	profiles := buildTranscodeProfiles(
		Config{HardwareDecode: false},
		"movie.mkv",
		"segment-%05d.m4s",
		"index.m3u8",
		buildProfileOptions{
			SourceVideoCodec: "h264",
			SourceAudioCodec: "dts",
		},
	)
	if len(profiles) == 0 {
		t.Fatal("expected at least one transcode fallback profile")
	}
	if profiles[0].Name == "remux_copy" {
		t.Fatalf("unexpected remux profile for dts audio: %+v", profiles[0])
	}
}

func TestTakeSessionsForMovieRemovesOnlyMatchingMovieSessions(t *testing.T) {
	t.Parallel()
	manager := New(Config{})
	t.Cleanup(manager.Close)
	manager.sessions["sess-a"] = &sessionState{session: Session{ID: "sess-a", MovieID: "movie-a"}}
	manager.sessions["sess-b"] = &sessionState{session: Session{ID: "sess-b", MovieID: "movie-a"}}
	manager.sessions["sess-c"] = &sessionState{session: Session{ID: "sess-c", MovieID: "movie-b"}}

	stale := manager.takeSessionsForMovie("movie-a")
	if len(stale) != 2 {
		t.Fatalf("stale session count = %d, want 2", len(stale))
	}
	if _, ok := manager.sessions["sess-c"]; !ok {
		t.Fatal("expected unrelated movie session to remain registered")
	}
	if _, ok := manager.sessions["sess-a"]; ok {
		t.Fatal("expected sess-a to be removed")
	}
	if _, ok := manager.sessions["sess-b"]; ok {
		t.Fatal("expected sess-b to be removed")
	}
}

func TestManagerCloseClearsAllSessions(t *testing.T) {
	t.Parallel()
	manager := New(Config{})
	t.Cleanup(manager.Close)
	manager.sessions["sess-a"] = &sessionState{session: Session{ID: "sess-a", MovieID: "movie-a"}}
	manager.sessions["sess-b"] = &sessionState{session: Session{ID: "sess-b", MovieID: "movie-b"}}

	manager.Close()

	if len(manager.sessions) != 0 {
		t.Fatalf("session count = %d, want 0", len(manager.sessions))
	}
}

func TestPreferHardwareTranscodeProfileKeepsRemuxFirst(t *testing.T) {
	t.Parallel()
	profiles := []transcodeProfile{
		{Name: "remux_copy", SessionKind: "remux-hls"},
		{Name: "h264_nvenc", SessionKind: "transcode-hls"},
		{Name: "h264_amf", SessionKind: "transcode-hls"},
	}
	got := preferHardwareTranscodeProfile(profiles, "h264_amf")
	if got[0].Name != "remux_copy" {
		t.Fatalf("first profile = %q, want remux_copy", got[0].Name)
	}
	if got[1].Name != "h264_amf" {
		t.Fatalf("first transcode profile = %q, want h264_amf", got[1].Name)
	}
}

func TestPreferHardwareTranscodeProfileIgnoresLibx264StickyPreference(t *testing.T) {
	t.Parallel()
	profiles := []transcodeProfile{
		{Name: "h264_amf", SessionKind: "transcode-hls"},
	}
	got := preferHardwareTranscodeProfile(profiles, "libx264")
	if got[0].Name != "h264_amf" {
		t.Fatalf("profile = %q, want h264_amf to stay ahead of software fallback", got[0].Name)
	}
}

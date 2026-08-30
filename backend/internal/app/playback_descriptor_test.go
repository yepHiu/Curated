package app

import (
	"testing"

	"curated-backend/internal/contracts"
	"curated-backend/internal/playback"
	"curated-backend/internal/storage"
)

func TestChoosePlaybackDurationSecPrefersProbedDuration(t *testing.T) {
	t.Parallel()

	got := choosePlaybackDurationSec(7234.5, 7200, 4)
	if got != 7234.5 {
		t.Fatalf("choosePlaybackDurationSec() = %v, want 7234.5", got)
	}
}

func TestChoosePlaybackDurationSecFallsBackToLargestSaneDuration(t *testing.T) {
	t.Parallel()

	got := choosePlaybackDurationSec(0, 7200, 4)
	if got != 7200 {
		t.Fatalf("choosePlaybackDurationSec() = %v, want 7200", got)
	}
}

func TestBuildDirectPlaybackDescriptorKeepsResolvedDurationAndClampsResume(t *testing.T) {
	t.Parallel()

	dto := buildDirectPlaybackDescriptor(
		"movie-1",
		contracts.MovieDetailDTO{
			MovieListItemDTO: contracts.MovieListItemDTO{
				Location: `D:\media\movie-1.mkv`,
			},
		},
		&storage.PlaybackProgressRow{
			PositionSec: 10800,
			DurationSec: 4,
		},
		7200,
		playbackDecision{},
	)

	if dto.DurationSec != 7200 {
		t.Fatalf("durationSec = %v, want 7200", dto.DurationSec)
	}
	if dto.ResumePositionSec != 7200 {
		t.Fatalf("resumePositionSec = %v, want 7200", dto.ResumePositionSec)
	}
	if dto.CanDirectPlay {
		t.Fatal("expected mkv direct playback to be marked as browser-unsafe")
	}
}

func TestBuildDirectPlaybackDescriptorMarksMP4AsDirectPlayable(t *testing.T) {
	t.Parallel()

	dto := buildDirectPlaybackDescriptor(
		"movie-2",
		contracts.MovieDetailDTO{
			MovieListItemDTO: contracts.MovieListItemDTO{
				Location: `D:\media\movie-2.mp4`,
			},
		},
		nil,
		3600,
		playbackDecision{},
	)

	if !dto.CanDirectPlay {
		t.Fatal("expected mp4 direct playback to remain browser-playable")
	}
	if dto.MimeType != "video/mp4" {
		t.Fatalf("mimeType = %q, want video/mp4", dto.MimeType)
	}
}

func TestBuildPlaybackDecisionPrefersDirectForBrowserSafeSource(t *testing.T) {
	t.Parallel()

	decision := buildPlaybackDecision(playbackDecisionInput{
		Location: "D:\\media\\movie-1.mp4",
		MediaInfo: playback.MediaInfo{
			Container:  "mp4",
			VideoCodec: "h264",
			AudioCodec: "aac",
		},
		StreamPushEnabled: true,
	})

	if decision.Mode != contracts.PlaybackModeDirect {
		t.Fatalf("mode = %q, want %q", decision.Mode, contracts.PlaybackModeDirect)
	}
	if decision.SessionKind != playbackSessionKindDirectFile {
		t.Fatalf("sessionKind = %q, want %q", decision.SessionKind, playbackSessionKindDirectFile)
	}
	if decision.PreferRemux {
		t.Fatal("direct playback must not request remux")
	}
	if decision.ReasonCode != "browser_direct_play_supported" {
		t.Fatalf("reasonCode = %q, want browser_direct_play_supported", decision.ReasonCode)
	}
}

func TestBuildPlaybackDecisionPrefersRemuxHLSForBrowserUnsafeContainer(t *testing.T) {
	t.Parallel()

	decision := buildPlaybackDecision(playbackDecisionInput{
		Location: "D:\\media\\movie-2.mkv",
		MediaInfo: playback.MediaInfo{
			Container:  "matroska",
			VideoCodec: "h264",
			AudioCodec: "aac",
		},
		StreamPushEnabled: true,
	})

	if decision.Mode != contracts.PlaybackModeHLS {
		t.Fatalf("mode = %q, want %q", decision.Mode, contracts.PlaybackModeHLS)
	}
	if decision.SessionKind != playbackSessionKindRemuxHLS {
		t.Fatalf("sessionKind = %q, want %q", decision.SessionKind, playbackSessionKindRemuxHLS)
	}
	if !decision.PreferRemux {
		t.Fatal("expected remux-first HLS plan for h264/aac source")
	}
	if decision.ReasonCode != "browser_container_unsupported" {
		t.Fatalf("reasonCode = %q, want browser_container_unsupported", decision.ReasonCode)
	}
}

func TestBuildPlaybackDecisionFallsBackToTranscodeHLSWhenAudioNeedsConversion(t *testing.T) {
	t.Parallel()

	decision := buildPlaybackDecision(playbackDecisionInput{
		Location: "D:\\media\\movie-3.mkv",
		MediaInfo: playback.MediaInfo{
			Container:  "matroska",
			VideoCodec: "h264",
			AudioCodec: "dts",
		},
		StreamPushEnabled: true,
	})

	if decision.Mode != contracts.PlaybackModeHLS {
		t.Fatalf("mode = %q, want %q", decision.Mode, contracts.PlaybackModeHLS)
	}
	if decision.SessionKind != playbackSessionKindTranscodeHLS {
		t.Fatalf("sessionKind = %q, want %q", decision.SessionKind, playbackSessionKindTranscodeHLS)
	}
	if decision.PreferRemux {
		t.Fatal("expected transcode fallback when audio codec is not HLS-friendly")
	}
	if decision.ReasonCode != "browser_container_unsupported" {
		t.Fatalf("reasonCode = %q, want browser_container_unsupported", decision.ReasonCode)
	}
}

func TestBuildPlaybackDecisionHonorsClientVideoCodecs(t *testing.T) {
	t.Parallel()

	base := func(clientVideoCodecs []string) playbackDecisionInput {
		return playbackDecisionInput{
			Location: "D:/media/movie-hevc.mp4",
			MediaInfo: playback.MediaInfo{
				Container:  "mp4",
				VideoCodec: "hevc",
				AudioCodec: "aac",
			},
			StreamPushEnabled: true,
			ClientVideoCodecs: clientVideoCodecs,
		}
	}

	// No capability report keeps the static whitelist (legacy behavior).
	if decision := buildPlaybackDecision(base(nil)); decision.Mode != contracts.PlaybackModeDirect {
		t.Fatalf("mode without client codecs = %q, want direct", decision.Mode)
	}

	// A browser that cannot decode HEVC must get the HLS plan instead of a
	// descriptor the video element will fail to decode.
	fallback := buildPlaybackDecision(base([]string{"h264"}))
	if fallback.Mode != contracts.PlaybackModeHLS {
		t.Fatalf("mode without hevc support = %q, want hls", fallback.Mode)
	}
	if fallback.ReasonCode != "browser_codec_unsupported" {
		t.Fatalf("reasonCode = %q, want browser_codec_unsupported", fallback.ReasonCode)
	}
	if fallback.SessionKind != playbackSessionKindTranscodeHLS {
		t.Fatalf("sessionKind = %q, want transcode-hls; hevc is never remux-eligible", fallback.SessionKind)
	}

	// A browser reporting HEVC keeps direct play.
	if decision := buildPlaybackDecision(base([]string{"h264", "hvc1"})); decision.Mode != contracts.PlaybackModeDirect {
		t.Fatalf("mode with hevc support = %q, want direct", decision.Mode)
	}

	// The capability report must not affect other containers.
	webm := buildPlaybackDecision(playbackDecisionInput{
		Location: "D:/media/movie.webm",
		MediaInfo: playback.MediaInfo{
			Container:  "webm",
			VideoCodec: "vp9",
			AudioCodec: "opus",
		},
		StreamPushEnabled: true,
		ClientVideoCodecs: []string{"h264"},
	})
	if webm.Mode != contracts.PlaybackModeDirect {
		t.Fatalf("webm mode = %q, want direct; client codecs only narrow the mp4 family", webm.Mode)
	}
}

func TestBuildPlaybackDecisionTreatsUnknownCodecAsUnsupportedWhenClientNarrows(t *testing.T) {
	t.Parallel()

	decision := buildPlaybackDecision(playbackDecisionInput{
		Location: "D:/media/movie-unknown.mp4",
		MediaInfo: playback.MediaInfo{
			Container:  "mp4",
			VideoCodec: "",
			AudioCodec: "aac",
		},
		StreamPushEnabled: true,
		ClientVideoCodecs: []string{"h264"},
	})

	if decision.Mode != contracts.PlaybackModeHLS {
		t.Fatalf("mode = %q, want hls for unknown codec under an explicit capability set", decision.Mode)
	}
}

func TestBuildPlaybackDecisionRemuxesUnstableTimestampsWhenStreamPushEnabled(t *testing.T) {
	t.Parallel()

	decision := buildPlaybackDecision(playbackDecisionInput{
		Location: "D:\\media\\movie-vfr.mp4",
		MediaInfo: playback.MediaInfo{
			Container:    "mp4",
			VideoCodec:   "h264",
			AudioCodec:   "aac",
			RFrameRate:   "60/1",
			AvgFrameRate: "24/1",
		},
		StreamPushEnabled: true,
	})

	if decision.Mode != contracts.PlaybackModeHLS {
		t.Fatalf("mode = %q, want %q", decision.Mode, contracts.PlaybackModeHLS)
	}
	if decision.SessionKind != playbackSessionKindRemuxHLS {
		t.Fatalf("sessionKind = %q, want %q", decision.SessionKind, playbackSessionKindRemuxHLS)
	}
	if !decision.PreferRemux {
		t.Fatal("expected remux-first HLS for h264/aac with unstable timestamps")
	}
	if decision.CanDirectPlay {
		t.Fatal("unstable timestamps must not stay on browser direct play")
	}
	if decision.ReasonCode != "source_timestamps_unstable" {
		t.Fatalf("reasonCode = %q, want source_timestamps_unstable", decision.ReasonCode)
	}
}

func TestBuildPlaybackDecisionKeepsDirectPlayWhenStreamPushDisabledDespiteUnstableTimestamps(t *testing.T) {
	t.Parallel()

	decision := buildPlaybackDecision(playbackDecisionInput{
		Location: "D:\\media\\movie-vfr.mp4",
		MediaInfo: playback.MediaInfo{
			Container:    "mp4",
			VideoCodec:   "h264",
			AudioCodec:   "aac",
			RFrameRate:   "60/1",
			AvgFrameRate: "24/1",
		},
		StreamPushEnabled: false,
	})

	if decision.Mode != contracts.PlaybackModeDirect {
		t.Fatalf("mode = %q, want %q", decision.Mode, contracts.PlaybackModeDirect)
	}
	if decision.SessionKind != playbackSessionKindDirectFile {
		t.Fatalf("sessionKind = %q, want %q", decision.SessionKind, playbackSessionKindDirectFile)
	}
	if decision.ReasonCode != "browser_direct_play_supported" {
		t.Fatalf("reasonCode = %q, want browser_direct_play_supported", decision.ReasonCode)
	}
}

func TestBuildPlaybackDecisionTranscodesUnstableTimestampsWhenRemuxIneligible(t *testing.T) {
	t.Parallel()

	decision := buildPlaybackDecision(playbackDecisionInput{
		Location: "D:\\media\\movie-hevc-vfr.mp4",
		MediaInfo: playback.MediaInfo{
			Container:    "mp4",
			VideoCodec:   "hevc",
			AudioCodec:   "aac",
			RFrameRate:   "60/1",
			AvgFrameRate: "24/1",
		},
		StreamPushEnabled: true,
	})

	if decision.Mode != contracts.PlaybackModeHLS {
		t.Fatalf("mode = %q, want %q", decision.Mode, contracts.PlaybackModeHLS)
	}
	if decision.SessionKind != playbackSessionKindTranscodeHLS {
		t.Fatalf("sessionKind = %q, want transcode-hls; hevc is not remux-eligible", decision.SessionKind)
	}
	if decision.PreferRemux {
		t.Fatal("hevc must not request remux")
	}
	if decision.ReasonCode != "source_timestamps_unstable" {
		t.Fatalf("reasonCode = %q, want source_timestamps_unstable", decision.ReasonCode)
	}
}

func TestBuildPlaybackDecisionKeepsDirectPlayForMatchingFrameRates(t *testing.T) {
	t.Parallel()

	decision := buildPlaybackDecision(playbackDecisionInput{
		Location: "D:\\media\\movie-cfr.mp4",
		MediaInfo: playback.MediaInfo{
			Container:    "mp4",
			VideoCodec:   "h264",
			AudioCodec:   "aac",
			RFrameRate:   "30000/1001",
			AvgFrameRate: "30/1",
		},
		StreamPushEnabled: true,
	})

	if decision.Mode != contracts.PlaybackModeDirect {
		t.Fatalf("mode = %q, want %q", decision.Mode, contracts.PlaybackModeDirect)
	}
	if decision.ReasonCode != "browser_direct_play_supported" {
		t.Fatalf("reasonCode = %q, want browser_direct_play_supported", decision.ReasonCode)
	}
}

func TestBuildPlaybackDecisionTranscodesNegativePrimingPTS(t *testing.T) {
	t.Parallel()

	decision := buildPlaybackDecision(playbackDecisionInput{
		Location: "D:\\test_curated\\DVDMS-981\\DVDMS-981.mp4",
		MediaInfo: playback.MediaInfo{
			Container:           "mp4",
			VideoCodec:          "h264",
			AudioCodec:          "aac",
			RFrameRate:          "30000/1001",
			AvgFrameRate:        "1283454521/42825027",
			HasNegativeVideoPTS: true,
		},
		StreamPushEnabled: true,
	})

	if decision.Mode != contracts.PlaybackModeHLS {
		t.Fatalf("mode = %q, want %q", decision.Mode, contracts.PlaybackModeHLS)
	}
	if decision.SessionKind != playbackSessionKindTranscodeHLS {
		t.Fatalf("sessionKind = %q, want transcode-hls for negative priming PTS", decision.SessionKind)
	}
	if decision.PreferRemux {
		t.Fatal("negative priming PTS must not stay on stream-copy remux")
	}
	if decision.ReasonCode != "source_timestamps_unstable" {
		t.Fatalf("reasonCode = %q, want source_timestamps_unstable", decision.ReasonCode)
	}
}

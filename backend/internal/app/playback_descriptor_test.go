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

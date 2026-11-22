package app

import (
	"curated-backend/internal/playback"
	"testing"
)

func TestNativeDecisionPreservesForceAndTimestampProtection(t *testing.T) {
	for _, tc := range []struct {
		name                                string
		supported, force, broken, transcode bool
		kind                                string
	}{
		{"native mkv", true, false, false, false, "direct-file"},
		{"force push", true, true, false, false, "remux-hls"},
		{"unknown decoder", false, false, false, false, "remux-hls"},
		{"decode failure", false, false, false, true, "transcode-hls"},
		{"negative pts", true, false, true, false, "transcode-hls"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := buildPlaybackDecision(playbackDecisionInput{Location: "source.mkv", StreamPushEnabled: true, ForceStreamPush: tc.force, NativeDirect: &tc.supported, ForceTranscode: tc.transcode, MediaInfo: playback.MediaInfo{Container: "matroska", VideoCodec: "h264", AudioCodec: "aac", HasNegativeVideoPTS: tc.broken}})
			if got.SessionKind != tc.kind {
				t.Fatalf("got %+v", got)
			}
		})
	}
	legacy := buildPlaybackDecision(playbackDecisionInput{Location: "source.mkv", StreamPushEnabled: true, MediaInfo: playback.MediaInfo{Container: "matroska", VideoCodec: "h264", AudioCodec: "aac"}})
	if legacy.SessionKind != "remux-hls" {
		t.Fatalf("legacy changed: %+v", legacy)
	}
}

func TestNativeLargeMP4IndexUsesOnlyEligibleRemux(t *testing.T) {
	supported := true
	base := playbackDecisionInput{Location: "source.mp4", NativeDirect: &supported, StreamPushEnabled: true,
		MediaInfo: playback.MediaInfo{Container: "mov", VideoCodec: "h264", AudioCodec: "aac", MP4InitializationBytes: 16650985}}
	got := buildPlaybackDecision(base)
	if got.SessionKind != "remux-hls" || !got.PreferRemux || got.ReasonCode != "native_large_mp4_index" {
		t.Fatalf("unexpected decision: %+v", got)
	}
	for _, tc := range []struct {
		name   string
		change func(*playbackDecisionInput)
		want   string
	}{
		{"small", func(i *playbackDecisionInput) { i.MediaInfo.MP4InitializationBytes = 4 << 20 }, "direct-file"},
		{"unknown", func(i *playbackDecisionInput) { i.MediaInfo.MP4InitializationBytes = 0 }, "direct-file"},
		{"legacy", func(i *playbackDecisionInput) { i.NativeDirect = nil }, "direct-file"},
		{"disabled", func(i *playbackDecisionInput) { i.StreamPushEnabled = false }, "direct-file"},
		{"hevc", func(i *playbackDecisionInput) { i.MediaInfo.VideoCodec = "hevc" }, "direct-file"},
		{"negative pts", func(i *playbackDecisionInput) { i.MediaInfo.HasNegativeVideoPTS = true }, "transcode-hls"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := base
			tc.change(&input)
			if result := buildPlaybackDecision(input); result.SessionKind != tc.want {
				t.Fatalf("got %+v", result)
			}
		})
	}
}

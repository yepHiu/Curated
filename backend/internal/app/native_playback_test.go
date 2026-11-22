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

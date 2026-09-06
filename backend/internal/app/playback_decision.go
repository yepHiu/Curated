package app

import (
	"path/filepath"
	"strings"

	"curated-backend/internal/contracts"
	"curated-backend/internal/playback"
)

const (
	playbackSessionKindDirectFile   = "direct-file"
	playbackSessionKindRemuxHLS     = "remux-hls"
	playbackSessionKindTranscodeHLS = "transcode-hls"
)

type playbackDecisionInput struct {
	Location          string
	MediaInfo         playback.MediaInfo
	StreamPushEnabled bool
	ForceStreamPush   bool
	// ClientVideoCodecs carries the browser-reported decodable mp4-family video
	// codecs from the `clientVideoCodecs` descriptor query parameter. Nil keeps
	// the static whitelist behavior for callers that report nothing.
	ClientVideoCodecs []string
}

type playbackDecision struct {
	Mode             contracts.PlaybackMode
	SessionKind      string
	ReasonCode       string
	ReasonMessage    string
	PreferRemux      bool
	CanDirectPlay    bool
	SourceContainer  string
	SourceVideoCodec string
	SourceAudioCodec string
}

func buildPlaybackDecision(input playbackDecisionInput) playbackDecision {
	container := normalizeSourceContainer(input.MediaInfo.Container, input.Location)
	videoCodec := normalizeCodecName(input.MediaInfo.VideoCodec)
	audioCodec := normalizeCodecName(input.MediaInfo.AudioCodec)
	directEligible := isBrowserDirectPlayCandidate(input.Location, container, videoCodec, audioCodec, input.ClientVideoCodecs)
	remuxEligible := canRemuxToHLS(videoCodec, audioCodec)
	// Discarded priming IDRs / negative PTS still jitter after stream-copy remux
	// because B-frames keep the original cadence. Re-encode those to CFR.
	if remuxEligible && input.MediaInfo.HasNegativeVideoPTS {
		remuxEligible = false
	}

	decision := playbackDecision{
		Mode:             contracts.PlaybackModeDirect,
		SessionKind:      playbackSessionKindDirectFile,
		ReasonCode:       "browser_direct_play_supported",
		ReasonMessage:    "Source can stay on direct browser playback.",
		CanDirectPlay:    directEligible,
		SourceContainer:  container,
		SourceVideoCodec: videoCodec,
		SourceAudioCodec: audioCodec,
	}

	if input.StreamPushEnabled && input.ForceStreamPush && strings.TrimSpace(input.Location) != "" {
		decision.Mode = contracts.PlaybackModeHLS
		decision.SessionKind = chooseHLSSessionKind(remuxEligible)
		decision.ReasonCode = "force_stream_push"
		decision.ReasonMessage = "Stream push is forced by player settings."
		decision.PreferRemux = remuxEligible
		decision.CanDirectPlay = false
		return decision
	}

	if directEligible {
		if input.StreamPushEnabled && input.MediaInfo.TimestampsUnstable() {
			decision.Mode = contracts.PlaybackModeHLS
			decision.SessionKind = chooseHLSSessionKind(remuxEligible)
			decision.PreferRemux = remuxEligible
			decision.CanDirectPlay = false
			decision.ReasonCode = "source_timestamps_unstable"
			decision.ReasonMessage = "Source frame timestamps look unstable for browser direct play, so HLS is preferred."
			if input.MediaInfo.HasNegativeVideoPTS {
				decision.ReasonMessage = "Source has negative priming timestamps; HLS transcode is used to keep a stable frame cadence."
			}
			return decision
		}
		return decision
	}

	if !input.StreamPushEnabled {
		decision.ReasonCode = "browser_direct_play_unavailable"
		decision.ReasonMessage = "Direct playback is the only available path because stream push is disabled."
		return decision
	}

	decision.Mode = contracts.PlaybackModeHLS
	decision.SessionKind = chooseHLSSessionKind(remuxEligible)
	decision.PreferRemux = remuxEligible
	decision.CanDirectPlay = false
	if browserContainerSupportedByExtension(input.Location) {
		decision.ReasonCode = "browser_codec_unsupported"
		decision.ReasonMessage = "Container is browser-friendly, but source codecs still need an HLS fallback."
		return decision
	}
	decision.ReasonCode = "browser_container_unsupported"
	decision.ReasonMessage = "Source container is not safe for direct browser playback, so HLS is required."
	return decision
}

func chooseHLSSessionKind(preferRemux bool) string {
	if preferRemux {
		return playbackSessionKindRemuxHLS
	}
	return playbackSessionKindTranscodeHLS
}

func normalizeSourceContainer(raw string, location string) string {
	text := strings.ToLower(strings.TrimSpace(raw))
	if text != "" {
		parts := strings.Split(text, ",")
		if len(parts) > 0 {
			text = strings.TrimSpace(parts[0])
		}
	}
	if text != "" {
		return text
	}
	return strings.TrimPrefix(strings.ToLower(filepath.Ext(strings.TrimSpace(location))), ".")
}

func normalizeCodecName(raw string) string {
	text := strings.ToLower(strings.TrimSpace(raw))
	switch text {
	case "avc1":
		return "h264"
	case "hev1", "hvc1":
		return "hevc"
	case "mp4a":
		return "aac"
	default:
		return text
	}
}

func browserContainerSupportedByExtension(location string) bool {
	switch strings.ToLower(filepath.Ext(strings.TrimSpace(location))) {
	case ".mp4", ".m4v", ".webm", ".ogv", ".m3u8":
		return true
	default:
		return false
	}
}

func isBrowserDirectPlayCandidate(location string, container string, videoCodec string, audioCodec string, clientVideoCodecs []string) bool {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(location)))
	switch ext {
	case ".mp4", ".m4v":
		return mp4FamilyDirectPlayCandidate(videoCodec, audioCodec, clientVideoCodecs)
	case ".webm":
		return isCodecEmptyOrOneOf(videoCodec, "vp8", "vp9", "av1") &&
			isCodecEmptyOrOneOf(audioCodec, "opus", "vorbis")
	case ".ogv":
		return isCodecEmptyOrOneOf(videoCodec, "theora") &&
			isCodecEmptyOrOneOf(audioCodec, "vorbis", "opus")
	}

	switch container {
	case "mp4", "mov":
		return mp4FamilyDirectPlayCandidate(videoCodec, audioCodec, clientVideoCodecs)
	case "webm":
		return isCodecEmptyOrOneOf(videoCodec, "vp8", "vp9", "av1") &&
			isCodecEmptyOrOneOf(audioCodec, "opus", "vorbis")
	case "ogg":
		return isCodecEmptyOrOneOf(videoCodec, "theora") &&
			isCodecEmptyOrOneOf(audioCodec, "vorbis", "opus")
	default:
		return false
	}
}

// mp4FamilyDirectPlayCandidate applies the static mp4 whitelist plus the
// browser-reported codec capability. HEVC and AV1 support varies across
// browsers and hardware, so a reported capability set overrides the whitelist.
func mp4FamilyDirectPlayCandidate(videoCodec string, audioCodec string, clientVideoCodecs []string) bool {
	if !(isCodecEmptyOrOneOf(videoCodec, "h264", "hevc", "av1") &&
		isCodecEmptyOrOneOf(audioCodec, "aac", "mp3")) {
		return false
	}
	if len(clientVideoCodecs) == 0 {
		return true
	}
	normalized := normalizeCodecName(videoCodec)
	if normalized == "" {
		// The client narrowed capability to an explicit set; an unknown codec
		// must prefer the HLS fallback over a likely decode error.
		return false
	}
	for _, candidate := range clientVideoCodecs {
		if normalizeCodecName(candidate) == normalized {
			return true
		}
	}
	return false
}

func canRemuxToHLS(videoCodec string, audioCodec string) bool {
	return isCodecOneOf(videoCodec, "h264") &&
		isCodecOneOf(audioCodec, "aac", "mp3", "ac3", "eac3")
}

func isCodecEmptyOrOneOf(codec string, allowed ...string) bool {
	if strings.TrimSpace(codec) == "" {
		return true
	}
	return isCodecOneOf(codec, allowed...)
}

func isCodecOneOf(codec string, allowed ...string) bool {
	normalized := normalizeCodecName(codec)
	for _, candidate := range allowed {
		if normalized == candidate {
			return true
		}
	}
	return false
}

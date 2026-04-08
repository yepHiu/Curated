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
	directEligible := isBrowserDirectPlayCandidate(input.Location, container, videoCodec, audioCodec)
	remuxEligible := canRemuxToHLS(videoCodec, audioCodec)

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

func isBrowserDirectPlayCandidate(location string, container string, videoCodec string, audioCodec string) bool {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(location)))
	switch ext {
	case ".mp4", ".m4v":
		return isCodecEmptyOrOneOf(videoCodec, "h264", "hevc", "av1") &&
			isCodecEmptyOrOneOf(audioCodec, "aac", "mp3")
	case ".webm":
		return isCodecEmptyOrOneOf(videoCodec, "vp8", "vp9", "av1") &&
			isCodecEmptyOrOneOf(audioCodec, "opus", "vorbis")
	case ".ogv":
		return isCodecEmptyOrOneOf(videoCodec, "theora") &&
			isCodecEmptyOrOneOf(audioCodec, "vorbis", "opus")
	}

	switch container {
	case "mp4", "mov":
		return isCodecEmptyOrOneOf(videoCodec, "h264", "hevc", "av1") &&
			isCodecEmptyOrOneOf(audioCodec, "aac", "mp3")
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

package playback

import (
	"regexp"
	"strconv"
	"strings"
)

const (
	hlsSegmentDurationSec = 2.0
	throttlePauseLeadSec  = 90.0
	throttleResumeLeadSec = 12.0
)

type throttleAction int

const (
	throttleNone throttleAction = iota
	throttlePause
	throttleResume
)

var hlsSegmentIndexPattern = regexp.MustCompile(`(?i)^segment-(\d+)\.(m4s|ts)$`)

// Stream-copy GOPs need not be two seconds. Use the published EXTINF durations
// rather than multiplying the filename index by the target segment duration.
func mediaTimeFromPlaylist(playlist, name string) (float64, bool) {
	if !hlsSegmentIndexPattern.MatchString(name) {
		return 0, false
	}
	position, duration := 0.0, 0.0
	for _, raw := range strings.Split(playlist, "\n") {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "#EXTINF:") {
			value, _, _ := strings.Cut(strings.TrimPrefix(line, "#EXTINF:"), ",")
			duration, _ = strconv.ParseFloat(value, 64)
		} else if line != "" && !strings.HasPrefix(line, "#") {
			if duration <= 0 || !isFiniteFloat(duration) {
				return 0, false
			}
			position += duration
			if line == name {
				return position, true
			}
			duration = 0
		}
	}
	return 0, false
}

func nextThrottleAction(paused bool, writtenSec, requestedSec, pauseLeadSec, resumeLeadSec float64) throttleAction {
	lead := writtenSec - requestedSec
	if !paused && lead > pauseLeadSec {
		return throttlePause
	}
	if paused && lead < resumeLeadSec {
		return throttleResume
	}
	return throttleNone
}

func mediaTimeFromHLSFileName(name string, segmentDurationSec float64) (float64, bool) {
	if segmentDurationSec <= 0 {
		segmentDurationSec = hlsSegmentDurationSec
	}
	match := hlsSegmentIndexPattern.FindStringSubmatch(strings.TrimSpace(name))
	if len(match) < 2 {
		return 0, false
	}
	index, err := strconv.Atoi(match[1])
	if err != nil || index < 0 {
		return 0, false
	}
	return float64(index) * segmentDurationSec, true
}

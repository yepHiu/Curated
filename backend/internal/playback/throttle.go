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

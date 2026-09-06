package playback

import (
	"math"
	"strconv"
	"strings"
)

// timestampUnstableRelativeDiff is the minimum |r-avg|/max(r,avg) that treats
// a source as unsafe for browser progressive MP4 demux. NTSC 30000/1001 vs 30/1
// is ~0.1% and must stay on direct play.
const timestampUnstableRelativeDiff = 0.02

// TimestampsUnstable reports whether ffprobe frame rates look hostile to
// browser direct play (VFR or a broken avg_frame_rate). Empty rates mean "not
// probed" and do not divert playback.
func (m MediaInfo) TimestampsUnstable() bool {
	if m.HasNegativeVideoPTS {
		return true
	}
	r, rOK := parseFrameRate(m.RFrameRate)
	avg, avgOK := parseFrameRate(m.AvgFrameRate)
	if rOK && avgOK {
		den := math.Max(r, avg)
		if den <= 0 {
			return false
		}
		return math.Abs(r-avg)/den > timestampUnstableRelativeDiff
	}
	return rOK && isExplicitlyUnknownFrameRate(m.AvgFrameRate)
}

func parseFrameRate(raw string) (float64, bool) {
	text := strings.TrimSpace(raw)
	if text == "" || isExplicitlyUnknownFrameRate(text) {
		return 0, false
	}
	if strings.Contains(text, "/") {
		parts := strings.SplitN(text, "/", 2)
		num, numErr := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		den, denErr := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if numErr != nil || denErr != nil || den == 0 || num <= 0 || math.IsInf(num/den, 0) {
			return 0, false
		}
		return num / den, true
	}
	value, err := strconv.ParseFloat(text, 64)
	if err != nil || value <= 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, false
	}
	return value, true
}

func isExplicitlyUnknownFrameRate(raw string) bool {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "0/0", "N/A", "NAN":
		return true
	default:
		return false
	}
}

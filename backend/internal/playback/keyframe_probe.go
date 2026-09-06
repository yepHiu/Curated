package playback

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"curated-backend/internal/executil"
)

// keyframeProbeWindowSec bounds how far before the requested start we scan for a
// video keyframe. Remux uses that keyframe for stream-copy; transcode uses it
// for a fast input seek. Sources whose GOP is longer than this window skip remux
// and transcode from the requested timestamp instead.
const keyframeProbeWindowSec = 30.0

// probeKeyframeAtOrBeforeFunc is overridable so tests can exercise keyframe
// alignment without a real ffprobe binary.
var probeKeyframeAtOrBeforeFunc = probeKeyframeAtOrBefore

type ffprobePacketList struct {
	Packets []struct {
		PTSTime string `json:"pts_time"`
		Flags   string `json:"flags"`
	} `json:"packets"`
}

// probeKeyframeAtOrBefore returns the timestamp of the last video keyframe at or
// before targetSec, so a stream-copy HLS session can start aligned to a real
// keyframe instead of re-encoding the whole file.
func probeKeyframeAtOrBefore(ctx context.Context, sourcePath string, ffmpegCommand string, targetSec float64, windowSec float64) (float64, bool) {
	if targetSec <= 0 {
		return 0, targetSec == 0
	}
	if windowSec <= 0 {
		windowSec = keyframeProbeWindowSec
	}
	windowStart := targetSec - windowSec
	if windowStart < 0 {
		windowStart = 0
	}
	// ffprobe read_intervals are half-open; extend the end slightly so a
	// keyframe exactly at the requested target still counts as eligible.
	interval := fmt.Sprintf("%.3f%%%.3f", windowStart, targetSec+0.001)

	for _, candidate := range ffprobeCommandCandidates(ffmpegCommand) {
		probeCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
		cmd := executil.CommandContext(
			probeCtx,
			candidate,
			"-v", "error",
			"-select_streams", "v:0",
			"-show_entries", "packet=pts_time,flags",
			"-of", "json",
			"-read_intervals", interval,
			sourcePath,
		)
		output, err := cmd.Output()
		cancel()
		if err != nil {
			continue
		}

		var raw ffprobePacketList
		if err := json.Unmarshal(output, &raw); err != nil {
			continue
		}

		best := -1.0
		for _, packet := range raw.Packets {
			if !strings.Contains(packet.Flags, "K") {
				continue
			}
			pts, err := strconv.ParseFloat(strings.TrimSpace(packet.PTSTime), 64)
			if err != nil || pts < 0 || pts > targetSec {
				continue
			}
			if pts > best {
				best = pts
			}
		}
		if best >= 0 {
			return best, true
		}
	}
	return 0, false
}

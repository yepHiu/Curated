package playback

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const (
	sessionSeekKindStart = "start"
	sessionSeekKindSwap  = "swap"
)

type ffmpegProgress struct {
	Speed      float64
	OutTimeSec float64
	HasSpeed   bool
	HasOutTime bool
}

func sessionSeekKindForOrigin(originSec float64) string {
	if originSec > 0.001 {
		return sessionSeekKindSwap
	}
	return sessionSeekKindStart
}

func formatEncoderSpeed(speed float64) string {
	if speed <= 0 || !isFiniteFloat(speed) {
		return ""
	}
	return fmt.Sprintf("%.2fx", speed)
}

func isFiniteFloat(value float64) bool {
	return value == value && value < 1e300 && value > -1e300
}

func parseFFmpegProgressBlock(block string) ffmpegProgress {
	var parsed ffmpegProgress
	for _, rawLine := range strings.Split(block, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "progress=") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(key) {
		case "speed":
			speed, ok := parseFFmpegSpeed(value)
			if !ok {
				continue
			}
			parsed.Speed = speed
			parsed.HasSpeed = true
		case "out_time_us":
			micros, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
			if err != nil || micros < 0 {
				continue
			}
			parsed.OutTimeSec = micros / 1_000_000
			parsed.HasOutTime = true
		case "out_time_ms":
			if parsed.HasOutTime {
				continue
			}
			millis, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
			if err != nil || millis < 0 {
				continue
			}
			parsed.OutTimeSec = millis / 1_000
			parsed.HasOutTime = true
		}
	}
	return parsed
}

func parseFFmpegSpeed(raw string) (float64, bool) {
	text := strings.TrimSpace(strings.ToLower(raw))
	text = strings.TrimSuffix(text, "x")
	if text == "" || text == "n/a" || text == "nan" {
		return 0, false
	}
	speed, err := strconv.ParseFloat(text, 64)
	if err != nil || speed < 0 || !isFiniteFloat(speed) {
		return 0, false
	}
	return speed, true
}

func consumeFFmpegProgress(r io.Reader, onUpdate func(ffmpegProgress)) {
	if r == nil || onUpdate == nil {
		return
	}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 1024), 64*1024)
	var block strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		if block.Len() > 0 {
			block.WriteByte('\n')
		}
		block.WriteString(line)
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "progress=") {
			continue
		}
		parsed := parseFFmpegProgressBlock(block.String())
		block.Reset()
		if parsed.HasSpeed || parsed.HasOutTime {
			onUpdate(parsed)
		}
	}
}

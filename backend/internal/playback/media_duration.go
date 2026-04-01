package playback

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

type mediaDurationCacheKey struct {
	path        string
	size        int64
	modTimeUnix int64
}

var mediaDurationCache sync.Map

var ffmpegDurationPattern = regexp.MustCompile(`Duration:\s*(\d{2}):(\d{2}):(\d{2}(?:\.\d+)?)`)

func ProbeMediaDuration(ctx context.Context, sourcePath string, ffmpegCommand string) (float64, error) {
	cleanPath := strings.TrimSpace(sourcePath)
	if cleanPath == "" {
		return 0, fmt.Errorf("media duration probe source path is empty")
	}

	info, err := os.Stat(cleanPath)
	if err != nil {
		return 0, err
	}

	key := mediaDurationCacheKey{
		path:        filepath.Clean(cleanPath),
		size:        info.Size(),
		modTimeUnix: info.ModTime().UnixNano(),
	}
	if cached, ok := mediaDurationCache.Load(key); ok {
		if duration, ok := cached.(float64); ok && duration > 0 {
			return duration, nil
		}
	}

	duration, err := probeDurationViaFFprobe(ctx, cleanPath, ffmpegCommand)
	if err != nil {
		duration, err = probeDurationViaFFmpeg(ctx, cleanPath, ffmpegCommand)
		if err != nil {
			return 0, err
		}
	}

	if duration <= 0 {
		return 0, fmt.Errorf("media duration probe returned non-positive duration")
	}

	mediaDurationCache.Store(key, duration)
	return duration, nil
}

func probeDurationViaFFprobe(ctx context.Context, sourcePath string, ffmpegCommand string) (float64, error) {
	var lastErr error
	for _, candidate := range ffprobeCommandCandidates(ffmpegCommand) {
		cmd := exec.CommandContext(
			ctx,
			candidate,
			"-v", "error",
			"-show_entries", "format=duration",
			"-of", "default=noprint_wrappers=1:nokey=1",
			sourcePath,
		)
		output, err := cmd.Output()
		if err != nil {
			lastErr = err
			continue
		}
		duration, err := parseNumericDuration(string(output))
		if err != nil {
			lastErr = err
			continue
		}
		return duration, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("ffprobe command not available")
	}
	return 0, lastErr
}

func probeDurationViaFFmpeg(ctx context.Context, sourcePath string, ffmpegCommand string) (float64, error) {
	cmd := exec.CommandContext(ctx, resolveFFmpegCommand(ffmpegCommand), "-i", sourcePath)
	cmd.Stdout = io.Discard
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	_ = cmd.Run()

	match := ffmpegDurationPattern.FindStringSubmatch(stderr.String())
	if len(match) != 4 {
		text := strings.TrimSpace(stderr.String())
		if text == "" {
			text = "duration probe output did not contain a Duration field"
		}
		return 0, fmt.Errorf("%s", text)
	}

	hours, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, err
	}
	minutes, err := strconv.Atoi(match[2])
	if err != nil {
		return 0, err
	}
	seconds, err := strconv.ParseFloat(match[3], 64)
	if err != nil {
		return 0, err
	}
	return float64(hours*3600+minutes*60) + seconds, nil
}

func parseNumericDuration(raw string) (float64, error) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return 0, fmt.Errorf("empty duration output")
	}
	duration, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0, err
	}
	if duration <= 0 {
		return 0, fmt.Errorf("duration output %q is not positive", text)
	}
	return duration, nil
}

func ffprobeCommandCandidates(ffmpegCommand string) []string {
	candidates := make([]string, 0, 3)
	seen := map[string]struct{}{}
	add := func(candidate string) {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			return
		}
		if _, ok := seen[candidate]; ok {
			return
		}
		seen[candidate] = struct{}{}
		candidates = append(candidates, candidate)
	}

	ffmpegResolved := resolveFFmpegCommand(ffmpegCommand)
	if strings.Contains(ffmpegResolved, "/") || strings.Contains(ffmpegResolved, `\`) {
		dir := filepath.Dir(ffmpegResolved)
		ext := filepath.Ext(ffmpegResolved)
		name := strings.TrimSuffix(filepath.Base(ffmpegResolved), ext)
		if strings.EqualFold(name, "ffmpeg") {
			add(filepath.Join(dir, "ffprobe"+ext))
		}
	}

	if isDefaultFFmpegCommand(strings.TrimSpace(ffmpegCommand)) {
		if bundled, ok := findBundledFFmpegCommand(); ok {
			dir := filepath.Dir(bundled)
			ext := filepath.Ext(bundled)
			add(filepath.Join(dir, "ffprobe"+ext))
		}
	}

	add(defaultFFprobeBinaryName())
	return candidates
}

func defaultFFprobeBinaryName() string {
	if runtime.GOOS == "windows" {
		return "ffprobe.exe"
	}
	return "ffprobe"
}

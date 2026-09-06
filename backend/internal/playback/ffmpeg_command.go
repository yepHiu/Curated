package playback

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func resolveFFmpegCommand(configured string) string {
	cmd := strings.TrimSpace(configured)
	if isDefaultFFmpegCommand(cmd) {
		if bundled, ok := findBundledFFmpegCommand(); ok {
			return bundled
		}
		if cmd != "" {
			return resolveFFmpegExecutable(cmd)
		}
		return resolveFFmpegExecutable(defaultFFmpegBinaryName())
	}
	if cmd == "" {
		return defaultFFmpegBinaryName()
	}
	return resolveFFmpegExecutable(cmd)
}

// Scoop launches a forwarding process. Suspend/kill must address FFmpeg itself,
// otherwise its child continues encoding after the shim has stopped.
func resolveFFmpegExecutable(command string) string {
	path, err := exec.LookPath(command)
	if err != nil || runtime.GOOS != "windows" {
		return command
	}
	raw, err := os.ReadFile(strings.TrimSuffix(path, filepath.Ext(path)) + ".shim")
	if err != nil {
		return path
	}
	for _, line := range strings.Split(string(raw), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) != "path" {
			continue
		}
		target := strings.Trim(strings.TrimSpace(value), "\"")
		if filepath.IsAbs(target) && strings.EqualFold(filepath.Base(target), "ffmpeg.exe") {
			if info, err := os.Stat(target); err == nil && !info.IsDir() {
				return target
			}
		}
	}
	return path
}

// ResolveFFmpegCommand exposes the configured/bundled command resolver to other
// backend features that execute bounded FFmpeg jobs (for example GIF clips).
func ResolveFFmpegCommand(configured string) string {
	return resolveFFmpegCommand(configured)
}

func findBundledFFmpegCommand() (string, bool) {
	for _, candidate := range bundledFFmpegCandidates() {
		if candidate == "" {
			continue
		}
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() {
			continue
		}
		return candidate, true
	}
	return "", false
}

func bundledFFmpegCandidates() []string {
	name := defaultFFmpegBinaryName()
	candidates := make([]string, 0, 4)
	seen := map[string]struct{}{}
	add := func(base string) {
		base = strings.TrimSpace(base)
		if base == "" {
			return
		}
		candidate := filepath.Clean(filepath.Join(base, "third_party", "ffmpeg", "bin", name))
		if _, ok := seen[candidate]; ok {
			return
		}
		seen[candidate] = struct{}{}
		candidates = append(candidates, candidate)
	}
	if exe, err := os.Executable(); err == nil {
		add(filepath.Dir(exe))
	}
	if cwd, err := os.Getwd(); err == nil {
		add(cwd)
		add(filepath.Join(cwd, "backend"))
	}
	return candidates
}

func isDefaultFFmpegCommand(cmd string) bool {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return true
	}
	if strings.Contains(cmd, "/") || strings.Contains(cmd, `\`) {
		return false
	}
	if strings.EqualFold(cmd, "ffmpeg") {
		return true
	}
	return runtime.GOOS == "windows" && strings.EqualFold(cmd, "ffmpeg.exe")
}

func defaultFFmpegBinaryName() string {
	if runtime.GOOS == "windows" {
		return "ffmpeg.exe"
	}
	return "ffmpeg"
}

package nativeplayer

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

type Config struct {
	Enabled bool
	Preset  string
	Command string
	Args    []string
}

type Launcher struct {
	cfg Config
	mu  sync.RWMutex
}

func New(cfg Config) *Launcher {
	return &Launcher{cfg: cfg}
}

func (l *Launcher) Enabled() bool {
	if l == nil {
		return false
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.cfg.Enabled
}

func (l *Launcher) Command() string {
	if l == nil {
		return ""
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	cmd := strings.TrimSpace(l.cfg.Command)
	if cmd != "" {
		return cmd
	}
	return DefaultCommandForPreset(l.cfg.Preset)
}

func (l *Launcher) SetConfig(cfg Config) {
	if l == nil {
		return
	}
	l.mu.Lock()
	l.cfg = cfg
	l.mu.Unlock()
}

func (l *Launcher) Launch(ctx context.Context, mediaTarget string, startPositionSec float64, title string) error {
	if l == nil {
		return fmt.Errorf("native player is disabled")
	}
	l.mu.RLock()
	cfg := l.cfg
	l.mu.RUnlock()
	if !cfg.Enabled {
		return fmt.Errorf("native player is disabled")
	}
	preset := NormalizePreset(cfg.Preset, cfg.Command)
	cmdName := strings.TrimSpace(cfg.Command)
	if cmdName == "" {
		cmdName = DefaultCommandForPreset(preset)
	}
	args := make([]string, 0, len(cfg.Args)+5)
	args = append(args, cfg.Args...)
	args = append(args, buildPresetArgs(preset, startPositionSec, title)...)
	args = append(args, mediaTarget)

	cmd := exec.CommandContext(ctx, cmdName, args...)
	if runtime.GOOS == "windows" && filepath.IsAbs(mediaTarget) {
		cmd.Dir = filepath.Dir(mediaTarget)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

func NormalizePreset(value string, command string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "mpv":
		return "mpv"
	case "potplayer":
		return "potplayer"
	case "custom":
		return "custom"
	}

	cmdLower := strings.ToLower(filepath.Base(strings.TrimSpace(command)))
	switch {
	case strings.Contains(cmdLower, "potplayer"):
		return "potplayer"
	case cmdLower == "", strings.Contains(cmdLower, "mpv"):
		return "mpv"
	default:
		return "custom"
	}
}

func DefaultCommandForPreset(preset string) string {
	switch NormalizePreset(preset, "") {
	case "potplayer":
		return "PotPlayerMini64.exe"
	default:
		return "mpv"
	}
}

func buildPresetArgs(preset string, startPositionSec float64, title string) []string {
	switch NormalizePreset(preset, "") {
	case "potplayer":
		return buildPotPlayerArgs(startPositionSec)
	case "custom":
		return nil
	default:
		return buildMPVArgs(startPositionSec, title)
	}
}

func buildMPVArgs(startPositionSec float64, title string) []string {
	args := make([]string, 0, 2)
	if startPositionSec > 0 {
		args = append(args, fmt.Sprintf("--start=%.3f", startPositionSec))
	}
	if strings.TrimSpace(title) != "" {
		args = append(args, "--force-media-title="+title)
	}
	return args
}

func buildPotPlayerArgs(startPositionSec float64) []string {
	if startPositionSec <= 0 {
		return nil
	}
	return []string{"/seek=" + formatPotPlayerSeek(startPositionSec)}
}

func formatPotPlayerSeek(startPositionSec float64) string {
	totalMillis := int64(startPositionSec * 1000)
	if totalMillis < 0 {
		totalMillis = 0
	}
	hours := totalMillis / 3_600_000
	minutes := (totalMillis % 3_600_000) / 60_000
	seconds := (totalMillis % 60_000) / 1000
	millis := totalMillis % 1000
	return fmt.Sprintf("%02d:%02d:%02d.%03d", hours, minutes, seconds, millis)
}

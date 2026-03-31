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
	return strings.TrimSpace(l.cfg.Command)
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
	cmdName := strings.TrimSpace(cfg.Command)
	if cmdName == "" {
		return fmt.Errorf("native player command is empty")
	}
	args := make([]string, 0, len(cfg.Args)+4)
	args = append(args, cfg.Args...)
	if startPositionSec > 0 {
		args = append(args, fmt.Sprintf("--start=%.3f", startPositionSec))
	}
	if strings.TrimSpace(title) != "" {
		args = append(args, "--force-media-title="+title)
	}
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

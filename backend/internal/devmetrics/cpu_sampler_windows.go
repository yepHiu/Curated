//go:build windows

package devmetrics

import (
	"context"
	"math"
	"os"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/process"
)

type windowsCPUSampler struct {
	process *process.Process
}

func NewCPUSampler() CPUSampler {
	proc, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return unsupportedCPUSampler{}
	}

	// Prime gopsutil's delta-based counters so the first externally visible
	// snapshot can reflect the period after backend startup.
	_, _ = cpu.Percent(0, false)
	_, _ = proc.CPUPercentWithContext(context.Background())

	return &windowsCPUSampler{process: proc}
}

func (s *windowsCPUSampler) Snapshot(ctx context.Context) CPUSnapshot {
	systemSamples, err := cpu.Percent(0, false)
	if err != nil || len(systemSamples) == 0 {
		return CPUSnapshot{Supported: false}
	}

	backendCPUPercent := 0.0
	if s != nil && s.process != nil {
		if procPercent, err := s.process.CPUPercentWithContext(ctx); err == nil {
			backendCPUPercent = normalizeProcessCPUPercent(procPercent)
		}
	}

	return CPUSnapshot{
		Supported:         true,
		SampledAt:         time.Now().UTC().Format(time.RFC3339),
		SystemCPUPercent:  clampCPUPercent(systemSamples[0]),
		BackendCPUPercent: backendCPUPercent,
	}
}

func normalizeProcessCPUPercent(value float64) float64 {
	if value <= 0 {
		return 0
	}
	if value > 100 {
		cores := runtime.NumCPU()
		if cores > 0 {
			value = value / float64(cores)
		}
	}
	return clampCPUPercent(value)
}

func clampCPUPercent(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return math.Round(value*100) / 100
}

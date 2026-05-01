// Package devmetrics collects runtime metrics for the development performance monitor.
package devmetrics

import "context"

// CPUSnapshot holds a point-in-time CPU usage reading for the dev performance bar.
type CPUSnapshot struct {
	Supported         bool
	SampledAt         string
	SystemCPUPercent  float64
	BackendCPUPercent float64
}

// CPUSampler provides CPU usage snapshots for the dev performance bar.
type CPUSampler interface {
	Snapshot(ctx context.Context) CPUSnapshot
}

type unsupportedCPUSampler struct{}

func (unsupportedCPUSampler) Snapshot(ctx context.Context) CPUSnapshot {
	_ = ctx
	return CPUSnapshot{Supported: false}
}

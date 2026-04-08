package devmetrics

import "context"

type CPUSnapshot struct {
	Supported         bool
	SampledAt         string
	SystemCPUPercent  float64
	BackendCPUPercent float64
}

type CPUSampler interface {
	Snapshot(ctx context.Context) CPUSnapshot
}

type unsupportedCPUSampler struct{}

func (unsupportedCPUSampler) Snapshot(ctx context.Context) CPUSnapshot {
	_ = ctx
	return CPUSnapshot{Supported: false}
}

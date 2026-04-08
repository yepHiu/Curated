package app

import (
	"context"
	"testing"

	"curated-backend/internal/devmetrics"
)

type stubCPUSampler struct {
	snapshot devmetrics.CPUSnapshot
}

func (s stubCPUSampler) Snapshot(ctx context.Context) devmetrics.CPUSnapshot {
	_ = ctx
	return s.snapshot
}

func TestGetDevPerformanceSummaryMapsSamplerSnapshot(t *testing.T) {
	t.Parallel()

	app := &App{
		devCPUSampler: stubCPUSampler{
			snapshot: devmetrics.CPUSnapshot{
				Supported:         true,
				SampledAt:         "2026-04-09T00:00:00Z",
				SystemCPUPercent:  21.5,
				BackendCPUPercent: 8.75,
			},
		},
	}

	dto := app.GetDevPerformanceSummary(context.Background())
	if !dto.Supported {
		t.Fatal("expected supported summary")
	}
	if dto.SampledAt != "2026-04-09T00:00:00Z" {
		t.Fatalf("sampledAt = %q, want RFC3339 time", dto.SampledAt)
	}
	if dto.SystemCPUPercent != 21.5 {
		t.Fatalf("system cpu = %v, want 21.5", dto.SystemCPUPercent)
	}
	if dto.BackendCPUPercent != 8.75 {
		t.Fatalf("backend cpu = %v, want 8.75", dto.BackendCPUPercent)
	}
}

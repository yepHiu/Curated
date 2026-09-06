package storage

import (
	"context"
	"path/filepath"
	"testing"
)

func TestMediaProbeCacheRoundTrip(t *testing.T) {
	t.Parallel()
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "movie.mkv")
	if cached, err := store.GetMediaProbeCache(ctx, path); err != nil {
		t.Fatalf("get empty cache failed: %v", err)
	} else if cached != nil {
		t.Fatalf("expected no cached row, got %+v", cached)
	}

	row := MediaProbeCacheRow{
		Path:         path,
		SizeBytes:    4096,
		MtimeUnixNs:  1723800000000000000,
		Container:    "matroska,webm",
		VideoCodec:   "h264",
		AudioCodec:   "aac",
		DurationSec:  7182.5,
		RFrameRate:          "24000/1001",
		AvgFrameRate:        "24000/1001",
		HasNegativeVideoPTS: 1,
		ProbeSchema:         MediaProbeCacheSchema,
	}
	if err := store.UpsertMediaProbeCache(ctx, row); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}

	cached, err := store.GetMediaProbeCache(ctx, path)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if cached == nil {
		t.Fatal("expected cached row")
	}
	if *cached != row {
		t.Fatalf("cached row = %+v, want %+v", *cached, row)
	}

	row.VideoCodec = "hevc"
	row.SizeBytes = 8192
	if err := store.UpsertMediaProbeCache(ctx, row); err != nil {
		t.Fatalf("second upsert failed: %v", err)
	}
	cached, err = store.GetMediaProbeCache(ctx, path)
	if err != nil {
		t.Fatalf("get after upsert failed: %v", err)
	}
	if cached == nil || cached.VideoCodec != "hevc" || cached.SizeBytes != 8192 {
		t.Fatalf("expected upsert to replace the row, got %+v", cached)
	}
}

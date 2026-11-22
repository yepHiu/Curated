package app

import (
	"context"
	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/playback"
	"curated-backend/internal/storage"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestNativePlaybackSourceBindingWithRealMedia(t *testing.T) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	path := filepath.Join(t.TempDir(), "ABC-001.mkv")
	cmd := exec.CommandContext(ctx, ffmpeg, "-v", "error", "-f", "lavfi", "-i", "testsrc=size=128x96:rate=24", "-f", "lavfi", "-i", "sine=frequency=440:sample_rate=48000", "-t", "2", "-c:v", "libx264", "-pix_fmt", "yuv420p", "-bf", "0", "-c:a", "aac", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("fixture: %v %s", err, out)
	}
	store, err := storage.NewSQLiteStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	movie, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{TaskID: "test", Path: path, FileName: "ABC-001.mkv", Number: "ABC-001"})
	if err != nil {
		t.Fatal(err)
	}
	a := &App{store: store, cfg: config.Config{Player: config.PlayerConfig{FFmpegCommand: ffmpeg}}}
	plan, err := a.PlanNativePlayback(ctx, movie.MovieID)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Source.VideoCodec != "h264" || plan.Source.Width != 128 || plan.Source.AudioSampleRate != 48000 || plan.Source.VideoStreams != 1 || plan.FileID == "" {
		t.Fatalf("bad source: %+v", plan)
	}
	selection := contracts.NativePlaybackSelection{Version: 1, SourceToken: plan.Source.Token, DirectPlay: true}
	dto, err := a.ResolveNativePlayback(ctx, movie.MovieID, selection, nil)
	if err != nil {
		t.Fatal(err)
	}
	if dto.Mode != "direct" || dto.SessionID != "" || dto.FileID != plan.FileID {
		t.Fatalf("unexpected descriptor: %+v", dto)
	}
	selection.DirectPlay = false
	if _, err := a.ResolveNativePlayback(ctx, movie.MovieID, selection, nil); err == nil {
		t.Fatal("unsupported source must not silently direct-play without HLS")
	}
	streams := playback.New(playback.Config{Enabled: true, SessionRoot: t.TempDir(), FFmpegCommand: ffmpeg})
	defer streams.Close()
	a.streams = streams
	selection.ForceTranscode = true
	hls, err := a.ResolveNativePlayback(ctx, movie.MovieID, selection, nil)
	if err != nil {
		t.Fatal(err)
	}
	if hls.Mode != "hls" || hls.SessionKind != "transcode-hls" {
		t.Fatalf("unexpected fallback: %+v", hls)
	}
	if err := a.DeletePlaybackSession(hls.SessionID); err != nil {
		t.Fatal(err)
	}
	selection.DirectPlay, selection.ForceTranscode = true, false
	if err := os.Chtimes(path, time.Now(), time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := a.ResolveNativePlayback(ctx, movie.MovieID, selection, nil); err == nil {
		t.Fatal("stale source token accepted")
	}
}

package app

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"curated-backend/internal/contracts"
	"curated-backend/internal/playback"
)

type nativePlaybackKey struct{}
type nativePlaybackChoice struct {
	info           playback.MediaInfo
	direct         bool
	forceTranscode bool
}

// Native playback queries the exact source before starting any encoder. Detailed
// probes reuse playback's size/mtime memory cache; legacy persisted probes remain
// unchanged and cannot falsely claim that new decoder fields are known.
func (a *App) nativePlaybackSource(ctx context.Context, movieID string) (contracts.NativePlaybackPlanDTO, playback.MediaInfo, error) {
	fileStarted := time.Now()
	detail, err := a.store.GetMoviePlaybackDetail(ctx, movieID)
	if err != nil {
		return contracts.NativePlaybackPlanDTO{}, playback.MediaInfo{}, err
	}
	before, err := os.Stat(detail.Location)
	if err != nil {
		return contracts.NativePlaybackPlanDTO{}, playback.MediaInfo{}, err
	}
	contracts.RecordPlaybackTiming(ctx, "file", fileStarted)
	probeStarted := time.Now()
	info, err := playback.ProbeMediaInfo(ctx, detail.Location, a.cfg.Player.FFmpegCommand)
	contracts.RecordPlaybackTiming(ctx, "probe", probeStarted)
	if err != nil {
		return contracts.NativePlaybackPlanDTO{}, info, err
	}
	container := normalizeSourceContainer(info.Container, detail.Location)
	if container == "mov" || container == "mp4" {
		info.MP4InitializationBytes = playback.MP4InitializationBytes(detail.Location)
	}
	after, err := os.Stat(detail.Location)
	if err != nil {
		return contracts.NativePlaybackPlanDTO{}, info, err
	}
	if before.Size() != after.Size() || before.ModTime() != after.ModTime() {
		return contracts.NativePlaybackPlanDTO{}, info, fmt.Errorf("source changed during playback planning")
	}
	encoded, _ := json.Marshal(struct {
		FileID   string
		Size     int64
		Modified int64
		Info     playback.MediaInfo
	}{
		detail.SelectedFileID, after.Size(), after.ModTime().UnixNano(), info,
	})
	digest := sha256.Sum256(encoded)
	fps := nativeFrameRate(info.AvgFrameRate)
	if fps == 0 {
		fps = nativeFrameRate(info.RFrameRate)
	}
	return contracts.NativePlaybackPlanDTO{FileID: detail.SelectedFileID, Source: contracts.NativePlaybackSource{
		Token: fmt.Sprintf("%x", digest), Container: normalizeSourceContainer(info.Container, detail.Location),
		VideoCodec: info.VideoCodec, VideoProfile: info.VideoProfile, VideoLevel: info.VideoLevel,
		PixelFormat: info.PixelFormat, Width: info.Width, Height: info.Height, FrameRate: fps, BitRate: info.BitRate,
		AudioCodec: info.AudioCodec, AudioProfile: info.AudioProfile, AudioChannels: info.AudioChannels,
		AudioSampleRate: info.AudioSampleRate, VideoStreams: info.VideoStreams, AudioStreams: info.AudioStreams,
	}}, info, nil
}

func nativeFrameRate(raw string) float64 {
	parts := strings.Split(raw, "/")
	n, _ := strconv.ParseFloat(parts[0], 64)
	if len(parts) == 2 {
		d, _ := strconv.ParseFloat(parts[1], 64)
		if d <= 0 {
			return 0
		}
		n /= d
	}
	if math.IsNaN(n) || math.IsInf(n, 0) || n <= 0 || n > 1000 {
		return 0
	}
	return n
}

func (a *App) PlanNativePlayback(ctx context.Context, movieID string) (contracts.NativePlaybackPlanDTO, error) {
	plan, _, err := a.nativePlaybackSource(ctx, movieID)
	return plan, err
}

func (a *App) ResolveNativePlayback(ctx context.Context, movieID string, selection contracts.NativePlaybackSelection, start *float64) (contracts.PlaybackDescriptorDTO, error) {
	if selection.Version != 1 {
		return contracts.PlaybackDescriptorDTO{}, fmt.Errorf("unsupported native playback version")
	}
	plan, info, err := a.nativePlaybackSource(ctx, movieID)
	if err != nil {
		return contracts.PlaybackDescriptorDTO{}, err
	}
	if selection.SourceToken != plan.Source.Token {
		return contracts.PlaybackDescriptorDTO{}, fmt.Errorf("source changed; request a new playback plan")
	}
	// Pin the resolved file so a default-file change between requests cannot
	// redirect playback or its resume position to another part.
	ctx = contracts.WithMovieFileSelection(ctx, plan.FileID)
	ctx = context.WithValue(ctx, nativePlaybackKey{}, nativePlaybackChoice{info, selection.DirectPlay, selection.ForceTranscode})
	return a.resolvePlayback(ctx, movieID, nil, start, false)
}

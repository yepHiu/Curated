package playback

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"curated-backend/internal/executil"
)

// MediaInfo is the lightweight probe result used by playback planning.
// It intentionally keeps only the fields needed to choose direct play, remux, or transcode.
type MediaInfo struct {
	Container           string
	VideoCodec          string
	AudioCodec          string
	DurationSec         float64
	RFrameRate          string
	AvgFrameRate        string
	HasNegativeVideoPTS bool
}

type mediaInfoCacheKey struct {
	path        string
	size        int64
	modTimeUnix int64
}

type ffprobeMediaInfo struct {
	Streams []struct {
		CodecType    string `json:"codec_type"`
		CodecName    string `json:"codec_name"`
		RFrameRate   string `json:"r_frame_rate"`
		AvgFrameRate string `json:"avg_frame_rate"`
	} `json:"streams"`
	Packets []struct {
		CodecType string `json:"codec_type"`
		PtsTime   string `json:"pts_time"`
	} `json:"packets"`
	Format struct {
		FormatName string `json:"format_name"`
		Duration   string `json:"duration"`
	} `json:"format"`
}

var mediaInfoCache sync.Map

// ProbeMediaInfo returns a lightweight MediaInfo snapshot using a size+modtime cache and ffprobe.
func ProbeMediaInfo(ctx context.Context, sourcePath string, ffmpegCommand string) (MediaInfo, error) {
	cleanPath := strings.TrimSpace(sourcePath)
	if cleanPath == "" {
		return MediaInfo{}, fmt.Errorf("media probe source path is empty")
	}

	info, err := os.Stat(cleanPath)
	if err != nil {
		return MediaInfo{}, err
	}

	key := mediaInfoCacheKey{
		path:        filepath.Clean(cleanPath),
		size:        info.Size(),
		modTimeUnix: info.ModTime().UnixNano(),
	}
	if cached, ok := mediaInfoCache.Load(key); ok {
		if mediaInfo, ok := cached.(MediaInfo); ok {
			return mediaInfo, nil
		}
	}

	probed, err := probeMediaInfoViaFFprobe(ctx, cleanPath, ffmpegCommand)
	if err != nil {
		return MediaInfo{}, err
	}

	mediaInfoCache.Store(key, probed)
	return probed, nil
}

func probeMediaInfoViaFFprobe(ctx context.Context, sourcePath string, ffmpegCommand string) (MediaInfo, error) {
	var lastErr error
	for _, candidate := range ffprobeCommandCandidates(ffmpegCommand) {
		cmd := executil.CommandContext(
			ctx,
			candidate,
			"-v", "error",
			"-read_intervals", "%+#16",
			"-show_entries", "format=format_name,duration:stream=codec_type,codec_name,r_frame_rate,avg_frame_rate:packet=codec_type,pts_time",
			"-of", "json",
			sourcePath,
		)
		output, err := cmd.Output()
		if err != nil {
			lastErr = err
			continue
		}

		var raw ffprobeMediaInfo
		if err := json.Unmarshal(output, &raw); err != nil {
			lastErr = err
			continue
		}

		mediaInfo := mediaInfoFromFFprobe(raw)
		if mediaInfo.Container == "" && mediaInfo.VideoCodec == "" && mediaInfo.AudioCodec == "" && mediaInfo.DurationSec <= 0 {
			lastErr = fmt.Errorf("ffprobe returned an empty media probe result")
			continue
		}
		return mediaInfo, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("ffprobe command not available")
	}
	return MediaInfo{}, lastErr
}

func mediaInfoFromFFprobe(raw ffprobeMediaInfo) MediaInfo {
	mediaInfo := MediaInfo{
		Container: strings.TrimSpace(raw.Format.FormatName),
	}
	if raw.Format.Duration != "" {
		duration, err := parseNumericDuration(raw.Format.Duration)
		if err == nil {
			mediaInfo.DurationSec = duration
		}
	}
	for _, stream := range raw.Streams {
		switch strings.ToLower(strings.TrimSpace(stream.CodecType)) {
		case "video":
			if mediaInfo.VideoCodec == "" {
				mediaInfo.VideoCodec = strings.TrimSpace(stream.CodecName)
				mediaInfo.RFrameRate = strings.TrimSpace(stream.RFrameRate)
				mediaInfo.AvgFrameRate = strings.TrimSpace(stream.AvgFrameRate)
			}
		case "audio":
			if mediaInfo.AudioCodec == "" {
				mediaInfo.AudioCodec = strings.TrimSpace(stream.CodecName)
			}
		}
	}
	mediaInfo.HasNegativeVideoPTS = videoPacketsHaveNegativePTS(raw.Packets)
	return mediaInfo
}

func videoPacketsHaveNegativePTS(packets []struct {
	CodecType string `json:"codec_type"`
	PtsTime   string `json:"pts_time"`
}) bool {
	const minNegativeSec = -0.0005
	for _, packet := range packets {
		if strings.ToLower(strings.TrimSpace(packet.CodecType)) != "video" {
			continue
		}
		pts, err := strconv.ParseFloat(strings.TrimSpace(packet.PtsTime), 64)
		if err != nil || math.IsNaN(pts) || math.IsInf(pts, 0) {
			continue
		}
		if pts < minNegativeSec {
			return true
		}
	}
	return false
}

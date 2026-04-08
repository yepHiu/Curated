package playback

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// MediaInfo is the lightweight probe result used by playback planning.
// It intentionally keeps only the fields needed to choose direct play, remux, or transcode.
type MediaInfo struct {
	Container   string
	VideoCodec  string
	AudioCodec  string
	DurationSec float64
}

type mediaInfoCacheKey struct {
	path        string
	size        int64
	modTimeUnix int64
}

type ffprobeMediaInfo struct {
	Streams []struct {
		CodecType string `json:"codec_type"`
		CodecName string `json:"codec_name"`
	} `json:"streams"`
	Format struct {
		FormatName string `json:"format_name"`
		Duration   string `json:"duration"`
	} `json:"format"`
}

var mediaInfoCache sync.Map

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
		cmd := exec.CommandContext(
			ctx,
			candidate,
			"-v", "error",
			"-show_entries", "format=format_name,duration:stream=codec_type,codec_name",
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
				}
			case "audio":
				if mediaInfo.AudioCodec == "" {
					mediaInfo.AudioCodec = strings.TrimSpace(stream.CodecName)
				}
			}
		}
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

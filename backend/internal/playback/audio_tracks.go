package playback

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"curated-backend/internal/contracts"
	"curated-backend/internal/executil"
)

// ProbeAudioTracks reports audio-relative indexes used by FFmpeg's 0:a:N selector.
func ProbeAudioTracks(ctx context.Context, source, ffmpeg string) ([]contracts.SubtitleAudioTrack, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	for _, candidate := range ffprobeCommandCandidates(ffmpeg) {
		output, e := executil.CommandContext(ctx, candidate, "-v", "error", "-select_streams", "a", "-show_entries", "stream=codec_name,channels:stream_tags=language,title", "-of", "json", source).Output()
		if e != nil {
			continue
		}
		var raw struct {
			Streams []struct {
				Codec    string `json:"codec_name"`
				Channels int    `json:"channels"`
				Tags     struct {
					Language string `json:"language"`
					Title    string `json:"title"`
				} `json:"tags"`
			} `json:"streams"`
		}
		if e = json.Unmarshal(output, &raw); e != nil {
			return nil, e
		}
		result := []contracts.SubtitleAudioTrack{}
		for i, t := range raw.Streams {
			result = append(result, contracts.SubtitleAudioTrack{Index: i, Language: t.Tags.Language, Title: t.Tags.Title, Codec: t.Codec, Channels: t.Channels})
		}
		return result, nil
	}
	return nil, fmt.Errorf("could not read audio tracks")
}

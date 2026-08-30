package playback

import (
	"encoding/json"
	"testing"
)

func TestMediaInfoFromFFprobeCapturesVideoFrameRates(t *testing.T) {
	t.Parallel()

	rawJSON := []byte(`{
		"streams": [
			{"codec_type":"audio","codec_name":"aac"},
			{"codec_type":"video","codec_name":"h264","r_frame_rate":"60/1","avg_frame_rate":"24/1"}
		],
		"format": {"format_name":"mov,mp4,m4a,3gp,3g2,mj2","duration":"120.5"}
	}`)
	var raw ffprobeMediaInfo
	if err := json.Unmarshal(rawJSON, &raw); err != nil {
		t.Fatal(err)
	}

	got := mediaInfoFromFFprobe(raw)
	if got.Container != "mov,mp4,m4a,3gp,3g2,mj2" {
		t.Fatalf("container = %q", got.Container)
	}
	if got.VideoCodec != "h264" || got.AudioCodec != "aac" {
		t.Fatalf("codecs = %q/%q", got.VideoCodec, got.AudioCodec)
	}
	if got.DurationSec != 120.5 {
		t.Fatalf("duration = %v", got.DurationSec)
	}
	if got.RFrameRate != "60/1" || got.AvgFrameRate != "24/1" {
		t.Fatalf("frame rates = %q / %q", got.RFrameRate, got.AvgFrameRate)
	}
	if !got.TimestampsUnstable() {
		t.Fatal("expected captured rates to be treated as unstable")
	}
}

func TestMediaInfoFromFFprobeDetectsNegativeVideoPTS(t *testing.T) {
	t.Parallel()

	rawJSON := []byte(`{
		"streams": [
			{"codec_type":"video","codec_name":"h264","r_frame_rate":"30000/1001","avg_frame_rate":"1283454521/42825027"},
			{"codec_type":"audio","codec_name":"aac"}
		],
		"packets": [
			{"codec_type":"video","pts_time":"-0.033367"},
			{"codec_type":"video","pts_time":"0.000000"}
		],
		"format": {"format_name":"mov,mp4,m4a,3gp,3g2,mj2","duration":"9702.08"}
	}`)
	var raw ffprobeMediaInfo
	if err := json.Unmarshal(rawJSON, &raw); err != nil {
		t.Fatal(err)
	}

	got := mediaInfoFromFFprobe(raw)
	if !got.HasNegativeVideoPTS {
		t.Fatal("expected negative video PTS to be detected")
	}
	if !got.TimestampsUnstable() {
		t.Fatal("negative priming PTS must divert even when average fps matches 29.97")
	}
}

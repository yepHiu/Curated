package playback

import (
	"strings"
	"testing"
)

func TestParseFFmpegProgressBlockReadsSpeedAndOutTime(t *testing.T) {
	t.Parallel()

	parsed := parseFFmpegProgressBlock(strings.Join([]string{
		"frame=12",
		"fps=24.00",
		"out_time_us=2500000",
		"speed=1.24x",
		"progress=continue",
	}, "\n"))
	if !parsed.HasSpeed || parsed.Speed < 1.239 || parsed.Speed > 1.241 {
		t.Fatalf("speed = %#v, want 1.24", parsed)
	}
	if !parsed.HasOutTime || parsed.OutTimeSec < 2.499 || parsed.OutTimeSec > 2.501 {
		t.Fatalf("out time = %#v, want 2.5s", parsed)
	}
}

func TestParseFFmpegProgressBlockIgnoresMissingSpeed(t *testing.T) {
	t.Parallel()

	parsed := parseFFmpegProgressBlock("out_time_us=1000000\nspeed=N/A\nprogress=continue\n")
	if parsed.HasSpeed {
		t.Fatalf("expected no speed, got %#v", parsed)
	}
	if !parsed.HasOutTime || parsed.OutTimeSec != 1 {
		t.Fatalf("out time = %#v, want 1s", parsed)
	}
}

func TestConsumeFFmpegProgressEmitsCompleteBlocks(t *testing.T) {
	t.Parallel()

	var updates []ffmpegProgress
	consumeFFmpegProgress(strings.NewReader(strings.Join([]string{
		"out_time_us=500000",
		"speed=0.80x",
		"progress=continue",
		"out_time_us=1500000",
		"speed=1.10x",
		"progress=end",
	}, "\n")+"\n"), func(parsed ffmpegProgress) {
		updates = append(updates, parsed)
	})
	if len(updates) != 2 {
		t.Fatalf("update count = %d, want 2", len(updates))
	}
	if updates[0].Speed < 0.79 || updates[0].Speed > 0.81 || updates[0].OutTimeSec != 0.5 {
		t.Fatalf("first update = %#v", updates[0])
	}
	if updates[1].Speed < 1.09 || updates[1].Speed > 1.11 || updates[1].OutTimeSec != 1.5 {
		t.Fatalf("second update = %#v", updates[1])
	}
}

func TestSessionSeekKindAndSpeedFormatting(t *testing.T) {
	t.Parallel()

	if got := sessionSeekKindForOrigin(0); got != sessionSeekKindStart {
		t.Fatalf("origin 0 kind = %q, want %s", got, sessionSeekKindStart)
	}
	if got := sessionSeekKindForOrigin(12.5); got != sessionSeekKindSwap {
		t.Fatalf("mid-stream kind = %q, want %s", got, sessionSeekKindSwap)
	}
	if got := formatEncoderSpeed(1.246); got != "1.25x" {
		t.Fatalf("speed label = %q, want 1.25x", got)
	}
	if got := formatEncoderSpeed(0); got != "" {
		t.Fatalf("zero speed label = %q, want empty", got)
	}
}

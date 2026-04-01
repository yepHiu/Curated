package app

import (
	"testing"

	"curated-backend/internal/config"
	"curated-backend/internal/playback"
)

func TestShouldPreferHLSRespectsForceStreamPush(t *testing.T) {
	t.Parallel()

	app := &App{
		cfg: config.Config{
			Player: config.PlayerConfig{
				StreamPushEnabled: true,
				ForceStreamPush:   true,
			},
		},
		streams: playback.New(playback.Config{Enabled: true}),
	}

	if !app.shouldPreferHLS("C:\\media\\movie.mp4") {
		t.Fatal("expected forceStreamPush to route mp4 playback through HLS")
	}
}

func TestShouldPreferHLSKeepsDirectPlayForMP4ByDefault(t *testing.T) {
	t.Parallel()

	app := &App{
		cfg: config.Config{
			Player: config.PlayerConfig{
				StreamPushEnabled: true,
				ForceStreamPush:   false,
			},
		},
		streams: playback.New(playback.Config{Enabled: true}),
	}

	if app.shouldPreferHLS("C:\\media\\movie.mp4") {
		t.Fatal("expected mp4 playback to remain direct when forceStreamPush is disabled")
	}
}

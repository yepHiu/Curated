package app

import (
	"os"
	"path/filepath"
	"testing"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
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

func TestSetPlayerSettingsPatch_EnablingStreamPushDoesNotEnableForceStreamPush(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "library-config.cfg")
	if err := os.WriteFile(path, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := &App{
		cfg: config.Config{
			Player: config.PlayerConfig{
				StreamPushEnabled:   false,
				ForceStreamPush:     false,
				SeekForwardStepSec:  10,
				SeekBackwardStepSec: 10,
			},
		},
		librarySettingsPath: path,
	}

	on := true
	if err := a.SetPlayerSettingsPatch(contracts.PatchPlayerSettingsDTO{
		StreamPushEnabled: &on,
	}); err != nil {
		t.Fatal(err)
	}

	if !a.cfg.Player.StreamPushEnabled {
		t.Fatal("expected StreamPushEnabled true after patch")
	}
	if a.cfg.Player.ForceStreamPush {
		t.Fatal("enabling HLS stream push must not turn on forceStreamPush")
	}
}

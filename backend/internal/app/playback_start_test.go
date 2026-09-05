package app

import (
	"context"
	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/playback"
	"curated-backend/internal/storage"
	"path/filepath"
	"testing"
)

func TestExplicitDirectPlaybackDoesNotStartForcedHLS(t *testing.T) {
	ctx := context.Background()
	store, err := storage.NewSQLiteStore(filepath.Join(t.TempDir(), "playback.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	movie, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{TaskID: "test", Path: "D:/fixtures/ABC-001.mp4", FileName: "ABC-001.mp4", Number: "ABC-001"})
	if err != nil {
		t.Fatal(err)
	}
	streams := playback.New(playback.Config{Enabled: true, SessionRoot: t.TempDir(), FFmpegCommand: "missing-ffmpeg"})
	t.Cleanup(streams.Close)
	a := &App{store: store, streams: streams, cfg: config.Config{Player: config.PlayerConfig{ForceStreamPush: true}}}
	dto, err := a.CreatePlaybackSession(ctx, movie.MovieID, contracts.PlaybackModeDirect, 42)
	if err != nil {
		t.Fatal(err)
	}
	if dto.Mode != contracts.PlaybackModeDirect || dto.SessionID != "" || dto.URL != "/api/library/movies/"+movie.MovieID+"/stream" {
		t.Fatalf("not direct: %+v", dto)
	}
	if dto.ResumePositionSec != 42 {
		t.Fatalf("lost target: %+v", dto)
	}
	if len(streams.ListSessionSnapshots(10)) != 0 {
		t.Fatal("direct request started ffmpeg")
	}
}

func TestPlaybackStartOverridesSavedPositionBeforeStartup(t *testing.T) {
	position := 1200.0
	if got := normalizePlaybackStart(&position, 600, 7200); got != 1200 {
		t.Fatalf("got %v", got)
	}
	position = 0
	if got := normalizePlaybackStart(&position, 600, 7200); got != 0 {
		t.Fatalf("got %v", got)
	}
	if got := normalizePlaybackStart(nil, 7190, 7200); got != 0 {
		t.Fatalf("near-end got %v", got)
	}
}

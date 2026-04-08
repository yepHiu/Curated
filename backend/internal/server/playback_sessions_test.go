package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
)

func TestHandleGetRecentPlaybackSessions_OK(t *testing.T) {
	t.Parallel()

	h := NewHandler(Deps{
		Cfg:    config.Config{},
		Logger: zap.NewNop(),
		PlaybackResolver: stubPlaybackResolver{
			sessions: []contracts.PlaybackSessionStatusDTO{
				{SessionID: "sess-2", MovieID: "movie-2", SessionKind: "transcode-hls"},
				{SessionID: "sess-1", MovieID: "movie-1", SessionKind: "remux-hls"},
			},
		},
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/api/playback/sessions/recent?limit=2")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var body contracts.PlaybackSessionListDTO
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 2 {
		t.Fatalf("recent session count = %d, want 2", len(body.Items))
	}
	if body.Items[0].SessionID != "sess-2" {
		t.Fatalf("first session = %q, want sess-2", body.Items[0].SessionID)
	}
}

func TestHandleGetPlaybackSessionStatus_OK(t *testing.T) {
	t.Parallel()

	h := NewHandler(Deps{
		Cfg:    config.Config{},
		Logger: zap.NewNop(),
		PlaybackResolver: stubPlaybackResolver{
			sessionByID: contracts.PlaybackSessionStatusDTO{
				SessionID:        "sess-1",
				MovieID:          "movie-1",
				SessionKind:      "remux-hls",
				TranscodeProfile: "remux_copy",
			},
		},
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/api/playback/sessions/sess-1")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var body contracts.PlaybackSessionStatusDTO
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.SessionID != "sess-1" {
		t.Fatalf("sessionId = %q, want sess-1", body.SessionID)
	}
	if body.TranscodeProfile != "remux_copy" {
		t.Fatalf("transcodeProfile = %q, want remux_copy", body.TranscodeProfile)
	}
}

func TestHandleGetPlaybackSessionStatus_NotFound(t *testing.T) {
	t.Parallel()

	h := NewHandler(Deps{
		Cfg:              config.Config{},
		Logger:           zap.NewNop(),
		PlaybackResolver: stubPlaybackResolver{},
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/api/playback/sessions/missing")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func (s stubPlaybackResolver) GetPlaybackSession(ctx context.Context, sessionID string) (contracts.PlaybackSessionStatusDTO, error) {
	_ = ctx
	_ = sessionID
	if s.sessionByID.SessionID == "" {
		return contracts.PlaybackSessionStatusDTO{}, http.ErrMissingFile
	}
	return s.sessionByID, nil
}

func (s stubPlaybackResolver) ListRecentPlaybackSessions(ctx context.Context, limit int) (contracts.PlaybackSessionListDTO, error) {
	_ = ctx
	_ = limit
	return contracts.PlaybackSessionListDTO{Items: s.sessions}, nil
}

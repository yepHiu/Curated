package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"curated-backend/internal/storage"
)

func TestHandlePostCuratedFrameAllowsNearbyDuplicate(t *testing.T) {
	t.Parallel()
	store, srv := newCuratedFramesP1Server(t)
	movieID := addMovieForCuratedFramesP1Test(t, store, "CFP2-DUP")

	if err := store.InsertCuratedFrame(context.Background(), storage.CuratedFrameMeta{
		ID: "existing-frame", MovieID: movieID, Title: "Existing Frame", Code: "CFP2-DUP",
		Actors: []string{"Mina"}, PositionSec: 42.0, CapturedAt: "2026-04-11T10:00:00Z", Tags: []string{"favorite"},
	}, makeTestPNG(t, 32, 18)); err != nil {
		t.Fatal(err)
	}

	body, err := json.Marshal(map[string]any{
		"id":          "duplicate-frame",
		"movieId":     movieID,
		"title":       "Duplicate Frame",
		"code":        "CFP2-DUP",
		"actors":      []string{"Mina"},
		"positionSec": 44.9,
		"capturedAt":  "2026-04-11T10:00:01Z",
		"tags":        []string{},
		"imageBase64": "cG5n",
	})
	if err != nil {
		t.Fatal(err)
	}

	resp, err := http.Post(srv.URL+"/api/curated-frames", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("duplicate status = %d, want 204", resp.StatusCode)
	}

	page, err := store.QueryCuratedFrames(context.Background(), storage.CuratedFrameQuery{MovieID: movieID, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 {
		t.Fatalf("curated frame total = %d, want 2", page.Total)
	}
}

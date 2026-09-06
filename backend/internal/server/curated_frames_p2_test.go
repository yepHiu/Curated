package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
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
		"imageBase64": base64.StdEncoding.EncodeToString(makeTestPNG(t, 32, 18)),
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

func TestCuratedCaptureReplayValidationAndCache(t *testing.T) {
	t.Parallel()
	store, srv := newCuratedFramesP1Server(t)
	movieID := addMovieForCuratedFramesP1Test(t, store, "CF-REPLAY")
	payload := map[string]any{
		"id": "replay-frame", "movieId": movieID, "title": "Frame", "code": "CF-REPLAY",
		"actors": []string{}, "positionSec": 12.5, "capturedAt": "2026-09-06T10:00:00Z",
		"tags": []string{}, "imageBase64": base64.StdEncoding.EncodeToString(makeTestPNG(t, 32, 18)),
	}
	post := func(want int) {
		t.Helper()
		body, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := http.Post(srv.URL+"/api/curated-frames", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != want {
			t.Fatalf("post status = %d, want %d", resp.StatusCode, want)
		}
	}
	post(http.StatusNoContent)
	post(http.StatusNoContent)
	payload["positionSec"] = 13
	post(http.StatusConflict)
	payload["id"] = "invalid-frame"
	payload["imageBase64"] = base64.StdEncoding.EncodeToString([]byte("invalid image"))
	post(http.StatusBadRequest)
	for _, kind := range []string{"image", "thumbnail"} {
		url := srv.URL + "/api/curated-frames/replay-frame/" + kind
		resp, err := http.Get(url)
		if err != nil {
			t.Fatal(err)
		}
		etag := resp.Header.Get("ETag")
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK || etag == "" {
			t.Fatalf("missing image/cache: %d %q", resp.StatusCode, etag)
		}
		for _, condition := range []string{etag, "W/" + etag, "*"} {
			req, _ := http.NewRequest(http.MethodGet, url, nil)
			req.Header.Set("If-None-Match", condition)
			cached, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			body, err := io.ReadAll(cached.Body)
			cached.Body.Close()
			if err != nil || cached.StatusCode != http.StatusNotModified || len(body) != 0 {
				t.Fatalf("cache response: status=%d body=%d err=%v", cached.StatusCode, len(body), err)
			}
		}
	}
}

package storage

import (
	"bytes"
	"context"
	"testing"
)

func TestCuratedStablePagesMotionAndThumbnailFallback(t *testing.T) {
	s := newMigratedTestStore(t)
	ctx := context.Background()
	for _, id := range []string{"a", "c", "b"} {
		insertCuratedFrameForP1Test(t, s, CuratedFrameMeta{ID: id})
	}
	if err := s.UpsertCuratedFrameMotion(ctx, CuratedFrameMotionMeta{FrameID: "b", Status: "ready", ContentType: "image/gif"}); err != nil {
		t.Fatal(err)
	}
	for offset, id := range []string{"c", "b", "a"} {
		page, err := s.QueryCuratedFrames(ctx, CuratedFrameQuery{Limit: 1, Offset: offset})
		if err != nil {
			t.Fatal(err)
		}
		if len(page.Items) != 1 || page.Items[0].ID != id {
			t.Fatalf("page %d: %+v", offset, page)
		}
		if (page.Items[0].Motion != nil) != (id == "b") {
			t.Fatalf("motion: %+v", page.Items[0])
		}
	}
	original, err := s.GetCuratedFrameThumbnail(ctx, "a")
	if err != nil || !bytes.Equal(original, []byte("png-a")) {
		t.Fatalf("fallback: %q %v", original, err)
	}
	if _, err := s.db.Exec(`UPDATE curated_frames SET thumb_blob=? WHERE id='a'`, []byte("small")); err != nil {
		t.Fatal(err)
	}
	thumb, err := s.GetCuratedFrameThumbnail(ctx, "a")
	if err != nil || !bytes.Equal(thumb, []byte("small")) {
		t.Fatalf("thumbnail: %q %v", thumb, err)
	}
}

func TestCaptureReplayRequiresSameImmutableData(t *testing.T) {
	s := newMigratedTestStore(t)
	ctx := context.Background()
	meta := CuratedFrameMeta{ID: "retry", MovieID: "movie-retry", PositionSec: 12, CapturedAt: "2026-09-06T00:00:00Z"}
	insertCuratedFrameForP1Test(t, s, meta)
	for _, tc := range []struct {
		meta CuratedFrameMeta
		blob []byte
		want bool
	}{
		{meta, []byte("png-retry"), true},
		{meta, []byte("different"), false},
		{CuratedFrameMeta{ID: meta.ID, MovieID: "other", PositionSec: 12, CapturedAt: meta.CapturedAt}, []byte("png-retry"), false},
	} {
		got, err := s.MatchesCuratedFrameCapture(ctx, tc.meta, tc.blob)
		if err != nil || got != tc.want {
			t.Fatalf("got %v, %v; want %v", got, err, tc.want)
		}
	}
}

func TestCuratedCursorSurvivesNewerInsert(t *testing.T) {
	s := newMigratedTestStore(t)
	ctx := context.Background()
	for _, id := range []string{"a", "b", "c"} {
		insertCuratedFrameForP1Test(t, s, CuratedFrameMeta{ID: id})
	}
	first, err := s.QueryCuratedFrames(ctx, CuratedFrameQuery{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	insertCuratedFrameForP1Test(t, s, CuratedFrameMeta{ID: "d"})
	next, err := s.QueryCuratedFrames(ctx, CuratedFrameQuery{Limit: 2, Cursor: first.NextCursor, SkipTotal: true})
	if err != nil {
		t.Fatal(err)
	}
	if next.Total != -1 || len(next.Items) != 2 || next.Items[0].ID != "b" || next.Items[1].ID != "a" {
		t.Fatalf("unexpected cursor page: %+v", next)
	}
	if _, _, err := DecodeCuratedFrameCursor("invalid"); err == nil {
		t.Fatal("invalid cursor accepted")
	}
}

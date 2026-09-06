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

package storage

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestListCuratedFramesForExport_orderAndMissing(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(root, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	id1 := "11111111-1111-1111-1111-111111111111"
	id2 := "22222222-2222-2222-2222-222222222222"
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a} // minimal invalid but insert only needs bytes
	if err := store.InsertCuratedFrame(ctx, CuratedFrameMeta{
		ID: id1, MovieID: "m", Title: "T", Code: "C-1", Actors: []string{"A"},
		PositionSec: 1, CapturedAt: "2020-01-01T00:00:00Z", Tags: nil,
	}, png); err != nil {
		t.Fatal(err)
	}
	if err := store.InsertCuratedFrame(ctx, CuratedFrameMeta{
		ID: id2, MovieID: "m", Title: "T2", Code: "C-2", Actors: []string{"B"},
		PositionSec: 2, CapturedAt: "2020-01-02T00:00:00Z", Tags: nil,
	}, png); err != nil {
		t.Fatal(err)
	}

	got, err := store.ListCuratedFramesForExport(ctx, []string{id2, id1})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != id2 || got[1].ID != id1 {
		t.Fatalf("order: %+v", got)
	}

	_, err = store.ListCuratedFramesForExport(ctx, []string{id1, "missing"})
	if err == nil || !errors.Is(err, ErrCuratedFrameNotFound) {
		t.Fatalf("expected ErrCuratedFrameNotFound, got %v", err)
	}
}

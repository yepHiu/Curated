package storage

import (
	"context"
	"curated-backend/internal/contracts"
	"database/sql"
	"errors"
	"path/filepath"
	"sync"
	"testing"
)

// TestWishlistReconcileIdentityChanges 验证自动关联随番号变化撤销，人工选择保持。
func TestWishlistReconcileIdentityChanges(t *testing.T) {
	s := newSavedViewTestStore(t)
	ctx := context.Background()
	id, _, err := s.AddWishlist(ctx, "SSIS-001")
	if err != nil {
		t.Fatal(err)
	}
	movie, err := s.PersistScanMovie(ctx, contracts.ScanFileResultDTO{Number: "SSIS-001", Path: filepath.Join(t.TempDir(), "movie.mp4")})
	if err != nil {
		t.Fatal(err)
	}
	// assertLinks 同时检查详情的有效关联和默认墙状态。
	assertLinks := func(count int) {
		t.Helper()
		if err := s.ReconcileWishlist(ctx); err != nil {
			t.Fatal(err)
		}
		item, err := s.GetWishlist(ctx, id)
		if err != nil || len(item.MovieIDs) != count {
			t.Fatalf("links: %+v %v", item, err)
		}
	}
	assertLinks(1)
	if _, err = s.db.ExecContext(ctx, `UPDATE movies SET code='SSIS-002' WHERE id=?`, movie.MovieID); err != nil {
		t.Fatal(err)
	}
	assertLinks(0)
	if err = s.SetWishlistLink(ctx, id, movie.MovieID, false); err != nil {
		t.Fatal(err)
	}
	assertLinks(1)
	if err = s.SetWishlistLink(ctx, id, movie.MovieID, true); err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.ExecContext(ctx, `UPDATE movies SET code='SSIS-001' WHERE id=?`, movie.MovieID); err != nil {
		t.Fatal(err)
	}
	assertLinks(0)
}

// TestWishlistDurabilityAndGeneration 验证并发幂等、重开数据库与迟到任务隔离。
func TestWishlistDurabilityAndGeneration(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "wishlist.db")
	s, e := NewSQLiteStore(path)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Migrate(ctx); e != nil {
		t.Fatal(e)
	}
	var wg sync.WaitGroup
	ids := make(chan string, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { // 并发模拟不同站点重复提交同一作品。
			defer wg.Done()
			id, _, err := s.AddWishlist(ctx, "ssis_001")
			if err != nil {
				t.Error(err)
			}
			ids <- id
		}()
	}
	wg.Wait()
	close(ids)
	var id string
	for v := range ids {
		if id != "" && id != v {
			t.Fatal("duplicate identity")
		}
		id = v
	}
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
	s, e = NewSQLiteStore(path)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	item, e := s.GetWishlist(ctx, id)
	if e != nil || item.Code != "SSIS-001" {
		t.Fatalf("reopen: %+v %v", item, e)
	}
	job, attempt, e := s.ClaimWishlistJob(ctx)
	if e != nil || attempt != 1 || job.ID != id {
		t.Fatalf("job %+v %v", job, e)
	}
	if _, _, e = s.ClaimWishlistJob(ctx); !errors.Is(e, sql.ErrNoRows) {
		t.Fatalf("lease not exclusive %v", e)
	}
	code := "SSIS-002"
	if e = s.PatchWishlist(ctx, id, contracts.WishlistPatch{Version: item.Version, Code: &code}); e != nil {
		t.Fatal(e)
	}
	if e = s.SaveWishlistMetadata(ctx, id, job.Generation, contracts.WishlistMetadata{Title: "stale"}); !errors.Is(e, sql.ErrNoRows) {
		t.Fatalf("stale metadata applied %v", e)
	}
	if e = s.FinishWishlistJob(ctx, id, job.Generation, attempt, "ready", "", false); e != nil {
		t.Fatal(e)
	}
	next, _, e := s.ClaimWishlistJob(ctx)
	if e != nil || next.Generation != 2 {
		t.Fatalf("new generation lost %+v %v", next, e)
	}
	a := WishlistAssetFile{ID: "asset", ItemID: id, Generation: 2, Role: "cover", Path: "image.jpg", SHA256: "hash"}
	if e = s.SaveWishlistAsset(ctx, a); e != nil {
		t.Fatal(e)
	}
	item, e = s.GetWishlist(ctx, id)
	if e != nil || len(item.Assets) != 1 {
		t.Fatalf("hydrate assets %+v %v", item, e)
	}
	if e = s.DeleteWishlist(ctx, id); e != nil {
		t.Fatal(e)
	}
	if e = s.SaveWishlistAsset(ctx, a); !errors.Is(e, sql.ErrNoRows) {
		t.Fatalf("deleted asset resurrected %v", e)
	}
}

// TestWishlistCompletionTokensAndPaging 验证用户意愿、游标及撤销权限。
func TestWishlistCompletionTokensAndPaging(t *testing.T) {
	s := newSavedViewTestStore(t)
	ctx := context.Background()
	id, _, e := s.AddWishlist(ctx, "SSIS001")
	if e != nil {
		t.Fatal(e)
	}
	done := true
	if e = s.PatchWishlist(ctx, id, contracts.WishlistPatch{Version: 1, Completed: &done}); e != nil {
		t.Fatal(e)
	}
	same, created, e := s.AddWishlist(ctx, "SSIS-001")
	if e != nil || created || same != id {
		t.Fatal("repeat changed identity")
	}
	item, e := s.GetWishlist(ctx, id)
	if e != nil || item.Status != "completed" {
		t.Fatal("repeat restored completed item")
	}
	if e = s.PatchWishlist(ctx, id, contracts.WishlistPatch{Version: 1, Completed: &done}); !errors.Is(e, ErrWishlistConflict) {
		t.Fatal("stale version allowed")
	}
	if _, _, e = s.AddWishlist(ctx, "SSIS-002"); e != nil {
		t.Fatal(e)
	}
	page, e := s.ListWishlist(ctx, "all", "", "", 1)
	if e != nil || page.NextCursor == "" || page.PendingCount != 1 {
		t.Fatalf("page %+v %v", page, e)
	}
	next, e := s.ListWishlist(ctx, "all", "", page.NextCursor, 1)
	if e != nil || next.Items[0].ID == page.Items[0].ID {
		t.Fatal("cursor repeated")
	}
	if _, e = s.ListWishlist(ctx, "pending", "", page.NextCursor, 1); e == nil {
		t.Fatal("cursor accepted wrong filter")
	}
}

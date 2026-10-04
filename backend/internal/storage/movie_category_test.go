package storage

import (
	"context"
	"path/filepath"
	"testing"

	"curated-backend/internal/contracts"
)

// TestMovieCategoryPagination 验证分类先于分页及计数，并保留共享全库查询。
func TestMovieCategoryPagination(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "category.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	for _, code := range []string{"ABC-001", "FC2-1234567", "FC2 PPV 2345678"} {
		if _, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{Number: code, Path: filepath.Join(t.TempDir(), code+".mp4")}); err != nil {
			t.Fatal(err)
		}
	}
	for mode, total := range map[string]int{"library": 1, "fc2": 2, "": 3} {
		page, err := store.ListMovies(ctx, contracts.ListMoviesRequest{Mode: mode, Limit: 1})
		if err != nil {
			t.Fatal(err)
		}
		if page.Total != total || len(page.Items) != 1 {
			t.Fatalf("mode %s: %+v", mode, page)
		}
	}
	filters, err := NormalizeSavedViewFilters(contracts.SavedViewFiltersV1{SchemaVersion: 1, Mode: "fc2"})
	if err != nil || filters.Mode != "fc2" {
		t.Fatalf("FC2 saved view: %+v %v", filters, err)
	}
}

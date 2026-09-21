package metatube

import (
	"github.com/metatube-community/metatube-sdk-go/model"
	"testing"
)

// TestWishlistExactCandidates 验证相似首项不能误选、同源歧义不能静默挑一条。
func TestWishlistExactCandidates(t *testing.T) {
	results := []*model.MovieSearchResult{{ID: "one", Provider: "JavDB", Homepage: "https://example.com", Number: "SSIS-001-CD1", Title: "variant"}, {ID: "two", Provider: "JavDB", Homepage: "https://example.com", Number: "SSIS-001", Title: "exact"}}
	hit, e := exactWishlistResult("ssis001", results)
	if e != nil || hit.ID != "two" {
		t.Fatalf("exact %+v %v", hit, e)
	}
	results = append(results, &model.MovieSearchResult{ID: "three", Provider: "JavDB", Homepage: "https://example.com", Number: "SSIS001", Title: "other"})
	if _, e = exactWishlistResult("SSIS-001", results); e == nil {
		t.Fatal("ambiguous accepted")
	}
	if _, e = exactWishlistResult("SSIS-009", results); e == nil {
		t.Fatal("unmatched accepted")
	}
}

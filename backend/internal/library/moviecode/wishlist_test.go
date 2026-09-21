package moviecode

import "testing"

// TestWishlistIdentity 验证番号别名和分卷边界，不改变库中 ID 语义。
func TestWishlistIdentity(t *testing.T) {
	for _, pair := range [][2]string{{"ssis_001", "SSIS001"}, {"FC2-PPV-123456", "FC2123456"}} {
		_, a, e := WishlistIdentity(pair[0])
		_, b, e2 := WishlistIdentity(pair[1])
		if e != nil || e2 != nil || a != b {
			t.Fatalf("identity %v", pair)
		}
	}
	_, a, _ := WishlistIdentity("SSIS-001")
	_, b, _ := WishlistIdentity("SSIS-001-CD1")
	if a == b {
		t.Fatal("volume collapsed")
	}
	for _, s := range []string{"", "../movie", "https://site/123", "SSIS", "A B123"} {
		if _, _, e := WishlistIdentity(s); e == nil {
			t.Fatalf("accepted %s", s)
		}
	}
}

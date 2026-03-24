package moviecode

import "testing"

func TestNormalizeForStorageID(t *testing.T) {
	t.Parallel()
	if got, want := NormalizeForStorageID("  ABC_123 "), "abc-123"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if got, want := NormalizeForStorageID("EBWH-287"), "ebwh-287"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

//go:build !release

package config

import "testing"

func TestDefaultHTTPAddr(t *testing.T) {
	if got := DefaultHTTPAddr(); got != ":8080" {
		t.Fatalf("dev build DefaultHTTPAddr: got %q want %q", got, ":8080")
	}
}

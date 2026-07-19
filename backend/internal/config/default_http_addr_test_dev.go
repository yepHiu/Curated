//go:build !release

package config

import "testing"

func TestDefaultHTTPAddr(t *testing.T) {
	if got := DefaultHTTPAddr(); got != "127.0.0.1:8080" {
		t.Fatalf("dev build DefaultHTTPAddr: got %q want %q", got, "127.0.0.1:8080")
	}
}

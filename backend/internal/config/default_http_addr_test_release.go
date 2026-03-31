//go:build release

package config

import "testing"

func TestDefaultHTTPAddr(t *testing.T) {
	if got := DefaultHTTPAddr(); got != ":8081" {
		t.Fatalf("release build DefaultHTTPAddr: got %q want %q", got, ":8081")
	}
}

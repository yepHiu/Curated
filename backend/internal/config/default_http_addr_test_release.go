//go:build release

package config

import "testing"

func TestDefaultHTTPAddr(t *testing.T) {
	if got := DefaultHTTPAddr(); got != "127.0.0.1:8081" {
		t.Fatalf("release build DefaultHTTPAddr: got %q want %q", got, "127.0.0.1:8081")
	}
}

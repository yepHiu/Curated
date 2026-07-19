package config

import (
	"strings"
	"testing"
)

func TestHTTPAddrIsLoopback(t *testing.T) {
	t.Parallel()

	tests := []struct {
		addr string
		want bool
	}{
		{addr: "127.0.0.1:8080", want: true},
		{addr: "localhost:8080", want: true},
		{addr: "[::1]:8080", want: true},
		{addr: ":8080", want: false},
		{addr: "0.0.0.0:8080", want: false},
		{addr: "[::]:8080", want: false},
		{addr: "192.168.1.10:8080", want: false},
		{addr: "curated.local:8080", want: false},
		{addr: "invalid", want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.addr, func(t *testing.T) {
			t.Parallel()
			if got := HTTPAddrIsLoopback(tt.addr); got != tt.want {
				t.Fatalf("HTTPAddrIsLoopback(%q) = %v, want %v", tt.addr, got, tt.want)
			}
		})
	}
}

func TestValidateHTTPExposure(t *testing.T) {
	t.Parallel()

	if err := (Config{HttpAddr: "127.0.0.1:8080"}).ValidateHTTPExposure(false); err != nil {
		t.Fatalf("loopback listener should not require LAN opt-in or PIN: %v", err)
	}

	err := (Config{HttpAddr: ":8080"}).ValidateHTTPExposure(true)
	if err == nil || !strings.Contains(err.Error(), "lanEnabled=true") {
		t.Fatalf("wildcard without LAN opt-in error = %v", err)
	}

	err = (Config{HttpAddr: ":8080", LANEnabled: true}).ValidateHTTPExposure(false)
	if err == nil || !strings.Contains(err.Error(), "configure an application PIN") {
		t.Fatalf("LAN without PIN error = %v", err)
	}

	if err := (Config{HttpAddr: ":8080", LANEnabled: true}).ValidateHTTPExposure(true); err != nil {
		t.Fatalf("explicit LAN listener with PIN should be valid: %v", err)
	}
}

package config

import (
	"net"
	"strings"
	"testing"
)

// TestHTTPAddrIsLoopback 覆盖 loopback、wildcard 与显式主机判定。
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

// TestValidateHTTPExposure 确认非 loopback 必须显式 lanEnabled，但不要求 PIN。
func TestValidateHTTPExposure(t *testing.T) {
	t.Parallel()

	if err := (Config{HttpAddr: "127.0.0.1:8080"}).ValidateHTTPExposure(); err != nil {
		t.Fatalf("loopback listener should not require LAN opt-in: %v", err)
	}

	err := (Config{HttpAddr: ":8080"}).ValidateHTTPExposure()
	if err == nil || !strings.Contains(err.Error(), "lanEnabled=true") {
		t.Fatalf("wildcard without LAN opt-in error = %v", err)
	}

	if err := (Config{HttpAddr: ":8080", LANEnabled: true}).ValidateHTTPExposure(); err != nil {
		t.Fatalf("explicit LAN listener without PIN should be valid: %v", err)
	}
}

func TestResolveListenAddr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  Config
		want string
	}{
		{name: "lan off forces loopback", cfg: Config{HttpAddr: "0.0.0.0:8081", LANEnabled: false}, want: "127.0.0.1:8081"},
		{name: "lan on promotes loopback", cfg: Config{HttpAddr: "127.0.0.1:8080", LANEnabled: true}, want: "0.0.0.0:8080"},
		{name: "lan on promotes wildcard", cfg: Config{HttpAddr: ":8081", LANEnabled: true}, want: "0.0.0.0:8081"},
		{name: "lan on keeps explicit host", cfg: Config{HttpAddr: "192.168.1.8:8081", LANEnabled: true}, want: "192.168.1.8:8081"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.cfg.ResolveListenAddr(); got != tt.want {
				t.Fatalf("ResolveListenAddr() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLANAccessURLsFiltersPrivateIPv4(t *testing.T) {
	t.Parallel()

	prev := interfaceAddrs
	t.Cleanup(func() { interfaceAddrs = prev })
	interfaceAddrs = func() ([]net.Addr, error) {
		return []net.Addr{
			&net.IPNet{IP: net.ParseIP("127.0.0.1"), Mask: net.CIDRMask(8, 32)},
			&net.IPNet{IP: net.ParseIP("192.168.1.8"), Mask: net.CIDRMask(24, 32)},
			&net.IPNet{IP: net.ParseIP("10.0.0.5"), Mask: net.CIDRMask(8, 32)},
			&net.IPNet{IP: net.ParseIP("8.8.8.8"), Mask: net.CIDRMask(32, 32)},
			&net.IPNet{IP: net.ParseIP("169.254.1.1"), Mask: net.CIDRMask(16, 32)},
		}, nil
	}

	got := LANAccessURLs("127.0.0.1:8081")
	if len(got) != 2 || got[0] != "http://10.0.0.5:8081" || got[1] != "http://192.168.1.8:8081" {
		t.Fatalf("LANAccessURLs = %#v", got)
	}
}

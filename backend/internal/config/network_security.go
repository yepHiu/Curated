package config

import (
	"fmt"
	"net"
	"strings"
)

// HTTPAddrIsLoopback reports whether addr binds only to the local machine.
// Empty hosts such as :8080 are wildcard listeners and therefore are not loopback-only.
func HTTPAddrIsLoopback(addr string) bool {
	host, _, err := net.SplitHostPort(strings.TrimSpace(addr))
	if err != nil {
		return false
	}
	host = strings.Trim(strings.TrimSpace(host), "[]")
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// ValidateHTTPExposure enforces explicit LAN opt-in and a configured PIN for every
// listener that can accept requests from outside the local machine.
func (c Config) ValidateHTTPExposure(pinEnabled bool) error {
	addr := strings.TrimSpace(c.HttpAddr)
	if HTTPAddrIsLoopback(addr) {
		return nil
	}
	if !c.LANEnabled {
		return fmt.Errorf("refuse non-loopback httpAddr %q: set lanEnabled=true to opt in to LAN access", addr)
	}
	if !pinEnabled {
		return fmt.Errorf("refuse LAN access on httpAddr %q: configure an application PIN before enabling LAN mode", addr)
	}
	return nil
}

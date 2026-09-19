package config

import (
	"fmt"
	"net"
	"sort"
	"strings"
)

var interfaceAddrs = net.InterfaceAddrs

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

// ResolveListenAddr returns the TCP address Curated should bind for the current LAN preference.
// When LAN is off, the host is forced to 127.0.0.1 while keeping the configured port.
// When LAN is on, loopback or unspecified hosts become 0.0.0.0; an explicit non-loopback host is kept.
func (c Config) ResolveListenAddr() string {
	addr := strings.TrimSpace(c.HttpAddr)
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	if !c.LANEnabled {
		return net.JoinHostPort("127.0.0.1", port)
	}
	host = strings.Trim(strings.TrimSpace(host), "[]")
	if host == "" || strings.EqualFold(host, "localhost") {
		return net.JoinHostPort("0.0.0.0", port)
	}
	ip := net.ParseIP(host)
	if ip != nil && (ip.IsLoopback() || ip.IsUnspecified()) {
		return net.JoinHostPort("0.0.0.0", port)
	}
	return addr
}

// LANAccessURLs lists http://<private-ipv4>:<port> candidates for the current listen port.
// Link-local and loopback addresses are omitted because they are not useful LAN entry URLs.
func LANAccessURLs(httpAddr string) []string {
	_, port, err := net.SplitHostPort(strings.TrimSpace(httpAddr))
	if err != nil || strings.TrimSpace(port) == "" {
		return nil
	}
	addrs, err := interfaceAddrs()
	if err != nil {
		return nil
	}
	seen := map[string]struct{}{}
	urls := make([]string, 0, 4)
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet == nil {
			continue
		}
		ip := ipNet.IP.To4()
		if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() || ip.IsMulticast() {
			continue
		}
		if !ip.IsPrivate() {
			continue
		}
		host := ip.String()
		if _, exists := seen[host]; exists {
			continue
		}
		seen[host] = struct{}{}
		urls = append(urls, "http://"+net.JoinHostPort(host, port))
	}
	sort.Strings(urls)
	return urls
}

// ValidateHTTPExposure enforces explicit LAN opt-in for every listener that
// can accept requests from outside the local machine. PIN lock is independent.
func (c Config) ValidateHTTPExposure() error {
	addr := strings.TrimSpace(c.HttpAddr)
	if HTTPAddrIsLoopback(addr) {
		return nil
	}
	if !c.LANEnabled {
		return fmt.Errorf("refuse non-loopback httpAddr %q: set lanEnabled=true to opt in to LAN access", addr)
	}
	return nil
}
